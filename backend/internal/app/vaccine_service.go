package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
)

const vaccineRuleVersion = "2026-10-21.1"

const (
	vaccineCoreSource  = "https://www.aaha.org/resources/2020-aahaaafp-feline-vaccination-guidelines/core-vaccines-for-pet-cats/"
	vaccineWSAVASource = "https://wsava.org/wp-content/uploads/2024/04/WSAVA-Vaccination-guidelines-2024.pdf"
)

// VaccineService supplies conservative, explainable follow-up date suggestions.
type VaccineService struct {
	records   repository.DailyRecordRepo
	cats      repository.CatRepo
	members   repository.MemberRepo
	families  repository.FamilyRepo
	reminders *ReminderService
}

func NewVaccineService(records repository.DailyRecordRepo, cats repository.CatRepo, members repository.MemberRepo, families repository.FamilyRepo, reminders *ReminderService) *VaccineService {
	return &VaccineService{records: records, cats: cats, members: members, families: families, reminders: reminders}
}

type VaccineSuggestion struct {
	Status              string `json:"status"`
	CatID               string `json:"cat_id"`
	VaccineCode         string `json:"vaccine_code,omitempty"`
	VaccineName         string `json:"vaccine_name,omitempty"`
	DoseNumber          string `json:"dose_number,omitempty"`
	AgeWeeks            *int   `json:"age_weeks,omitempty"`
	PriorDosesRecorded  int    `json:"prior_doses_recorded"`
	DateWindowStart     string `json:"date_window_start,omitempty"`
	DateWindowEnd       string `json:"date_window_end,omitempty"`
	DefaultReminderDate string `json:"default_reminder_date,omitempty"`
	Title               string `json:"title,omitempty"`
	Reason              string `json:"reason"`
	RuleID              string `json:"rule_id,omitempty"`
	RuleVersion         string `json:"rule_version,omitempty"`
	SourceURL           string `json:"source_url,omitempty"`
	Timezone            string `json:"timezone"`
	DateSource          string `json:"date_source,omitempty"`
}

type VaccineReminderInput struct {
	ScheduledDate string `json:"scheduled_date"`
	DateSource    string `json:"date_source"`
	Timezone      string `json:"timezone"`
}

// SuggestNext reads the saved record, cat profile and recorded vaccine history in the authorized family.
func (s *VaccineService) SuggestNext(ctx context.Context, familyID, userID, recordID string) (*VaccineSuggestion, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	record, err := s.records.FindByID(ctx, recordID)
	if err != nil || record == nil || record.DeletedAt != nil || record.FamilyID != familyID || record.RecordType != "vaccine" {
		return nil, errors.NotFound(errors.CodeNotFound, "vaccine record not found")
	}
	if len(record.CatIDs) != 1 {
		return nil, errors.NotFound(errors.CodeNotFound, "vaccine record not found")
	}
	cat, err := s.cats.FindByID(ctx, record.CatIDs[0])
	if err != nil || cat == nil || cat.DeletedAt != nil || cat.FamilyID != familyID {
		return nil, errors.NotFound(errors.CodeNotFound, "vaccine record not found")
	}
	var payload map[string]any
	if record.Payload != "" {
		_ = json.Unmarshal([]byte(record.Payload), &payload)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	loc, locationErr := s.familyLocation(ctx, familyID, payload)
	if locationErr != nil {
		return nil, locationErr
	}
	vaccineDate := vaccineRecordDate(record, payload, loc)
	name := payloadString(payload, "vaccine_name")
	code := strings.ToLower(strings.TrimSpace(payloadString(payload, "vaccine_code")))
	dose := payloadString(payload, "dose_number")
	if dose == "" {
		dose = payloadNumberString(payload, "dose_number")
	}
	stage := strings.ToLower(strings.TrimSpace(payloadString(payload, "series_stage")))
	if name == "" {
		name = vaccineDisplayName(code)
	}
	result := &VaccineSuggestion{
		Status: "needs_input", CatID: cat.ID, VaccineCode: code, VaccineName: name, DoseNumber: dose,
		Timezone: loc.String(), RuleVersion: vaccineRuleVersion,
		Title:  fmt.Sprintf("联系诊所确认/预约%s", nameOrGeneric(name)),
		Reason: "请补充疫苗类型、针次、接种史或兽医提供的下一次日期；系统不会按默认间隔猜测。",
	}
	if birthdayAge := ageInWeeks(cat.Birthday, vaccineDate); birthdayAge != nil {
		result.AgeWeeks = birthdayAge
	} else if suppliedAge := payloadInt(payload, "age_weeks"); suppliedAge != nil {
		result.AgeWeeks = suppliedAge
	}
	history, historyErr := s.records.ListByFamily(ctx, repository.DailyRecordQuery{
		FamilyID: familyID, CatID: cat.ID, Types: []string{"vaccine"}, Limit: 1000,
	})
	if historyErr != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to read vaccine history", historyErr)
	}
	var latestPreviousDose *time.Time
	for _, prior := range history {
		if prior.ID == record.ID || prior.DeletedAt != nil {
			continue
		}
		var priorPayload map[string]any
		if json.Unmarshal([]byte(prior.Payload), &priorPayload) != nil || !strings.EqualFold(payloadString(priorPayload, "vaccine_code"), code) || code == "" {
			continue
		}
		priorDate := vaccineRecordDate(prior, priorPayload, loc)
		if !priorDate.Before(vaccineDate) {
			continue
		}
		result.PriorDosesRecorded++
		if latestPreviousDose == nil || priorDate.After(*latestPreviousDose) {
			priorDateCopy := priorDate
			latestPreviousDose = &priorDateCopy
		}
	}
	previousDoseGapDays := 0
	hasPreviousDose := latestPreviousDose != nil
	if latestPreviousDose != nil {
		previousDoseGapDays = calendarDayDifference(vaccineDate, *latestPreviousDose)
	}

	if due := strings.TrimSpace(payloadString(payload, "next_due_date")); due != "" {
		source := strings.TrimSpace(payloadString(payload, "next_due_source"))
		if source == "user" || source == "veterinarian" || source == "product_label" {
			if dueDate, parseErr := time.ParseInLocation("2006-01-02", due, loc); parseErr == nil && dueDate.After(vaccineDate) && dueDate.After(time.Now().In(loc)) {
				result.Status = "suggested"
				result.DateWindowStart, result.DateWindowEnd, result.DefaultReminderDate = due, due, due
				result.DateSource = source
				result.RuleID = "user-provided-date"
				result.SourceURL = vaccineSourceFor(code)
				result.Title = fmt.Sprintf("联系诊所确认/预约%s", nameOrGeneric(name))
				result.Reason = "采用记录中由用户、兽医或产品说明提供的日期；实际安排请再次核对兽医意见和产品说明。"
				return result, nil
			}
		}
		result.Reason = "记录中的下一次日期无效、已过期或缺少日期来源，请核对后手动设置未来日期。"
		return result, nil
	}

	if code == "rabies" {
		result.Status = "unsupported"
		result.SourceURL = vaccineWSAVASource
		result.Reason = "狂犬疫苗的下一次时间取决于当地规定和具体产品免疫持续期，请按当地规定、产品说明或兽医给出的日期手动设置。"
		return result, nil
	}
	if code != "fvrhcp" && code != "felv" {
		result.Status = "unsupported"
		result.Reason = "当前规则未覆盖此疫苗，请按兽医意见或产品说明手动选择日期。"
		return result, nil
	}
	if dose == "" || dose == "unknown" || stage == "unknown" {
		result.Reason = "针次或系列阶段不明确，请补充接种信息，或按兽医提供的日期手动设置待办。"
		return result, nil
	}
	if dose != "1" && dose != "2" && dose != "3" {
		result.Reason = "当前针次无法匹配首免规则，请按兽医意见或产品说明手动设置下一次日期。"
		return result, nil
	}
	if dose == "1" && result.PriorDosesRecorded > 0 {
		result.Reason = "已有同类型疫苗记录，但本次填写为首针；请先核对接种史和针次。"
		return result, nil
	}
	if doseNumber, _ := strconv.Atoi(dose); doseNumber > 1 && result.PriorDosesRecorded == 0 {
		previousDateRaw := strings.TrimSpace(payloadString(payload, "previous_dose_date"))
		previousDate, previousDateErr := time.ParseInLocation("2006-01-02", previousDateRaw, loc)
		if previousDateRaw == "" || previousDateErr != nil || !previousDate.Before(vaccineDate) {
			result.Reason = "此前针次未在本应用的接种记录中找到，请补充有效的上一针日期或核对针次。"
			return result, nil
		}
		previousDoseGapDays = calendarDayDifference(vaccineDate, previousDate)
		hasPreviousDose = true
	}

	minDays, maxDays, defaultDays := 0, 0, 0
	switch code {
	case "fvrhcp":
		result.SourceURL = vaccineCoreSource
		if result.AgeWeeks == nil {
			result.Reason = "猫咪出生日期和本次接种时年龄均未知，无法判断幼猫系列或成年猫初免；请补充年龄或手动设置日期。"
			return result, nil
		}
		if *result.AgeWeeks >= 6 && *result.AgeWeeks < 16 && stage == "primary" {
			if dose != "1" && hasPreviousDose && (previousDoseGapDays < 21 || previousDoseGapDays > 28) {
				result.Reason = "上一针与本次接种的间隔不在当前幼猫规则的 3–4 周窗口内，请核对接种史和兽医意见。"
				return result, nil
			}
			minDays, maxDays, defaultDays = 21, 28, 28
			result.RuleID = "fvrhcp-kitten-primary-3-4-weeks"
			result.Reason = "适用于已确认的幼猫猫三联初免：常见安排为每 3–4 周复核/接种，直到达到至少 16 周龄；窗口仅供预约核对，实际剂次由兽医和产品说明确定。"
		} else if *result.AgeWeeks >= 52 && stage == "primary" && dose == "1" && strings.TrimSpace(payloadString(payload, "product_name")) != "" {
			minDays, maxDays, defaultDays = 14, 28, 21
			result.RuleID = "fvrhcp-adult-primary-2-4-weeks"
			result.Reason = "适用于已确认的成年猫猫三联初免/接种史不完整情形，常见候选间隔为 2–4 周；具体间隔需核对产品、剂型和兽医意见。"
		} else {
			result.Reason = "年龄、针次、系列阶段或产品信息不足以安全套用猫三联规则；请按兽医意见或产品说明手动设置日期。"
			return result, nil
		}
	case "felv":
		result.SourceURL = vaccineWSAVASource
		if result.AgeWeeks == nil || *result.AgeWeeks < 8 {
			result.Reason = "FeLV 初免需核对猫咪年龄、抗原检测和产品说明；请补充信息或按兽医提供的日期手动设置。"
			return result, nil
		}
		if dose != "1" || stage != "primary" {
			result.Reason = "FeLV 后续加强还需结合初免完成日期、产品说明和暴露风险；请按兽医意见手动设置下一次日期。"
			return result, nil
		}
		minDays, maxDays, defaultDays = 21, 28, 28
		result.RuleID = "felv-primary-second-dose-3-4-weeks"
		result.Reason = "适用于已确认的 FeLV 初免首剂：常见第二剂窗口为 3–4 周；接种前需按兽医建议核对 FeLV 检测结果和产品说明。后续加强不由此规则推断。"
	}

	start := vaccineDate.AddDate(0, 0, minDays)
	end := vaccineDate.AddDate(0, 0, maxDays)
	defaultDate := vaccineDate.AddDate(0, 0, defaultDays)
	result.DateWindowStart = start.Format("2006-01-02")
	result.DateWindowEnd = end.Format("2006-01-02")
	result.DefaultReminderDate = defaultDate.Format("2006-01-02")
	if !defaultDate.After(time.Now().In(loc)) {
		result.Status = "needs_input"
		result.DefaultReminderDate = ""
		result.Reason += " 该窗口已经过去，请核对实际接种安排并手动选择未来日期。"
		return result, nil
	}
	result.Status = "suggested"
	result.Title = fmt.Sprintf("联系诊所确认/预约%s", nameOrGeneric(name))
	return result, nil
}

// CreateReminder re-evaluates the saved record and creates a source-linked, idempotent reminder.
func (s *VaccineService) CreateReminder(ctx context.Context, familyID, userID, recordID string, in VaccineReminderInput) (*ReminderEnvelope, error) {
	suggestion, err := s.SuggestNext(ctx, familyID, userID, recordID)
	if err != nil {
		return nil, err
	}
	if s.reminders == nil {
		return nil, errors.New(errors.TypeInternal, errors.CodeInvalidRequest, "reminder service is unavailable")
	}
	if existing, lookupErr := s.reminders.FindVaccineReminderBySource(ctx, familyID, userID, recordID); lookupErr != nil {
		return nil, lookupErr
	} else if existing != nil {
		return existing, nil
	}
	date, err := time.Parse("2006-01-02", strings.TrimSpace(in.ScheduledDate))
	if err != nil {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "scheduled_date must use YYYY-MM-DD")
	}
	zone := strings.TrimSpace(in.Timezone)
	if zone == "" {
		zone = suggestion.Timezone
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "invalid timezone")
	}
	dateSource := strings.TrimSpace(in.DateSource)
	if dateSource != "rule_suggestion" && dateSource != "user" && dateSource != "veterinarian" && dateSource != "product_label" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "date_source must identify a rule suggestion, user, veterinarian, or product label")
	}
	if dateSource == "rule_suggestion" {
		if suggestion.Status != "suggested" || date.Format("2006-01-02") < suggestion.DateWindowStart || date.Format("2006-01-02") > suggestion.DateWindowEnd {
			return nil, errors.InvalidRequest(errors.CodeValidationFailed, "rule-suggested date must be inside the current recommendation window")
		}
	}
	scheduledAt := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, loc)
	metadata := map[string]any{
		"vaccine_code":   suggestion.VaccineCode,
		"vaccine_name":   suggestion.VaccineName,
		"dose_number":    suggestion.DoseNumber,
		"date_source":    dateSource,
		"confirmed_date": date.Format("2006-01-02"),
		"suggestion":     suggestion,
	}
	return s.reminders.Create(ctx, familyID, userID, ReminderInput{
		CatID: suggestion.CatID, Type: "vaccine", Title: suggestion.Title,
		Subtitle: fmt.Sprintf("%s · 计划日期 %s", nameOrGeneric(suggestion.VaccineName), date.Format("2006-01-02")),
		Time:     "09:00", Icon: "vaccine", Rule: suggestion.RuleID + "@" + suggestion.RuleVersion,
		ScheduledAt: &scheduledAt, Timezone: zone, SourceRecordID: recordID, Metadata: metadata,
	})
}

func (s *VaccineService) CompleteReminderWithRecord(ctx context.Context, familyID, userID, reminderID, recordID string) (*ReminderEnvelope, error) {
	if s.reminders == nil {
		return nil, errors.New(errors.TypeInternal, errors.CodeInvalidRequest, "reminder service is unavailable")
	}
	return s.reminders.CompleteWithRecord(ctx, familyID, userID, reminderID, recordID)
}

func (s *VaccineService) UpdateReminderDate(ctx context.Context, familyID, userID, reminderID, scheduledDate, timezone string) (*ReminderEnvelope, error) {
	if s.reminders == nil {
		return nil, errors.New(errors.TypeInternal, errors.CodeInvalidRequest, "reminder service is unavailable")
	}
	return s.reminders.UpdateVaccineSchedule(ctx, familyID, userID, reminderID, scheduledDate, timezone)
}

func (s *VaccineService) CancelReminder(ctx context.Context, familyID, userID, reminderID string) error {
	if s.reminders == nil {
		return errors.New(errors.TypeInternal, errors.CodeInvalidRequest, "reminder service is unavailable")
	}
	return s.reminders.CancelVaccineReminder(ctx, familyID, userID, reminderID)
}

func (s *VaccineService) familyLocation(ctx context.Context, familyID string, payload map[string]any) (*time.Location, error) {
	if s.families == nil {
		return nil, errors.New(errors.TypeInternal, errors.CodeInvalidRequest, "family repository is unavailable")
	}
	family, err := s.families.FindByID(ctx, familyID)
	if err != nil || family == nil || family.DeletedAt != nil {
		return nil, errors.NotFound(errors.CodeNotFound, "family not found")
	}
	familyLocation, err := time.LoadLocation(family.Timezone)
	if err != nil {
		familyLocation = time.UTC
	}
	if name := strings.TrimSpace(payloadString(payload, "timezone")); name != "" {
		if loc, loadErr := time.LoadLocation(name); loadErr == nil {
			return loc, nil
		}
	}
	return familyLocation, nil
}

func vaccineRecordDate(record *model.DailyRecord, payload map[string]any, loc *time.Location) time.Time {
	if raw := strings.TrimSpace(payloadString(payload, "vaccination_date")); raw != "" {
		if date, err := time.ParseInLocation("2006-01-02", raw, loc); err == nil {
			return date
		}
	}
	return record.OccurredAt.In(loc)
}

func ageInWeeks(birthday string, at time.Time) *int {
	birthDate, err := time.Parse("2006-01-02", strings.TrimSpace(birthday))
	if err != nil {
		return nil
	}
	date := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
	days := int(date.Sub(birthDate).Hours() / 24)
	if days < 0 {
		return nil
	}
	weeks := days / 7
	return &weeks
}

func calendarDayDifference(later, earlier time.Time) int {
	ly, lm, ld := later.Date()
	ey, em, ed := earlier.Date()
	laterDay := time.Date(ly, lm, ld, 0, 0, 0, 0, time.UTC)
	earlierDay := time.Date(ey, em, ed, 0, 0, 0, 0, time.UTC)
	return int(laterDay.Sub(earlierDay).Hours() / 24)
}

func payloadString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func payloadNumberString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	if number, ok := value.(float64); ok && number == float64(int(number)) {
		return strconv.Itoa(int(number))
	}
	return ""
}

func payloadInt(payload map[string]any, key string) *int {
	value, ok := payload[key]
	if !ok || value == nil {
		return nil
	}
	var number int
	switch v := value.(type) {
	case float64:
		if v != float64(int(v)) {
			return nil
		}
		number = int(v)
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return nil
		}
		number = int(parsed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return nil
		}
		number = parsed
	default:
		return nil
	}
	if number < 0 || number > 2000 {
		return nil
	}
	return &number
}

func vaccineDisplayName(code string) string {
	switch code {
	case "fvrhcp":
		return "猫三联"
	case "felv":
		return "FeLV"
	case "rabies":
		return "狂犬疫苗"
	case "other":
		return "其他疫苗"
	case "unknown":
		return "不确定疫苗"
	default:
		return "疫苗"
	}
}

func vaccineSourceFor(code string) string {
	if code == "felv" || code == "rabies" {
		return vaccineWSAVASource
	}
	return vaccineCoreSource
}

func nameOrGeneric(name string) string {
	if strings.TrimSpace(name) == "" {
		return "下一针"
	}
	return strings.TrimSpace(name) + "下一针"
}
