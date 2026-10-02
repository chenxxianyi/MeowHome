package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

type toolRecordRepo struct {
	recordMemory
	rows    []*model.DailyRecord
	err     error
	queries []repository.DailyRecordQuery
}

func (r *toolRecordRepo) ListByFamily(_ context.Context, q repository.DailyRecordQuery) ([]*model.DailyRecord, error) {
	r.queries = append(r.queries, q)
	if r.err != nil {
		return nil, r.err
	}
	from, _ := time.Parse(time.RFC3339, q.From)
	to, _ := time.Parse(time.RFC3339, q.To)
	out := make([]*model.DailyRecord, 0)
	for _, row := range r.rows {
		if row.FamilyID != q.FamilyID || !recordHasCat(row, q.CatID) || row.OccurredAt.Before(from) || row.OccurredAt.After(to) {
			continue
		}
		if len(q.Types) > 0 && row.RecordType != q.Types[0] {
			continue
		}
		if q.BeforeOccurredAt != nil && (row.OccurredAt.After(*q.BeforeOccurredAt) || (row.OccurredAt.Equal(*q.BeforeOccurredAt) && row.ID >= q.BeforeID)) {
			continue
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OccurredAt.Equal(out[j].OccurredAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].OccurredAt.After(out[j].OccurredAt)
	})
	if len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

type toolHealthRepo struct {
	profile *model.CatHealthProfile
	err     error
}

func (*toolHealthRepo) Create(context.Context, *model.CatHealthProfile) error { return nil }
func (r *toolHealthRepo) FindByCatID(_ context.Context, _ string) (*model.CatHealthProfile, error) {
	return r.profile, r.err
}
func (*toolHealthRepo) Update(context.Context, *model.CatHealthProfile) error { return nil }

func toolRecord(id, kind, payload string, at time.Time) *model.DailyRecord {
	return &model.DailyRecord{Base: model.Base{ID: id}, FamilyID: "fam", CatIDs: []string{"cat"}, RecordType: kind, Payload: payload, OccurredAt: at, Note: "原始记录"}
}

func toolJSON(t *testing.T, result *AgentToolResult) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(result.Content, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAgentToolDefinitionsAndScope(t *testing.T) {
	defs := AgentToolDefinitions()
	want := []string{"listRecords", "getCatProfile", "getTrends", "createReminderDraft"}
	if len(defs) != len(want) {
		t.Fatalf("tool count = %d", len(defs))
	}
	for i, def := range defs {
		if def.Name != want[i] || !json.Valid(def.Parameters) {
			t.Fatalf("invalid definition %+v", def)
		}
	}
	s, _ := newAgentServiceForTest()
	scope := AgentToolScope{FamilyID: "fam", UserID: "user"}
	for _, tc := range []struct{ name, args string }{
		{"unknown", `{}`},
		{"listRecords", `{"cat_id":"cat","family_id":"other"}`},
		{"listRecords", `{"cat_id":"cat","days":0}`},
		{"listRecords", `{"cat_id":"cat","limit":51}`},
		{"listRecords", `{"cat_id":"cat","type":"unknown"}`},
		{"getTrends", `{"cat_id":"cat","metric":"weight","days":91}`},
		{"getTrends", `{"cat_id":"cat","metric":"blood","days":7}`},
		{"getCatProfile", `{"cat_id":"cat"}{"cat_id":"cat"}`},
	} {
		if _, err := s.ExecuteTool(context.Background(), scope, tc.name, tc.args); err == nil {
			t.Errorf("accepted %s %s", tc.name, tc.args)
		}
	}
	if _, err := s.ExecuteTool(context.Background(), AgentToolScope{FamilyID: "other", UserID: "user"}, "listRecords", `{"cat_id":"cat"}`); err == nil {
		t.Fatal("cross-family member accepted")
	}
	if _, err := s.ExecuteTool(context.Background(), scope, "listRecords", `{"cat_id":"other"}`); err == nil {
		t.Fatal("cross-family cat accepted")
	}
	candidates, err := s.ToolCatCandidates(context.Background(), scope)
	if err != nil || len(candidates) != 1 || candidates[0]["id"] != "cat" {
		t.Fatalf("cat candidates: %+v %v", candidates, err)
	}
}

func TestAgentListRecordsBoundsAndReadFailure(t *testing.T) {
	s, _ := newAgentServiceForTest()
	base := s.clock()
	repo := &toolRecordRepo{rows: []*model.DailyRecord{toolRecord("a", "feeding", `{"amount":30}`, base.Add(-time.Hour)), toolRecord("b", "feeding", `{"amount":20}`, base.Add(-2*time.Hour)), toolRecord("old", "feeding", `{"amount":10}`, base.Add(-10*24*time.Hour))}}
	s.records = repo
	scope := AgentToolScope{FamilyID: "fam", UserID: "user"}
	result, err := s.ExecuteTool(context.Background(), scope, "listRecords", `{"cat_id":"cat","type":"feeding","limit":1}`)
	if err != nil {
		t.Fatal(err)
	}
	data := toolJSON(t, result)
	if !result.Truncated || len(result.Evidence) != 1 || data["truncated"] != true || len(data["records"].([]any)) != 1 {
		t.Fatalf("bad bounded records: %+v %+v", data, result)
	}
	if repo.queries[0].Limit != 2 || repo.queries[0].FamilyID != "fam" || repo.queries[0].CatID != "cat" || repo.queries[0].From == "" || repo.queries[0].To == "" {
		t.Fatalf("unscoped query: %+v", repo.queries[0])
	}
	result, err = s.ExecuteTool(context.Background(), scope, "listRecords", `{"cat_id":"cat","days":1,"limit":50}`)
	if err != nil || len(toolJSON(t, result)["records"].([]any)) != 2 {
		t.Fatalf("record boundary: %v", err)
	}
	repo.err = errors.New("read failed")
	if _, err = s.ExecuteTool(context.Background(), scope, "listRecords", `{"cat_id":"cat"}`); !errors.Is(err, repo.err) {
		t.Fatalf("read error lost: %v", err)
	}
}

func TestAgentGetCatProfileMissingVersusFailure(t *testing.T) {
	s, _ := newAgentServiceForTest()
	profiles := &toolHealthRepo{err: repository.ErrNotFound}
	s.SetHealthProfileRepo(profiles)
	scope := AgentToolScope{FamilyID: "fam", UserID: "user"}
	result, err := s.ExecuteTool(context.Background(), scope, "getCatProfile", `{"cat_id":"cat"}`)
	if err != nil || toolJSON(t, result)["profile_recorded"] != false {
		t.Fatalf("missing profile: %v", err)
	}
	profiles.err = nil
	profiles.profile = &model.CatHealthProfile{CatID: "cat", Diseases: `["慢性病"]`, Allergies: `[]`}
	result, err = s.ExecuteTool(context.Background(), scope, "getCatProfile", `{"cat_id":"cat"}`)
	if err != nil || toolJSON(t, result)["profile_recorded"] != true || len(result.Evidence) != 1 {
		t.Fatalf("existing profile: %v", err)
	}
	profiles.err = errors.New("database failed")
	if _, err = s.ExecuteTool(context.Background(), scope, "getCatProfile", `{"cat_id":"cat"}`); !errors.Is(err, profiles.err) {
		t.Fatalf("read error lost: %v", err)
	}
	profiles.err = nil
	profiles.profile = nil
	if _, err = s.ExecuteTool(context.Background(), scope, "getCatProfile", `{"cat_id":"cat"}`); err != repository.ErrInvalidQuery {
		t.Fatalf("nil profile accepted: %v", err)
	}
}

func TestAgentGetTrendsAggregatesAllPagesAndMarksMissing(t *testing.T) {
	s, _ := newAgentServiceForTest()
	base := s.clock()
	rows := make([]*model.DailyRecord, 0, 1003)
	for i := 0; i < 1002; i++ {
		rows = append(rows, toolRecord(fmt.Sprintf("r%04d", i), "drinking", `{"amount":2}`, base.Add(-time.Duration(i)*time.Second)))
	}
	rows = append(rows, toolRecord("missing", "drinking", `{}`, base.Add(-3*time.Hour)))
	repo := &toolRecordRepo{rows: rows}
	s.records = repo
	result, err := s.ExecuteTool(context.Background(), AgentToolScope{FamilyID: "fam", UserID: "user"}, "getTrends", `{"cat_id":"cat","metric":"drinking","days":2}`)
	if err != nil {
		t.Fatal(err)
	}
	data := toolJSON(t, result)
	points := data["points"].([]any)
	if len(points) != 1 || points[0].(map[string]any)["value"] != float64(2004) || data["records_without_value"] != float64(1) || len(data["missing_dates"].([]any)) != 1 || len(repo.queries) < 2 || !result.Truncated || len(result.Evidence) != 50 {
		t.Fatalf("incomplete trends: pages=%d data=%+v evidence=%d", len(repo.queries), data, len(result.Evidence))
	}
}

func TestAgentGetTrendsKeepsExplicitZero(t *testing.T) {
	s, _ := newAgentServiceForTest()
	s.records = &toolRecordRepo{rows: []*model.DailyRecord{
		toolRecord("zero", "drinking", `{"amount":0}`, s.clock().Add(-time.Hour)),
		toolRecord("unknown", "drinking", `{}`, s.clock().Add(-2*time.Hour)),
	}}
	result, err := s.ExecuteTool(context.Background(), AgentToolScope{FamilyID: "fam", UserID: "user"}, "getTrends", `{"cat_id":"cat","metric":"drinking","days":1}`)
	if err != nil {
		t.Fatal(err)
	}
	data := toolJSON(t, result)
	points := data["points"].([]any)
	if len(points) != 1 || points[0].(map[string]any)["value"] != float64(0) || points[0].(map[string]any)["observations"] != float64(1) || len(data["missing_dates"].([]any)) != 0 || data["records_without_value"] != float64(1) {
		t.Fatalf("zero and missing conflated: %+v", data)
	}
}

func TestAgentReminderToolSavesPendingDraftOnly(t *testing.T) {
	s, repo := newAgentServiceForTest()
	repo.messages["target"] = &model.AgentMessage{Base: model.Base{ID: "target"}, FamilyID: "fam", Role: string(RoleAssistant), Visibility: string(VisibilityFamily), Type: AgentTypeChatAnswer, GeneratedAt: s.clock()}
	scope := AgentToolScope{FamilyID: "fam", UserID: "user", DraftTargetMessageID: "target", ExpectedDraftVersion: 0}
	result, err := s.ExecuteTool(context.Background(), scope, "createReminderDraft", `{"cat_id":"cat","type":"custom","title":"喂药","scheduled_at":"2026-10-02T09:00:00+08:00","timezone":"Asia/Shanghai"}`)
	if err != nil {
		t.Fatal(err)
	}
	if result.Draft == nil || result.Draft.MessageID != "target" || result.Draft.Version != 1 || repo.messages["target"].ActionStatus != DraftPending || len(repo.reminders) != 0 {
		t.Fatalf("draft changed reminder state: %+v %+v", result, repo.reminders)
	}
	if _, err = s.ExecuteTool(context.Background(), scope, "createReminderDraft", `{"cat_id":"cat","title":"again"}`); err == nil {
		t.Fatal("stale draft version accepted")
	}
	if _, err = s.ExecuteTool(context.Background(), AgentToolScope{FamilyID: "fam", UserID: "user"}, "createReminderDraft", `{"cat_id":"cat","title":"test"}`); err == nil {
		t.Fatal("missing server target accepted")
	}
}

func TestAgentReminderToolReusesTurnDraftOnRetry(t *testing.T) {
	s, repo := newAgentServiceForTest()
	scope := AgentToolScope{FamilyID: "fam", UserID: "user", SessionID: "session", TurnID: "turn-1"}
	args := `{"cat_id":"cat","type":"custom","title":"喂药","scheduled_at":"2026-10-02T09:00:00+08:00","timezone":"Asia/Shanghai"}`
	first, err := s.ExecuteTool(context.Background(), scope, "createReminderDraft", args)
	if err != nil || first.Draft == nil {
		t.Fatalf("first draft: %v", err)
	}
	second, err := s.ExecuteTool(context.Background(), scope, "createReminderDraft", args)
	if err != nil || second.Draft == nil || first.Draft.MessageID != second.Draft.MessageID || second.Draft.Version != 1 || len(repo.messages) != 1 || len(repo.reminders) != 0 {
		t.Fatalf("retry duplicated draft: first=%+v second=%+v err=%v", first, second, err)
	}
	if _, err := s.ExecuteTool(context.Background(), scope, "createReminderDraft", `{"cat_id":"cat","title":"different"}`); err == nil {
		t.Fatal("changed content reused the turn")
	}
}
