package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/platform/config"
)

func testConfig(base string) config.AI {
	return config.AI{Enabled: true, BaseURL: base + "/v1", APIKey: "test-secret", Model: "test-model", Timeout: time.Second, MaxRetries: 1}
}

func TestProviderRetriesShareTurnRequestBudget(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) != 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = io.WriteString(w, `{"model":"test-model","choices":[{"finish_reason":"stop","message":{"content":"ok"}}]}`)
	}))
	defer server.Close()
	cfg := testConfig(server.URL)
	cfg.MaxRetries = 2
	p := NewChatCompletionsProvider(cfg)
	ctx := app.WithLLMRequestBudget(context.Background(), 4)
	request := app.LLMRequest{Messages: []app.LLMMessage{{Role: "user", Content: "query"}}}
	if _, err := p.ChatWithTools(ctx, request, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ChatWithTools(ctx, request, nil); !errors.Is(err, app.ErrLLMRequestBudgetExhausted) {
		t.Fatalf("missing shared budget: %v", err)
	}
	if calls.Load() != 4 {
		t.Fatalf("HTTP requests=%d want=4 including retries", calls.Load())
	}
	if _, err := p.ChatWithTools(app.WithLLMRequestBudget(context.Background(), 1), request, nil); !errors.Is(err, app.ErrLLMRequestBudgetExhausted) {
		t.Fatalf("enhancement budget not enforced: %v", err)
	}
	if calls.Load() != 5 {
		t.Fatalf("enhancement retried beyond one HTTP request: %d", calls.Load())
	}
}

func TestChatCompletionsProviderToolRoundTrip(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("bad request path or authorization")
			w.WriteHeader(400)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		var req wireRequest
		if json.Unmarshal(body, &req) != nil || req.Model != "test-model" || req.Stream {
			t.Errorf("bad wire request: %s", body)
			w.WriteHeader(400)
			return
		}
		if calls.Add(1) == 1 {
			if len(req.Tools) != 1 || req.Tools[0].Function.Name != "listRecords" {
				t.Error("tool schema missing")
			}
			_, _ = io.WriteString(w, `{"model":"test-model","choices":[{"finish_reason":"tool_calls","message":{"content":null,"tool_calls":[{"id":"call-1","type":"function","function":{"name":"listRecords","arguments":"{\"days\":2}"}}]}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
			return
		}
		if len(req.Messages) != 3 || req.Messages[1].ToolCalls[0].ID != "call-1" || req.Messages[2].ToolCallID != "call-1" {
			t.Errorf("tool result not associated: %+v", req.Messages)
		}
		_, _ = io.WriteString(w, `{"model":"test-model","choices":[{"finish_reason":"stop","message":{"content":"查到两条原始记录","tool_calls":[]}}],"usage":{"prompt_tokens":20,"completion_tokens":9,"total_tokens":29}}`)
	}))
	defer server.Close()
	p := NewChatCompletionsProvider(testConfig(server.URL))
	tool := app.ToolDef{Name: "listRecords", Description: "读取记录", Parameters: json.RawMessage(`{"type":"object","properties":{"days":{"type":"integer"}}}`)}
	first, err := p.ChatWithTools(context.Background(), app.LLMRequest{Messages: []app.LLMMessage{{Role: "user", Content: "查两天记录"}}}, []app.ToolDef{tool})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.ToolCalls) != 1 || first.ToolCalls[0].ID != "call-1" || first.FinishReason != "tool_calls" || first.Usage.TotalTokens != 15 {
		t.Fatalf("tool call lost: %+v", first)
	}
	second, err := p.ChatWithTools(context.Background(), app.LLMRequest{Messages: []app.LLMMessage{
		{Role: "user", Content: "查两天记录"},
		{Role: "assistant", ToolCalls: first.ToolCalls},
		{Role: "tool", ToolCallID: "call-1", Content: `{"count":2}`},
	}}, []app.ToolDef{tool})
	if err != nil || second.Content != "查到两条原始记录" || second.FinishReason != "stop" {
		t.Fatalf("final answer: %+v %v", second, err)
	}
}

func TestChatCompletionsProviderKeepsMultipleToolCallIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"model":"test-model","choices":[{"finish_reason":"tool_calls","message":{"content":null,"tool_calls":[{"id":"call-a","type":"function","function":{"name":"listRecords","arguments":"{}"}},{"id":"call-b","type":"function","function":{"name":"getTrends","arguments":"{}"}}]}}]}`)
	}))
	defer server.Close()
	p := NewChatCompletionsProvider(testConfig(server.URL))
	got, err := p.ChatWithTools(context.Background(), app.LLMRequest{Messages: []app.LLMMessage{{Role: "user", Content: "查记录和趋势"}}}, nil)
	if err != nil || len(got.ToolCalls) != 2 || got.ToolCalls[0].ID != "call-a" || got.ToolCalls[1].ID != "call-b" {
		t.Fatalf("multiple calls: %+v %v", got, err)
	}
}

func TestChatCompletionsProviderFailureBoundaries(t *testing.T) {
	t.Run("missing configuration", func(t *testing.T) {
		p := NewChatCompletionsProvider(config.AI{Enabled: true})
		if _, err := p.ChatWithTools(context.Background(), app.LLMRequest{Messages: []app.LLMMessage{{Role: "user", Content: "hi"}}}, nil); !errors.Is(err, app.ErrLLMUnavailable) {
			t.Fatalf("missing config: %v", err)
		}
	})
	t.Run("insecure remote URL", func(t *testing.T) {
		p := NewChatCompletionsProvider(testConfig("http://example.com"))
		if _, err := p.ChatWithTools(context.Background(), app.LLMRequest{Messages: []app.LLMMessage{{Role: "user", Content: "hi"}}}, nil); !errors.Is(err, app.ErrLLMUnavailable) {
			t.Fatalf("remote HTTP accepted: %v", err)
		}
	})
	t.Run("429 retry and malformed JSON", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) == 1 {
				w.WriteHeader(429)
				return
			}
			_, _ = io.WriteString(w, "{invalid")
		}))
		defer server.Close()
		p := NewChatCompletionsProvider(testConfig(server.URL))
		if _, err := p.ChatWithTools(context.Background(), app.LLMRequest{Messages: []app.LLMMessage{{Role: "user", Content: "hi"}}}, nil); !errors.Is(err, app.ErrLLMInvalidResponse) || calls.Load() != 2 {
			t.Fatalf("retry/invalid JSON: %v, calls=%d", err, calls.Load())
		}
	})
	t.Run("400 is not retried", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(400) }))
		defer server.Close()
		p := NewChatCompletionsProvider(testConfig(server.URL))
		if _, err := p.ChatWithTools(context.Background(), app.LLMRequest{Messages: []app.LLMMessage{{Role: "user", Content: "hi"}}}, nil); !errors.Is(err, app.ErrLLMUnavailable) || calls.Load() != 1 || strings.Contains(err.Error(), "test-secret") {
			t.Fatalf("400 boundary: %v, calls=%d", err, calls.Load())
		}
	})
	t.Run("5xx is retried", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(503) }))
		defer server.Close()
		p := NewChatCompletionsProvider(testConfig(server.URL))
		if _, err := p.ChatWithTools(context.Background(), app.LLMRequest{Messages: []app.LLMMessage{{Role: "user", Content: "hi"}}}, nil); !errors.Is(err, app.ErrLLMUnavailable) || calls.Load() != 2 {
			t.Fatalf("5xx retry: %v, calls=%d", err, calls.Load())
		}
	})
	t.Run("response size", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, strings.Repeat("x", int(maxResponseBytes)+1))
		}))
		defer server.Close()
		p := NewChatCompletionsProvider(testConfig(server.URL))
		if _, err := p.ChatWithTools(context.Background(), app.LLMRequest{Messages: []app.LLMMessage{{Role: "user", Content: "hi"}}}, nil); !errors.Is(err, app.ErrLLMInvalidResponse) {
			t.Fatalf("oversized response: %v", err)
		}
	})
	t.Run("timeout and cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
			_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"late"}}]}`)
		}))
		defer server.Close()
		cfg := testConfig(server.URL)
		cfg.Timeout, cfg.MaxRetries = 10*time.Millisecond, 0
		p := NewChatCompletionsProvider(cfg)
		req := app.LLMRequest{Messages: []app.LLMMessage{{Role: "user", Content: "hi"}}}
		if _, err := p.ChatWithTools(context.Background(), req, nil); !errors.Is(err, app.ErrLLMUnavailable) {
			t.Fatalf("timeout: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := p.ChatWithTools(ctx, req, nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation: %v", err)
		}
	})
}
