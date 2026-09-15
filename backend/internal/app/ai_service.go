package app

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
)

// AIService 提供自然语言记录解析与每日摘要。
//
// 当前实现为确定性规则解析（无需外部依赖）。配置 AI_ENABLED=true 并
// 提供 AI_BASE_URL/AI_API_KEY/AI_MODEL 后，可在 parseWithProvider 中
// 替换为真实 LLM 调用——上层接口与数据结构保持不变。
type AIService struct {
	records  repository.DailyRecordRepo
	reports  repository.AnalysisReportRepo
	sessions repository.AIRepo
	cats     repository.CatRepo
	members  repository.MemberRepo
}

// NewAIService 创建 AI 服务。
func NewAIService(
	records repository.DailyRecordRepo,
	reports repository.AnalysisReportRepo,
	sessions repository.AIRepo,
	cats repository.CatRepo,
	members repository.MemberRepo,
) *AIService {
	return &AIService{records: records, reports: reports, sessions: sessions, cats: cats, members: members}
}

// AIParsedField 单个解析字段。
type AIParsedField struct {
	Key        string `json:"key"`
	Value      any    `json:"value"`
	Confidence string `json:"confidence"`
	Note       string `json:"note,omitempty"`
}

// AIParsedRecord 解析出的一条记录。
type AIParsedRecord struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	CatID  string          `json:"catId,omitempty"`
	Fields []AIParsedField `json:"fields"`
}

// AIParseSessionEnvelope 解析会话响应（与前端 AIParseSession 对齐）。
type AIParseSessionEnvelope struct {
	ID            string           `json:"id"`
	OriginalInput string           `json:"originalInput"`
	ParsedAt      string           `json:"parsedAt"`
	Model         string           `json:"model"`
	Records       []AIParsedRecord `json:"records"`
}

// AIEvidence 摘要引用的证据。
type AIEvidence struct {
	Type    string `json:"type"`
	CatID   string `json:"catId,omitempty"`
	Time    string `json:"time"`
	Content string `json:"content"`
}

// AISummaryEnvelope 每日摘要响应（与前端 AIAnalysisReport 对齐）。
type AISummaryEnvelope struct {
	ID          string       `json:"id"`
	GeneratedAt string       `json:"generatedAt"`
	RecordCount int          `json:"recordCount"`
	Body        string       `json:"body"`
	Evidence    []AIEvidence `json:"evidence"`
}

// 关键词 → 记录类型。顺序即匹配优先级。
var keywordRules = []struct {
	recordType string
	keywords   []string
}{
	{"vomit", []string{"呕吐", "吐了", "吐了", "吐", "干呕"}},
	{"medication", []string{"喂药", "服药", "吃药", "用药", "药"}},
	{"weight", []string{"体重", "称重", "公斤", "kg"}},
	{"drinking", []string{"喝水", "饮水", "喝了水"}},
	{"elimination", []string{"排便", "拉稀", "软便", "便便", "猫砂", "尿"}},
	{"mental", []string{"精神", "没精神", "萎靡", "活泼"}},
	{"feeding", []string{"喂食", "吃了", "没吃", "猫粮", "进食", "食欲", "罐头"}},
}

var (
	reNumber = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(g|克|ml|毫升|kg|公斤|次|片|粒|袋|盒|支)`)
	reClock  = regexp.MustCompile(`(\d{1,2})[:：](\d{2})`)
)

// Parse 把一段自然语言解析为结构化记录。
func (s *AIService) Parse(ctx context.Context, familyID, userID, text string) (*AIParseSessionEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "input is required")
	}

	now := time.Now().UTC()
	sessionID := model.NewBase(familyID, userID).ID

	cats, _ := s.cats.ListByFamily(ctx, familyID)
	parsed := parseText(text, cats)

	raw, _ := json.Marshal(parsed)
	session := &model.AIParseSession{
		Base:          model.Base{ID: sessionID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID:      familyID,
		OriginalInput: text,
		Status:        "ready",
		Model:         "rule-based-v1",
		Result:        string(raw),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to persist AI session", err)
	}

	return &AIParseSessionEnvelope{
		ID:            sessionID,
		OriginalInput: text,
		ParsedAt:      now.Format(time.RFC3339),
		Model:         "rule-based-v1",
		Records:       parsed,
	}, nil
}

// Summary 生成家庭当日摘要并落库。
func (s *AIService) Summary(ctx context.Context, familyID, userID string) (*AISummaryEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}

	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	rows, err := s.records.ListByFamily(ctx, repository.DailyRecordQuery{
		FamilyID: familyID,
		From:     dayStart.UTC().Format(time.RFC3339),
		To:       dayStart.Add(24*time.Hour - time.Millisecond).UTC().Format(time.RFC3339),
		Limit:    500,
	})
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to load records", err)
	}

	cats, _ := s.cats.ListByFamily(ctx, familyID)
	nameOf := map[string]string{}
	for _, c := range cats {
		nameOf[c.ID] = c.Name
	}

	evidence := make([]AIEvidence, 0, len(rows))
	abnormal := []string{}
	for _, r := range rows {
		catID := ""
		if len(r.CatIDs) > 0 {
			catID = r.CatIDs[0]
		}
		content := r.Title
		if content == "" {
			content = r.Note
		}
		if content == "" {
			content = labelOf(r.RecordType)
		}
		evidence = append(evidence, AIEvidence{
			Type:    r.RecordType,
			CatID:   catID,
			Time:    r.OccurredAt.Format("15:04"),
			Content: content,
		})
		if r.Severity == "danger" || r.Severity == "warning" {
			abnormal = append(abnormal, fmt.Sprintf("%s%s", nameOf[catID], content))
		}
	}

	body := buildSummaryBody(len(rows), abnormal, nameOf, cats)

	report := &model.AnalysisReport{
		Base:        model.Base{ID: model.NewBase(familyID, userID).ID, CreatedBy: userID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		FamilyID:    familyID,
		RecordCount: len(rows),
		Content:     body,
	}
	if ev, err := json.Marshal(evidence); err == nil {
		report.Evidence = string(ev)
	}
	if err := s.reports.Create(ctx, report); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to persist report", err)
	}

	return &AISummaryEnvelope{
		ID:          report.ID,
		GeneratedAt: report.CreatedAt.Format(time.RFC3339),
		RecordCount: report.RecordCount,
		Body:        body,
		Evidence:    evidence,
	}, nil
}

func buildSummaryBody(recordCount int, abnormal []string, nameOf map[string]string, cats []*model.Cat) string {
	if recordCount == 0 {
		return "今日尚无记录。添加喂食、饮水或排便记录后，这里会自动生成摘要。"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "今日共记录 %d 条数据。", recordCount)
	if len(abnormal) == 0 {
		b.WriteString("各项指标未见异常，继续保持规律记录。")
	} else {
		fmt.Fprintf(&b, "其中 %d 项需要关注：%s。建议持续观察。", len(abnormal), strings.Join(abnormal, "；"))
	}
	if len(cats) > 0 {
		names := make([]string, 0, len(cats))
		for _, c := range cats {
			names = append(names, c.Name)
		}
		fmt.Fprintf(&b, "当前家庭成员：%s。", strings.Join(names, "、"))
	}
	return b.String()
}

// parseText 按句子切分，逐句识别猫咪与记录类型。
func parseText(text string, cats []*model.Cat) []AIParsedRecord {
	sentences := splitSentences(text)
	out := []AIParsedRecord{}

	for idx, sent := range sentences {
		recordType := matchRecordType(sent)
		if recordType == "" {
			continue
		}
		catID := matchCat(sent, cats)

		fields := []AIParsedField{}
		if t := matchClock(sent); t != "" {
			fields = append(fields, AIParsedField{Key: "time", Value: t, Confidence: "medium"})
		}
		if n, unit := matchNumber(sent); n != "" {
			key := "amount"
			note := ""
			switch recordType {
			case "feeding":
				key, note = "consumed", "AI 推断为实际进食量"
			case "vomit":
				key = "count"
			case "weight":
				key = "weight"
			}
			fields = append(fields, AIParsedField{Key: key, Value: n + unit, Confidence: "medium", Note: note})
		}
		fields = append(fields, AIParsedField{Key: "notes", Value: sent, Confidence: "low"})

		out = append(out, AIParsedRecord{
			ID:     "rec-" + strconv.Itoa(idx+1),
			Type:   recordType,
			CatID:  catID,
			Fields: fields,
		})
	}
	return out
}

func splitSentences(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == '。' || r == '！' || r == '？' || r == '\n' || r == '；' || r == ';' || r == '.'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if s := strings.TrimSpace(f); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func matchRecordType(sentence string) string {
	for _, rule := range keywordRules {
		for _, kw := range rule.keywords {
			if strings.Contains(sentence, kw) {
				return rule.recordType
			}
		}
	}
	return ""
}

func matchCat(sentence string, cats []*model.Cat) string {
	for _, c := range cats {
		if c.Name != "" && strings.Contains(sentence, c.Name) {
			return c.ID
		}
	}
	return ""
}

func matchClock(sentence string) string {
	if m := reClock.FindStringSubmatch(sentence); m != nil {
		return fmt.Sprintf("%02s:%s", m[1], m[2])
	}
	switch {
	case strings.Contains(sentence, "早上") || strings.Contains(sentence, "早餐"):
		return "08:00"
	case strings.Contains(sentence, "中午") || strings.Contains(sentence, "午餐"):
		return "12:00"
	case strings.Contains(sentence, "下午"):
		return "15:00"
	case strings.Contains(sentence, "晚上") || strings.Contains(sentence, "晚餐"):
		return "19:00"
	case strings.Contains(sentence, "夜里") || strings.Contains(sentence, "半夜"):
		return "23:00"
	}
	return ""
}

func matchNumber(sentence string) (string, string) {
	if m := reNumber.FindStringSubmatch(sentence); m != nil {
		return m[1], m[2]
	}
	return "", ""
}
