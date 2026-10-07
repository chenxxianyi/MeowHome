package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

func TestChatInvokesToolsPersistsUsageAndReusesAnswer(t *testing.T) {
	s, repo := newAgentServiceForTest()
	s.records = &toolRecordRepo{rows: []*model.DailyRecord{toolRecord("record-1", "feeding", `{"amount":30}`, s.clock().Add(-time.Hour))}}
	calls := 0
	s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(ctx context.Context, req LLMRequest, _ []ToolDef) (*LLMResponse, error) {
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 25*time.Second {
			t.Fatal("missing shared deadline")
		}
		calls++
		if calls == 1 {
			return &LLMResponse{Model: "fake", Usage: LLMUsage{TotalTokens: 10}, ToolCalls: []LLMToolCall{{ID: "call", Name: "listRecords", Arguments: `{"cat_id":"cat","type":"feeding"}`}}}, nil
		}
		return &LLMResponse{Model: "fake", Usage: LLMUsage{TotalTokens: 20}, Content: `{"answer":"有一条已记录的进食记录，不能仅由次数判断进食正常。","source_ids":["record-1"]}`}, nil
	}})
	in := AgentChatRequest{Message: "小白最近吃得怎么样", ClientMessageID: "client"}
	first, err := s.Chat(context.Background(), "fam", "user", in)
	if err != nil || first.Degraded || first.Status != "completed" || len(first.Message.Evidence) != 1 {
		t.Fatalf("tool chat not connected: %+v %v", first, err)
	}
	second, err := s.Chat(context.Background(), "fam", "user", in)
	if err != nil || second.Message.ID != first.Message.ID || calls != 2 || len(repo.sessions) != 1 {
		t.Fatalf("retry called model again: %v", err)
	}
	if repo.messages[first.Message.ID].TotalTokens != 30 {
		t.Fatal("usage not persisted")
	}
	page, err := s.ListSessionMessages(context.Background(), "fam", "user", first.SessionID, "", 20)
	if err != nil || len(page.Messages) != 2 {
		t.Fatalf("tool message leaked: %+v %v", page, err)
	}
}

func TestChatDraftSurvivesGenerationFailureAndRetry(t *testing.T) {
	s, repo := newAgentServiceForTest()
	calls := 0
	s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(context.Context, LLMRequest, []ToolDef) (*LLMResponse, error) {
		calls++
		if calls == 1 {
			return &LLMResponse{ToolCalls: []LLMToolCall{{ID: "draft", Name: "createReminderDraft", Arguments: `{"cat_id":"cat","title":"剪指甲","scheduled_at":"2026-10-02T09:00:00+08:00"}`}}}, nil
		}
		return nil, ErrLLMUnavailable
	}})
	in := AgentChatRequest{Message: "给小白创建剪指甲提醒", ClientMessageID: "client"}
	first, err := s.Chat(context.Background(), "fam", "user", in)
	if err != nil || !first.Degraded || !strings.Contains(first.Message.Body, "草稿已保存") {
		t.Fatalf("lost draft: %+v %v", first, err)
	}
	second, err := s.Chat(context.Background(), "fam", "user", in)
	if err != nil || first.Message.ID != second.Message.ID || calls != 2 || len(repo.reminders) != 0 {
		t.Fatalf("retry duplicated work: %v", err)
	}
	draft := repo.messages[draftIDForTurn(first.TurnID)]
	if draft == nil || draft.Visibility != "private" || draft.ActionStatus != "pending" {
		t.Fatalf("invalid saved draft: %+v", draft)
	}
}

func TestChatRunningBusyAndInterruptedLeaseRecovery(t *testing.T) {
	s, repo := newAgentServiceForTest()
	request := AgentChatRequest{Message: "查询记录", ClientMessageID: "key"}
	client := request.ClientMessageID
	now := s.clock()
	lease := now.Add(time.Minute)
	user := &model.AgentMessage{Base: model.Base{ID: "turn"}, FamilyID: "fam", UserID: "user", Role: "user", Body: request.Message, ClientMessageID: &client, GeneratedAt: now}
	session := &model.AgentSession{Base: model.Base{ID: "session"}, FamilyID: "fam", UserID: "user"}
	if claimed, err := repo.ClaimChatTurn(context.Background(), session, user, "", "old-token", now, lease); !claimed || err != nil {
		t.Fatal(err)
	}
	response, err := s.Chat(context.Background(), "fam", "user", request)
	if err != nil || response.Status != "running" || response.Message != nil || response.TurnID != "turn" {
		t.Fatalf("running not recoverable: %+v %v", response, err)
	}
	if _, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{SessionID: "session", Message: "另一个问题", ClientMessageID: "other"}); err == nil {
		t.Fatal("busy session accepted new turn")
	}
	s.clock = func() time.Time { return lease.Add(time.Second) }
	response, err = s.Chat(context.Background(), "fam", "user", request)
	if err != nil || response.Status != "completed" || response.TurnID != "turn" || len(repo.sessions) != 1 {
		t.Fatalf("orphan recovery failed: %+v %v", response, err)
	}
	if err := repo.FinishChatTurn(context.Background(), user, "old-token", nil); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("old worker published: %v", err)
	}
}

func TestChatContextIncludesOnlyFiveCompletedPairs(t *testing.T) {
	s, repo := newAgentServiceForTest()
	var sessionID string
	for i := 0; i < 7; i++ {
		s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(_ context.Context, req LLMRequest, _ []ToolDef) (*LLMResponse, error) {
			var users int
			for _, m := range req.Messages[3:] {
				if m.Role == "user" {
					users++
				}
			}
			expected := i + 1
			if expected > 6 {
				expected = 6
			}
			if users != expected {
				t.Fatalf("history users=%d want=%d", users, expected)
			}
			return &LLMResponse{Content: `{"answer":"请明确猫咪与查询时间。","source_ids":[]}`}, nil
		}})
		response, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{SessionID: sessionID, Message: "请查询", ClientMessageID: string(rune('a' + i))})
		if err != nil {
			t.Fatal(err)
		}
		sessionID = response.SessionID
	}
	if len(repo.messages) != 14 {
		t.Fatal("lost history")
	}
}

func TestChatTrendFailureShowsOnlyActualValuesAndEvidence(t *testing.T) {
	s, _ := newAgentServiceForTest()
	s.records = &toolRecordRepo{rows: []*model.DailyRecord{toolRecord("feed", "feeding", `{"amount":30}`, s.clock().Add(-time.Hour))}}
	calls := 0
	s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(context.Context, LLMRequest, []ToolDef) (*LLMResponse, error) {
		calls++
		if calls == 1 {
			return &LLMResponse{ToolCalls: []LLMToolCall{{ID: "trend", Name: "getTrends", Arguments: `{"cat_id":"cat","metric":"feeding","days":2}`}}}, nil
		}
		return nil, ErrLLMUnavailable
	}})
	response, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{Message: "小白这两天吃多少", ClientMessageID: "key"})
	if err != nil || !response.Degraded || !strings.Contains(response.Message.Body, "30 g") || !strings.Contains(response.Message.Body, "1 天缺少") || len(response.Message.Evidence) != 1 || response.Message.Evidence[0].SourceID != "feed" {
		t.Fatalf("fallback lost real facts: %+v %v", response, err)
	}
}

func TestChatMessagesCannotBeConvertedIntoDrafts(t *testing.T) {
	s, repo := newAgentServiceForTest()
	response, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{Message: "查询", ClientMessageID: "key"})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range repo.messages {
		if m.SessionID != response.SessionID || m.Type != AgentTypeChatAnswer {
			continue
		}
		if _, err := s.EditDraft(context.Background(), "fam", "user", m.ID, AgentReminderEditRequest{Reminder: &AgentReminderInput{CatID: "cat", Title: "测试提醒", Type: "custom"}}); err == nil {
			t.Fatal("canonical chat history converted into draft")
		}
	}
}
