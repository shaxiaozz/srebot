package approval

import (
	"testing"
)

func TestPolicy_Denylist_HighestPriority(t *testing.T) {
	// Denylist should block even with yolo or whitelist
	p := NewPolicy([]string{"shell"}, false, true) // yolo=true, shell whitelisted
	args := map[string]any{"command": "rm -rf /"}
	result := p.Check("shell", args)
	if result != Deny {
		t.Errorf("denylist should block shell even with yolo; got %v, want Deny", result)
	}
}

func TestPolicy_Yolo_AllowsNonDenylist(t *testing.T) {
	p := NewPolicy([]string{}, false, true) // yolo=true, empty whitelist
	args := map[string]any{"command": "ls -la"}
	result := p.Check("shell", args)
	if result != Allow {
		t.Errorf("yolo mode should allow non-denylist shell; got %v, want Allow", result)
	}
}

func TestPolicy_Yolo_BlocksDenylist(t *testing.T) {
	p := NewPolicy([]string{}, false, true) // yolo=true
	args := map[string]any{"command": "shutdown -h now"}
	result := p.Check("shell", args)
	if result != Deny {
		t.Errorf("yolo mode should still block denylist; got %v, want Deny", result)
	}
}

func TestPolicy_ReadOnlyAutoApprove_AllowsReadFile(t *testing.T) {
	p := NewPolicy([]string{}, true, false) // readOnlyAutoApprove=true
	args := map[string]any{"path": "somefile.txt"}
	result := p.Check("read_file", args)
	if result != Allow {
		t.Errorf("read-only auto-approve should allow read_file; got %v, want Allow", result)
	}
}

func TestPolicy_ReadOnlyAutoApprove_DoesNotAllowShell(t *testing.T) {
	p := NewPolicy([]string{}, true, false) // readOnlyAutoApprove=true
	args := map[string]any{"command": "ls"}
	result := p.Check("shell", args)
	if result != AskUser {
		t.Errorf("read-only auto-approve should not allow shell; got %v, want AskUser", result)
	}
}

func TestPolicy_Whitelist_Allows(t *testing.T) {
	p := NewPolicy([]string{"custom_tool"}, false, false)
	args := map[string]any{}
	result := p.Check("custom_tool", args)
	if result != Allow {
		t.Errorf("whitelist should allow custom_tool; got %v, want Allow", result)
	}
}

func TestPolicy_AskUser_Default(t *testing.T) {
	p := NewPolicy([]string{}, false, false) // no whitelist, no yolo
	args := map[string]any{"command": "echo hello"}
	result := p.Check("shell", args)
	if result != AskUser {
		t.Errorf("default should ask user; got %v, want AskUser", result)
	}
}

func TestPolicy_Decision_Order(t *testing.T) {
	// yolo=true, shell in whitelist, but we test that denylist wins
	p := NewPolicy([]string{"shell"}, true, true)

	tests := []struct {
		name       string
		toolName   string
		args       map[string]any
		wantResult Decision
	}{
		{
			name:       "denylist blocks despite yolo and whitelist",
			toolName:   "shell",
			args:       map[string]any{"command": "rm -rf /"},
			wantResult: Deny,
		},
		{
			name:       "yolo allows non-denylist",
			toolName:   "shell",
			args:       map[string]any{"command": "echo hi"},
			wantResult: Allow,
		},
		{
			name:       "whitelist allows",
			toolName:   "shell",
			args:       map[string]any{},
			wantResult: Allow, // yolo overrides before whitelist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Check(tt.toolName, tt.args)
			if result != tt.wantResult {
				t.Errorf("got %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestPolicy_Edge_Cases(t *testing.T) {
	p := NewPolicy([]string{}, false, false)

	tests := []struct {
		name       string
		toolName   string
		args       map[string]any
		wantResult Decision
	}{
		{
			name:       "shell with non-string command",
			toolName:   "shell",
			args:       map[string]any{"command": 123},
			wantResult: AskUser,
		},
		{
			name:       "shell with missing command",
			toolName:   "shell",
			args:       map[string]any{},
			wantResult: AskUser,
		},
		{
			name:       "unknown tool",
			toolName:   "unknown_tool",
			args:       map[string]any{},
			wantResult: AskUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Check(tt.toolName, tt.args)
			if result != tt.wantResult {
				t.Errorf("got %v, want %v", result, tt.wantResult)
			}
		})
	}
}
