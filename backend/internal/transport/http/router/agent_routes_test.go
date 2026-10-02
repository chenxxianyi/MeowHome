package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/config"
	"github.com/meowhome/backend/internal/platform/token"
	"github.com/meowhome/backend/internal/transport/http/handler"
	my "github.com/meowhome/backend/internal/transport/http/middleware"
	"go.uber.org/zap"
)

type routeMemberRepo struct{ repository.MemberRepo }

func (routeMemberRepo) FindByUser(_ context.Context, userID string) ([]*model.Member, error) {
	if userID == "member" {
		return []*model.Member{{FamilyID: "fam", UserID: userID, Role: "member"}}, nil
	}
	return nil, nil
}

type routeAgentRepo struct{ repository.AgentRepo }

func (routeAgentRepo) FindMessage(context.Context, string, string) (*model.AgentMessage, error) {
	return nil, repository.ErrNotFound
}

func (routeAgentRepo) ListMessages(context.Context, repository.AgentMessageQuery) ([]*model.AgentMessage, error) {
	return []*model.AgentMessage{}, nil
}

type routeRecordRepo struct{ repository.DailyRecordRepo }

func (routeRecordRepo) FindByID(_ context.Context, id string) (*model.DailyRecord, error) {
	if id == "other-record" {
		return &model.DailyRecord{Base: model.Base{ID: id}, FamilyID: "other"}, nil
	}
	return nil, repository.ErrNotFound
}

type routeHealth struct{}

func (routeHealth) Liveness() (any, error)  { return map[string]string{"status": "up"}, nil }
func (routeHealth) Readiness() (any, error) { return map[string]string{"status": "up"}, nil }

type routeActionRepo struct {
	repository.AgentRepo
	message  *model.AgentMessage
	reminder *model.Reminder
}

func (r *routeActionRepo) FindMessage(_ context.Context, familyID, id string) (*model.AgentMessage, error) {
	if r.message == nil || r.message.FamilyID != familyID || r.message.ID != id {
		return nil, repository.ErrNotFound
	}
	copy := *r.message
	return &copy, nil
}
func (r *routeActionRepo) UpdateMessageIfVersion(_ context.Context, familyID string, m *model.AgentMessage, version int, status string) error {
	if r.message == nil || r.message.FamilyID != familyID || r.message.DraftVersion != version || r.message.ActionStatus != status {
		return repository.ErrConflict
	}
	copy := *m
	r.message = &copy
	return nil
}
func (r *routeActionRepo) ConfirmMessageAndCreateReminder(_ context.Context, familyID, actorID, messageID string, expected int, now time.Time, reminder *model.Reminder) (bool, error) {
	if r.message == nil || r.message.FamilyID != familyID || r.message.ID != messageID || r.message.DraftVersion != expected || r.message.ActionStatus != "pending" {
		return false, repository.ErrConflict
	}
	r.reminder = reminder
	r.message.ActionStatus = "confirmed"
	r.message.ConfirmedReminderID = reminder.ID
	return true, nil
}

type routeReminderRepo struct {
	repository.ReminderRepo
	actions *routeActionRepo
}

func (r routeReminderRepo) FindByID(_ context.Context, id string) (*model.Reminder, error) {
	if r.actions.reminder != nil && r.actions.reminder.ID == id {
		return r.actions.reminder, nil
	}
	return nil, repository.ErrNotFound
}

func TestAgentAndRecordRoutesUseRealAuthChain(t *testing.T) {
	const secret = "test-route-secret"
	members := routeMemberRepo{}
	records := routeRecordRepo{}
	agent := app.NewAgentService(routeAgentRepo{}, records, nil, nil, nil, members, true)
	record := app.NewRecordService(records, members, nil)
	h := handler.New(nil, nil, nil, nil, record, nil, nil, nil, nil, agent)
	cfg := &config.Config{}
	cfg.Auth.JWTSecret = secret
	engine := New(cfg, zap.NewNop(), nil, routeHealth{}, h)
	tok, err := token.Sign([]byte(secret), "member", time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		method, path, bearer, body, override string
		want                                 int
	}{
		{"GET", "/api/v1/families/fam/agent/messages", "", "", "", http.StatusUnauthorized},
		{"GET", "/api/v1/families/fam/agent/messages", tok, "", "", http.StatusOK},
		{"GET", "/api/v1/families/fam/agent/messages?limit=0", tok, "", "", http.StatusBadRequest},
		{"GET", "/api/v1/families/other/agent/messages", tok, "", "", http.StatusForbidden},
		{"GET", "/api/v1/families/fam/agent/messages/other", tok, "", "", http.StatusNotFound},
		{"POST", "/api/v1/families/fam/agent/messages/other/dismiss", tok, "", "", http.StatusNotFound},
		{"POST", "/api/v1/families/fam/agent/messages/other/confirm", tok, `{"expected_version":1}`, "", http.StatusNotFound},
		{"POST", "/api/v1/families/fam/agent/messages/other/draft", tok, "", "PATCH", http.StatusBadRequest},
		{"POST", "/api/v1/families/fam/agent/messages/other/confirm", tok, `{"expected_version":1,"padding":"` + strings.Repeat("x", 17000) + `"}`, "", http.StatusRequestEntityTooLarge},
		{"POST", "/api/v1/internal/agent/patrol", tok, `{"family_id":"fam"}`, "", http.StatusForbidden},
		{"GET", "/api/v1/families/fam/records/other-record", tok, "", "", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			body := strings.NewReader(tc.body)
			req := httptest.NewRequest(tc.method, tc.path, body)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			if tc.override != "" {
				req.Header.Set("X-HTTP-Method-Override", tc.override)
			}
			if tc.bearer != "" {
				req.Header.Set("Authorization", "Bearer "+tc.bearer)
			}
			w := httptest.NewRecorder()
			my.MethodOverride(engine).ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status %d, want %d, body %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
	for attempt := 2; attempt <= 4; attempt++ {
		req := httptest.NewRequest("POST", "/api/v1/internal/agent/patrol", strings.NewReader(`{"family_id":"fam"}`))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		my.MethodOverride(engine).ServeHTTP(w, req)
		want := http.StatusForbidden
		if attempt == 4 {
			want = http.StatusTooManyRequests
		}
		if w.Code != want {
			t.Fatalf("patrol attempt %d returned %d, want %d", attempt, w.Code, want)
		}
	}
}

func TestAgentDraftRoutesEditConfirmAndDismiss(t *testing.T) {
	const secret = "test-route-secret"
	now := time.Now().UTC()
	actions := &routeActionRepo{message: &model.AgentMessage{Base: model.Base{ID: "message-1", CreatedAt: now}, FamilyID: "fam", Role: "assistant", Visibility: "family", Type: app.AgentTypePatrolAbnormal, Severity: "info", Title: "护理提示", Body: "查看记录", GeneratedAt: now, Model: "rule-engine-v1"}}
	agent := app.NewAgentService(actions, nil, routeReminderRepo{actions: actions}, nil, nil, routeMemberRepo{}, true)
	h := handler.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, agent)
	cfg := &config.Config{}
	cfg.Auth.JWTSecret = secret
	engine := my.MethodOverride(New(cfg, zap.NewNop(), nil, routeHealth{}, h))
	tok, err := token.Sign([]byte(secret), "member", time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body, override string, want int) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tok)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if override != "" {
			req.Header.Set("X-HTTP-Method-Override", override)
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s returned %d, want %d: %s", method, path, w.Code, want, w.Body.String())
		}
	}
	base := "/api/v1/families/fam/agent/messages/message-1"
	scheduled := now.Add(2 * time.Hour).Format(time.RFC3339)
	editBody := `{"expected_version":0,"reminder":{"cat_id":"both","type":"custom","title":"检查猫砂","scheduled_at":"` + scheduled + `","timezone":"UTC"}}`
	call("POST", base+"/draft", editBody, "PATCH", http.StatusOK)
	if actions.message.DraftVersion != 1 || actions.message.ActionStatus != "pending" {
		t.Fatalf("draft state: %+v", actions.message)
	}
	call("POST", base+"/confirm", `{"expected_version":1}`, "", http.StatusOK)
	if actions.reminder == nil || actions.reminder.FamilyID != "fam" || actions.message.ConfirmedReminderID != actions.reminder.ID {
		t.Fatalf("confirmation state: %+v %+v", actions.message, actions.reminder)
	}
	call("POST", base+"/dismiss", "", "", http.StatusNoContent)
	if actions.message.DisplayStatus != "dismissed" || actions.message.ActionStatus != "confirmed" || actions.reminder == nil {
		t.Fatalf("dismiss state: %+v %+v", actions.message, actions.reminder)
	}
}
