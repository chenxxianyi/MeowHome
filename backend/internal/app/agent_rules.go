package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
)

// RuleCandidate 是确定性规则引擎的中间结果；它不依赖 LLM。
type RuleCandidate struct {
	RuleID      string
	RuleVersion string
	CatID       string
	Severity    string
	Title       string
	Body        string
	Scope       string
	WindowStart time.Time
	WindowEnd   time.Time
	Evidence    []AgentEvidence
	Actions     []AgentAction
	DedupKey    string
}

func evidenceForRecord(r *model.DailyRecord, loc *time.Location, excerpt string) AgentEvidence {
	return AgentEvidence{SourceType: "record", SourceID: r.ID, CatIDs: r.CatIDs, OccurredAt: r.OccurredAt, Excerpt: excerpt}
}

func encodeJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

// evaluateRules 按家庭本地日历计算 R-01 至 R-06。缺失数据只产生“未记录”提示，
// 不把缺失解释成健康或疾病结论。
func evaluateRules(now time.Time, loc *time.Location, familyCreatedAt time.Time, cats []*model.Cat, records []*model.DailyRecord, reminders []*model.Reminder) []RuleCandidate {
	localNow := now.In(loc)
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)
	yesterdayStart := today.AddDate(0, 0, -1)
	byCat := map[string][]*model.DailyRecord{}
	seenByCat := map[string]map[string]bool{}
	for _, r := range records {
		if r.DeletedAt != nil || r.OccurredAt.After(now) {
			continue
		}
		for _, catID := range r.CatIDs {
			if seenByCat[catID] == nil {
				seenByCat[catID] = map[string]bool{}
			}
			if seenByCat[catID][r.ID] {
				continue
			}
			seenByCat[catID][r.ID] = true
			byCat[catID] = append(byCat[catID], r)
		}
	}
	validCats := make([]*model.Cat, 0, len(cats))
	for _, c := range cats {
		if c.DeletedAt == nil {
			validCats = append(validCats, c)
		}
	}
	catName := func(id string) string {
		for _, c := range validCats {
			if c.ID == id {
				return c.Name
			}
		}
		return "该猫咪"
	}
	out := make([]RuleCandidate, 0)
	for _, c := range validCats {
		var evidence []AgentEvidence
		for _, r := range byCat[c.ID] {
			if r.Severity == "danger" && !r.OccurredAt.Before(yesterdayStart) && r.OccurredAt.Before(today) {
				evidence = append(evidence, evidenceForRecord(r, loc, r.Note))
			}
		}
		if len(evidence) == 0 {
			continue
		}
		key := fmt.Sprintf("%s:%s:%s:%s", c.FamilyID, RuleR01, c.ID, yesterdayStart.Format("2006-01-02"))
		out = append(out, RuleCandidate{RuleID: RuleR01, RuleVersion: "1", CatID: c.ID, Severity: "danger", Title: c.Name + "昨日有高风险记录", Body: fmt.Sprintf("%s 昨日有 %d 条原记录标为 danger，请核对原始记录。", catName(c.ID), len(evidence)), Scope: "cat", WindowStart: yesterdayStart.UTC(), WindowEnd: today.UTC(), Evidence: evidence, Actions: []AgentAction{{Type: "view_records", Title: "查看原始记录", CatID: c.ID, Days: 2}}, DedupKey: key})
	}
	// R-02：同一只猫同一类型在 7 天内至少三条（呕吐或腹泻）记录。
	weekStart := today.AddDate(0, 0, -6)
	for _, c := range validCats {
		var vomit, diarrhea []AgentEvidence
		for _, r := range byCat[c.ID] {
			if r.OccurredAt.Before(weekStart) || r.OccurredAt.After(now) {
				continue
			}
			if r.RecordType == "vomit" {
				vomit = append(vomit, evidenceForRecord(r, loc, r.Note))
				continue
			}
			if r.RecordType != "elimination" {
				continue
			}
			p := decodePayload(r.Payload)
			form, _ := p["form"].(string)
			if form == "稀便" || form == "水样" {
				diarrhea = append(diarrhea, evidenceForRecord(r, loc, form))
			}
		}
		if len(vomit) >= 3 || len(diarrhea) >= 3 {
			typeName, ev := "呕吐", vomit
			if len(diarrhea) >= 3 && len(diarrhea) > len(vomit) {
				typeName, ev = "腹泻", diarrhea
			}
			out = append(out, RuleCandidate{RuleID: RuleR02, RuleVersion: "1", CatID: c.ID, Severity: "warning", Title: c.Name + "近 7 天有多次" + typeName, Body: "检测到同一只猫在 7 天窗口内同类型记录达到 3 条，建议核对原始记录并持续观察。", Scope: "cat", WindowStart: weekStart.UTC(), WindowEnd: now.UTC(), Evidence: ev, Actions: []AgentAction{{Type: "view_records", Title: "查看相关记录", CatID: c.ID, Days: 7}}, DedupKey: fmt.Sprintf("%s:%s:%s:%s", c.FamilyID, RuleR02, c.ID, weekStart.Format("2006-01-02"))})
		}
	}
	// R-03：待办用药提醒已超过计划时间 24 小时。
	for _, r := range reminders {
		if r.DeletedAt != nil || r.State != "todo" || r.Type != "medication" || r.ScheduledAt == nil || now.Sub(*r.ScheduledAt) <= 24*time.Hour {
			continue
		}
		out = append(out, RuleCandidate{RuleID: RuleR03, RuleVersion: "1", CatID: r.CatID, Severity: "warning", Title: "用药提醒待确认", Body: "这条用药提醒已超过计划时间 24 小时，当前只能确认提醒状态，不能据此判断是否漏服。", Scope: "cat", WindowStart: r.ScheduledAt.UTC(), WindowEnd: now.UTC(), Evidence: []AgentEvidence{{SourceType: "reminder", SourceID: r.ID, CatIDs: []string{r.CatID}, OccurredAt: *r.ScheduledAt, Excerpt: r.Title}}, Actions: []AgentAction{{Type: "view_records", Title: "查看用药记录", CatID: r.CatID, Days: 3}}, DedupKey: fmt.Sprintf("%s:%s:%s:%s", r.FamilyID, RuleR03, r.ID, today.Format("2006-01-02"))})
	}
	// R-04：仅周日 20:00 后按家庭本地日历检查；从未称重时以猫咪建档日为起点。
	if localNow.Weekday() == time.Sunday && localNow.Hour() >= 20 {
		cutoff := today.AddDate(0, 0, -14)
		for _, c := range validCats {
			var latest *model.DailyRecord
			for _, r := range byCat[c.ID] {
				if r.RecordType == "weight" && (latest == nil || r.OccurredAt.After(latest.OccurredAt)) {
					latest = r
				}
			}
			start := c.CreatedAt
			evidence := AgentEvidence{SourceType: "cat_profile", SourceID: c.ID, CatIDs: []string{c.ID}, OccurredAt: c.CreatedAt, Excerpt: "猫咪建档起点；暂无称重记录"}
			if latest != nil {
				start = latest.OccurredAt
				evidence = AgentEvidence{SourceType: "weight", SourceID: latest.ID, CatIDs: []string{c.ID}, OccurredAt: latest.OccurredAt, Excerpt: "最近一次称重记录"}
			}
			if start.IsZero() {
				continue
			}
			startLocal := start.In(loc)
			startDate := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), 0, 0, 0, 0, loc)
			if startDate.Before(cutoff) {
				out = append(out, RuleCandidate{RuleID: RuleR04, RuleVersion: "1", CatID: c.ID, Severity: "warning", Title: c.Name + "超过 14 天未称重", Body: "当前没有足够新的体重记录；这是记录覆盖提示，不代表体重异常。", Scope: "cat", WindowStart: cutoff.UTC(), WindowEnd: now.UTC(), Evidence: []AgentEvidence{evidence}, Actions: []AgentAction{{Type: "view_trend", Title: "查看体重趋势", CatID: c.ID, Days: 30}}, DedupKey: fmt.Sprintf("%s:%s:%s:%s", c.FamilyID, RuleR04, c.ID, today.Format("2006-01-02"))})
			}
		}
	}
	// R-05：疫苗/驱虫提醒未来 7 天内到期。
	localSevenDaysLater := localNow.AddDate(0, 0, 7).UTC()
	for _, r := range reminders {
		if r.DeletedAt != nil || (r.Type != "vaccine" && r.Type != "deworm") || r.State != "todo" || r.ScheduledAt == nil || r.ScheduledAt.Before(now) || !r.ScheduledAt.Before(localSevenDaysLater) {
			continue
		}
		out = append(out, RuleCandidate{RuleID: RuleR05, RuleVersion: "1", CatID: r.CatID, Severity: "info", Title: "护理计划即将到期", Body: fmt.Sprintf("%s 计划在 7 天内到期，请核对日期和猫咪归属。", r.Title), Scope: "cat", WindowStart: now.UTC(), WindowEnd: r.ScheduledAt.UTC(), Evidence: []AgentEvidence{{SourceType: "reminder", SourceID: r.ID, CatIDs: []string{r.CatID}, OccurredAt: *r.ScheduledAt, Excerpt: r.Title}}, Actions: []AgentAction{{Type: "view_records", Title: "查看护理计划", CatID: r.CatID, Days: 14}}, DedupKey: fmt.Sprintf("%s:%s:%s:%s", r.FamilyID, RuleR05, r.ID, today.Format("2006-01-02"))})
	}
	// R-06：昨天家庭总记录数为零，只生成一条家庭级消息，避免每只猫重复提示。
	hasFamilyRecord := false
	for _, r := range records {
		if r.DeletedAt == nil && !r.OccurredAt.Before(yesterdayStart) && r.OccurredAt.Before(today) {
			hasFamilyRecord = true
			break
		}
	}
	if len(validCats) > 0 && !familyCreatedAt.IsZero() && !familyCreatedAt.After(yesterdayStart) && !hasFamilyRecord {
		catIDs := make([]string, 0, len(validCats))
		for _, c := range validCats {
			catIDs = append(catIDs, c.ID)
		}
		out = append(out, RuleCandidate{RuleID: RuleR06, RuleVersion: "1", Severity: "info", Title: "家庭昨日暂无记录", Body: "昨日没有找到本家庭的照护记录，仅表示记录缺失，不能推断实际照护情况。", Scope: "family", WindowStart: yesterdayStart.UTC(), WindowEnd: today.UTC(), Evidence: []AgentEvidence{{SourceType: "family_scope", SourceID: validCats[0].FamilyID, CatIDs: catIDs, OccurredAt: yesterdayStart.UTC(), Excerpt: fmt.Sprintf("查询范围：%s 至 %s，本家庭有效记录为零", yesterdayStart.Format("2006-01-02"), today.Format("2006-01-02"))}}, Actions: []AgentAction{{Type: "view_records", Title: "查看家庭记录", Days: 2}}, DedupKey: fmt.Sprintf("%s:%s:%s", validCats[0].FamilyID, RuleR06, yesterdayStart.Format("2006-01-02"))})
	}
	for i := range out {
		out[i].DedupKey += ":v" + out[i].RuleVersion
	}
	return out
}
