package config

import "testing"

func TestDefaults_HasSaneValues(t *testing.T) {
	r := Defaults()
	if r.Agents.Defaults.Provider != "openai" {
		t.Errorf("Provider = %q, want openai", r.Agents.Defaults.Provider)
	}
	if r.Agents.Defaults.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %q", r.Agents.Defaults.BaseURL)
	}
	if r.Agents.Defaults.Model != "gpt-4o-mini" {
		t.Errorf("Model = %q", r.Agents.Defaults.Model)
	}
	if r.Agents.Defaults.TimeoutSec != 120 {
		t.Errorf("TimeoutSec = %d", r.Agents.Defaults.TimeoutSec)
	}
	if r.Agents.Defaults.ContextWindowTokens != 65536 {
		t.Errorf("ContextWindowTokens = %d", r.Agents.Defaults.ContextWindowTokens)
	}
	if r.Memory.MaxHistoryEntries != 1000 {
		t.Errorf("MaxHistoryEntries = %d", r.Memory.MaxHistoryEntries)
	}
	if r.Approval.YOLO {
		t.Error("YOLO should default false")
	}
	if !r.Approval.ReadOnlyAutoApprove {
		t.Error("ReadOnlyAutoApprove should default true")
	}
	if r.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q", r.Logging.Level)
	}
}

func TestResolve_FlattensRoot(t *testing.T) {
	r := Defaults()
	r.Agents.Defaults.APIKey = "sk-x"
	res := r.Resolve()
	if res.Agent.APIKey != "sk-x" {
		t.Errorf("Agent.APIKey = %q", res.Agent.APIKey)
	}
	if res.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q", res.Logging.Level)
	}
	if res.Memory.MaxHistoryEntries != 1000 {
		t.Errorf("Memory.MaxHistoryEntries = %d", res.Memory.MaxHistoryEntries)
	}
}

func TestValidate_RejectsEmptyAPIKey(t *testing.T) {
	res := Defaults().Resolve()
	res.Agent.APIKey = ""
	err := res.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "apiKey") {
		t.Errorf("error %q should mention apiKey", err)
	}
}

func TestValidate_RejectsZeroContextWindow(t *testing.T) {
	res := Defaults().Resolve()
	res.Agent.APIKey = "sk-test"
	res.Agent.ContextWindowTokens = 0
	err := res.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "contextWindowTokens") {
		t.Errorf("error %q should mention contextWindowTokens", err)
	}
}

func TestValidate_OKWhenMinimalFieldsSet(t *testing.T) {
	res := Defaults().Resolve()
	res.Agent.APIKey = "sk-test"
	if err := res.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// containsStr reports whether substr is within s.
func containsStr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
