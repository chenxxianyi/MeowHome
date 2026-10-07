package app

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
)

func TestPatrolEnhancementPreservesFactsAndCanBeDisabled(t *testing.T) {
	s, repo := newAgentServiceForTest()
	s.SetEnhancementEnabled(true)
	m := &model.AgentMessage{Base: model.Base{ID: "message"}, FamilyID: "fam", Role: "assistant", Visibility: "family", RuleID: RuleR02, Severity: "warning", Body: "小白最近7天有3次呕吐记录。", RuleBody: "小白最近7天有3次呕吐记录。", Evidence: `[{"source_type":"record","source_id":"record"}]`, ActionSuggestions: `[{"type":"view_records","title":"查看记录"}]`}
	repo.messages[m.ID] = m
	calls := 0
	s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(ctx context.Context, req LLMRequest, tools []ToolDef) (*LLMResponse, error) {
		calls++
		if len(tools) != 0 || len(req.Messages) != 2 || strings.Contains(req.Messages[1].Content, "source_id") {
			t.Fatal("uncontrolled enhancement input")
		}
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 5*time.Second {
			t.Fatal("missing enhancement budget")
		}
		return &LLMResponse{Model: "fake", Content: `{"body":"猫管家提醒：小白最近7天有3次呕吐记录。请核对原始记录。"}`, Usage: LLMUsage{TotalTokens: 15}}, nil
	}})
	s.enhanceMessage(context.Background(), "fam", m.ID)
	s.enhanceMessage(context.Background(), "fam", m.ID)
	stored := repo.messages[m.ID]
	if calls != 1 || stored.EnhanceStatus != "completed" || stored.Body != m.Body || stored.RuleBody != m.Body || stored.Evidence != m.Evidence || stored.ActionSuggestions != m.ActionSuggestions || stored.Severity != "warning" || stored.TotalTokens != 15 {
		t.Fatalf("enhancement changed facts: %+v", stored)
	}
	shown, err := s.GetMessage(context.Background(), "fam", "user", m.ID)
	if err != nil || shown.Body != stored.EnhancedBody {
		t.Fatalf("enhanced body not displayed: %+v %v", shown, err)
	}
	s.SetEnhancementEnabled(false)
	shown, err = s.GetMessage(context.Background(), "fam", "user", m.ID)
	if err != nil || shown.Body != m.Body || shown.Model == "fake" {
		t.Fatal("disabled enhancement did not restore template")
	}
}

func TestEnhancementRejectsChangedFactsAndSkipsDanger(t *testing.T) {
	for _, tc := range []struct {
		name, reply                        string
		fail, danger, disabled, concurrent bool
	}{
		{name: "new-diagnosis", reply: `{"body":"小白患有胃炎。"}`},
		{name: "changed-number", reply: `{"body":"小白最近7天有8次呕吐记录。"}`},
		{name: "delay", reply: `{"body":"小白最近7天有3次呕吐记录。可以一周后再处理。"}`},
		{name: "dose", reply: `{"body":"小白最近7天有3次呕吐记录。服用10毫克。"}`},
		{name: "too-long", reply: encodeJSON(map[string]string{"body": strings.Repeat("文", 601)})},
		{name: "extra-field", reply: `{"body":"小白最近7天有3次呕吐记录。","severity":"info"}`},
		{name: "trailing-json", reply: `{"body":"小白最近7天有3次呕吐记录。"}{}`},
		{name: "failure", fail: true},
		{name: "danger", danger: true},
		{name: "disabled", disabled: true},
		{name: "concurrent-dismiss", reply: `{"body":"小白最近7天有3次呕吐记录。"}`, concurrent: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, repo := newAgentServiceForTest()
			s.SetEnhancementEnabled(!tc.disabled)
			severity := "warning"
			if tc.danger {
				severity = "danger"
			}
			m := &model.AgentMessage{Base: model.Base{ID: "message"}, FamilyID: "fam", Role: "assistant", Visibility: "family", RuleID: RuleR02, Severity: severity, Body: "小白最近7天有3次呕吐记录。"}
			repo.messages[m.ID] = m
			calls := 0
			s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(context.Context, LLMRequest, []ToolDef) (*LLMResponse, error) {
				calls++
				if tc.concurrent {
					repo.messages[m.ID].DisplayStatus = DraftDismissed
				}
				if tc.fail {
					return nil, ErrLLMUnavailable
				}
				return &LLMResponse{Content: tc.reply}, nil
			}})
			s.enhanceMessage(context.Background(), "fam", m.ID)
			stored := repo.messages[m.ID]
			if stored.EnhancedBody != "" || stored.Body != m.Body || (tc.concurrent && stored.DisplayStatus != DraftDismissed) {
				t.Fatalf("unsafe/stale output persisted: %+v", stored)
			}
			if (tc.danger || tc.disabled) && calls != 0 {
				t.Fatal("danger/disabled called model")
			}
		})
	}
}

func TestAgentFamilyRolloutLeavesHistoryReadable(t *testing.T) {
	s, repo := newAgentServiceForTest()
	s.SetAllowedFamilies([]string{"allowed"})
	repo.messages["old"] = &model.AgentMessage{Base: model.Base{ID: "old"}, FamilyID: "fam", Visibility: "family", Role: "assistant"}
	if _, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{Message: "查询", ClientMessageID: "key"}); err == nil {
		t.Fatal("non-rollout family called agent")
	}
	if _, err := s.PatrolSystem(context.Background(), "fam"); err == nil {
		t.Fatal("non-rollout family patrolled")
	}
	if _, err := s.GetMessage(context.Background(), "fam", "user", "old"); err != nil {
		t.Fatal("history was hidden")
	}
	if err := s.DismissMessage(context.Background(), "fam", "user", "old"); err != nil {
		t.Fatal("historical dismiss blocked")
	}
}

func TestAgentSwitchMatrix(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, aiEnabled := range []bool{false, true} {
			for _, enhance := range []bool{false, true} {
				t.Run(fmt.Sprintf("agent=%t/ai=%t/enhance=%t", enabled, aiEnabled, enhance), func(t *testing.T) {
					s, repo := newAgentServiceForTest()
					s.enabled = enabled
					s.SetEnhancementEnabled(aiEnabled && enhance)
					repo.messages["rule"] = &model.AgentMessage{Base: model.Base{ID: "rule"}, FamilyID: "fam", Role: "assistant", Visibility: "family", Type: AgentTypePatrolAbnormal, Severity: "warning", RuleID: RuleR06, Body: "昨日没有记录。", RuleBody: "昨日没有记录。"}
					s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(_ context.Context, _ LLMRequest, tools []ToolDef) (*LLMResponse, error) {
						if !aiEnabled {
							return nil, ErrLLMUnavailable
						}
						if len(tools) == 0 {
							return &LLMResponse{Content: `{"body":"猫管家提醒：昨日没有记录。"}`}, nil
						}
						return &LLMResponse{Content: `{"answer":"请明确猫咪和时间。","source_ids":[]}`}, nil
					}})
					response, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{Message: "查询", ClientMessageID: "key"})
					if enabled {
						if err != nil || response.Degraded == aiEnabled {
							t.Fatalf("incorrect chat switch: %+v %v", response, err)
						}
					} else if err == nil {
						t.Fatal("disabled chat accepted")
					}
					s.enhanceMessage(context.Background(), "fam", "rule")
					if (repo.messages["rule"].EnhanceStatus == "completed") != (enabled && aiEnabled && enhance) {
						t.Fatal("incorrect enhancement switch")
					}
					if _, err := s.GetMessage(context.Background(), "fam", "user", "rule"); err != nil {
						t.Fatal("switch hid history")
					}
					scheduled := s.clock().Add(time.Hour)
					_, editErr := s.EditDraft(context.Background(), "fam", "user", "rule", AgentReminderEditRequest{Reminder: &AgentReminderInput{CatID: "cat", Type: "custom", Title: "护理", ScheduledAt: &scheduled, Timezone: "Asia/Shanghai"}})
					if (editErr == nil) != enabled {
						t.Fatalf("incorrect draft edit switch: %v", editErr)
					}
					_, confirmErr := s.ConfirmDraft(context.Background(), "fam", "user", "rule", 1)
					if (confirmErr == nil) != enabled {
						t.Fatalf("incorrect confirm switch: %v", confirmErr)
					}
					if err := s.DismissMessage(context.Background(), "fam", "user", "rule"); err != nil {
						t.Fatal("switch prevented dismiss")
					}
				})
			}
		}
	}
}

func TestEnhancementTimeoutPreservesTemplateAndWorkerStops(t *testing.T) {
	s, repo := newAgentServiceForTest()
	s.SetEnhancementEnabled(true)
	repo.messages["rule"] = &model.AgentMessage{Base: model.Base{ID: "rule"}, FamilyID: "fam", Role: "assistant", Visibility: "family", Severity: "warning", RuleID: RuleR06, Body: "昨日没有记录。"}
	s.SetLLMProvider(FakeLLMProvider{ChatFunc: func(ctx context.Context, _ LLMRequest, _ []ToolDef) (*LLMResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	s.enhanceMessage(ctx, "fam", "rule")
	if repo.messages["rule"].EnhanceStatus != "failed" || repo.messages["rule"].Body != "昨日没有记录。" || repo.messages["rule"].EnhancedBody != "" {
		t.Fatal("timeout destroyed rule template")
	}
	workerCtx, stop := context.WithCancel(context.Background())
	s.StartEnhancementWorker(workerCtx)
	stop()
	waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Second)
	defer waitCancel()
	if err := s.WaitEnhancementWorker(waitCtx); err != nil {
		t.Fatal("worker did not stop")
	}
}
