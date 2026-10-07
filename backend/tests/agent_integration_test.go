package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/infrastructure/persistence/mysql"
	"github.com/meowhome/backend/internal/platform/config"
	"github.com/meowhome/backend/internal/platform/token"
	"github.com/meowhome/backend/internal/transport/http/handler"
	"github.com/meowhome/backend/internal/transport/http/router"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func agentDBFixture(t *testing.T) (*gorm.DB, *app.AgentService, *mysql.AgentRepo, string, string, string) {
	t.Helper()
	db := SetupTestDB(t)
	now := time.Now().UTC()
	familyID, userID, catID := ulid.Make().String(), ulid.Make().String(), ulid.Make().String()
	require.NoError(t, db.Create(&model.Family{Base: model.Base{ID: familyID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, Name: "Agent test", Timezone: "Asia/Shanghai", Currency: "CNY"}).Error)
	require.NoError(t, db.Create(&model.Member{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: userID, Role: "owner", Timezone: "Asia/Shanghai"}).Error)
	require.NoError(t, db.Omit("Birthday").Create(&model.Cat{Base: model.Base{ID: catID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, Name: "小白", Gender: "unknown"}).Error)
	repo := mysql.NewAgentRepo(db)
	service := app.NewAgentService(repo, mysql.NewDailyRecordRepo(db), mysql.NewReminderRepo(db), mysql.NewCatRepo(db), mysql.NewFamilyRepo(db), mysql.NewMemberRepo(db), true)
	return db, service, repo, familyID, userID, catID
}

func TestAgentMySQLMigrationRepeatAndConcurrentFirstTurn(t *testing.T) {
	db, service, _, familyID, userID, _ := agentDBFixture(t)
	require.NoError(t, applyMigrations(db, t))
	started, release := make(chan struct{}), make(chan struct{})
	service.SetLLMProvider(app.FakeLLMProvider{ChatFunc: func(context.Context, app.LLMRequest, []app.ToolDef) (*app.LLMResponse, error) {
		close(started)
		<-release
		return &app.LLMResponse{Content: `{"answer":"请明确猫咪与时间。","source_ids":[]}`}, nil
	}})
	input := app.AgentChatRequest{Message: "查询记录", ClientMessageID: "client"}
	var answer *app.AgentChatResponse
	var firstErr error
	var worker sync.WaitGroup
	worker.Add(1)
	go func() {
		defer worker.Done()
		answer, firstErr = service.Chat(context.Background(), familyID, userID, input)
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		close(release)
		worker.Wait()
		t.Fatal("chat did not reach provider")
	}
	other, err := service.Chat(context.Background(), familyID, userID, input)
	close(release)
	worker.Wait()
	require.NoError(t, firstErr)
	require.NoError(t, err)
	require.Equal(t, "running", other.Status)
	require.Equal(t, answer.SessionID, other.SessionID)
	require.Equal(t, answer.TurnID, other.TurnID)
	again, err := service.Chat(context.Background(), familyID, userID, input)
	require.NoError(t, err)
	require.Equal(t, answer.Message.ID, again.Message.ID)
	var count int64
	require.NoError(t, db.Model(&model.AgentSession{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestAgentMySQLPrivateDraftConfirmationAndAuditRollback(t *testing.T) {
	db, service, repo, familyID, userID, catID := agentDBFixture(t)
	now := time.Now().UTC()
	sessionID, messageID := ulid.Make().String(), ulid.Make().String()
	require.NoError(t, repo.CreateSession(context.Background(), &model.AgentSession{Base: model.Base{ID: sessionID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: userID, Status: "active"}))
	target := &model.AgentMessage{Base: model.Base{ID: messageID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, SessionID: sessionID, UserID: userID, Role: "assistant", Visibility: "private", Type: "reminder_draft", Title: "剪指甲", Body: "核对提醒", GeneratedAt: now, Model: "test"}
	require.NoError(t, repo.CreateMessage(context.Background(), target))
	scheduled := now.Add(2 * time.Hour)
	draft, err := service.EditDraft(context.Background(), familyID, userID, messageID, app.AgentReminderEditRequest{Reminder: &app.AgentReminderInput{CatID: catID, Type: "custom", Title: "剪指甲", ScheduledAt: &scheduled, Timezone: "Asia/Shanghai"}})
	require.NoError(t, err)
	otherID := ulid.Make().String()
	require.NoError(t, db.Create(&model.Member{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: otherID, Role: "member"}).Error)
	_, err = service.ConfirmDraft(context.Background(), familyID, otherID, messageID, draft.Version)
	require.Error(t, err)
	// A DB trigger makes the audit insert fail after reminder/message writes.
	require.NoError(t, db.Exec("CREATE TRIGGER agent_test_audit_failure BEFORE INSERT ON audit_logs FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='agent test audit failure'").Error)
	_, err = service.ConfirmDraft(context.Background(), familyID, userID, messageID, draft.Version)
	require.Error(t, err)
	var count int64
	require.NoError(t, db.Model(&model.Reminder{}).Count(&count).Error)
	require.EqualValues(t, 0, count)
	stored, err := repo.FindMessage(context.Background(), familyID, messageID)
	require.NoError(t, err)
	require.Equal(t, "pending", stored.ActionStatus)
	require.NoError(t, db.Exec("DROP TRIGGER agent_test_audit_failure").Error)
	first, err := service.ConfirmDraft(context.Background(), familyID, userID, messageID, draft.Version)
	require.NoError(t, err)
	second, err := service.ConfirmDraft(context.Background(), familyID, userID, messageID, draft.Version)
	require.NoError(t, err)
	require.Equal(t, first.Reminder.ID, second.Reminder.ID)
	_, err = service.ConfirmDraft(context.Background(), familyID, otherID, messageID, draft.Version)
	require.Error(t, err)
}

func TestAgentMySQLLeaseFencingAndEnhancementVersion(t *testing.T) {
	_, _, repo, familyID, userID, _ := agentDBFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	key := "client"
	session := &model.AgentSession{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: userID, Status: "active"}
	user := &model.AgentMessage{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: userID, Role: "user", Visibility: "private", Type: "chat_answer", Title: "查询", Body: "查询", GeneratedAt: now, Model: "test", ClientMessageID: &key}
	claimed, err := repo.ClaimChatTurn(ctx, session, user, "", "old", now, now.Add(time.Second))
	require.NoError(t, err)
	require.True(t, claimed)
	claimed, err = repo.ClaimChatTurn(ctx, session, user, "", "new", now.Add(2*time.Second), now.Add(time.Minute))
	require.NoError(t, err)
	require.True(t, claimed)
	user.RunStatus = "failed"
	require.ErrorIs(t, repo.FinishChatTurn(ctx, user, "old", nil), repository.ErrConflict)
	require.NoError(t, repo.FinishChatTurn(ctx, user, "new", nil))
	m := &model.AgentMessage{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, Role: "assistant", Visibility: "family", Type: "patrol_abnormal", Severity: "warning", Title: "规则", Body: "规则提示", GeneratedAt: now, Model: "rule-engine-v1", RuleID: app.RuleR06}
	require.NoError(t, repo.CreateMessage(ctx, m))
	claimed, err = repo.ClaimEnhancement(ctx, m)
	require.NoError(t, err)
	require.True(t, claimed)
	stale := *m
	m.DisplayStatus = "dismissed"
	require.NoError(t, repo.UpdateMessageIfVersion(ctx, familyID, m, 0, ""))
	stale.EnhancedBody = "规则提示"
	stale.EnhanceStatus = "completed"
	require.True(t, errors.Is(repo.SaveEnhancement(ctx, &stale), repository.ErrConflict))
	stored, err := repo.FindMessage(ctx, familyID, m.ID)
	require.NoError(t, err)
	require.Empty(t, stored.EnhancedBody)
}

type agentIntegrationHealth struct{}

func (agentIntegrationHealth) Liveness() (any, error)  { return map[string]string{"status": "up"}, nil }
func (agentIntegrationHealth) Readiness() (any, error) { return map[string]string{"status": "up"}, nil }

func TestAgentMySQLChatRoutesAndOwnerIsolation(t *testing.T) {
	db, service, _, familyID, userID, _ := agentDBFixture(t)
	const secret = "agent-integration-secret"
	cfg := &config.Config{}
	cfg.Auth.JWTSecret = secret
	h := handler.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, service)
	engine := router.New(cfg, zap.NewNop(), nil, agentIntegrationHealth{}, h)
	bearer, err := token.Sign([]byte(secret), userID, time.Hour, time.Now().UTC())
	require.NoError(t, err)
	request := func(method, path, body, bearer string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+bearer)
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, req)
		return recorder
	}
	base := "/api/v1/families/" + familyID + "/agent"
	first := request("POST", base+"/chat", `{"client_message_id":"key","message":"查询记录"}`, bearer)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	var envelope struct {
		Data app.AgentChatResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &envelope))
	again := request("POST", base+"/chat", `{"client_message_id":"key","message":"查询记录"}`, bearer)
	require.Equal(t, http.StatusOK, again.Code)
	var repeated struct {
		Data app.AgentChatResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(again.Body.Bytes(), &repeated))
	require.Equal(t, envelope.Data.Message.ID, repeated.Data.Message.ID)
	require.Equal(t, http.StatusOK, request("GET", base+"/sessions", "", bearer).Code)
	require.Equal(t, http.StatusOK, request("GET", base+"/sessions/"+envelope.Data.SessionID+"/messages", "", bearer).Code)
	otherID := ulid.Make().String()
	now := time.Now().UTC()
	require.NoError(t, db.Create(&model.Member{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: otherID, Role: "member"}).Error)
	other, err := token.Sign([]byte(secret), otherID, time.Hour, now)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, request("GET", base+"/sessions/"+envelope.Data.SessionID+"/messages", "", other).Code)
	require.Equal(t, http.StatusNotFound, request("GET", base+"/messages/"+envelope.Data.Message.ID, "", other).Code)
}

func TestAgentMySQLChatRouteInterruptedDraftRecovery(t *testing.T) {
	db, service, repo, familyID, userID, catID := agentDBFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)
	session := &model.AgentSession{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: userID, Status: "active"}
	key := "interrupted"
	user := &model.AgentMessage{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: userID, Role: "user", Visibility: "private", Type: "chat_answer", Title: "创建提醒", Body: "创建剪指甲提醒", GeneratedAt: now, Model: "test", ClientMessageID: &key}
	claimed, err := repo.ClaimChatTurn(ctx, session, user, "", "old-worker", now, now.Add(time.Minute))
	require.NoError(t, err)
	require.True(t, claimed)
	args := fmt.Sprintf(`{"cat_id":%q,"title":"剪指甲","type":"custom","scheduled_at":%q,"timezone":"Asia/Shanghai"}`, catID, now.Add(2*time.Hour).Format(time.RFC3339))
	scope := app.AgentToolScope{FamilyID: familyID, UserID: userID, SessionID: session.ID, TurnID: user.TurnID}
	tool, err := service.ExecuteTool(ctx, scope, "createReminderDraft", args)
	require.NoError(t, err)
	require.NotNil(t, tool.Draft)
	// This is the state left by a process interruption after draft persistence.
	calls := 0
	service.SetLLMProvider(app.FakeLLMProvider{ChatFunc: func(context.Context, app.LLMRequest, []app.ToolDef) (*app.LLMResponse, error) {
		calls++
		if calls == 1 {
			return &app.LLMResponse{ToolCalls: []app.LLMToolCall{{ID: "draft", Name: "createReminderDraft", Arguments: args}}, Model: "test"}, nil
		}
		return &app.LLMResponse{Content: `{"answer":"请核对提醒草稿并手动确认。","source_ids":[]}`, Model: "test"}, nil
	}})
	const secret = "agent-recovery-test-secret"
	cfg := &config.Config{}
	cfg.Auth.JWTSecret = secret
	engine := router.New(cfg, zap.NewNop(), nil, agentIntegrationHealth{}, handler.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, service))
	bearer, err := token.Sign([]byte(secret), userID, time.Hour, now)
	require.NoError(t, err)
	chat := func(sessionID, key, message string) *httptest.ResponseRecorder {
		body, err := json.Marshal(app.AgentChatRequest{SessionID: sessionID, ClientMessageID: key, Message: message})
		require.NoError(t, err)
		req := httptest.NewRequest("POST", "/api/v1/families/"+familyID+"/agent/chat", strings.NewReader(string(body)))
		req.Header.Set("Authorization", "Bearer "+bearer)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, req)
		return response
	}
	decode := func(response *httptest.ResponseRecorder) app.AgentChatResponse {
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var envelope struct {
			Data app.AgentChatResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
		return envelope.Data
	}
	active := decode(chat("", key, user.Body))
	require.Equal(t, "running", active.Status)
	require.Nil(t, active.Message)
	require.Equal(t, session.ID, active.SessionID)
	require.Equal(t, user.TurnID, active.TurnID)
	require.Equal(t, http.StatusConflict, chat(session.ID, "another", "另一轮查询").Code)
	require.Zero(t, calls)
	// Expire the persisted lease; a fresh admission recovers the canonical turn.
	require.NoError(t, db.Model(&model.AgentMessage{}).Where("id = ?", user.ID).Update("run_lease_until", now.Add(-time.Minute)).Error)
	recovered := decode(chat("", key, user.Body))
	require.Equal(t, "completed", recovered.Status)
	require.False(t, recovered.Degraded)
	require.Equal(t, session.ID, recovered.SessionID)
	require.Equal(t, user.TurnID, recovered.TurnID)
	require.Equal(t, 2, calls)
	repeated := decode(chat("", key, user.Body))
	require.Equal(t, recovered.Message.ID, repeated.Message.ID)
	require.Equal(t, 2, calls)
	var drafts []model.AgentMessage
	require.NoError(t, db.Where("turn_id = ? AND type = ?", user.TurnID, "reminder_draft").Find(&drafts).Error)
	require.Len(t, drafts, 1)
	require.Equal(t, tool.Draft.MessageID, drafts[0].ID)
	require.Equal(t, tool.Draft.Version, drafts[0].DraftVersion)
	require.Equal(t, "pending", drafts[0].ActionStatus)
	var count int64
	require.NoError(t, db.Model(&model.Reminder{}).Count(&count).Error)
	require.Zero(t, count)
	require.ErrorIs(t, repo.FinishChatTurn(ctx, user, "old-worker", nil), repository.ErrConflict)
	history, err := service.ListSessionMessages(ctx, familyID, userID, session.ID, "", 20)
	require.NoError(t, err)
	require.Len(t, history.Messages, 3, "user, draft and answer are visible; internal tool messages are hidden")
	continued := decode(chat(session.ID, "continue", "继续核对"))
	require.Equal(t, session.ID, continued.SessionID)
	require.NotEqual(t, recovered.TurnID, continued.TurnID)
}
