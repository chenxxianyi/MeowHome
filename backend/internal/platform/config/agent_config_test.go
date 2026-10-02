package config

import "testing"

func TestAgentEnvironmentBindingsAndValidation(t *testing.T) {
	t.Setenv("AI_AGENT_ENABLED", "true")
	t.Setenv("AI_AGENT_LLM_ENHANCE", "true")
	t.Setenv("AI_AGENT_PATROL_TIMES", "07:30,19:45")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Agent.Enabled || !cfg.Agent.LLMEnhance || cfg.Agent.PatrolTimes != "07:30,19:45" {
		t.Fatalf("Agent env binding failed: %+v", cfg.Agent)
	}
	t.Setenv("AI_AGENT_PATROL_TIMES", "7:30")
	if _, err := Load(); err == nil {
		t.Fatal("invalid patrol time accepted")
	}
}
