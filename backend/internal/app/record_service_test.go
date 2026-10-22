package app

import (
	"context"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

type recordServiceMembers struct {
	repository.MemberRepo
	familyIDs []string
}

func (m recordServiceMembers) FindByUser(context.Context, string) ([]*model.Member, error) {
	rows := make([]*model.Member, 0, len(m.familyIDs))
	for _, familyID := range m.familyIDs {
		rows = append(rows, &model.Member{FamilyID: familyID})
	}
	return rows, nil
}

type recordServiceCats struct {
	repository.CatRepo
	cats map[string]*model.Cat
}

func (c recordServiceCats) FindByID(_ context.Context, id string) (*model.Cat, error) {
	cat := c.cats[id]
	if cat == nil {
		return nil, repository.ErrNotFound
	}
	return cat, nil
}

type recordServiceRecords struct {
	repository.DailyRecordRepo
	rows map[string]*model.DailyRecord
}

func (r *recordServiceRecords) Create(_ context.Context, row *model.DailyRecord) error {
	r.rows[row.ID] = row
	return nil
}

func (r *recordServiceRecords) FindByID(_ context.Context, id string) (*model.DailyRecord, error) {
	row := r.rows[id]
	if row == nil || row.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}
	return row, nil
}

func (r *recordServiceRecords) Update(_ context.Context, row *model.DailyRecord, expectedUpdatedAt time.Time) error {
	existing := r.rows[row.ID]
	if existing == nil || existing.DeletedAt != nil || existing.FamilyID != row.FamilyID {
		return repository.ErrNotFound
	}
	if !existing.UpdatedAt.Equal(expectedUpdatedAt) {
		return repository.ErrConflict
	}
	r.rows[row.ID] = row
	return nil
}

func (r *recordServiceRecords) Delete(_ context.Context, familyID, id string) error {
	row := r.rows[id]
	if row == nil || row.DeletedAt != nil || row.FamilyID != familyID {
		return repository.ErrNotFound
	}
	now := time.Now().UTC()
	row.DeletedAt = &now
	return nil
}

type recordServiceAudit struct {
	repository.AuditRepo
	actions []string
}

func (a *recordServiceAudit) Create(_ context.Context, row *model.AuditLog) error {
	a.actions = append(a.actions, row.Action)
	return nil
}

func newRecordServiceForTest() (*RecordService, *recordServiceRecords, *recordServiceAudit) {
	records := &recordServiceRecords{rows: map[string]*model.DailyRecord{}}
	audit := &recordServiceAudit{}
	cats := recordServiceCats{cats: map[string]*model.Cat{
		"cat-a": {Base: model.Base{ID: "cat-a"}, FamilyID: "family-a"},
		"cat-b": {Base: model.Base{ID: "cat-b"}, FamilyID: "family-b"},
	}}
	service := NewRecordService(records, recordServiceMembers{familyIDs: []string{"family-a"}}, audit, cats)
	return service, records, audit
}

func TestRecordCreateRejectsCatsOutsideFamilyAndDuplicateIDs(t *testing.T) {
	service, records, _ := newRecordServiceForTest()
	for _, tc := range []struct {
		name   string
		catIDs []string
	}{
		{name: "foreign cat", catIDs: []string{"cat-b"}},
		{name: "duplicate cat", catIDs: []string{"cat-a", "cat-a"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), "family-a", "user-a", RecordInput{Type: "feeding", CatIDs: tc.catIDs})
			if err == nil {
				t.Fatal("expected validation error")
			}
			if len(records.rows) != 0 {
				t.Fatalf("invalid record was persisted: %d rows", len(records.rows))
			}
		})
	}
}

func TestRecordUpdatePreservesCreationMetadataAndDeleteHidesRecord(t *testing.T) {
	service, records, audit := newRecordServiceForTest()
	createdAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	records.rows["record-a"] = &model.DailyRecord{
		Base:       model.Base{ID: "record-a", CreatedBy: "original-user", CreatedAt: createdAt, UpdatedAt: createdAt},
		FamilyID:   "family-a",
		CatIDs:     []string{"cat-a"},
		RecordType: "feeding",
		OccurredAt: createdAt,
		Source:     "manual",
		Severity:   "normal",
		Title:      "原标题",
	}

	updated, err := service.Update(context.Background(), "family-a", "editor-user", "record-a", RecordInput{
		Type: "feeding", CatIDs: []string{"cat-a"}, OccurredAt: time.Now().UTC().Format(time.RFC3339),
		ExpectedUpdatedAt: createdAt.Format(time.RFC3339Nano),
		Severity:          "normal", Title: "已更正", Note: "更正说明", Source: "manual",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.CreatedBy != "original-user" || updated.CreatedAt != createdAt.Format(time.RFC3339Nano) {
		t.Fatalf("creation metadata changed: %+v", updated)
	}
	if updated.UpdatedAt == createdAt.Format(time.RFC3339Nano) || updated.Title != "已更正" {
		t.Fatalf("record was not updated: %+v", updated)
	}
	if _, err := service.Update(context.Background(), "family-a", "editor-user", "record-a", RecordInput{
		Type: "feeding", CatIDs: []string{"cat-a"}, ExpectedUpdatedAt: createdAt.Format(time.RFC3339Nano), Title: "过期版本",
	}); err == nil {
		t.Fatal("stale record version was accepted")
	}

	if err := service.Delete(context.Background(), "family-a", "editor-user", "record-a"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := service.Get(context.Background(), "family-a", "editor-user", "record-a"); err == nil {
		t.Fatal("soft-deleted record remained readable")
	}
	if got := audit.actions; len(got) != 2 || got[0] != "record.update" || got[1] != "record.delete" {
		t.Fatalf("audit actions = %v", got)
	}
}

func TestRecordMutationCannotAccessAnotherFamily(t *testing.T) {
	service, records, _ := newRecordServiceForTest()
	createdAt := time.Now().UTC().Truncate(time.Second)
	foreign := &model.DailyRecord{
		Base:       model.Base{ID: "record-b", CreatedBy: "user-b", CreatedAt: createdAt, UpdatedAt: createdAt},
		FamilyID:   "family-b",
		CatIDs:     []string{"cat-b"},
		RecordType: "feeding",
		OccurredAt: createdAt,
		Title:      "保留的记录",
	}
	records.rows[foreign.ID] = foreign

	in := RecordInput{Type: "feeding", CatIDs: []string{"cat-a"}, Title: "不应写入"}
	if _, err := service.Update(context.Background(), "family-a", "user-a", foreign.ID, in); err == nil {
		t.Fatal("cross-family update was accepted")
	}
	if err := service.Delete(context.Background(), "family-a", "user-a", foreign.ID); err == nil {
		t.Fatal("cross-family delete was accepted")
	}
	if foreign.Title != "保留的记录" || foreign.DeletedAt != nil {
		t.Fatalf("foreign record changed: %+v", foreign)
	}
}

func TestGenericRecordMutationCannotChangeVaccineLinkage(t *testing.T) {
	service, records, _ := newRecordServiceForTest()
	createdAt := time.Now().UTC().Add(-time.Hour)
	ordinary := &model.DailyRecord{
		Base:       model.Base{ID: "feeding-a", CreatedBy: "user-a", CreatedAt: createdAt, UpdatedAt: createdAt},
		FamilyID:   "family-a",
		CatIDs:     []string{"cat-a"},
		RecordType: "feeding",
		OccurredAt: createdAt,
		Title:      "喂食记录",
	}
	vaccine := &model.DailyRecord{
		Base:       model.Base{ID: "vaccine-a", CreatedBy: "user-a", CreatedAt: createdAt, UpdatedAt: createdAt},
		FamilyID:   "family-a",
		CatIDs:     []string{"cat-a"},
		RecordType: "vaccine",
		OccurredAt: createdAt,
		Title:      "猫三联",
	}
	records.rows[ordinary.ID] = ordinary
	records.rows[vaccine.ID] = vaccine

	input := RecordInput{Type: "vaccine", CatIDs: []string{"cat-a"}, Title: "更正后的接种记录"}
	if _, err := service.Update(context.Background(), "family-a", "user-a", ordinary.ID, input); err == nil {
		t.Fatal("generic update changed an ordinary record into a vaccine record")
	}
	if _, err := service.Update(context.Background(), "family-a", "user-a", vaccine.ID, input); err == nil {
		t.Fatal("generic update changed a vaccine record")
	}
	if err := service.Delete(context.Background(), "family-a", "user-a", vaccine.ID); err == nil {
		t.Fatal("generic delete changed a vaccine record")
	}
	if vaccine.Title != "猫三联" || vaccine.DeletedAt != nil {
		t.Fatalf("vaccine linkage source changed: %+v", vaccine)
	}
}
