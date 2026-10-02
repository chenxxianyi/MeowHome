package app

import (
	"context"
	"encoding/json"
	"errors"
)

// ErrLLMUnavailable 表示模型功能已请求但供应商没有可用配置或当前不可用。
var ErrLLMUnavailable = errors.New("agent LLM unavailable")
var ErrLLMInvalidResponse = errors.New("agent LLM response is invalid")

// LLMProvider 隔离供应商协议。工具结果通过 tool_call_id 与原调用关联。
type LLMProvider interface {
	ChatWithTools(ctx context.Context, req LLMRequest, tools []ToolDef) (*LLMResponse, error)
}

type LLMRequest struct {
	Model       string
	Messages    []LLMMessage
	Temperature *float64
	MaxTokens   int
}

type LLMMessage struct {
	Role       string // system | developer | user | assistant | tool
	Content    string
	ToolCallID string
	ToolCalls  []LLMToolCall
}

type ToolDef struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

type LLMToolCall struct {
	ID        string
	Name      string
	Arguments string // 模型原始 JSON 字符串，由应用层校验
}

type LLMUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type LLMResponse struct {
	Content      string
	ToolCalls    []LLMToolCall
	Usage        LLMUsage
	Model        string
	FinishReason string
}

// FakeLLMProvider 用于不依赖真实密钥的工具循环和降级测试。
type FakeLLMProvider struct {
	ChatFunc func(context.Context, LLMRequest, []ToolDef) (*LLMResponse, error)
}

func (f FakeLLMProvider) ChatWithTools(ctx context.Context, req LLMRequest, tools []ToolDef) (*LLMResponse, error) {
	if f.ChatFunc == nil {
		return nil, ErrLLMUnavailable
	}
	return f.ChatFunc(ctx, req, tools)
}
