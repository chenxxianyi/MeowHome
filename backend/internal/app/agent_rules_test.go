package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
)

func TestEvaluateRulesDeterministicBoundaries(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 27, 20, 0, 0, 0, loc).UTC() // 周日 20:00
	cats := []*model.Cat{{Base: model.Base{ID: "cat-a"}, FamilyID: "fam-a", Name: "小白"}, {Base: model.Base{ID: "cat-b"}, FamilyID: "fam-a", Name: "小黑"}}
	oldWeight := now.AddDate(0, 0, -15)
	records := []*model.DailyRecord{
		{Base: model.Base{ID: "danger-1"}, FamilyID: "fam-a", CatIDs: []string{"cat-a"}, RecordType: "vomit", Severity: "danger", OccurredAt: now.Add(-30 * time.Hour)},
		{Base: model.Base{ID: "weight-1"}, FamilyID: "fam-a", CatIDs: []string{"cat-a"}, RecordType: "weight", OccurredAt: oldWeight},
	}
	reminders := []*model.Reminder{{Base: model.Base{ID: "rem-1"}, FamilyID: "fam-a", CatID: "cat-a", Type: "medication", Title: "晚间用药", State: "todo", ScheduledAt: ptrTime(now.Add(-25 * time.Hour))}}
	got := evaluateRules(now, loc, now.AddDate(0, 0, -30), cats, records, reminders)
	seen := map[string]bool{}
	for _, item := range got {
		seen[item.RuleID] = true
		if item.DedupKey == "" || len(item.Evidence) == 0 {
			t.Fatalf("rule %s missing dedup/evidence", item.RuleID)
		}
	}
	for _, rule := range []string{RuleR01, RuleR03, RuleR04} {
		if !seen[rule] {
			t.Errorf("expected %s", rule)
		}
	}
}

func ptrTime(v time.Time) *time.Time { return &v }

func TestEvaluateRulesR02RequiresThreeSameType(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, loc).UTC()
	cat := []*model.Cat{{Base: model.Base{ID: "cat-a"}, FamilyID: "fam-a", Name: "小白"}}
	makeRecords := func(n int) []*model.DailyRecord {
		out := make([]*model.DailyRecord, n)
		for i := range out {
			out[i] = &model.DailyRecord{Base: model.Base{ID: fmt.Sprintf("v-%d", i)}, FamilyID: "fam-a", CatIDs: []string{"cat-a"}, RecordType: "vomit", OccurredAt: now.Add(-time.Duration(i+1) * 24 * time.Hour)}
		}
		return out
	}
	if got := evaluateRules(now, loc, now.AddDate(0, 0, -30), cat, makeRecords(2), nil); hasRule(got, RuleR02) {
		t.Fatal("two vomiting records must not trigger R-02")
	}
	if got := evaluateRules(now, loc, now.AddDate(0, 0, -30), cat, makeRecords(3), nil); !hasRule(got, RuleR02) {
		t.Fatal("three vomiting records must trigger R-02")
	}
}

func TestEvaluateRulesR05IsLeftClosedRightOpen(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, loc).UTC()
	cat := []*model.Cat{{Base: model.Base{ID: "cat-a"}, FamilyID: "fam-a", Name: "小白"}}
	for _, at := range []time.Time{now, now.Add(6 * 24 * time.Hour)} {
		got := evaluateRules(now, loc, now.AddDate(0, 0, -30), cat, nil, []*model.Reminder{{Base: model.Base{ID: "r"}, FamilyID: "fam-a", CatID: "cat-a", Type: "vaccine", Title: "疫苗", State: "todo", ScheduledAt: ptrTime(at)}})
		if !hasRule(got, RuleR05) {
			t.Fatalf("expected R-05 at %v", at)
		}
	}
	at := now.Add(7 * 24 * time.Hour)
	got := evaluateRules(now, loc, now.AddDate(0, 0, -30), cat, nil, []*model.Reminder{{Base: model.Base{ID: "r"}, FamilyID: "fam-a", CatID: "cat-a", Type: "vaccine", Title: "疫苗", State: "todo", ScheduledAt: ptrTime(at)}})
	if hasRule(got, RuleR05) {
		t.Fatal("exactly seven days must not trigger R-05")
	}
}

func TestEvaluateRulesR06EmitsOneFamilyMessage(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, loc).UTC()
	cats := []*model.Cat{{Base: model.Base{ID: "a"}, FamilyID: "fam-a", Name: "小白"}, {Base: model.Base{ID: "b"}, FamilyID: "fam-a", Name: "小黑"}}
	got := evaluateRules(now, loc, now.AddDate(0, 0, -30), cats, nil, nil)
	count := 0
	for _, item := range got {
		if item.RuleID == RuleR06 {
			count++
			if item.CatID != "" || item.Scope != "family" {
				t.Fatal("R-06 must be a family-level message")
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected one R-06, got %d", count)
	}
}

func TestEvaluateRulesR04DoesNotFlagNewFamilyOrExactBoundary(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 27, 20, 0, 0, 0, loc).UTC()
	cat := []*model.Cat{{Base: model.Base{ID: "a"}, FamilyID: "fam-a", Name: "小白"}}
	exactWeight := &model.DailyRecord{Base: model.Base{ID: "w"}, FamilyID: "fam-a", CatIDs: []string{"a"}, RecordType: "weight", OccurredAt: now.Add(-14 * 24 * time.Hour)}
	if got := evaluateRules(now, loc, now.AddDate(0, 0, -30), cat, []*model.DailyRecord{exactWeight}, nil); hasRule(got, RuleR04) {
		t.Fatal("exactly fourteen elapsed days must not trigger R-04")
	}
	if got := evaluateRules(now, loc, now.Add(-13*24*time.Hour), cat, nil, nil); hasRule(got, RuleR04) {
		t.Fatal("family created less than fourteen days ago must not trigger R-04")
	}
}

func TestEvaluateRulesR06SkipsFamilyCreatedYesterday(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, loc).UTC()
	cat := []*model.Cat{{Base: model.Base{ID: "a"}, FamilyID: "fam-a", Name: "小白"}}
	if got := evaluateRules(now, loc, now.Add(-12*time.Hour), cat, nil, nil); hasRule(got, RuleR06) {
		t.Fatal("family without a complete yesterday must not trigger R-06")
	}
}

func hasRule(items []RuleCandidate, id string) bool {
	for _, item := range items {
		if item.RuleID == id {
			return true
		}
	}
	return false
}
