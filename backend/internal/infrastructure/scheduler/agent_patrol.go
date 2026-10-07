// Package scheduler 提供单实例 Agent 巡检和事件扫描调度。
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"go.uber.org/zap"
)

type patrolRunner interface {
	PatrolSystemAt(context.Context, string, time.Time) (*app.AgentPatrolResponse, error)
	PatrolDangerRecordSystem(context.Context, string, string) error
}

// AgentPatrolScheduler 串行扫描家庭。多实例部署前需增加数据库领取锁。
type AgentPatrolScheduler struct {
	agent         patrolRunner
	families      repository.FamilyRepo
	records       repository.AgentEventRecordRepo
	progress      repository.AgentTaskProgressRepo
	enabled       bool
	times         []patrolTime
	interval      time.Duration
	clock         func() time.Time
	logf          func(string, ...any)
	familyAllowed func(string) bool
	done          chan struct{}
}

type patrolTime struct{ hour, minute int }

func NewAgentPatrolScheduler(agent patrolRunner, families repository.FamilyRepo, records repository.AgentEventRecordRepo, progress repository.AgentTaskProgressRepo, enabled bool, patrolTimes string) (*AgentPatrolScheduler, error) {
	times, err := parsePatrolTimes(patrolTimes)
	if err != nil {
		return nil, err
	}
	return &AgentPatrolScheduler{agent: agent, families: families, records: records, progress: progress, enabled: enabled, times: times, interval: 30 * time.Second, clock: func() time.Time { return time.Now().UTC() }, logf: func(string, ...any) {}, done: make(chan struct{})}, nil
}

func parsePatrolTimes(raw string) ([]patrolTime, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "08:00,20:00"
	}
	seen := map[patrolTime]bool{}
	out := make([]patrolTime, 0)
	for _, item := range strings.Split(raw, ",") {
		parts := strings.Split(strings.TrimSpace(item), ":")
		if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 2 {
			return nil, fmt.Errorf("invalid agent patrol time %q", item)
		}
		var hour, minute int
		if _, err := fmt.Sscanf(parts[0], "%d", &hour); err != nil || hour < 0 || hour > 23 {
			return nil, fmt.Errorf("invalid agent patrol hour %q", item)
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &minute); err != nil || minute < 0 || minute > 59 {
			return nil, fmt.Errorf("invalid agent patrol minute %q", item)
		}
		pt := patrolTime{hour, minute}
		if !seen[pt] {
			seen[pt] = true
			out = append(out, pt)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("agent patrol times cannot be empty")
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].hour == out[j].hour {
			return out[i].minute < out[j].minute
		}
		return out[i].hour < out[j].hour
	})
	return out, nil
}

func (s *AgentPatrolScheduler) SetFamilyFilter(allowed func(string) bool) { s.familyAllowed = allowed }

func (s *AgentPatrolScheduler) SetLogger(logger *zap.Logger) {
	if logger != nil {
		s.logf = func(event string, _ ...any) { logger.Warn("agent_scheduler_failure", zap.String("event", event)) }
	}
}

// Start 立即补跑最近两个到期时段，随后每 30 秒扫描。Wait 用于退出等待。
func (s *AgentPatrolScheduler) Start(ctx context.Context) {
	if !s.enabled || s.agent == nil || s.families == nil || s.records == nil || s.progress == nil {
		close(s.done)
		return
	}
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			if err := s.RunOnce(ctx, s.clock().UTC()); err != nil && ctx.Err() == nil {
				s.logf("agent patrol scan failed: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *AgentPatrolScheduler) Wait(ctx context.Context) error {
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RunOnce 按家庭游标扫描；单家庭失败不影响后续家庭。
func (s *AgentPatrolScheduler) RunOnce(ctx context.Context, now time.Time) error {
	if !s.enabled {
		return nil
	}
	if s.agent == nil || s.families == nil || s.records == nil || s.progress == nil {
		return repository.ErrInvalidQuery
	}
	beforeID := ""
	const pageSize = 50
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		families, err := s.families.List(ctx, beforeID, pageSize)
		if err != nil {
			return err
		}
		for _, family := range families {
			if s.familyAllowed != nil && !s.familyAllowed(family.ID) {
				continue
			}
			if family == nil || family.DeletedAt != nil {
				continue
			}
			familyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			if err := s.runFamily(familyCtx, family, now); err != nil && ctx.Err() == nil {
				s.logf("agent patrol failed family=%s: %v", family.ID, err)
			}
			cancel()
		}
		if len(families) < pageSize {
			return nil
		}
		last := families[len(families)-1]
		if last == nil || last.ID == beforeID {
			return repository.ErrInvalidQuery
		}
		beforeID = last.ID
	}
}

type dueWindow struct {
	key string
	at  time.Time
}

func (s *AgentPatrolScheduler) dueWindows(family *model.Family, now time.Time) []dueWindow {
	loc, err := time.LoadLocation(family.Timezone)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	local := now.In(loc)
	date := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	all := make([]dueWindow, 0, 2*len(s.times))
	for _, day := range []time.Time{date.AddDate(0, 0, -1), date} {
		for _, slot := range s.times {
			at := time.Date(day.Year(), day.Month(), day.Day(), slot.hour, slot.minute, 0, 0, loc)
			if !at.After(now) {
				all = append(all, dueWindow{fmt.Sprintf("patrol:%s:%02d:%02d", day.Format("2006-01-02"), slot.hour, slot.minute), at.UTC()})
			}
		}
	}
	if len(all) > 2 {
		return all[len(all)-2:]
	}
	return all
}

func (s *AgentPatrolScheduler) runFamily(ctx context.Context, family *model.Family, now time.Time) error {
	var firstErr error
	for _, slot := range s.dueWindows(family, now) {
		p, err := s.getProgress(ctx, family.ID, slot.key)
		if err != nil {
			return err
		}
		if p.Status == "success" {
			continue
		}
		_, err = s.agent.PatrolSystemAt(ctx, family.ID, slot.at)
		if saveErr := s.saveResult(ctx, p, err, now); saveErr != nil {
			return saveErr
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if err := s.scanEvents(ctx, family.ID, now); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (s *AgentPatrolScheduler) getProgress(ctx context.Context, familyID, key string) (*repository.AgentTaskProgress, error) {
	p, err := s.progress.Get(ctx, familyID, key)
	if errors.Is(err, repository.ErrNotFound) {
		return &repository.AgentTaskProgress{FamilyID: familyID, TaskKey: key}, nil
	}
	return p, err
}

func (s *AgentPatrolScheduler) saveResult(ctx context.Context, p *repository.AgentTaskProgress, runErr error, now time.Time) error {
	now = now.UTC()
	if runErr == nil {
		p.Status, p.LastSuccessAt = "success", &now
	} else {
		p.Status, p.LastFailureAt = "failed", &now
	}
	return s.progress.Save(ctx, p)
}

func (s *AgentPatrolScheduler) scanEvents(ctx context.Context, familyID string, now time.Time) error {
	p, err := s.getProgress(ctx, familyID, "danger-events")
	if err != nil {
		return err
	}
	const batchSize = 100
	// 每次唤醒最多处理 100 条；下轮沿持久化游标继续。
	rows, err := s.records.ListCreatedAfter(ctx, repository.CreatedRecordQuery{FamilyID: familyID, AfterAt: p.CursorCreatedAt, AfterID: p.CursorID, Limit: batchSize})
	if err != nil {
		_ = s.saveResult(ctx, p, err, now)
		return err
	}
	for _, row := range rows {
		if row == nil {
			return repository.ErrInvalidQuery
		}
		if row.Severity == "danger" {
			if err := s.agent.PatrolDangerRecordSystem(ctx, familyID, row.ID); err != nil {
				_ = s.saveResult(ctx, p, err, now)
				return err
			}
		}
		at := row.CreatedAt.UTC()
		p.CursorCreatedAt, p.CursorID = &at, row.ID
		if err := s.progress.Save(ctx, p); err != nil {
			return err
		}
	}
	return s.saveResult(ctx, p, nil, now)
}
