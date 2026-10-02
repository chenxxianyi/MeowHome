package app

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	apperr "github.com/meowhome/backend/internal/platform/errors"
)

func observedNumber(payload map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, exists := payload[key]
		if !exists || value == nil {
			continue
		}
		var number float64
		switch v := value.(type) {
		case float64:
			number = v
		case int:
			number = float64(v)
		case json.Number:
			parsed, err := v.Float64()
			if err != nil {
				continue
			}
			number = parsed
		case string:
			parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil {
				continue
			}
			number = parsed
		default:
			continue
		}
		if !math.IsNaN(number) && !math.IsInf(number, 0) {
			return number, true
		}
	}
	return 0, false
}

func (s *AgentService) toolGetTrends(ctx context.Context, scope AgentToolScope, raw string) (*AgentToolResult, error) {
	var args struct {
		CatID  string `json:"cat_id"`
		Metric string `json:"metric"`
		Days   int    `json:"days"`
	}
	if err := decodeToolArgs(raw, &args); err != nil {
		return nil, err
	}
	if _, err := s.toolCat(ctx, scope.FamilyID, args.CatID); err != nil {
		return nil, err
	}
	units := map[string]string{"weight": "kg", "feeding": "g", "drinking": "ml", "vomit": "count"}
	if units[args.Metric] == "" || args.Days < 1 || args.Days > 90 {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "trend arguments are invalid")
	}
	loc, err := s.toolLocation(ctx, scope.FamilyID)
	if err != nil {
		return nil, err
	}
	now := s.clock().UTC()
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	from := today.AddDate(0, 0, -(args.Days - 1))
	q := repository.DailyRecordQuery{FamilyID: scope.FamilyID, CatID: args.CatID, Types: []string{args.Metric}, From: from.UTC().Format(time.RFC3339), To: now.Format(time.RFC3339), Limit: 1000}
	records := make([]*model.DailyRecord, 0)
	for {
		batch, err := s.records.ListByFamily(ctx, q)
		if err != nil {
			return nil, err
		}
		for _, record := range batch {
			if record == nil || record.FamilyID != scope.FamilyID || record.DeletedAt != nil || !recordHasCat(record, args.CatID) || record.RecordType != args.Metric || record.OccurredAt.Before(from) || record.OccurredAt.After(now) {
				return nil, repository.ErrInvalidQuery
			}
		}
		records = append(records, batch...)
		if len(batch) < q.Limit {
			break
		}
		last := batch[len(batch)-1]
		if q.BeforeOccurredAt != nil && (last.OccurredAt.After(*q.BeforeOccurredAt) || (last.OccurredAt.Equal(*q.BeforeOccurredAt) && last.ID >= q.BeforeID)) {
			return nil, repository.ErrInvalidQuery
		}
		at := last.OccurredAt
		q.BeforeOccurredAt, q.BeforeID = &at, last.ID
	}
	type point struct {
		Date         string   `json:"date"`
		Value        float64  `json:"value"`
		Observations int      `json:"observations"`
		SourceIDs    []string `json:"source_ids"`
	}
	byDay := map[string]*point{}
	lastWeightAt := map[string]time.Time{}
	missingValues := 0
	evidence := make([]AgentEvidence, 0)
	evidenceTruncated := false
	for _, r := range records {
		day := r.OccurredAt.In(loc).Format("2006-01-02")
		p := decodePayload(r.Payload)
		value := 0.0
		observed := false
		switch args.Metric {
		case "weight":
			value, observed = observedNumber(p, "weight")
			observed = observed && value > 0
		case "feeding":
			value, observed = observedNumber(p, "consumedAmount", "consumed", "amount")
			observed = observed && value >= 0
		case "drinking":
			value, observed = observedNumber(p, "amount")
			observed = observed && value >= 0
		case "vomit":
			value, observed = observedNumber(p, "count")
			if !observed || value <= 0 {
				value = 1
			}
			observed = true // 一条呕吐事件至少代表一次观测
		}
		if !observed {
			missingValues++
			continue
		}
		pt := byDay[day]
		if pt == nil {
			pt = &point{Date: day, SourceIDs: []string{}}
			byDay[day] = pt
		}
		if args.Metric == "weight" {
			if r.OccurredAt.After(lastWeightAt[day]) {
				pt.Value = value
				lastWeightAt[day] = r.OccurredAt
			}
		} else {
			pt.Value += value
		}
		pt.Observations++
		if len(pt.SourceIDs) < 10 {
			pt.SourceIDs = append(pt.SourceIDs, r.ID)
		} else {
			evidenceTruncated = true
		}
		if len(evidence) < 50 {
			evidence = append(evidence, evidenceForRecord(r, loc, toolExcerpt(r.Note, 300)))
		} else {
			evidenceTruncated = true
		}
	}
	points := make([]point, 0, len(byDay))
	missingDates := make([]string, 0)
	for day := from; !day.After(today); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		if p := byDay[key]; p != nil {
			points = append(points, *p)
		} else {
			missingDates = append(missingDates, key)
		}
	}
	return toolResult(map[string]any{"cat_id": args.CatID, "metric": args.Metric, "unit": units[args.Metric], "points": points, "missing_dates": missingDates, "records_without_value": missingValues, "evidence_truncated": evidenceTruncated, "window_start": from.UTC(), "window_end": now}, evidence, evidenceTruncated)
}

func (s *AgentService) toolCreateReminderDraft(ctx context.Context, scope AgentToolScope, raw string) (*AgentToolResult, error) {
	var args struct {
		CatID       string     `json:"cat_id"`
		Type        string     `json:"type"`
		Title       string     `json:"title"`
		ScheduledAt *time.Time `json:"scheduled_at"`
		Timezone    string     `json:"timezone"`
	}
	if err := decodeToolArgs(raw, &args); err != nil {
		return nil, err
	}
	if args.CatID != "both" {
		if _, err := s.toolCat(ctx, scope.FamilyID, args.CatID); err != nil {
			return nil, err
		}
	}
	if args.Type == "" {
		args.Type = "custom"
	}
	input := &AgentReminderInput{CatID: args.CatID, Type: args.Type, Title: args.Title, ScheduledAt: args.ScheduledAt, Timezone: args.Timezone}
	if err := s.validateReminder(ctx, scope.FamilyID, input); err != nil {
		return nil, err
	}
	if scope.DraftTargetMessageID == "" && scope.TurnID != "" {
		target, err := s.ensureTurnDraftTarget(ctx, scope)
		if err != nil {
			return nil, err
		}
		scope.DraftTargetMessageID = target.ID
		if target.ActionStatus == DraftPending {
			if target.DraftPayload != encodeJSON(input) {
				return nil, apperr.Conflict(CodeAgentConflict, "draft content changed for the same turn")
			}
			draft := &AgentReminderDraft{MessageID: target.ID, Version: target.DraftVersion, Reminder: input}
			if target.DraftExpiresAt != nil {
				draft.ExpiresAt = *target.DraftExpiresAt
			}
			result, err := toolResult(map[string]any{"message_id": draft.MessageID, "version": draft.Version, "expires_at": draft.ExpiresAt, "status": "pending", "reminder": draft.Reminder}, nil, false)
			if err != nil {
				return nil, err
			}
			result.Draft = draft
			return result, nil
		}
	}
	if scope.DraftTargetMessageID == "" {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "draft target is unavailable")
	}
	draft, err := s.EditDraft(ctx, scope.FamilyID, scope.UserID, scope.DraftTargetMessageID, AgentReminderEditRequest{ExpectedVersion: scope.ExpectedDraftVersion, Reminder: input})
	if err != nil {
		return nil, err
	}
	result, err := toolResult(map[string]any{"message_id": draft.MessageID, "version": draft.Version, "expires_at": draft.ExpiresAt, "status": "pending", "reminder": draft.Reminder}, nil, false)
	if err != nil {
		return nil, err
	}
	result.Draft = draft
	return result, nil
}
