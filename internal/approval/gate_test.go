package approval

import (
	"context"
	"errors"
	"testing"
)

// stubPrompter is a test double that returns pre-programmed results.
type stubPrompter struct {
	results []ConfirmResult
	errs    []error
	idx     int
}

func (s *stubPrompter) Confirm(msg string) (ConfirmResult, error) {
	if s.idx >= len(s.results) {
		return ConfirmNo, errors.New("no more results")
	}
	r := s.results[s.idx]
	var e error
	if s.idx < len(s.errs) && s.errs[s.idx] != nil {
		e = s.errs[s.idx]
	}
	s.idx++
	return r, e
}

func TestCLIGate_Allow_Path(t *testing.T) {
	p := NewPolicy([]string{"read_file"}, false, false)
	prompter := &stubPrompter{}
	gate := NewCLIGate(p, prompter)

	err := gate.Authorize(context.Background(), "read_file", map[string]any{"path": "file.txt"})
	if err != nil {
		t.Errorf("Allow path should return nil; got %v", err)
	}
	if prompter.idx > 0 {
		t.Errorf("prompter should not be called for Allow; idx=%d", prompter.idx)
	}
}

func TestCLIGate_Deny_Path(t *testing.T) {
	p := NewPolicy([]string{}, false, false) // no whitelist
	prompter := &stubPrompter{}
	gate := NewCLIGate(p, prompter)

	err := gate.Authorize(context.Background(), "unknown_tool", map[string]any{})
	// unknown_tool will trigger AskUser, not Deny
	if err == nil {
		t.Errorf("AskUser with ConfirmNo should return error; got nil")
	}
}

func TestCLIGate_Deny_Denylist(t *testing.T) {
	p := NewPolicy([]string{}, false, false)
	prompter := &stubPrompter{}
	gate := NewCLIGate(p, prompter)

	err := gate.Authorize(context.Background(), "shell", map[string]any{"command": "rm -rf /"})
	if err == nil {
		t.Errorf("denylist block should return error; got nil")
	}
	if !errors.Is(err, errors.New("blocked by denylist: rm -rf /")) {
		// errors.Is won't match formatted errors; check message
		if err.Error() != "blocked by denylist: rm -rf /" {
			t.Errorf("error message should contain 'blocked by denylist'; got %v", err)
		}
	}
}

func TestCLIGate_Deny_Policy(t *testing.T) {
	// Create a policy that explicitly denies a tool
	// Actually, our policy doesn't have a way to explicitly deny (only Allow/Deny/AskUser)
	// So we test that non-whitelisted, non-yolo, non-readonly tools return AskUser
	p := NewPolicy([]string{}, false, false)
	prompter := &stubPrompter{
		results: []ConfirmResult{ConfirmNo},
	}
	gate := NewCLIGate(p, prompter)

	err := gate.Authorize(context.Background(), "some_tool", map[string]any{})
	if err == nil {
		t.Errorf("user denied should return error; got nil")
	}
	if err.Error() != "user denied" {
		t.Errorf("error message should be 'user denied'; got %v", err)
	}
}

func TestCLIGate_AskUser_ConfirmYes(t *testing.T) {
	p := NewPolicy([]string{}, false, false)
	prompter := &stubPrompter{
		results: []ConfirmResult{ConfirmYes},
	}
	gate := NewCLIGate(p, prompter)

	err := gate.Authorize(context.Background(), "some_tool", map[string]any{})
	if err != nil {
		t.Errorf("ConfirmYes should return nil; got %v", err)
	}
}

func TestCLIGate_AskUser_ConfirmNo(t *testing.T) {
	p := NewPolicy([]string{}, false, false)
	prompter := &stubPrompter{
		results: []ConfirmResult{ConfirmNo},
	}
	gate := NewCLIGate(p, prompter)

	err := gate.Authorize(context.Background(), "some_tool", map[string]any{})
	if err == nil {
		t.Errorf("ConfirmNo should return error; got nil")
	}
	if err.Error() != "user denied" {
		t.Errorf("error should be 'user denied'; got %v", err)
	}
}

func TestCLIGate_AskUser_ConfirmSessionAll(t *testing.T) {
	p := NewPolicy([]string{}, false, false)
	prompter := &stubPrompter{
		results: []ConfirmResult{ConfirmSessionAll},
	}
	gate := NewCLIGate(p, prompter)

	// First call with ConfirmSessionAll
	err := gate.Authorize(context.Background(), "tool1", map[string]any{})
	if err != nil {
		t.Errorf("ConfirmSessionAll should return nil; got %v", err)
	}

	// Second call should auto-approve without prompting
	err = gate.Authorize(context.Background(), "tool2", map[string]any{})
	if err != nil {
		t.Errorf("subsequent tool should auto-approve; got %v", err)
	}

	// Verify prompter was only called once
	if prompter.idx != 1 {
		t.Errorf("prompter should be called only once; idx=%d", prompter.idx)
	}
}

func TestCLIGate_ContextCanceled(t *testing.T) {
	p := NewPolicy([]string{}, false, false)
	prompter := &stubPrompter{}
	gate := NewCLIGate(p, prompter)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := gate.Authorize(ctx, "some_tool", map[string]any{})
	if err == nil {
		t.Errorf("canceled context should return error; got nil")
	}
	if err != context.Canceled {
		t.Errorf("error should be context.Canceled; got %v", err)
	}
}

func TestCLIGate_PrompterError(t *testing.T) {
	p := NewPolicy([]string{}, false, false)
	prompterErr := errors.New("prompter failed")
	prompter := &stubPrompter{
		results: []ConfirmResult{ConfirmNo},
		errs:    []error{prompterErr},
	}
	gate := NewCLIGate(p, prompter)

	err := gate.Authorize(context.Background(), "some_tool", map[string]any{})
	if !errors.Is(err, prompterErr) {
		t.Errorf("should propagate prompter error; got %v", err)
	}
}

func TestCLIGate_DenylistOverridesYolo(t *testing.T) {
	p := NewPolicy([]string{}, false, true) // yolo=true
	prompter := &stubPrompter{}
	gate := NewCLIGate(p, prompter)

	err := gate.Authorize(context.Background(), "shell", map[string]any{"command": "shutdown -h now"})
	if err == nil {
		t.Errorf("denylist should override yolo; got nil")
	}
	if err.Error() != "blocked by denylist: shutdown/reboot/halt/poweroff" {
		t.Errorf("error should mention denylist; got %v", err)
	}
}

func TestCLIGate_AllowShellNonDenylist_WithYolo(t *testing.T) {
	p := NewPolicy([]string{}, false, true) // yolo=true
	prompter := &stubPrompter{}
	gate := NewCLIGate(p, prompter)

	err := gate.Authorize(context.Background(), "shell", map[string]any{"command": "echo hello"})
	if err != nil {
		t.Errorf("yolo should allow non-denylist shell; got %v", err)
	}
}

// mockPolicy is a test double that returns a specific decision.
type mockPolicy struct {
	decision Decision
}

func (m *mockPolicy) Check(toolName string, args map[string]any) Decision {
	return m.decision
}

func TestCLIGate_UnexpectedPolicyDecision(t *testing.T) {
	// This tests the defensive code path (should never happen with real Policy implementations)
	mockP := &mockPolicy{decision: Decision(999)} // invalid decision
	prompter := &stubPrompter{}
	gate := &cliGate{
		policy:   mockP,
		prompter: prompter,
	}

	err := gate.Authorize(context.Background(), "some_tool", map[string]any{})
	if err == nil {
		t.Errorf("unexpected decision should return error; got nil")
	}
	if !errors.Is(err, errors.New("unexpected policy decision: 999")) {
		// Direct comparison won't work, check error message
		if err.Error() != "unexpected policy decision: 999" {
			t.Errorf("error message should be about unexpected decision; got %v", err)
		}
	}
}
