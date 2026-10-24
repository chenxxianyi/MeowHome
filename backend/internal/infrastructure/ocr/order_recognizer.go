// Package ocr 实现订单截图识别的 OpenAI-compatible 视觉提供方。
//
// 与 ai.ChatCompletionsProvider 相同，本包只负责协议往返：图片以 data URL
// 随用户消息发送，模型输出原文返回给应用层解析。解析、校验与业务落库
// 留在 app 层，Provider 可替换为本地方案或其它云服务。
package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/platform/config"
	apperr "github.com/meowhome/backend/internal/platform/errors"
)

const maxResponseBytes int64 = 1 << 20
const maxRoundTrip = 25 * time.Second

// ChatCompletionsProvider 通过 OpenAI-compatible 视觉接口识别订单截图。
type ChatCompletionsProvider struct {
	config config.OCR
	client *http.Client
}

// NewChatCompletionsProvider 创建 OCR 提供方。
func NewChatCompletionsProvider(cfg config.OCR) *ChatCompletionsProvider {
	timeout := cfg.Timeout
	if timeout <= 0 || timeout > maxRoundTrip {
		timeout = maxRoundTrip
	}
	return &ChatCompletionsProvider{
		config: cfg,
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Available 表示配置是否齐备（启用、地址、密钥与模型）。
func (p *ChatCompletionsProvider) Available() bool {
	return p.config.Enabled && strings.TrimSpace(p.config.BaseURL) != "" &&
		strings.TrimSpace(p.config.APIKey) != "" && strings.TrimSpace(p.config.Model) != ""
}

// recognizePrompt 约束模型输出严格的订单 JSON。
const recognizePrompt = `你是订单截图识别器，只输出 JSON，不要输出任何解释。
请从订单截图中提取：平台（taobao/jd/pdd/other）、订单日期（YYYY-MM-DD）、订单号（可选，忽略收货信息）、
币种（默认 CNY）、实付总额（数字，单位元）、商品明细数组。
商品明细每项包含：raw_title（截图中的原始标题）、name（规范商品名，去掉促销词）、
specification（规格，如 2kg/袋）、quantity（整数）、unit（件/袋/包/盒/罐/克/千克 等）、
line_amount（该行金额，数字，单位元；未显示时省略）、confidence（high/medium/low）、
evidence（依据的截图文案，可省略）。
只允许出现商品明细与金额，不要输出地址、电话、收件人、支付账号等隐私信息。
格式示例：
{"platform":"taobao","order_date":"2026-10-21","order_no":"","currency":"CNY","total_amount":199.00,"lines":[{"raw_title":"皇家成猫粮 K36 2kg","name":"皇家成猫粮","specification":"2kg","quantity":1,"unit":"袋","line_amount":99.00,"confidence":"high"}]}`

type imagePart struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

type textPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type wireMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type wireRequest struct {
	Model       string        `json:"model"`
	Messages    []wireMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type wireResponse struct {
	Choices []struct {
		Message struct {
			Content *string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// RecognizeOrder 把若干张订单截图交给视觉模型，返回模型输出原文。
// 未配置或不可用时返回 OCR_UNAVAILABLE；网络/协议失败返回 OCR_FAILED。
func (p *ChatCompletionsProvider) RecognizeOrder(ctx context.Context, images [][]byte) (string, error) {
	if !p.Available() {
		return "", apperr.New(apperr.TypeExternal, apperr.CodeOCRUnavailable,
			"OCR 未配置：请在 .env 中启用 OCR_ENABLED 并填写 OCR_BASE_URL / OCR_API_KEY / OCR_MODEL")
	}
	if len(images) == 0 {
		return "", apperr.InvalidRequest(apperr.CodeValidationFailed, "至少需要一张订单截图")
	}

	parts := make([]any, 0, len(images)+1)
	parts = append(parts, textPart{Type: "text", Text: "请识别这张订单截图。"})
	for _, img := range images {
		url, err := dataURL(img)
		if err != nil {
			return "", apperr.InvalidRequest(apperr.CodeValidationFailed, "订单截图格式不受支持")
		}
		part := imagePart{Type: "image_url"}
		part.ImageURL.URL = url
		parts = append(parts, part)
	}

	body, err := json.Marshal(wireRequest{
		Model: p.config.Model,
		Messages: []wireMessage{
			{Role: "system", Content: recognizePrompt},
			{Role: "user", Content: parts},
		},
		Temperature: 0,
		MaxTokens:   2048,
	})
	if err != nil {
		return "", apperr.Wrap(apperr.TypeExternal, apperr.CodeOCRFailed, "编码 OCR 请求失败", err)
	}

	endpoint := strings.TrimRight(strings.TrimSpace(p.config.BaseURL), "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", apperr.Wrap(apperr.TypeExternal, apperr.CodeOCRFailed, "创建 OCR 请求失败", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.config.APIKey))

	resp, err := p.client.Do(req)
	if err != nil {
		return "", apperr.Wrap(apperr.TypeExternal, apperr.CodeOCRFailed, "OCR 服务请求失败", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return "", apperr.Wrap(apperr.TypeExternal, apperr.CodeOCRFailed, "读取 OCR 响应失败", err)
	}
	if int64(len(payload)) > maxResponseBytes {
		return "", apperr.New(apperr.TypeExternal, apperr.CodeOCRFailed, "OCR 响应超出大小限制")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", apperr.New(apperr.TypeExternal, apperr.CodeOCRFailed,
			fmt.Sprintf("OCR 服务返回异常状态（HTTP %d）", resp.StatusCode))
	}

	var parsed wireResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return "", apperr.Wrap(apperr.TypeExternal, apperr.CodeOCRFailed, "OCR 响应不是有效 JSON", err)
	}
	if parsed.Error != nil {
		return "", apperr.New(apperr.TypeExternal, apperr.CodeOCRFailed, "OCR 服务返回错误: "+parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Content == nil {
		return "", apperr.New(apperr.TypeExternal, apperr.CodeOCRFailed, "OCR 服务未返回内容")
	}
	text := strings.TrimSpace(*parsed.Choices[0].Message.Content)
	if text == "" {
		return "", apperr.New(apperr.TypeExternal, apperr.CodeOCRFailed, "OCR 服务返回空内容")
	}
	return text, nil
}

// dataURL 把图片字节包成 data URL；不支持的格式返回错误。
func dataURL(image []byte) (string, error) {
	mime := ""
	switch {
	case len(image) >= 3 && bytes.Equal(image[:3], []byte{0xff, 0xd8, 0xff}):
		mime = "image/jpeg"
	case len(image) >= 8 && bytes.Equal(image[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		mime = "image/png"
	case len(image) >= 6 && (string(image[:6]) == "GIF87a" || string(image[:6]) == "GIF89a"):
		mime = "image/gif"
	case len(image) >= 12 && string(image[:4]) == "RIFF" && string(image[8:12]) == "WEBP":
		mime = "image/webp"
	default:
		return "", errors.New("unsupported image format")
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(image), nil
}
