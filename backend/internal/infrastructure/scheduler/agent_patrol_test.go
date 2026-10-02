package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

type familyListMemory struct{ families []*model.Family }

func (f familyListMemory) Create(context.Context, *model.Family) error             { return nil }
func (f familyListMemory) FindByID(context.Context, string) (*model.Family, error) { return nil, nil }
func (f familyListMemory) List(_ context.Context, beforeID string, limit int) ([]*model.Family, error) {
	out := []*model.Family{}
	for _, item := range f.families {
		if beforeID != "" && item.ID <= beforeID {
			continue
		}
		out = append(out, item)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}
func (f familyListMemory) Update(context.Context, *model.Family) error { return nil }
func (f familyListMemory) Delete(context.Context, string) error        { return nil }

type progressMemory struct {
	rows map[string]*repository.AgentTaskProgress
}

func (p *progressMemory) Get(_ context.Context, familyID, key string) (*repository.AgentTaskProgress, error) {
	if p.rows == nil || p.rows[familyID+"/"+key] == nil {
		return nil, repository.ErrNotFound
	}
	copy := *p.rows[familyID+"/"+key]
	return &copy, nil
}
func (p *progressMemory) Save(_ context.Context, row *repository.AgentTaskProgress) error {
	if p.rows == nil {
		p.rows = map[string]*repository.AgentTaskProgress{}
	}
	copy := *row
	p.rows[row.FamilyID+"/"+row.TaskKey] = &copy
	return nil
}

type eventRecordMemory struct{ rows []*model.DailyRecord }

func (e eventRecordMemory) ListCreatedAfter(_ context.Context, q repository.CreatedRecordQuery) ([]*model.DailyRecord, error) {
	out := []*model.DailyRecord{}
	for _, r := range e.rows {
		if r.FamilyID != q.FamilyID || r.DeletedAt != nil {
			continue
		}
		if q.AfterAt != nil && (r.CreatedAt.Before(*q.AfterAt) || (r.CreatedAt.Equal(*q.AfterAt) && r.ID <= q.AfterID)) {
			continue
		}
		out = append(out, r)
		if len(out) == q.Limit {
			break
		}
	}
	return out, nil
}

type patrolCounter struct {
	calls      map[string][]time.Time
	events     []string
	failFamily string
	failEvent  string
}

func (p *patrolCounter) PatrolSystemAt(_ context.Context, familyID string, at time.Time) (*app.AgentPatrolResponse, error) {
	if p.calls == nil {
		p.calls = map[string][]time.Time{}
	}
	p.calls[familyID] = append(p.calls[familyID], at)
	if familyID == p.failFamily {
		return nil, errors.New("patrol failed")
	}
	return &app.AgentPatrolResponse{}, nil
}
func (p *patrolCounter) PatrolDangerRecordSystem(_ context.Context, familyID, recordID string) error {
	if recordID == p.failEvent {
		return errors.New("event failed")
	}
	p.events = append(p.events, familyID+"/"+recordID)
	return nil
}

func newTestScheduler(t *testing.T, runner *patrolCounter, families []*model.Family, events []*model.DailyRecord, state *progressMemory, enabled bool) *AgentPatrolScheduler {
	t.Helper()
	s, err := NewAgentPatrolScheduler(runner, familyListMemory{families: families}, eventRecordMemory{rows: events}, state, enabled, "08:00,20:00")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPatrolWindowsPersistAcrossRestart(t *testing.T) {
	times, err := parsePatrolTimes("20:00,08:00,08:00")
	if err != nil || len(times) != 2 || times[0].hour != 8 || times[1].hour != 20 {
		t.Fatalf("bad times: %+v %v", times, err)
	}
	fam := &model.Family{Base: model.Base{ID: "fam"}, Timezone: "Asia/Shanghai"}
	state := &progressMemory{}
	counter := &patrolCounter{}
	s := newTestScheduler(t, counter, []*model.Family{fam}, nil, state, true)
	at0759 := time.Date(2026, 9, 27, 23, 59, 0, 0, time.UTC)
	if err := s.RunOnce(context.Background(), at0759); err != nil {
		t.Fatal(err)
	}
	for _, at := range counter.calls["fam"] {
		if at.Equal(time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)) {
			t.Fatal("08:00 ran at 07:59")
		}
	}
	counter.calls = map[string][]time.Time{}
	at0800 := at0759.Add(time.Minute)
	if err := s.RunOnce(context.Background(), at0800); err != nil {
		t.Fatal(err)
	}
	if len(counter.calls["fam"]) != 1 || !counter.calls["fam"][0].Equal(at0800) {
		t.Fatalf("08:00 not run once: %+v", counter.calls)
	}
	restarted := newTestScheduler(t, counter, []*model.Family{fam}, nil, state, true)
	if err := restarted.RunOnce(context.Background(), at0800); err != nil {
		t.Fatal(err)
	}
	if len(counter.calls["fam"]) != 1 {
		t.Fatal("restart repeated successful slot")
	}
	if err := restarted.RunOnce(context.Background(), at0800.Add(12*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(counter.calls["fam"]) != 2 {
		t.Fatalf("20:00 missing: %+v", counter.calls)
	}
	if state.rows["fam/danger-events"].Status != "success" {
		t.Fatal("zero event scan not persisted")
	}
}

func TestSundayWindowCatchupUsesScheduledTime(t *testing.T) {
	fam := &model.Family{Base: model.Base{ID: "fam"}, Timezone: "Asia/Shanghai"}
	s := newTestScheduler(t, &patrolCounter{}, []*model.Family{fam}, nil, &progressMemory{}, true)
	monday0759 := time.Date(2026, 9, 27, 23, 59, 0, 0, time.UTC) // 周一 07:59
	windows := s.dueWindows(fam, monday0759)
	if len(windows) == 0 || windows[len(windows)-1].at.In(time.FixedZone("CST", 8*3600)).Weekday() != time.Sunday || windows[len(windows)-1].at.In(time.FixedZone("CST", 8*3600)).Hour() != 20 {
		t.Fatalf("Sunday 20:00 not selected: %+v", windows)
	}
}

func TestEventCursorReplayFailureAndBackfill(t *testing.T) {
	fam := &model.Family{Base: model.Base{ID: "fam"}, Timezone: "UTC"}
	now := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	events := []*model.DailyRecord{
		{Base: model.Base{ID: "a", CreatedAt: now.Add(-time.Minute)}, FamilyID: "fam", Severity: "normal", OccurredAt: now.AddDate(0, 0, -20)},
		{Base: model.Base{ID: "b", CreatedAt: now}, FamilyID: "fam", Severity: "danger", OccurredAt: now.AddDate(0, 0, -20)},
	}
	state := &progressMemory{}
	runner := &patrolCounter{failEvent: "b"}
	s := newTestScheduler(t, runner, []*model.Family{fam}, events, state, true)
	if err := s.RunOnce(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	p := state.rows["fam/danger-events"]
	if p.CursorID != "a" || p.Status != "failed" {
		t.Fatalf("failure advanced cursor: %+v", p)
	}
	runner.failEvent = ""
	s = newTestScheduler(t, runner, []*model.Family{fam}, events, state, true)
	if err := s.RunOnce(context.Background(), now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(runner.events) != 1 || runner.events[0] != "fam/b" || state.rows["fam/danger-events"].CursorID != "b" {
		t.Fatalf("backfill failed: %+v %+v", runner.events, state.rows)
	}
}

func TestFamilyFailureDoesNotStopNextFamily(t *testing.T) {
	state := &progressMemory{}
	runner := &patrolCounter{failFamily: "a"}
	families := []*model.Family{{Base: model.Base{ID: "a"}, Timezone: "UTC"}, {Base: model.Base{ID: "b"}, Timezone: "UTC"}}
	s := newTestScheduler(t, runner, families, nil, state, true)
	if err := s.RunOnce(context.Background(), time.Date(2026, 9, 28, 20, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls["b"]) == 0 {
		t.Fatal("second family not scanned")
	}
	if state.rows["a/patrol:2026-09-28:20:00"].Status != "failed" {
		t.Fatal("failure not persisted")
	}
}

func TestSchedulerDisabledDoesNotScan(t *testing.T) {
	runner := &patrolCounter{}
	s := newTestScheduler(t, runner, []*model.Family{{Base: model.Base{ID: "fam"}, Timezone: "UTC"}}, nil, &progressMemory{}, false)
	if err := s.RunOnce(context.Background(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 0 {
		t.Fatal("disabled scheduler ran")
	}
}
