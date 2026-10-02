package app

import (
	"context"
	"testing"
)

func TestFakeLLMProviderDrivesToolResultRoundTrip(t *testing.T) {
	calls := 0
	fake := FakeLLMProvider{ChatFunc: func(ctx context.Context, req LLMRequest, tools []ToolDef) (*LLMResponse, error) {
		calls++
		if calls == 1 {
			if len(req.Messages) != 1 || req.Messages[0].Role != "user" {
				t.Fatal("first request lost user message")
			}
			return &LLMResponse{ToolCalls: []LLMToolCall{{ID: "call-1", Name: "listRecords", Arguments: `{"days":2}`}}, FinishReason: "tool_calls"}, nil
		}
		if len(req.Messages) != 3 || req.Messages[1].ToolCalls[0].ID != "call-1" || req.Messages[2].ToolCallID != "call-1" {
			t.Fatalf("tool result lost linkage: %+v", req.Messages)
		}
		return &LLMResponse{Content: "查询完成", FinishReason: "stop"}, nil
	}}
	ctx := context.Background()
	first, err := fake.ChatWithTools(ctx, LLMRequest{Messages: []LLMMessage{{Role: "user", Content: "近两天记录"}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	final, err := fake.ChatWithTools(ctx, LLMRequest{Messages: []LLMMessage{
		{Role: "user", Content: "近两天记录"},
		{Role: "assistant", ToolCalls: first.ToolCalls},
		{Role: "tool", ToolCallID: first.ToolCalls[0].ID, Content: `{"count":2}`},
	}}, nil)
	if err != nil || calls != 2 || final.Content != "查询完成" {
		t.Fatalf("fake round trip: %+v %v calls=%d", final, err, calls)
	}
}
