package app

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

type agentRepoMemory struct {
	sessions  map[string]*model.AgentSession
	messages  map[string]*model.AgentMessage
	reminders map[string]*model.Reminder
	audits    []string
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
func (r *agentRepoMemory) FindSession(_ context.Context, familyID, userID, id string) (*model.AgentSession, error) {
	s, ok := r.sessions[id]
	if !ok || s.FamilyID != familyID || s.UserID != userID {
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
	if r.messages[m.ID] != nil {
		return repository.ErrDuplicateKey
	}
	for _, old := range r.messages {
		if m.ClientMessageID != nil && old.ClientMessageID != nil && old.FamilyID == m.FamilyID && old.UserID == m.UserID && *old.ClientMessageID == *m.ClientMessageID && old.Role == m.Role {
			return repository.ErrDuplicateKey
		}
		if m.DedupKey != nil && old.DedupKey != nil && *m.DedupKey == *old.DedupKey {
			return repository.ErrDuplicateKey
		}
	}
	r.messages[m.ID] = m
	return nil
}
func (r *agentRepoMemory) CreatePatrolMessage(ctx context.Context, m *model.AgentMessage, from, to time.Time, maxNonDanger int64) error {
	if m.Severity != "danger" {
		count, err := r.CountNonDangerMessages(ctx, m.FamilyID, from, to)
		if err != nil {
			return err
		}
		if count >= maxNonDanger {
			return repository.ErrDailyLimit
		}
	}
	return r.CreateMessage(ctx, m)
}
func (r *agentRepoMemory) FindMessage(_ context.Context, familyID, id string) (*model.AgentMessage, error) {
	m, ok := r.messages[id]
	if !ok || m.FamilyID != familyID {
		return nil, repository.ErrNotFound
	}
	copy := *m
	return &copy, nil
}
func (r *agentRepoMemory) FindMessageByDedup(_ context.Context, familyID, dedupKey string) (*model.AgentMessage, error) {
	for _, m := range r.messages {
		if m.FamilyID == familyID && m.DedupKey != nil && *m.DedupKey == dedupKey {
			return m, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *agentRepoMemory) FindMessageByClientID(_ context.Context, familyID, userID, sessionID, clientMessageID string) (*model.AgentMessage, error) {
	for _, m := range r.messages {
		if m.FamilyID == familyID && m.UserID == userID && m.SessionID == sessionID && m.ClientMessageID != nil && *m.ClientMessageID == clientMessageID && m.Role == string(RoleAssistant) {
			return m, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *agentRepoMemory) FindUserMessageByClientID(_ context.Context, familyID, userID, clientMessageID string) (*model.AgentMessage, error) {
	for _, m := range r.messages {
		if m.FamilyID == familyID && m.UserID == userID && m.ClientMessageID != nil && *m.ClientMessageID == clientMessageID && m.Role == string(RoleUser) {
			return m, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *agentRepoMemory) ListMessages(_ context.Context, q repository.AgentMessageQuery) ([]*model.AgentMessage, error) {
	out := []*model.AgentMessage{}
	for _, m := range r.messages {
		if m.FamilyID != q.FamilyID || (q.SessionID != "" && m.SessionID != q.SessionID) || (q.Visibility != "" && m.Visibility != q.Visibility) || (q.ExcludeTools && m.Role == string(RoleTool)) {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *agentRepoMemory) CountNonDangerMessages(_ context.Context, familyID string, from, to time.Time) (int64, error) {
	var count int64
	for _, m := range r.messages {
		if m.FamilyID == familyID && m.Visibility == string(VisibilityFamily) && m.Severity != "danger" && !m.GeneratedAt.Before(from) && m.GeneratedAt.Before(to) && m.DeletedAt == nil {
			count++
		}
	}
	return count, nil
}
func (r *agentRepoMemory) UpdateMessage(_ context.Context, familyID string, m *model.AgentMessage) error {
	if m.FamilyID != familyID {
		return repository.ErrNotFound
	}
	r.messages[m.ID] = m
	return nil
}
func (r *agentRepoMemory) UpdateMessageIfVersion(_ context.Context, familyID string, m *model.AgentMessage, expected int, expectedStatus string) error {
	old, ok := r.messages[m.ID]
	if !ok || old.FamilyID != familyID || old.DraftVersion != expected || old.ActionStatus != expectedStatus {
		return repository.ErrConflict
	}
	r.messages[m.ID] = m
	return nil
}
func (r *agentRepoMemory) ConfirmMessageAndCreateReminder(_ context.Context, familyID, actorUserID, messageID string, expected int, now time.Time, reminder *model.Reminder) (bool, error) {
	m, ok := r.messages[messageID]
	if !ok || m.FamilyID != familyID || reminder.FamilyID != familyID || actorUserID == "" {
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
	r.audits = append(r.audits, messageID)
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

func (f familyMemory) Create(context.Context, *model.Family) error { return nil }
func (f familyMemory) FindByID(_ context.Context, id string) (*model.Family, error) {
	if f.family.ID != id {
		return nil, repository.ErrNotFound
	}
	return f.family, nil
}
func (f familyMemory) List(context.Context, string, int) ([]*model.Family, error) {
	return []*model.Family{f.family}, nil
}
func (f familyMemory) Update(context.Context, *model.Family) error { return nil }
func (f familyMemory) Delete(context.Context, string) error        { return nil }

type catMemory struct{ cat *model.Cat }

func (c catMemory) Create(context.Context, *model.Cat) error { return nil }
func (c catMemory) FindByID(_ context.Context, id string) (*model.Cat, error) {
	if c.cat.ID != id {
		return nil, repository.ErrNotFound
	}
	return c.cat, nil
}
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

func (r reminderMemory) ListScheduled(_ context.Context, q repository.ReminderScheduleQuery) ([]*model.Reminder, error) {
	out := make([]*model.Reminder, 0)
	for _, m := range r.repo.reminders {
		if m.FamilyID != q.FamilyID || m.DeletedAt != nil || m.ScheduledAt == nil || (q.State != "" && m.State != q.State) {
			continue
		}
		if len(q.Types) > 0 {
			found := false
			for _, t := range q.Types {
				found = found || m.Type == t
			}
			if !found {
				continue
			}
		}
		if q.From != nil && m.ScheduledAt.Before(*q.From) {
			continue
		}
		if q.ToExclusive != nil && !m.ScheduledAt.Before(*q.ToExclusive) {
			continue
		}
		if q.CursorAt != nil && (m.ScheduledAt.Before(*q.CursorAt) || (m.ScheduledAt.Equal(*q.CursorAt) && m.ID <= q.CursorID)) {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScheduledAt.Equal(*out[j].ScheduledAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].ScheduledAt.Before(*out[j].ScheduledAt)
	})
	if len(out) > q.Limit {
		out = out[:q.Limit]
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

type eventRecordForTest struct {
	recordMemory
	record *model.DailyRecord
}

type manualAuditMemory struct{ logs []*model.AuditLog }

func (a *manualAuditMemory) Create(_ context.Context, log *model.AuditLog) error {
	a.logs = append(a.logs, log)
	return nil
}

func TestManualPatrolRequiresAdminAndWritesAuditForZeroMessages(t *testing.T) {
	s, _ := newAgentServiceForTest()
	s.families = familyMemory{family: &model.Family{Base: model.Base{ID: "fam", CreatedAt: s.clock()}, Timezone: "Asia/Shanghai"}}
	audit := &manualAuditMemory{}
	s.SetAuditRepo(audit)
	if _, err := s.Patrol(context.Background(), "fam", "user"); err == nil {
		t.Fatal("member manual patrol allowed")
	}
	if len(audit.logs) != 0 {
		t.Fatal("unauthorized patrol audited as success")
	}
	s.members = memberMemory{familyID: "fam", userID: "user", role: "admin"}
	if _, err := s.Patrol(context.Background(), "fam", "user"); err != nil {
		t.Fatal(err)
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != "agent.patrol.manual" || audit.logs[0].FamilyID != "fam" || audit.logs[0].UserID != "user" {
		t.Fatalf("missing scoped audit: %+v", audit.logs)
	}
}

func (r eventRecordForTest) FindByID(_ context.Context, id string) (*model.DailyRecord, error) {
	if r.record != nil && r.record.ID == id {
		return r.record, nil
	}
	return nil, repository.ErrNotFound
}

func newAgentServiceForTest() (*AgentService, *agentRepoMemory) {
	repo := newAgentRepoMemory()
	family := &model.Family{Base: model.Base{ID: "fam", CreatedAt: time.Now().Add(-30 * 24 * time.Hour)}, Timezone: "Asia/Shanghai"}
	cat := &model.Cat{Base: model.Base{ID: "cat"}, FamilyID: "fam", Name: "小白"}
	s := NewAgentService(repo, recordMemory{}, reminderMemory{repo: repo}, catMemory{cat: cat}, familyMemory{family: family}, memberMemory{familyID: "fam", userID: "user", role: "member"}, true)
	s.clock = func() time.Time { return time.Date(2026, 9, 28, 4, 0, 0, 0, time.UTC) }
	return s, repo
}

func TestDangerEventUsesCreatedRecordAndDeduplicates(t *testing.T) {
	s, repo := newAgentServiceForTest()
	r := &model.DailyRecord{Base: model.Base{ID: "record-1", CreatedAt: time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC)}, FamilyID: "fam", CatIDs: []string{"cat"}, Severity: "danger", OccurredAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), Note: "补录"}
	s.records = eventRecordForTest{record: r}
	for i := 0; i < 2; i++ {
		if err := s.PatrolDangerRecordSystem(context.Background(), "fam", r.ID); err != nil {
			t.Fatal(err)
		}
	}
	if len(repo.messages) != 1 {
		t.Fatalf("event message count = %d", len(repo.messages))
	}
	for _, m := range repo.messages {
		if m.Severity != "danger" || m.CatID != "cat" || m.RuleID != RuleR01 || m.DedupKey == nil {
			t.Fatalf("bad event message: %+v", m)
		}
	}
	if err := s.PatrolDangerRecordSystem(context.Background(), "other", r.ID); err != repository.ErrNotFound {
		t.Fatalf("cross-family record: %v", err)
	}
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
	scheduled := s.clock().Add(2 * time.Hour).Format(time.RFC3339)
	draft := &model.AgentMessage{Base: model.Base{ID: "message-1"}, FamilyID: "fam", UserID: "user", Visibility: string(VisibilityFamily), Role: string(RoleAssistant), Type: AgentTypeReminderDraft, ActionStatus: DraftPending, DraftVersion: 1, DraftExpiresAt: &expires, DraftPayload: fmt.Sprintf(`{"cat_id":"cat","type":"custom","title":"剪指甲","scheduled_at":%q}`, scheduled)}
	repo.messages[draft.ID] = draft
	first, err := s.ConfirmDraft(context.Background(), "fam", "user", draft.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.ConfirmDraft(context.Background(), "fam", "user", draft.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Reminder.ID != second.Reminder.ID || len(repo.reminders) != 1 || len(repo.audits) != 1 {
		t.Fatalf("confirmation is not idempotent: first=%+v second=%+v reminders=%d", first, second, len(repo.reminders))
	}
}

func TestAgentMessagePageCursorPointsToLastReturnedMessage(t *testing.T) {
	s, _ := newAgentServiceForTest()
	base := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	rows := []*model.AgentMessage{
		{Base: model.Base{ID: "c"}, GeneratedAt: base},
		{Base: model.Base{ID: "b"}, GeneratedAt: base},
		{Base: model.Base{ID: "a"}, GeneratedAt: base},
	}
	page := s.messageList(rows, 2)
	if len(page.Messages) != 2 || page.NextCursor != cursorFor(rows[1]) {
		t.Fatalf("cursor must resume after the last returned row: %+v", page)
	}
}

func TestPrivateAgentMessageUsesSessionOwner(t *testing.T) {
	s, repo := newAgentServiceForTest()
	repo.sessions["session"] = &model.AgentSession{Base: model.Base{ID: "session"}, FamilyID: "fam", UserID: "another-user"}
	m := &model.AgentMessage{Base: model.Base{ID: "message"}, FamilyID: "fam", SessionID: "session", UserID: "user", Visibility: string(VisibilityPrivate)}
	if s.canReadMessage(context.Background(), "fam", "user", m) {
		t.Fatal("private message must follow the session owner, not a denormalized message user_id")
	}
}

func TestPatrolScheduledRemindersPagesPastFiveHundred(t *testing.T) {
	s, repo := newAgentServiceForTest()
	now := s.clock()
	due := now.Add(-25 * time.Hour)
	for i := 0; i < 501; i++ {
		id := fmt.Sprintf("rem-%04d", i)
		repo.reminders[id] = &model.Reminder{Base: model.Base{ID: id}, FamilyID: "fam", Type: "medication", State: "todo", ScheduledAt: &due}
	}
	boundary := now.Add(-24 * time.Hour)
	repo.reminders["boundary"] = &model.Reminder{Base: model.Base{ID: "boundary"}, FamilyID: "fam", Type: "medication", State: "todo", ScheduledAt: &boundary}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	got, err := s.loadPatrolReminders(context.Background(), "fam", now, loc)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 501 {
		t.Fatalf("expected all 501 overdue reminders and no exact-24h reminder, got %d", len(got))
	}
}

func TestStaleDismissCannotOverwriteConfirmedDraft(t *testing.T) {
	repo := newAgentRepoMemory()
	m := &model.AgentMessage{Base: model.Base{ID: "draft"}, FamilyID: "fam", DraftVersion: 1, ActionStatus: DraftPending}
	repo.messages[m.ID] = m
	stale, err := repo.FindMessage(context.Background(), "fam", m.ID)
	if err != nil {
		t.Fatal(err)
	}
	m.ActionStatus = DraftConfirmed
	m.ConfirmedReminderID = "reminder"
	stale.ActionStatus = DraftDismissed
	stale.ConfirmedReminderID = ""
	if err := repo.UpdateMessageIfVersion(context.Background(), "fam", stale, 1, DraftPending); err != repository.ErrConflict {
		t.Fatalf("stale dismiss should conflict with confirmation, got %v", err)
	}
	if m.ActionStatus != DraftConfirmed || m.ConfirmedReminderID != "reminder" {
		t.Fatal("stale dismiss changed confirmed reminder state")
	}
}

func TestDismissConfirmedMessageOnlyChangesDisplayStatus(t *testing.T) {
	s, repo := newAgentServiceForTest()
	repo.messages["confirmed"] = &model.AgentMessage{Base: model.Base{ID: "confirmed"}, FamilyID: "fam", Visibility: string(VisibilityFamily), ActionStatus: DraftConfirmed, ConfirmedReminderID: "reminder", DraftVersion: 1}
	if err := s.DismissMessage(context.Background(), "fam", "user", "confirmed"); err != nil {
		t.Fatal(err)
	}
	if err := s.DismissMessage(context.Background(), "fam", "user", "confirmed"); err != nil {
		t.Fatalf("repeated dismiss should be idempotent: %v", err)
	}
	m := repo.messages["confirmed"]
	if m.DisplayStatus != DraftDismissed || m.ActionStatus != DraftConfirmed || m.ConfirmedReminderID != "reminder" {
		t.Fatalf("dismiss must preserve formal reminder state: %+v", m)
	}
}
