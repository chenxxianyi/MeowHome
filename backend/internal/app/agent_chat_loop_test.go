package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
)

func TestAgentTurnSingleToolAndAuthorizedCitations(t *testing.T) {
	s, _ := newAgentServiceForTest()
	s.records = &toolRecordRepo{rows: []*model.DailyRecord{toolRecord("record-1", "feeding", `{"amount":30}`, s.clock().Add(-time.Hour))}}
	calls := 0
	s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(_ context.Context, req LLMRequest, defs []ToolDef) (*LLMResponse, error) {
		calls++
		if len(defs) != 4 || len(req.Messages) < 4 || req.Messages[2].Role != "user" || !strings.Contains(req.Messages[2].Content, `"id":"cat"`) || strings.Contains(req.Messages[1].Content, `"name"`) {
			t.Fatalf("missing authorized context: %+v", req)
		}
		if calls == 1 {
			return &LLMResponse{Model: "fake", ToolCalls: []LLMToolCall{{ID: "call-1", Name: "listRecords", Arguments: `{"cat_id":"cat","type":"feeding","days":7}`}}}, nil
		}
		last := req.Messages[len(req.Messages)-1]
		if last.Role != "tool" || last.ToolCallID != "call-1" || !strings.Contains(last.Content, "record-1") {
			t.Fatalf("tool call linkage lost: %+v", last)
		}
		return &LLMResponse{Model: "fake", Content: `{"answer":"有一条进食记录。","source_ids":["record-1","forged"]}`}, nil
	}})
	result, err := s.runAgentTurn(context.Background(), AgentToolScope{FamilyID: "fam", UserID: "user"}, []LLMMessage{{Role: "user", Content: "小白进食记录"}})
	if err != nil || calls != 2 || result.Answer == "" || len(result.Evidence) != 1 || result.Evidence[0].SourceID != "record-1" || len(result.ToolResults) != 1 {
		t.Fatalf("turn result: %+v %v", result, err)
	}
}

func TestAgentTurnReusesDuplicateToolResult(t *testing.T) {
	s, _ := newAgentServiceForTest()
	repo := &toolRecordRepo{}
	s.records = repo
	modelCalls := 0
	s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(_ context.Context, req LLMRequest, _ []ToolDef) (*LLMResponse, error) {
		modelCalls++
		if modelCalls <= 2 {
			return &LLMResponse{ToolCalls: []LLMToolCall{{ID: "call-" + string(rune('0'+modelCalls)), Name: "listRecords", Arguments: `{"cat_id":"cat"}`}}}, nil
		}
		return &LLMResponse{Content: `{"answer":"没有找到已记录的结果，不能断定没有发生。","source_ids":[]}`}, nil
	}})
	result, err := s.runAgentTurn(context.Background(), AgentToolScope{FamilyID: "fam", UserID: "user"}, []LLMMessage{{Role: "user", Content: "查询"}})
	if err != nil || len(repo.queries) != 1 || len(result.ToolResults) != 2 || !result.ToolResults[1].Reused {
		t.Fatalf("duplicate call not reused: %+v %v", result, err)
	}
}

func TestAgentTurnStopsOnInvalidOutputBudgetAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		reply     *LLMResponse
		wantCalls int
	}{
		{"non-json", &LLMResponse{Content: "随意文字"}, 1},
		{"duplicate-call-id", &LLMResponse{ToolCalls: []LLMToolCall{{ID: "same", Name: "getCatProfile", Arguments: `{"cat_id":"cat"}`}, {ID: "same", Name: "getCatProfile", Arguments: `{"cat_id":"cat"}`}}}, 1},
		{"too-many-tools", &LLMResponse{ToolCalls: make([]LLMToolCall, 9)}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newAgentServiceForTest()
			calls := 0
			s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(context.Context, LLMRequest, []ToolDef) (*LLMResponse, error) { calls++; return tc.reply, nil }})
			if _, err := s.runAgentTurn(context.Background(), AgentToolScope{FamilyID: "fam", UserID: "user"}, []LLMMessage{{Role: "user", Content: "查询"}}); err == nil || calls != tc.wantCalls {
				t.Fatalf("missing stop: calls=%d err=%v", calls, err)
			}
		})
	}
	s, _ := newAgentServiceForTest()
	s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(ctx context.Context, _ LLMRequest, _ []ToolDef) (*LLMResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := s.runAgentTurn(ctx, AgentToolScope{FamilyID: "fam", UserID: "user"}, []LLMMessage{{Role: "user", Content: "查询"}}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline not propagated: %v", err)
	}
}

func TestAgentTurnInvalidToolArgumentsAreBounded(t *testing.T) {
	s, _ := newAgentServiceForTest()
	calls := 0
	s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(_ context.Context, req LLMRequest, _ []ToolDef) (*LLMResponse, error) {
		calls++
		if calls == 1 {
			return &LLMResponse{ToolCalls: []LLMToolCall{{ID: "bad", Name: "listRecords", Arguments: `{"cat_id":"cat","family_id":"other"}`}}}, nil
		}
		last := req.Messages[len(req.Messages)-1]
		var result map[string]string
		if last.Role != "tool" || json.Unmarshal([]byte(last.Content), &result) != nil || result["error"] == "" {
			t.Fatalf("unsafe tool failure: %+v", last)
		}
		return &LLMResponse{Content: `{"answer":"请指定猫咪和记录范围。","source_ids":[]}`}, nil
	}})
	if _, err := s.runAgentTurn(context.Background(), AgentToolScope{FamilyID: "fam", UserID: "user"}, []LLMMessage{{Role: "user", Content: "查询"}}); err != nil || calls != 2 {
		t.Fatalf("tool error turn: %v", err)
	}
}
