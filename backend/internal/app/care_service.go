package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
)

// CareService 由日常记录聚合出「今日状态 / 焦点项 / 趋势」。
type CareService struct {
	records   repository.DailyRecordRepo
	reminders repository.ReminderRepo
	cats      repository.CatRepo
	families  repository.FamilyRepo
	members   repository.MemberRepo
}

// NewCareService 创建照护聚合服务。
func NewCareService(
	records repository.DailyRecordRepo,
	reminders repository.ReminderRepo,
	cats repository.CatRepo,
	families repository.FamilyRepo,
	members repository.MemberRepo,
) *CareService {
	return &CareService{records: records, reminders: reminders, cats: cats, families: families, members: members}
}

// --- 响应结构（字段名与前端 types 严格对齐）---

// MetricRow 带目标值的指标行。
type MetricRow struct {
	Amount   float64 `json:"amount"`
	Expected float64 `json:"expected"`
	Unit     string  `json:"unit"`
	State    string  `json:"state"`
	Label    string  `json:"label"`
}

// StateRow 纯状态行。
type StateRow struct {
	State string `json:"state"`
	Label string `json:"label"`
}

// VomitRow 呕吐行。
type VomitRow struct {
	Count int    `json:"count"`
	State string `json:"state"`
	Label string `json:"label"`
}

// MedRow 用药行。
type MedRow struct {
	State string `json:"state"`
	Label string `json:"label"`
	Time  string `json:"time,omitempty"`
}

// TodayStatusData 今日状态。
type TodayStatusData struct {
	Food        MetricRow `json:"food"`
	Water       MetricRow `json:"water"`
	Elimination StateRow  `json:"elimination"`
	Vomit       VomitRow  `json:"vomit"`
	Medication  MedRow    `json:"medication"`
	Mental      StateRow  `json:"mental"`
}

// TrendEvent 趋势图上的事件标记。
type TrendEvent struct {
	Type  string `json:"type"`
	Label string `json:"label"`
}

// TrendPoint 单个趋势数据点。
type TrendPoint struct {
	Date   string       `json:"date"`
	Weight float64      `json:"weight"`
	Food   float64      `json:"food"`
	Water  float64      `json:"water"`
	Poop   float64      `json:"poop"`
	Vomit  float64      `json:"vomit"`
	Mental string       `json:"mental"`
	Events []TrendEvent `json:"events"`
}

// FocusAction 焦点项上的操作按钮。
type FocusAction struct {
	Label  string `json:"label"`
	Action string `json:"action"`
}

// FocusItem 需要主人关注的条目。
type FocusItem struct {
	ID       string        `json:"id"`
	CatID    string        `json:"catId"`
	Type     string        `json:"type"`
	Severity string        `json:"severity"`
	Title    string        `json:"title"`
	Body     string        `json:"body"`
	Evidence []string      `json:"evidence"`
	Actions  []FocusAction `json:"actions"`
}

// --- 今日状态 ---

// TodayStatus 聚合指定猫咪今日的状态。
func (s *CareService) TodayStatus(ctx context.Context, familyID, userID, catID string) (*TodayStatusData, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	loc := s.location(ctx, familyID)
	now := time.Now().In(loc)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.Add(24*time.Hour - time.Millisecond)

	rows, err := s.records.ListByFamily(ctx, repository.DailyRecordQuery{
		FamilyID: familyID,
		CatID:    catID,
		From:     dayStart.UTC().Format(time.RFC3339),
		To:       dayEnd.UTC().Format(time.RFC3339),
		Limit:    500,
	})
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to load records", err)
	}

	out := &TodayStatusData{
		Food:        MetricRow{Expected: 80, Unit: "g", State: "none", Label: "今日暂无喂食记录"},
		Water:       MetricRow{Expected: 150, Unit: "ml", State: "none", Label: "今日暂无饮水记录"},
		Elimination: StateRow{State: "none", Label: "今日暂无排便记录"},
		Vomit:       VomitRow{State: "none", Label: "无呕吐"},
		Medication:  MedRow{State: "none", Label: "无需用药"},
		Mental:      StateRow{State: "none", Label: "今日暂无精神记录"},
	}

	for _, r := range rows {
		p := decodePayload(r.Payload)
		switch r.RecordType {
		case "feeding":
			out.Food.Amount += num(p, "consumedAmount", "consumed", "amount")
			out.Food.State, out.Food.Label = rateMetric(out.Food.Amount, out.Food.Expected, "进食")
		case "drinking":
			out.Water.Amount += num(p, "amount")
			out.Water.State, out.Water.Label = rateMetric(out.Water.Amount, out.Water.Expected, "饮水")
		case "elimination":
			out.Elimination = StateRow{State: severityToState(r.Severity), Label: eliminationLabel(p, r.Severity)}
		case "vomit":
			c := int(num(p, "count"))
			if c == 0 {
				c = 1
			}
			out.Vomit.Count += c
			out.Vomit.State = "danger"
			out.Vomit.Label = fmt.Sprintf("今日呕吐 %d 次", out.Vomit.Count)
		case "mental":
			lv, _ := p["level"].(string)
			if lv == "normal" || lv == "" {
				out.Mental = StateRow{State: "normal", Label: "精神状态正常"}
			} else {
				out.Mental = StateRow{State: "warning", Label: "精神" + lv}
			}
		}
	}

	// 用药状态来自未完成的用药提醒
	if s.reminders != nil {
		pending, err := s.reminders.List(ctx, repository.ReminderQuery{FamilyID: familyID, CatID: catID, Status: "todo"})
		if err == nil {
			for _, rem := range pending {
				if rem.Type == "medication" {
					out.Medication = MedRow{State: "warning", Label: "待服药：" + rem.Title, Time: rem.TimeLabel}
					break
				}
			}
		}
	}

	return out, nil
}

// --- 趋势 ---

// Trends 返回最近 days 天的趋势数据点（按日期升序）。
func (s *CareService) Trends(ctx context.Context, familyID, userID, catID string, days int) ([]TrendPoint, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	if days <= 0 || days > 365 {
		days = 30
	}
	loc := s.location(ctx, familyID)
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	from := today.AddDate(0, 0, -(days - 1))

	rows, err := s.records.ListByFamily(ctx, repository.DailyRecordQuery{
		FamilyID: familyID,
		CatID:    catID,
		From:     from.UTC().Format(time.RFC3339),
		To:       today.Add(24*time.Hour - time.Millisecond).UTC().Format(time.RFC3339),
		Limit:    1000,
	})
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to load records", err)
	}

	byDay := map[string]*TrendPoint{}
	points := make([]TrendPoint, 0, days)
	for i := 0; i < days; i++ {
		d := from.AddDate(0, 0, i).Format("2006-01-02")
		byDay[d] = &TrendPoint{Date: d, Mental: "normal", Events: []TrendEvent{}}
	}

	for _, r := range rows {
		key := r.OccurredAt.In(loc).Format("2006-01-02")
		pt, ok := byDay[key]
		if !ok {
			continue
		}
		p := decodePayload(r.Payload)
		switch r.RecordType {
		case "weight":
			if w := num(p, "weight"); w > 0 {
				pt.Weight = w
			}
		case "feeding":
			pt.Food += num(p, "consumedAmount", "consumed", "amount")
		case "drinking":
			pt.Water += num(p, "amount")
		case "elimination":
			pt.Poop++
		case "vomit":
			c := num(p, "count")
			if c == 0 {
				c = 1
			}
			pt.Vomit += c
			pt.Events = append(pt.Events, TrendEvent{Type: "vomit", Label: "呕吐"})
		case "medication":
			pt.Events = append(pt.Events, TrendEvent{Type: "medication", Label: "用药"})
		case "mental":
			if lv, _ := p["level"].(string); lv != "" && lv != "normal" {
				pt.Mental = "low"
			}
		case "visit":
			pt.Events = append(pt.Events, TrendEvent{Type: "medical", Label: "就诊"})
		}
	}

	for i := 0; i < days; i++ {
		points = append(points, *byDay[from.AddDate(0, 0, i).Format("2006-01-02")])
	}
	return points, nil
}

// --- 焦点项 ---

// FocusItems 汇总当前需要关注的条目（异常记录 + 待办用药）。
func (s *CareService) FocusItems(ctx context.Context, familyID, userID string) ([]FocusItem, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}

	out := []FocusItem{}
	loc := s.location(ctx, familyID)
	now := time.Now().In(loc)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	rows, err := s.records.ListByFamily(ctx, repository.DailyRecordQuery{
		FamilyID: familyID,
		From:     dayStart.AddDate(0, 0, -2).UTC().Format(time.RFC3339),
		To:       dayStart.Add(24*time.Hour - time.Millisecond).UTC().Format(time.RFC3339),
		Types:    []string{"vomit", "symptom", "mental"},
		Limit:    100,
	})
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to load records", err)
	}

	for _, r := range rows {
		if r.Severity != "danger" && r.Severity != "warning" {
			continue
		}
		catID := ""
		if len(r.CatIDs) > 0 {
			catID = r.CatIDs[0]
		}
		title := r.Title
		if title == "" {
			title = focusTitle(r)
		}
		out = append(out, FocusItem{
			ID:       r.ID,
			CatID:    catID,
			Type:     r.RecordType,
			Severity: r.Severity,
			Title:    title,
			Body:     focusBody(r),
			Evidence: []string{r.OccurredAt.In(loc).Format("01-02 15:04") + " " + labelOf(r.RecordType)},
			Actions:  focusActions(r.RecordType),
		})
	}

	if s.reminders != nil {
		pending, err := s.reminders.List(ctx, repository.ReminderQuery{FamilyID: familyID, Status: "todo"})
		if err == nil {
			for _, rem := range pending {
				if rem.Type != "medication" {
					continue
				}
				out = append(out, FocusItem{
					ID:       rem.ID,
					CatID:    rem.CatID,
					Type:     "medication",
					Severity: "info",
					Title:    rem.Title,
					Body:     rem.Subtitle,
					Evidence: []string{"用药计划"},
					Actions: []FocusAction{
						{Label: "完成服药", Action: "done"},
						{Label: "稍后提醒", Action: "later"},
					},
				})
			}
		}
	}

	// danger 排在前，其次 warning，最后 info
	rank := map[string]int{"danger": 0, "warning": 1, "warn": 1, "info": 2}
	sort.SliceStable(out, func(i, j int) bool { return rank[out[i].Severity] < rank[out[j].Severity] })
	return out, nil
}

// --- 内部辅助 ---

// location 取家庭时区；解析失败时回落到 UTC+8。
func (s *CareService) location(ctx context.Context, familyID string) *time.Location {
	fallback := time.FixedZone("CST", 8*3600)
	if s.families == nil {
		return fallback
	}
	f, err := s.families.FindByID(ctx, familyID)
	if err != nil || f.Timezone == "" {
		return fallback
	}
	loc, err := time.LoadLocation(f.Timezone)
	if err != nil {
		return fallback
	}
	return loc
}

func decodePayload(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return map[string]any{}
	}
	return m
}

// num 从 payload 中按顺序取第一个可转成数字的键。
func num(p map[string]any, keys ...string) float64 {
	for _, k := range keys {
		switch v := p[k].(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case json.Number:
			if f, err := v.Float64(); err == nil {
				return f
			}
		case string:
			var f float64
			if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
				return f
			}
		}
	}
	return 0
}

func rateMetric(amount, expected float64, what string) (string, string) {
	if expected <= 0 {
		return "normal", what + "已记录"
	}
	ratio := amount / expected
	switch {
	case ratio >= 0.9:
		return "normal", what + "正常"
	case ratio >= 0.6:
		return "warning", fmt.Sprintf("%s偏低（%.0f/%.0f）", what, amount, expected)
	default:
		return "danger", fmt.Sprintf("%s明显不足（%.0f/%.0f）", what, amount, expected)
	}
}

func severityToState(sev string) string {
	if sev == "" {
		return "normal"
	}
	return sev
}

func eliminationLabel(p map[string]any, sev string) string {
	if form, _ := p["form"].(string); form != "" {
		return "排便：" + form
	}
	if sev == "danger" || sev == "warning" {
		return "排便异常"
	}
	return "排便正常"
}

func focusTitle(r *model.DailyRecord) string {
	switch r.RecordType {
	case "vomit":
		return "呕吐记录"
	case "symptom":
		return "异常症状"
	case "mental":
		return "精神异常"
	}
	return labelOf(r.RecordType)
}

func focusBody(r *model.DailyRecord) string {
	if r.Note != "" {
		return r.Note
	}
	p := decodePayload(r.Payload)
	if content, _ := p["content"].(string); content != "" {
		return "呕吐物：" + content
	}
	if symptom, _ := p["symptom"].(string); symptom != "" {
		return "症状：" + symptom
	}
	return ""
}

func focusActions(recordType string) []FocusAction {
	if recordType == "medication" {
		return []FocusAction{{Label: "完成服药", Action: "done"}, {Label: "稍后提醒", Action: "later"}}
	}
	return []FocusAction{
		{Label: "查看记录", Action: "view"},
		{Label: "继续观察", Action: "observe"},
		{Label: "添加记录", Action: "add"},
	}
}

func labelOf(recordType string) string {
	labels := map[string]string{
		"feeding": "喂食", "drinking": "饮水", "elimination": "排便", "vomit": "呕吐",
		"weight": "体重", "medication": "用药", "mental": "精神状态", "symptom": "异常症状",
		"visit": "就诊", "vaccine": "疫苗", "deworm": "驱虫", "food-change": "换粮",
		"behavior": "行为", "interaction": "双猫互动", "photo": "照片",
		"milestone": "成长事件", "custom": "自定义",
	}
	if l, ok := labels[recordType]; ok {
		return l
	}
	return recordType
}
