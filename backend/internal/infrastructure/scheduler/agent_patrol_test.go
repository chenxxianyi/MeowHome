package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/domain/model"
)

type familyListMemory struct{ families []*model.Family }
func (f familyListMemory) Create(context.Context, *model.Family) error { return nil }
func (f familyListMemory) FindByID(context.Context, string) (*model.Family, error) { return nil, nil }
func (f familyListMemory) List(_ context.Context, beforeID string, limit int) ([]*model.Family, error) { out := []*model.Family{}; for _, item := range f.families { if beforeID != "" && item.ID <= beforeID { continue }; out = append(out, item); if len(out) == limit { break } }; return out, nil }
func (f familyListMemory) Update(context.Context, *model.Family) error { return nil }
func (f familyListMemory) Delete(context.Context, string) error { return nil }

type patrolCounter struct{ count map[string]int }
func (p *patrolCounter) PatrolSystem(_ context.Context, familyID string) (*app.AgentPatrolResponse, error) { if p.count == nil { p.count = map[string]int{} }; p.count[familyID]++; return &app.AgentPatrolResponse{}, nil }

func TestParsePatrolTimesAndRunOnceAreDeterministic(t *testing.T) {
	times, err := parsePatrolTimes("20:00,08:00,08:00")
	if err != nil || len(times) != 2 || times[0].hour != 8 || times[1].hour != 20 { t.Fatalf("unexpected parsed times: %+v %v", times, err) }
	counter := &patrolCounter{}
	families := familyListMemory{families: []*model.Family{{Base: model.Base{ID: "fam"}, Timezone: "Asia/Shanghai"}}}
	s, err := NewAgentPatrolScheduler(counter, families, true, "08:00,20:00")
	if err != nil { t.Fatal(err) }
	now := time.Date(2026, 9, 28, 0, 5, 0, 0, time.UTC) // 本地 08:05
	if err := s.RunOnce(context.Background(), now); err != nil { t.Fatal(err) }
	if err := s.RunOnce(context.Background(), now.Add(20*time.Second)); err != nil { t.Fatal(err) }
	if counter.count["fam"] != 1 { t.Fatalf("same patrol slot ran %d times", counter.count["fam"]) }
	if err := s.RunOnce(context.Background(), now.Add(12*time.Hour)); err != nil { t.Fatal(err) }
	if counter.count["fam"] != 2 { t.Fatalf("next patrol slot did not run: %d", counter.count["fam"]) }
}

func TestSchedulerDisabledDoesNotScan(t *testing.T) {
	counter := &patrolCounter{}
	s, err := NewAgentPatrolScheduler(counter, familyListMemory{families: []*model.Family{{Base: model.Base{ID: "fam"}, Timezone: "UTC"}}}, false, "08:00")
	if err != nil { t.Fatal(err) }
	if err := s.RunOnce(context.Background(), time.Now().UTC()); err != nil { t.Fatal(err) }
	if len(counter.count) != 0 { t.Fatal("disabled scheduler ran patrol") }
}
