package app

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/platform/id"
)

// OCRDraftLine 一条商品行草稿（识别与用户修订共用同一结构）。
type OCRDraftLine struct {
	RawTitle      string
	Name          string
	Specification string
	Quantity      int
	Unit          string
	LineAmount    *int64 // 分；nil 表示未识别到
	Confidence    string // high/medium/low/none
	Evidence      string
}

// OCRDraft 订单草稿：平台、订单日期、实付总额与商品明细。
type OCRDraft struct {
	Platform    string
	OrderDate   string // YYYY-MM-DD；空表示未识别
	OrderNo     string
	TotalAmount *int64 // 分；nil 表示未识别
	Currency    string
	Lines       []OCRDraftLine
}

// draftJSON 模型输出的原始 JSON 结构（字段宽松：数字或字符串都接受）。
type draftJSON struct {
	Platform    json.RawMessage `json:"platform"`
	OrderDate   json.RawMessage `json:"order_date"`
	OrderNo     json.RawMessage `json:"order_no"`
	TotalAmount json.RawMessage `json:"total_amount"`
	Currency    json.RawMessage `json:"currency"`
	Lines       []draftLineJSON `json:"lines"`
}

type draftLineJSON struct {
	RawTitle      json.RawMessage `json:"raw_title"`
	Name          json.RawMessage `json:"name"`
	Specification json.RawMessage `json:"specification"`
	Quantity      json.RawMessage `json:"quantity"`
	Unit          json.RawMessage `json:"unit"`
	LineAmount    json.RawMessage `json:"line_amount"`
	Confidence    json.RawMessage `json:"confidence"`
	Evidence      json.RawMessage `json:"evidence"`
}

// extractJSONObject 从模型输出中提取第一个完整 JSON 对象。
// 容忍 markdown 代码块围栏、前后说明文字与嵌套花括号。
func extractJSONObject(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("empty OCR output")
	}
	// 优先处理 ```json ... ``` 围栏。
	if idx := strings.Index(s, "```"); idx >= 0 {
		rest := s[idx+3:]
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			rest = rest[nl+1:]
		}
		if end := strings.Index(rest, "```"); end >= 0 {
			rest = rest[:end]
		}
		s = strings.TrimSpace(rest)
	}
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return "", fmt.Errorf("no JSON object in OCR output")
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case ch == '\\':
				escaped = true
			case ch == '"':
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1], nil
			}
		}
	}
	return "", fmt.Errorf("unbalanced JSON object in OCR output")
}

// parseOCRJSON 把模型输出的 JSON 文本解析成订单草稿并做字段归一化。
func parseOCRJSON(raw string) (*OCRDraft, error) {
	fragment, err := extractJSONObject(raw)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(fragment))
	decoder.UseNumber()
	var payload draftJSON
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("invalid OCR JSON: %w", err)
	}
	draft := &OCRDraft{
		Platform:    normalizePlatform(rawString(payload.Platform)),
		OrderDate:   normalizeOrderDate(rawString(payload.OrderDate)),
		OrderNo:     strings.TrimSpace(rawString(payload.OrderNo)),
		TotalAmount: parseAmountCents(payload.TotalAmount),
		Currency:    normalizeCurrency(rawString(payload.Currency)),
	}
	for _, line := range payload.Lines {
		d := OCRDraftLine{
			RawTitle:      strings.TrimSpace(rawString(line.RawTitle)),
			Name:          strings.TrimSpace(rawString(line.Name)),
			Specification: strings.TrimSpace(rawString(line.Specification)),
			Unit:          strings.TrimSpace(rawString(line.Unit)),
			LineAmount:    parseAmountCents(line.LineAmount),
			Confidence:    normalizeConfidence(rawString(line.Confidence)),
			Evidence:      strings.TrimSpace(rawString(line.Evidence)),
		}
		d.Quantity = parseQuantity(line.Quantity)
		if d.Name == "" {
			d.Name = d.RawTitle
		}
		if d.Name == "" {
			continue
		}
		if d.Quantity <= 0 {
			d.Quantity = 1
			if d.Confidence == model.ConfidenceHigh {
				// 数量缺失时不允许保持高置信。
				d.Confidence = model.ConfidenceLow
			}
		}
		draft.Lines = append(draft.Lines, d)
	}
	return draft, nil
}

// validateDraft 校验草稿字段并返回问题摘要；返回空表示无需提示用户。
func validateDraft(draft *OCRDraft) []string {
	var issues []string
	if draft == nil {
		return []string{"未识别到订单内容"}
	}
	if draft.OrderDate != "" {
		if _, err := time.Parse("2006-01-02", draft.OrderDate); err != nil {
			issues = append(issues, "订单日期格式无法识别")
		}
	}
	if draft.TotalAmount != nil {
		if *draft.TotalAmount < 0 {
			issues = append(issues, "实付总额不能为负数")
		} else if *draft.TotalAmount > 100_000_000_00 {
			issues = append(issues, "实付总额超出合理范围")
		}
	}
	if len(draft.Lines) == 0 {
		issues = append(issues, "未识别到商品明细，请手动补充")
	}
	var sum int64
	complete := true
	for _, line := range draft.Lines {
		if line.Quantity <= 0 || line.Quantity > 10000 {
			issues = append(issues, fmt.Sprintf("商品「%s」数量超出范围", line.Name))
		}
		if line.LineAmount == nil {
			complete = false
			continue
		}
		if *line.LineAmount < 0 {
			issues = append(issues, fmt.Sprintf("商品「%s」金额不能为负数", line.Name))
		}
		sum += *line.LineAmount
	}
	if draft.TotalAmount != nil && complete && sum > 0 && *draft.TotalAmount > 0 {
		diff := sum - *draft.TotalAmount
		if diff < 0 {
			diff = -diff
		}
		tolerance := *draft.TotalAmount / 100
		if tolerance < 1 {
			tolerance = 1
		}
		if diff > tolerance {
			issues = append(issues, "明细金额合计与实付总额不一致")
		}
	}
	return issues
}

// draftStatus 根据商品明细数量推导导入状态：有明细为 parsed，无明细为 partial。
func draftStatus(draft *OCRDraft) string {
	if draft != nil && len(draft.Lines) > 0 {
		return model.PurchaseStatusParsed
	}
	return model.PurchaseStatusPartial
}

// draftToLines 把草稿转成可持久化的商品行。
func draftToLines(draft *OCRDraft, familyID, userID, importID string, now time.Time) []*model.PurchaseImportLine {
	if draft == nil {
		return nil
	}
	lines := make([]*model.PurchaseImportLine, 0, len(draft.Lines))
	for i, line := range draft.Lines {
		lines = append(lines, &model.PurchaseImportLine{
			Base:          model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
			FamilyID:      familyID,
			ImportID:      importID,
			LineNo:        i + 1,
			RawTitle:      line.RawTitle,
			Name:          line.Name,
			Specification: line.Specification,
			Quantity:      line.Quantity,
			Unit:          line.Unit,
			LineAmount:    line.LineAmount,
			Confidence:    line.Confidence,
			Evidence:      line.Evidence,
		})
	}
	return lines
}

// rawString 把 RawMessage 解释成字符串；数字按字面返回。
func rawString(v json.RawMessage) string {
	if len(v) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(v, &s); err == nil {
		return s
	}
	var n json.Number
	if err := json.Unmarshal(v, &n); err == nil {
		return n.String()
	}
	var b bool
	if err := json.Unmarshal(v, &b); err == nil {
		return strconv.FormatBool(b)
	}
	return ""
}

// parseAmountCents 解析金额为分；支持数字与带货币符号/分隔符的字符串。
func parseAmountCents(v json.RawMessage) *int64 {
	if len(v) == 0 {
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(v, &num); err == nil {
		return centsFromNumber(num.String())
	}
	var s string
	if err := json.Unmarshal(v, &s); err == nil {
		return centsFromString(s)
	}
	return nil
}

// centsFromNumber 把十进制数字字符串转成分。
func centsFromNumber(raw string) *int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return nil
	}
	cents := int64(math.Round(f * 100))
	return &cents
}

// centsFromString 把金额字符串转成分。
// 容忍 ¥/￥/$ 前缀、千分位逗号、末尾“元”/“人民币”与空白。
func centsFromString(raw string) *int64 {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	s = strings.TrimPrefix(s, "人民币")
	s = strings.Map(func(r rune) rune {
		switch r {
		case '¥', '￥', '$', ',', '元', ' ':
			return -1
		}
		return r
	}, s)
	if s == "" {
		return nil
	}
	if cents := centsFromNumber(s); cents != nil {
		return cents
	}
	return nil
}

// normalizePlatform 把平台别名归一到 taobao / jd / pdd / other。
func normalizePlatform(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case s == "":
		return ""
	case strings.Contains(s, "淘宝") || strings.Contains(s, "天猫") || s == "taobao" || s == "tmall":
		return "taobao"
	case strings.Contains(s, "京东") || s == "jd" || s == "jingdong":
		return "jd"
	case strings.Contains(s, "拼多多") || s == "pdd" || s == "pinduoduo":
		return "pdd"
	default:
		return "other"
	}
}

// normalizeCurrency 归一币种；空值默认 CNY。
func normalizeCurrency(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	if s == "" {
		return "CNY"
	}
	if len(s) > 8 {
		s = s[:8]
	}
	return s
}

// normalizeConfidence 归一置信度。
func normalizeConfidence(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case model.ConfidenceHigh, model.ConfidenceMedium, model.ConfidenceLow, model.ConfidenceNone:
		return s
	}
	// 兼容“高/中/低”与“高置信度”等写法。
	switch {
	case strings.Contains(s, "高"):
		return model.ConfidenceHigh
	case strings.Contains(s, "中"):
		return model.ConfidenceMedium
	case strings.Contains(s, "低"):
		return model.ConfidenceLow
	default:
		return model.ConfidenceNone
	}
}

// parseQuantity 解析数量；失败返回 0 由调用方兜底为 1。
func parseQuantity(v json.RawMessage) int {
	if len(v) == 0 {
		return 0
	}
	var n json.Number
	if err := json.Unmarshal(v, &n); err == nil {
		f, err := n.Float64()
		if err != nil {
			return 0
		}
		return int(math.Round(f))
	}
	var s string
	if err := json.Unmarshal(v, &s); err == nil {
		s = strings.TrimSpace(s)
		s = strings.TrimSuffix(s, "件")
		s = strings.TrimSuffix(s, "个")
		s = strings.TrimSuffix(s, "x")
		s = strings.TrimSuffix(s, "X")
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int(math.Round(f))
		}
	}
	return 0
}

// normalizeOrderDate 把常见日期写法归一到 YYYY-MM-DD。
// 支持 2026-10-21 / 2026/10/21 / 2026.10.21 / 2026年10月21日 / 带时间的日期串 / 仅“10月21日”。
func normalizeOrderDate(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// 只取日期部分：截断空格与 T 之后的时间。
	if idx := strings.IndexAny(s, " T"); idx > 0 {
		s = s[:idx]
	}
	s = strings.ReplaceAll(s, "年", "-")
	s = strings.ReplaceAll(s, "月", "-")
	s = strings.ReplaceAll(s, "日", "")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	parts := strings.Split(s, "-")
	if len(parts) == 3 {
		year, errY := strconv.Atoi(parts[0])
		month, errM := strconv.Atoi(parts[1])
		day, errD := strconv.Atoi(parts[2])
		if errY == nil && errM == nil && errD == nil && year >= 2000 && year <= 2100 && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		}
		return ""
	}
	if len(parts) == 2 {
		// “10月21日”或“10-21”：补当前年份。
		month, errM := strconv.Atoi(parts[0])
		day, errD := strconv.Atoi(parts[1])
		now := time.Now()
		if errM == nil && errD == nil && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			return fmt.Sprintf("%04d-%02d-%02d", now.Year(), month, day)
		}
	}
	return ""
}
