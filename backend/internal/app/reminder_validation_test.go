package app

import (
	"context"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
)

func TestReminderValidationUsesExplicitScopeAndUTC(t *testing.T) {
	now := time.Date(2026, 9, 28, 16, 0, 0, 0, time.UTC)
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	future := time.Date(2026, 9, 29, 8, 0, 0, 0, shanghai).Add(time.Second)
	cat := catMemory{cat: &model.Cat{FamilyID: "family-a", Base: model.Base{ID: "cat-a"}}}
	family := familyMemory{family: &model.Family{Base: model.Base{ID: "family-a"}, Timezone: "Asia/Shanghai"}}
	input := ReminderInput{CatID: "cat-a", Type: "medication", Title: "  服药提醒  ", ScheduledAt: &future}
	got, err := normalizeReminderInput(context.Background(), cat, family, "family-a", input, now, true, 200)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "服药提醒" || got.Timezone != "Asia/Shanghai" || got.ScheduledAt.Location() != time.UTC || !got.ScheduledAt.Equal(future) {
		t.Fatalf("normalization lost the absolute time or display timezone: %+v", got)
	}
	for _, when := range []time.Time{now, now.Add(-time.Second)} {
		bad := input
		bad.ScheduledAt = &when
		if _, err := normalizeReminderInput(context.Background(), cat, family, "family-a", bad, now, true, 200); err == nil {
			t.Fatal("past or present reminder time must be rejected")
		}
	}
	missingCat := input
	missingCat.CatID = ""
	if _, err := normalizeReminderInput(context.Background(), cat, family, "family-a", missingCat, now, true, 200); err == nil {
		t.Fatal("Agent reminders require an explicit cat or both")
	}
	legacy, err := normalizeReminderInput(context.Background(), cat, family, "family-a", ReminderInput{Title: "旧版提醒"}, now, false, 255)
	if err != nil || legacy.CatID != "both" || legacy.ScheduledAt != nil {
		t.Fatalf("legacy family reminder compatibility changed: %+v, %v", legacy, err)
	}
	foreign := input
	foreign.CatID = "cat-from-another-family"
	if _, err := normalizeReminderInput(context.Background(), cat, family, "family-a", foreign, now, true, 200); err == nil {
		t.Fatal("foreign cat must be rejected")
	}
}
