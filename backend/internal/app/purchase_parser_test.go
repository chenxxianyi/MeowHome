package app

import (
	"strings"
	"testing"
	"time"
)

// TestParseOCRJSONNormalOrder 覆盖正常淘宝订单：数字金额、日期、单商品行。
func TestParseOCRJSONNormalOrder(t *testing.T) {
	raw := `{"platform":"taobao","order_date":"2026-10-21","order_no":"TAO123","currency":"CNY","total_amount":199.00,"lines":[{"raw_title":"皇家成猫粮 K36 2kg 全价猫粮","name":"皇家成猫粮","specification":"2kg/袋","quantity":1,"unit":"袋","line_amount":99.00,"confidence":"high","evidence":"实付款 ¥199.00"}]}`
	draft, err := parseOCRJSON(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if draft.Platform != "taobao" {
		t.Errorf("platform = %q, want taobao", draft.Platform)
	}
	if draft.OrderDate != "2026-10-21" {
		t.Errorf("order_date = %q", draft.OrderDate)
	}
	if draft.OrderNo != "TAO123" {
		t.Errorf("order_no = %q", draft.OrderNo)
	}
	if draft.TotalAmount == nil || *draft.TotalAmount != 19900 {
		t.Fatalf("total = %v, want 19900 cents", draft.TotalAmount)
	}
	if draft.Currency != "CNY" {
		t.Errorf("currency = %q", draft.Currency)
	}
	if len(draft.Lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(draft.Lines))
	}
	line := draft.Lines[0]
	if line.Name != "皇家成猫粮" || line.Specification != "2kg/袋" || line.Quantity != 1 || line.Unit != "袋" {
		t.Errorf("line = %+v", line)
	}
	if line.LineAmount == nil || *line.LineAmount != 9900 {
		t.Errorf("line amount = %v", line.LineAmount)
	}
	if line.Confidence != "high" {
		t.Errorf("confidence = %q", line.Confidence)
	}
}

// TestParseOCRJSONMarkdownFence 覆盖模型把 JSON 包在 markdown 代码块中的输出。
func TestParseOCRJSONMarkdownFence(t *testing.T) {
	raw := "识别结果如下：\n```json\n{\"platform\":\"京东\",\"order_date\":\"2026/10/21\",\"total_amount\":\"1,299.00\",\"lines\":[{\"name\":\"猫砂 10kg\",\"quantity\":\"2件\",\"line_amount\":\"¥99.50\",\"confidence\":\"中\"}]}\n```\n以上。"
	draft, err := parseOCRJSON(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if draft.Platform != "jd" {
		t.Errorf("platform = %q, want jd（京东归一）", draft.Platform)
	}
	if draft.OrderDate != "2026-10-21" {
		t.Errorf("order_date = %q, want 2026-10-21", draft.OrderDate)
	}
	if draft.TotalAmount == nil || *draft.TotalAmount != 129900 {
		t.Errorf("total = %v, want 129900", draft.TotalAmount)
	}
	if len(draft.Lines) != 1 {
		t.Fatalf("lines = %d", len(draft.Lines))
	}
	line := draft.Lines[0]
	if line.Quantity != 2 {
		t.Errorf("quantity = %d, want 2（数量字符串归一）", line.Quantity)
	}
	if line.LineAmount == nil || *line.LineAmount != 9950 {
		t.Errorf("line amount = %v, want 9950", line.LineAmount)
	}
	if line.Confidence != "medium" {
		t.Errorf("confidence = %q, want medium（中→medium）", line.Confidence)
	}
}

// TestParseOCRJSONChineseDateAndCurrency 覆盖中文日期、人民币前缀与千分位。
func TestParseOCRJSONChineseDateAndCurrency(t *testing.T) {
	draft, err := parseOCRJSON(`{"platform":"拼多多","order_date":"2026年10月21日","total_amount":"￥199.00元","lines":[{"name":"猫抓板","quantity":3,"line_amount":"59.70"}]}`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if draft.Platform != "pdd" {
		t.Errorf("platform = %q", draft.Platform)
	}
	if draft.OrderDate != "2026-10-21" {
		t.Errorf("order_date = %q", draft.OrderDate)
	}
	if draft.TotalAmount == nil || *draft.TotalAmount != 19900 {
		t.Errorf("total = %v", draft.TotalAmount)
	}
	if draft.Lines[0].Quantity != 3 {
		t.Errorf("quantity = %d", draft.Lines[0].Quantity)
	}
}

// TestParseOCRJSONMissingLineAmount 覆盖未识别到行金额、缺规格与低置信字段。
func TestParseOCRJSONMissingLineAmount(t *testing.T) {
	draft, err := parseOCRJSON(`{"platform":"taobao","total_amount":50,"lines":[{"raw_title":"冻干 500g","name":"冻干","confidence":"low"}]}`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	line := draft.Lines[0]
	if line.LineAmount != nil {
		t.Errorf("line amount = %v, want nil", *line.LineAmount)
	}
	if line.Quantity != 1 {
		t.Errorf("quantity = %d, want 兜底 1", line.Quantity)
	}
	// 数量兜底后低置信仍保留。
	if line.Confidence != "low" {
		t.Errorf("confidence = %q", line.Confidence)
	}
	if line.Name != "冻干" || line.RawTitle != "冻干 500g" {
		t.Errorf("name/raw_title = %q/%q", line.Name, line.RawTitle)
	}
}

// TestParseOCRJSONNoLines 覆盖订单截图没有商品明细（只有金额）的场景。
func TestParseOCRJSONNoLines(t *testing.T) {
	draft, err := parseOCRJSON(`{"platform":"taobao","total_amount":88.8}`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(draft.Lines) != 0 {
		t.Fatalf("lines = %d, want 0", len(draft.Lines))
	}
	issues := validateDraft(draft)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "商品明细") {
			found = true
		}
	}
	if !found {
		t.Errorf("issues = %v, want 未识别到商品明细", issues)
	}
	if status := draftStatus(draft); status != "partial" {
		t.Errorf("status = %q, want partial", status)
	}
}

// TestExtractJSONObjectBalanced 覆盖嵌套花括号与前后文字。
func TestExtractJSONObjectBalanced(t *testing.T) {
	raw := "模型说明 {\"a\":{\"b\":1},\"c\":\"}\"} 结束"
	got, err := extractJSONObject(raw)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if got != `{"a":{"b":1},"c":"}"}` {
		t.Errorf("got %q", got)
	}
	if _, err := extractJSONObject("没有 JSON"); err == nil {
		t.Error("expected error for non-JSON output")
	}
}

// TestValidateDraftTotalMismatch 覆盖明细合计与实付总额不一致（优惠/分摊）。
func TestValidateDraftTotalMismatch(t *testing.T) {
	draft := &OCRDraft{
		TotalAmount: centsPtr(9900),
		Lines: []OCRDraftLine{
			{Name: "A", Quantity: 1, LineAmount: centsPtr(6000)},
			{Name: "B", Quantity: 1, LineAmount: centsPtr(5000)},
		},
	}
	issues := validateDraft(draft)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "不一致") {
			found = true
		}
	}
	if !found {
		t.Errorf("issues = %v, want 明细金额合计与实付总额不一致", issues)
	}
	// 容差范围内不应提示。
	draft.Lines[1].LineAmount = centsPtr(3890)
	if issues := validateDraft(draft); len(issues) != 0 {
		t.Errorf("issues = %v, want 空（在 1 元容差内）", issues)
	}
}

// TestNormalizeOrderDateVariants 覆盖常见日期写法。
func TestNormalizeOrderDateVariants(t *testing.T) {
	now := time.Now()
	currentYear := itoa(now.Year())
	cases := map[string]string{
		"2026-10-21":          "2026-10-21",
		"2026/10/21":          "2026-10-21",
		"2026.10.21":          "2026-10-21",
		"2026年10月21日":         "2026-10-21",
		"2026-10-21 12:00:00": "2026-10-21",
		"10-21":               currentYear + "-10-21",
		"10月21日":              currentYear + "-10-21",
	}
	for input, wantDate := range cases {
		if got := normalizeOrderDate(input); got != wantDate {
			t.Errorf("normalizeOrderDate(%q) = %q, want %q", input, got, wantDate)
		}
	}
	// 无效日期一律返回空串，由用户手工修正。
	for _, invalid := range []string{"", "2026-13-40", "abc"} {
		if got := normalizeOrderDate(invalid); got != "" {
			t.Errorf("normalizeOrderDate(%q) = %q, want empty", invalid, got)
		}
	}
}

func centsPtr(v int64) *int64 { return &v }

func itoa(v int) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	out := make([]byte, 0, 4)
	for v > 0 {
		out = append([]byte{digits[v%10]}, out...)
		v /= 10
	}
	return string(out)
}
