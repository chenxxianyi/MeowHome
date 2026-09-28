// Package scheduler 提供首版单实例 Agent 巡检调度。
package scheduler

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

type patrolRunner interface {
	PatrolSystem(context.Context, string) (*app.AgentPatrolResponse, error)
}

// AgentPatrolScheduler 按家庭时区触发日巡检。状态保存在进程内，部署多实例前仍需数据库领取锁。
type AgentPatrolScheduler struct {
	agent       patrolRunner
	families    repository.FamilyRepo
	enabled     bool
	times       []patrolTime
	interval    time.Duration
	clock       func() time.Time
	logf        func(string, ...any)
	mu          sync.Mutex
	lastRunKey  map[string]string
}

type patrolTime struct {
	hour   int
	minute int
}

// NewAgentPatrolScheduler 创建调度器。patrolTimes 使用逗号分隔的 HH:MM。
func NewAgentPatrolScheduler(agent patrolRunner, families repository.FamilyRepo, enabled bool, patrolTimes string) (*AgentPatrolScheduler, error) {
	times, err := parsePatrolTimes(patrolTimes)
	if err != nil {
		return nil, err
	}
	return &AgentPatrolScheduler{agent: agent, families: families, enabled: enabled, times: times, interval: 30 * time.Second, clock: func() time.Time { return time.Now().UTC() }, logf: func(string, ...any) {}, lastRunKey: map[string]string{}}, nil
}

func parsePatrolTimes(raw string) ([]patrolTime, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "08:00,20:00"
	}
	seen := map[patrolTime]bool{}
	out := make([]patrolTime, 0)
	for _, item := range strings.Split(raw, ",") {
		parts := strings.Split(strings.TrimSpace(item), ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid agent patrol time %q", item)
		}
		var hour, minute int
		if _, err := fmt.Sscanf(parts[0], "%d", &hour); err != nil || hour < 0 || hour > 23 {
			return nil, fmt.Errorf("invalid agent patrol hour %q", item)
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &minute); err != nil || minute < 0 || minute > 59 {
			return nil, fmt.Errorf("invalid agent patrol minute %q", item)
		}
		pt := patrolTime{hour: hour, minute: minute}
		if !seen[pt] {
			seen[pt] = true
			out = append(out, pt)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("agent patrol times cannot be empty")
	}
	sort.Slice(out, func(i, j int) bool { if out[i].hour == out[j].hour { return out[i].minute < out[j].minute }; return out[i].hour < out[j].hour })
	return out, nil
}

// Start 启动单实例 ticker；关闭 ctx 后等待当前家庭任务返回。
func (s *AgentPatrolScheduler) Start(ctx context.Context) {
	if !s.enabled || s.agent == nil || s.families == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.RunOnce(ctx, s.clock().UTC()); err != nil {
					s.logf("agent patrol scan failed: %v", err)
				}
			}
		}
	}()
}

// RunOnce 扫描家庭并执行当前最近一个巡检时点，供启动补跑和测试使用。
func (s *AgentPatrolScheduler) RunOnce(ctx context.Context, now time.Time) error {
	if !s.enabled {
		return nil
	}
	beforeID := ""
	const pageSize = 50
	for {
		families, err := s.families.List(ctx, beforeID, pageSize)
		if err != nil {
			return err
		}
		for _, family := range families {
			if family == nil || family.DeletedAt != nil {
				continue
			}
			slot, key, due := s.dueSlot(family, now)
			_ = slot
			if !due || s.alreadyRun(family.ID, key) {
				continue
			}
			familyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			_, patrolErr := s.agent.PatrolSystem(familyCtx, family.ID)
			cancel()
			if patrolErr != nil {
				s.logf("agent patrol failed family=%s: %v", family.ID, patrolErr)
				continue
			}
			s.markRun(family.ID, key)
		}
		if len(families) < pageSize {
			return nil
		}
		last := families[len(families)-1]
		if last == nil || last.ID == beforeID {
			return nil
		}
		beforeID = last.ID
	}
}

func (s *AgentPatrolScheduler) dueSlot(family *model.Family, now time.Time) (patrolTime, string, bool) {
	loc, err := time.LoadLocation(family.Timezone)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	local := now.In(loc)
	date := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	selectedDate := date
	selected := patrolTime{}
	found := false
	for _, slot := range s.times {
		scheduled := time.Date(date.Year(), date.Month(), date.Day(), slot.hour, slot.minute, 0, 0, loc)
		if !local.Before(scheduled) {
			selected, found = slot, true
		}
	}
	if !found {
		selected = s.times[len(s.times)-1]
		selectedDate = date.AddDate(0, 0, -1)
	}
	key := fmt.Sprintf("%s|%02d:%02d", selectedDate.Format("2006-01-02"), selected.hour, selected.minute)
	return selected, key, true
}

func (s *AgentPatrolScheduler) alreadyRun(familyID, key string) bool { s.mu.Lock(); defer s.mu.Unlock(); return s.lastRunKey[familyID] == key }
func (s *AgentPatrolScheduler) markRun(familyID, key string) { s.mu.Lock(); defer s.mu.Unlock(); s.lastRunKey[familyID] = key }
