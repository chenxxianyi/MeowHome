package app

import (
	"context"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

type agentRepoMemory struct {
	sessions  map[string]*model.AgentSession
	messages  map[string]*model.AgentMessage
	reminders map[string]*model.Reminder
}

func newAgentRepoMemory() *agentRepoMemory {
	return &agentRepoMemory{sessions: map[string]*model.AgentSession{}, messages: map[string]*model.AgentMessage{}, reminders: map[string]*model.Reminder{}}
}
func (r *agentRepoMemory) CreateSession(_ context.Context, s *model.AgentSession) error {
	r.sessions[s.ID] = s
	return nil
}
func (r *agentRepoMemory) UpdateSession(_ context.Context, s *model.AgentSession) error {
	r.sessions[s.ID] = s
	return nil
}
func (r *agentRepoMemory) FindSession(_ context.Context, id string) (*model.AgentSession, error) {
	s, ok := r.sessions[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return s, nil
}
func (r *agentRepoMemory) ListSessions(_ context.Context, familyID, userID, _ string, limit int) ([]*model.AgentSession, error) {
	out := []*model.AgentSession{}
	for _, s := range r.sessions {
		if s.FamilyID == familyID && s.UserID == userID {
			out = append(out, s)
		}
	}
	if len(out) > limit+1 {
		out = out[:limit+1]
	}
	return out, nil
}
func (r *agentRepoMemory) CreateMessage(_ context.Context, m *model.AgentMessage) error {
	for _, old := range r.messages {
		if m.ClientMessageID != "" && old.FamilyID == m.FamilyID && old.UserID == m.UserID && old.ClientMessageID == m.ClientMessageID && old.Role == m.Role {
			return repository.ErrDuplicateKey
		}
		if m.DedupKey != nil && old.DedupKey != nil && *m.DedupKey == *old.DedupKey {
			return repository.ErrDuplicateKey
		}
	}
	r.messages[m.ID] = m
	return nil
}
func (r *agentRepoMemory) FindMessage(_ context.Context, id string) (*model.AgentMessage, error) {
	m, ok := r.messages[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return m, nil
}
func (r *agentRepoMemory) FindMessageByDedup(_ context.Context, familyID, dedupKey string) (*model.AgentMessage, error) {
	for _, m := range r.messages {
		if m.FamilyID == familyID && m.DedupKey != nil && *m.DedupKey == dedupKey {
			return m, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *agentRepoMemory) FindMessageByClientID(_ context.Context, sessionID, clientMessageID string) (*model.AgentMessage, error) {
	for _, m := range r.messages {
		if m.SessionID == sessionID && m.ClientMessageID == clientMessageID && m.Role == string(RoleAssistant) {
			return m, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *agentRepoMemory) FindAnyMessageByClientID(_ context.Context, familyID, userID, clientMessageID string) (*model.AgentMessage, error) {
	for _, m := range r.messages {
		if m.FamilyID == familyID && m.UserID == userID && m.ClientMessageID == clientMessageID {
			return m, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *agentRepoMemory) ListMessages(_ context.Context, q repository.AgentMessageQuery) ([]*model.AgentMessage, error) {
	out := []*model.AgentMessage{}
	for _, m := range r.messages {
		if m.FamilyID != q.FamilyID || (q.SessionID != "" && m.SessionID != q.SessionID) || (q.Visibility != "" && m.Visibility != q.Visibility) {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}
func (r *agentRepoMemory) UpdateMessage(_ context.Context, m *model.AgentMessage) error {
	r.messages[m.ID] = m
	return nil
}
func (r *agentRepoMemory) UpdateMessageIfVersion(_ context.Context, m *model.AgentMessage, expected int) error {
	old, ok := r.messages[m.ID]
	if !ok || old.DraftVersion != expected {
		return repository.ErrConflict
	}
	r.messages[m.ID] = m
	return nil
}
func (r *agentRepoMemory) ConfirmMessageAndCreateReminder(_ context.Context, messageID string, expected int, now time.Time, reminder *model.Reminder) (bool, error) {
	m, ok := r.messages[messageID]
	if !ok {
		return false, repository.ErrNotFound
	}
	if m.ActionStatus == DraftConfirmed && m.ConfirmedReminderID != "" {
		*reminder = *r.reminders[m.ConfirmedReminderID]
		return false, nil
	}
	if m.DraftVersion != expected || m.ActionStatus != DraftPending || m.DraftExpiresAt == nil || !m.DraftExpiresAt.After(now) {
		return false, repository.ErrConflict
	}
	r.reminders[reminder.ID] = reminder
	m.ActionStatus = DraftConfirmed
	m.ConfirmedReminderID = reminder.ID
	r.messages[m.ID] = m
	return true, nil
}

type memberMemory struct{ familyID, userID, role string }

func (m memberMemory) Create(context.Context, *model.Member) error                   { return nil }
func (m memberMemory) FindByFamily(context.Context, string) ([]*model.Member, error) { return nil, nil }
func (m memberMemory) FindByUser(_ context.Context, userID string) ([]*model.Member, error) {
	if userID == m.userID {
		return []*model.Member{{FamilyID: m.familyID, UserID: userID, Role: m.role}}, nil
	}
	return nil, nil
}
func (m memberMemory) UpdateRole(context.Context, string, string) error { return nil }
func (m memberMemory) Delete(context.Context, string) error             { return nil }

type familyMemory struct{ family *model.Family }

func (f familyMemory) Create(context.Context, *model.Family) error             { return nil }
func (f familyMemory) FindByID(context.Context, string) (*model.Family, error) { return f.family, nil }
func (f familyMemory) List(context.Context, string, int) ([]*model.Family, error) {
	return []*model.Family{f.family}, nil
}
func (f familyMemory) Update(context.Context, *model.Family) error             { return nil }
func (f familyMemory) Delete(context.Context, string) error                    { return nil }

type catMemory struct{ cat *model.Cat }

func (c catMemory) Create(context.Context, *model.Cat) error             { return nil }
func (c catMemory) FindByID(context.Context, string) (*model.Cat, error) { return c.cat, nil }
func (c catMemory) ListByFamily(context.Context, string) ([]*model.Cat, error) {
	return []*model.Cat{c.cat}, nil
}
func (c catMemory) Update(context.Context, *model.Cat) error { return nil }
func (c catMemory) Delete(context.Context, string) error     { return nil }

type reminderMemory struct{ repo *agentRepoMemory }

func (r reminderMemory) Create(_ context.Context, m *model.Reminder) error {
	r.repo.reminders[m.ID] = m
	return nil
}
func (r reminderMemory) FindByID(_ context.Context, id string) (*model.Reminder, error) {
	m, ok := r.repo.reminders[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return m, nil
}
func (r reminderMemory) List(context.Context, repository.ReminderQuery) ([]*model.Reminder, error) {
	out := []*model.Reminder{}
	for _, m := range r.repo.reminders {
		out = append(out, m)
	}
	return out, nil
}
func (r reminderMemory) Update(context.Context, *model.Reminder) error { return nil }
func (r reminderMemory) Delete(context.Context, string) error          { return nil }

type recordMemory struct{}

func (recordMemory) Create(context.Context, *model.DailyRecord) error { return nil }
func (recordMemory) FindByID(context.Context, string) (*model.DailyRecord, error) {
	return nil, repository.ErrNotFound
}
func (recordMemory) ListByFamily(context.Context, repository.DailyRecordQuery) ([]*model.DailyRecord, error) {
	return nil, nil
}
func (recordMemory) Delete(context.Context, string) error { return nil }

func newAgentServiceForTest() (*AgentService, *agentRepoMemory) {
	repo := newAgentRepoMemory()
	family := &model.Family{Base: model.Base{ID: "fam", CreatedAt: time.Now().Add(-30 * 24 * time.Hour)}, Timezone: "Asia/Shanghai"}
	cat := &model.Cat{Base: model.Base{ID: "cat"}, FamilyID: "fam", Name: "小白"}
	s := NewAgentService(repo, recordMemory{}, reminderMemory{repo: repo}, catMemory{cat: cat}, familyMemory{family: family}, memberMemory{familyID: "fam", userID: "user", role: "member"}, true)
	s.clock = func() time.Time { return time.Date(2026, 9, 28, 4, 0, 0, 0, time.UTC) }
	return s, repo
}

func TestChatRecoversSessionWhenFirstResponseIsRetried(t *testing.T) {
	s, repo := newAgentServiceForTest()
	first, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{ClientMessageID: "client-1", Message: "小白最近怎么样"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{ClientMessageID: "client-1", Message: "小白最近怎么样"})
	if err != nil {
		t.Fatal(err)
	}
	if first.SessionID != second.SessionID || first.Message.ID != second.Message.ID {
		t.Fatalf("retry did not return original result: first=%+v second=%+v", first, second)
	}
	if len(repo.sessions) != 1 || len(repo.messages) != 2 {
		t.Fatalf("retry created duplicate state: sessions=%d messages=%d", len(repo.sessions), len(repo.messages))
	}
	if _, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{ClientMessageID: "client-1", Message: "不同内容"}); err == nil {
		t.Fatal("reusing client_message_id with different content must fail")
	}
}

func TestConfirmDraftIsIdempotentInRepositoryPath(t *testing.T) {
	s, repo := newAgentServiceForTest()
	expires := s.clock().Add(time.Hour)
	draft := &model.AgentMessage{Base: model.Base{ID: "message-1"}, FamilyID: "fam", UserID: "user", Visibility: string(VisibilityFamily), Role: string(RoleAssistant), Type: AgentTypeReminderDraft, ActionStatus: DraftPending, DraftVersion: 1, DraftExpiresAt: &expires, DraftPayload: `{"cat_id":"cat","type":"custom","title":"剪指甲"}`}
	repo.messages[draft.ID] = draft
	first, err := s.ConfirmDraft(context.Background(), "fam", "user", draft.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.ConfirmDraft(context.Background(), "fam", "user", draft.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Reminder.ID != second.Reminder.ID || len(repo.reminders) != 1 {
		t.Fatalf("confirmation is not idempotent: first=%+v second=%+v reminders=%d", first, second, len(repo.reminders))
	}
}
