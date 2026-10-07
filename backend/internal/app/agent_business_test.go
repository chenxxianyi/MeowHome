package app

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// These fixtures assert orchestration contracts with a scripted provider. They
// are also inputs for optional live evaluation; they do not measure real intent
// recognition or the quality of medical boundary language from a real model.
func TestAgentBusinessFixtures(t *testing.T) {
	var cases []struct {
		Name, Group, Message, Tool string
		Args                       json.RawMessage
		RecordCount                *int `json:"record_count"`
		Value                      *float64
	}
	data, err := os.ReadFile("testdata/agent_business_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 24 {
		t.Fatalf("expected 24 fixtures, got %d", len(cases))
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			s, repo := newAgentServiceForTest()
			s.healthProfiles = &toolHealthRepo{err: repository.ErrNotFound}
			s.records = &toolRecordRepo{rows: []*model.DailyRecord{
				toolRecord("feed", "feeding", `{"amount":30}`, s.clock().Add(-time.Hour)), toolRecord("water", "drinking", `{"amount":50}`, s.clock().Add(-time.Hour)), toolRecord("weight", "weight", `{"weight":4.2}`, s.clock().Add(-time.Hour)), toolRecord("vomit", "vomit", `{"count":1}`, s.clock().Add(-time.Hour)),
			}}
			calls := 0
			s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(_ context.Context, req LLMRequest, _ []ToolDef) (*LLMResponse, error) {
				calls++
				if !strings.Contains(req.Messages[0].Content, "先提问澄清") || !strings.Contains(req.Messages[0].Content, "不能诊断") {
					t.Fatal("missing product boundary")
				}
				if tc.Group == "fault" && (tc.Tool == "" || calls > 1) {
					return nil, ErrLLMUnavailable
				}
				if tc.Tool != "" && calls == 1 {
					return &LLMResponse{ToolCalls: []LLMToolCall{{ID: "fixture-call", Name: tc.Tool, Arguments: string(tc.Args)}}}, nil
				}
				if tc.Tool != "" {
					last := req.Messages[len(req.Messages)-1]
					if last.Role != "tool" || last.ToolCallID != "fixture-call" {
						t.Fatal("missing tool linkage")
					}
					var output map[string]any
					if err := json.Unmarshal([]byte(last.Content), &output); err != nil {
						t.Fatal(err)
					}
					if tc.Group == "security" {
						if output["error"] == nil {
							t.Fatal("unauthorized tool succeeded")
						}
					}
					if tc.RecordCount != nil {
						records, ok := output["records"].([]any)
						if !ok || len(records) != *tc.RecordCount {
							t.Fatalf("wrong record object/count: %+v", output)
						}
						for _, record := range records {
							ids := record.(map[string]any)["cat_ids"].([]any)
							if len(ids) != 1 || ids[0] != "cat" {
								t.Fatal("wrong record cat")
							}
						}
					}
					if tc.Value != nil {
						if output["cat_id"] != "cat" {
							t.Fatal("wrong trend cat")
						}
						points, ok := output["points"].([]any)
						if !ok || len(points) != 1 || points[0].(map[string]any)["value"] != *tc.Value {
							t.Fatalf("wrong numeric aggregation: %+v", output)
						}
					}
				}
				var args struct{ Type, Metric string }
				_ = json.Unmarshal(tc.Args, &args)
				source := "feed"
				kind := args.Type
				if kind == "" {
					kind = args.Metric
				}
				switch kind {
				case "drinking":
					source = "water"
				case "weight":
					source = "weight"
				case "vomit":
					source = "vomit"
				}
				if tc.Tool == "getCatProfile" {
					source = "cat"
				}
				return &LLMResponse{Content: encodeJSON(map[string]any{"answer": "请核对猫咪、时间和原始记录；有歧义时先确认查询对象。", "source_ids": []string{source, "forged-id"}})}, nil
			}})
			response, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{Message: tc.Message, ClientMessageID: tc.Name})
			if err != nil || response.Message == nil {
				t.Fatalf("chat fixture failed: %+v %v", response, err)
			}
			if response.Degraded != (tc.Group == "fault") {
				t.Fatal("wrong degradation state")
			}
			if tc.Group == "query" && len(response.Message.Evidence) != 1 {
				t.Fatalf("known citation lost: %+v", response.Message.Evidence)
			}
			for _, e := range response.Message.Evidence {
				if e.SourceID == "forged-id" {
					t.Fatal("forged citation escaped")
				}
			}
			if tc.Group == "clarify" && calls != 1 {
				t.Fatal("clarification executed tools")
			}
			if tc.Group == "draft" {
				m := repo.messages[draftIDForTurn(response.TurnID)]
				if m == nil || m.ActionStatus != DraftPending || m.Visibility != "private" {
					t.Fatal("draft was not pending/private")
				}
			}
			if len(repo.reminders) != 0 {
				t.Fatal("agent automatically wrote a final reminder")
			}
		})
	}
}
