package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	apperr "github.com/meowhome/backend/internal/platform/errors"
	"github.com/meowhome/backend/internal/platform/id"
	"go.uber.org/zap"
)

func (s *AgentService) Chat(ctx context.Context, familyID, userID string, in AgentChatRequest) (*AgentChatResponse, error) {
	if err := s.ensureEnabled(familyID); err != nil {
		return nil, err
	}
	if err := s.access(ctx, familyID, userID); err != nil {
		return nil, err
	}
	in.Message = strings.TrimSpace(in.Message)
	if in.Message == "" || len([]rune(in.Message)) > AgentChatMessageMaxLen || strings.TrimSpace(in.ClientMessageID) == "" || len(in.ClientMessageID) > 128 {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "invalid chat input")
	}
	now := s.clock().UTC()
	gen := id.ULIDGenerator{}
	session := &model.AgentSession{Base: model.Base{ID: gen.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: userID, Title: toolExcerpt(in.Message, 40), Status: "active"}
	user := &model.AgentMessage{Base: model.Base{ID: gen.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: userID, Role: string(RoleUser), Visibility: string(VisibilityPrivate), Type: AgentTypeChatAnswer, Severity: "info", Title: "用户消息", Body: in.Message, GeneratedAt: now, Model: agentChatPromptVersion, ClientMessageID: &in.ClientMessageID}
	token := gen.New()
	claimed, err := s.repo.ClaimChatTurn(ctx, session, user, in.SessionID, token, now, now.Add(35*time.Second))
	if err == repository.ErrConflict {
		return nil, apperr.Conflict(CodeAgentConflict, "session is busy or client_message_id has conflicting content")
	}
	if err == repository.ErrNotFound {
		return nil, apperr.NotFound(apperr.CodeNotFound, "agent session not found")
	}
	if err != nil {
		return nil, err
	}
	if !claimed {
		answer, e := s.repo.FindMessageByClientID(ctx, familyID, userID, session.ID, in.ClientMessageID)
		if e == nil {
			return chatResponse(session.ID, user.ID, in.ClientMessageID, answer), nil
		}
		if e != repository.ErrNotFound {
			return nil, e
		}
		return &AgentChatResponse{SessionID: session.ID, TurnID: user.ID, ClientMessageID: in.ClientMessageID, Status: "running"}, nil
	}
	started := time.Now()
	turnCtx, turnCancel := context.WithTimeout(ctx, 25*time.Second)
	defer turnCancel()
	// Cancellation interrupts the model, but final status is written with a short
	// independent context so a disconnected client can recover the saved answer.
	finishCtx := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	}
	history, historyErr := s.chatHistory(turnCtx, familyID, session.ID, user.ID)
	var result *agentTurnResult
	runErr := historyErr
	if runErr == nil {
		history = append(history, LLMMessage{Role: "user", Content: user.Body})
		result, runErr = s.runAgentTurnRecorded(turnCtx, AgentToolScope{FamilyID: familyID, UserID: userID, SessionID: session.ID, TurnID: user.ID}, history, func(tool agentTurnToolResult) error {
			stamp := s.clock().UTC()
			message := &model.AgentMessage{Base: model.Base{ID: gen.New(), CreatedBy: "system", CreatedAt: stamp, UpdatedAt: stamp}, FamilyID: familyID, SessionID: session.ID, UserID: userID, TurnID: user.ID, Role: string(RoleTool), Visibility: string(VisibilityPrivate), Type: AgentTypeChatAnswer, Severity: "info", Title: "工具结果", Body: tool.Content, GeneratedAt: stamp, Model: agentChatPromptVersion, ToolCallID: tool.Call.ID, ToolName: tool.Call.Name}
			return s.repo.CreateMessage(turnCtx, message)
		})
	}
	if historyErr != nil {
		user.RunStatus, user.RunError, user.DurationMS = "failed", "history_unavailable", time.Since(started).Milliseconds()
		writeCtx, cancel := finishCtx()
		defer cancel()
		if err := s.repo.FinishChatTurn(writeCtx, user, token, nil); err != nil {
			return nil, err
		}
		return nil, apperr.New(apperr.TypeExternal, CodeAgentUnavailable, "chat history unavailable; retry the same message")
	}
	if result == nil {
		result = &agentTurnResult{}
	}
	body := result.Answer
	degraded := runErr != nil
	if degraded {
		body = agentFallback(result)
	}
	stamp := s.clock().UTC()
	answer := &model.AgentMessage{Base: model.Base{ID: gen.New(), CreatedBy: "system", CreatedAt: stamp, UpdatedAt: stamp}, FamilyID: familyID, SessionID: session.ID, UserID: userID, TurnID: user.ID, Role: string(RoleAssistant), Visibility: string(VisibilityPrivate), Type: AgentTypeChatAnswer, Severity: "info", Title: "猫管家", Body: body, GeneratedAt: stamp, Model: result.Model, ClientMessageID: &in.ClientMessageID, RunStatus: "completed", Degraded: degraded, PromptTokens: result.Usage.PromptTokens, CompletionTokens: result.Usage.CompletionTokens, TotalTokens: result.Usage.TotalTokens, DurationMS: time.Since(started).Milliseconds(), Disclaimer: "仅根据已记录数据回答，不构成诊断或用药建议。", Evidence: encodeJSON(result.Evidence), ActionSuggestions: encodeJSON([]AgentAction{{Type: "view_records", Title: "查看记录", Days: 7}, {Type: "view_trend", Title: "查看趋势", Days: 7}})}
	if answer.Model == "" {
		answer.Model = agentChatPromptVersion
	}
	if degraded {
		answer.RunError = agentRunErrorCode(runErr)
		answer.Disclaimer = "模型未完成回答，以下为确定性降级信息。仅根据记录提示，不构成诊断或用药建议。"
	}
	if result.Draft != nil {
		answer.ActionSuggestions = encodeJSON([]AgentAction{{Type: "create_reminder", Title: "查看待确认草稿", Payload: map[string]any{"message_id": result.Draft.MessageID}}})
	}
	user.RunStatus, user.RunError, user.DurationMS = "completed", answer.RunError, answer.DurationMS
	writeCtx, cancel := finishCtx()
	defer cancel()
	if err := s.repo.FinishChatTurn(writeCtx, user, token, answer); err != nil {
		return nil, err
	}
	s.logger.Info("agent_chat_finished", zap.String("request_id", agentRequestID(ctx)), zap.String("family_id", familyID), zap.String("turn_id", user.ID), zap.String("message_id", answer.ID), zap.Bool("degraded", degraded), zap.String("reason", answer.RunError), zap.Int("total_tokens", answer.TotalTokens), zap.Int64("duration_ms", answer.DurationMS), zap.Int("tool_calls", len(result.ToolResults)))
	return chatResponse(session.ID, user.ID, in.ClientMessageID, answer), nil
}

func chatResponse(sessionID, turnID, clientID string, answer *model.AgentMessage) *AgentChatResponse {
	return &AgentChatResponse{SessionID: sessionID, TurnID: turnID, ClientMessageID: clientID, Status: "completed", Message: toAgentMessage(answer), Degraded: answer.Degraded || strings.Contains(answer.Disclaimer, "降级")}
}

// Only completed visible user/answer pairs enter context. Tool data is queried
// afresh; drafts and interrupted inputs cannot become fabricated history.
func (s *AgentService) chatHistory(ctx context.Context, familyID, sessionID, currentTurnID string) ([]LLMMessage, error) {
	rows, err := s.repo.ListMessages(ctx, repository.AgentMessageQuery{FamilyID: familyID, SessionID: sessionID, Visibility: string(VisibilityPrivate), ExcludeTools: true, Limit: 50})
	if err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].GeneratedAt.Equal(rows[j].GeneratedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].GeneratedAt.Before(rows[j].GeneratedAt)
	})
	users := map[string]*model.AgentMessage{}
	pairs := make([][2]LLMMessage, 0)
	for _, m := range rows {
		if m.Role == string(RoleUser) && m.Type == AgentTypeChatAnswer && m.ID != currentTurnID {
			users[m.ID] = m
		}
	}
	for _, m := range rows {
		if m.TurnID == currentTurnID || m.Type != AgentTypeChatAnswer {
			continue
		}
		if m.Role == string(RoleAssistant) && !m.Degraded {
			if u := users[m.TurnID]; u != nil {
				pairs = append(pairs, [2]LLMMessage{{Role: "user", Content: u.Body}, {Role: "assistant", Content: m.Body}})
			}
		}
	}
	if len(pairs) > 5 {
		pairs = pairs[len(pairs)-5:]
	}
	out := make([]LLMMessage, 0, len(pairs)*2)
	for _, p := range pairs {
		out = append(out, p[:]...)
	}
	return out, nil
}

func agentFallback(result *agentTurnResult) string {
	parts := []string{"模型暂时无法完成回答。你可以查看原始记录和趋势，或稍后发送新问题。"}
	seen := map[string]bool{}
	for _, tool := range result.ToolResults {
		if seen[tool.Call.Name] {
			continue
		}
		seen[tool.Call.Name] = true
		var data map[string]json.RawMessage
		if json.Unmarshal([]byte(tool.Content), &data) != nil || data["error"] != nil {
			continue
		}
		if tool.Call.Name == "listRecords" {
			var records []json.RawMessage
			if json.Unmarshal(data["records"], &records) == nil {
				parts = append(parts, fmt.Sprintf("本次查询返回 %d 条记录；未记录不代表没有发生，列表可能有截断，请查看原始记录。", len(records)))
			}
		}
		if tool.Call.Name == "getTrends" {
			var trend struct {
				Unit   string `json:"unit"`
				Points []struct {
					Date         string  `json:"date"`
					Value        float64 `json:"value"`
					Observations int     `json:"observations"`
				} `json:"points"`
				Missing []string `json:"missing_dates"`
			}
			if json.Unmarshal([]byte(tool.Content), &trend) == nil {
				points := trend.Points
				if len(points) > 7 {
					points = points[len(points)-7:]
				}
				rows := make([]string, 0, len(points))
				for _, point := range points {
					rows = append(rows, fmt.Sprintf("%s：%g %s（%d 条数值观测）", point.Date, point.Value, trend.Unit, point.Observations))
				}
				if len(rows) > 0 {
					parts = append(parts, "已查询的最近观测："+strings.Join(rows, "；"))
				}
				parts = append(parts, fmt.Sprintf("该窗口有 %d 天缺少数值观测；缺测不表示数值为零，请查看趋势页核对完整窗口。", len(trend.Missing)))
			}
		}
		if tool.Call.Name == "getCatProfile" {
			var profile struct {
				Name     string `json:"name"`
				Recorded bool   `json:"profile_recorded"`
			}
			if json.Unmarshal([]byte(tool.Content), &profile) == nil {
				state := "有健康档案记录"
				if !profile.Recorded {
					state = "未保存健康档案，不能据此判断健康状况"
				}
				parts = append(parts, fmt.Sprintf("已查询猫咪档案：%s；%s。", profile.Name, state))
			}
		}
	}
	if result.Draft != nil {
		parts = append(parts, "提醒草稿已保存，尚未创建正式提醒；请打开草稿核对并确认。")
	}
	return strings.Join(parts, "\n")
}

func agentRunErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "interrupted"
	case errors.Is(err, ErrLLMInvalidResponse):
		return "invalid_output"
	case errors.Is(err, ErrLLMUnavailable):
		return "unavailable"
	case errors.Is(err, ErrLLMRequestBudgetExhausted):
		return "request_budget"
	default:
		var appError *apperr.AppError
		if errors.As(err, &appError) {
			switch appError.Code {
			case CodeAgentConflict:
				return "conflict"
			case CodeAgentDraftExpired:
				return "draft_expired"
			case apperr.CodeNotFound:
				return "not_found"
			}
			if appError.Type == apperr.TypeProtocol {
				return "invalid_arguments"
			}
		}
		return "turn_failed"
	}
}
