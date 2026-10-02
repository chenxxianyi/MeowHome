// Package ai 实现明确声明 Chat Completions 工具调用协议的供应商适配器。
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/platform/config"
)

const maxResponseBytes int64 = 1 << 20
const maxRoundTrip = 25 * time.Second

type ChatCompletionsProvider struct {
	config config.AI
	client *http.Client
}

func NewChatCompletionsProvider(cfg config.AI) *ChatCompletionsProvider {
	timeout := cfg.Timeout
	if timeout <= 0 || timeout > maxRoundTrip {
		timeout = maxRoundTrip
	}
	return &ChatCompletionsProvider{config: cfg, client: &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

var _ app.LLMProvider = (*ChatCompletionsProvider)(nil)

type wireMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
}
type wireToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
type wireTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}
type wireRequest struct {
	Model       string        `json:"model"`
	Messages    []wireMessage `json:"messages"`
	Tools       []wireTool    `json:"tools,omitempty"`
	Stream      bool          `json:"stream"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}
type wireResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content   *string        `json:"content"`
			ToolCalls []wireToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func endpoint(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", app.ErrLLMUnavailable
	}
	if u.Scheme == "http" {
		ip := net.ParseIP(u.Hostname())
		if u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return "", app.ErrLLMUnavailable
		}
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(u.Path, "/chat/completions") {
		u.Path += "/chat/completions"
	}
	return u.String(), nil
}

func (p *ChatCompletionsProvider) ChatWithTools(ctx context.Context, req app.LLMRequest, tools []app.ToolDef) (*app.LLMResponse, error) {
	if !p.config.Enabled || strings.TrimSpace(p.config.APIKey) == "" {
		return nil, app.ErrLLMUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, maxRoundTrip)
	defer cancel()
	target, err := endpoint(p.config.BaseURL)
	if err != nil {
		return nil, err
	}
	model := req.Model
	if model == "" {
		model = p.config.Model
	}
	if model == "" || len(req.Messages) == 0 {
		return nil, app.ErrLLMUnavailable
	}
	wire := wireRequest{Model: model, Stream: false, Temperature: req.Temperature, MaxTokens: req.MaxTokens}
	for _, item := range req.Messages {
		if item.Role != "system" && item.Role != "developer" && item.Role != "user" && item.Role != "assistant" && item.Role != "tool" {
			return nil, fmt.Errorf("%w: unsupported role", app.ErrLLMInvalidResponse)
		}
		if item.Role == "tool" && item.ToolCallID == "" {
			return nil, fmt.Errorf("%w: missing tool call ID", app.ErrLLMInvalidResponse)
		}
		wm := wireMessage{Role: item.Role, Content: item.Content, ToolCallID: item.ToolCallID}
		for _, call := range item.ToolCalls {
			if call.ID == "" || call.Name == "" {
				return nil, fmt.Errorf("%w: incomplete outgoing tool call", app.ErrLLMInvalidResponse)
			}
			wc := wireToolCall{ID: call.ID, Type: "function"}
			wc.Function.Name, wc.Function.Arguments = call.Name, call.Arguments
			wm.ToolCalls = append(wm.ToolCalls, wc)
		}
		wire.Messages = append(wire.Messages, wm)
	}
	for _, tool := range tools {
		if tool.Name == "" || !json.Valid(tool.Parameters) {
			return nil, fmt.Errorf("%w: invalid tool definition", app.ErrLLMInvalidResponse)
		}
		wt := wireTool{Type: "function"}
		wt.Function.Name, wt.Function.Description, wt.Function.Parameters = tool.Name, tool.Description, tool.Parameters
		wire.Tools = append(wire.Tools, wt)
	}
	payload, err := json.Marshal(wire)
	if err != nil {
		return nil, app.ErrLLMInvalidResponse
	}
	attempts := p.config.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 3 {
		attempts = 3
	}
	for attempt := 0; attempt < attempts; attempt++ {
		result, retry, err := p.once(ctx, target, payload)
		if err == nil {
			return result, nil
		}
		if !retry || attempt+1 == attempts || ctx.Err() != nil {
			return nil, err
		}
		timer := time.NewTimer(time.Duration(100*(1<<attempt)) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, app.ErrLLMUnavailable
}

func (p *ChatCompletionsProvider) once(ctx context.Context, target string, payload []byte) (*app.LLMResponse, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(payload))
	if err != nil {
		return nil, false, app.ErrLLMUnavailable
	}
	request.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return nil, false, ctx.Err()
		}
		return nil, true, app.ErrLLMUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		retry := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
		return nil, retry, fmt.Errorf("%w: provider returned status %d", app.ErrLLMUnavailable, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || int64(len(body)) > maxResponseBytes {
		return nil, false, app.ErrLLMInvalidResponse
	}
	var parsed wireResponse
	if json.Unmarshal(body, &parsed) != nil || len(parsed.Choices) == 0 {
		return nil, false, app.ErrLLMInvalidResponse
	}
	choice := parsed.Choices[0]
	out := &app.LLMResponse{Model: parsed.Model, FinishReason: choice.FinishReason, Usage: app.LLMUsage{PromptTokens: parsed.Usage.PromptTokens, CompletionTokens: parsed.Usage.CompletionTokens, TotalTokens: parsed.Usage.TotalTokens}}
	if choice.Message.Content != nil {
		out.Content = *choice.Message.Content
	}
	for _, call := range choice.Message.ToolCalls {
		if call.Type != "function" || call.ID == "" || call.Function.Name == "" {
			return nil, false, app.ErrLLMInvalidResponse
		}
		out.ToolCalls = append(out.ToolCalls, app.LLMToolCall{ID: call.ID, Name: call.Function.Name, Arguments: call.Function.Arguments})
	}
	if out.Content == "" && len(out.ToolCalls) == 0 {
		return nil, false, app.ErrLLMInvalidResponse
	}
	return out, false, nil
}
