package tools

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

// stubGate is a test stub that returns a configured error.
type stubGate struct {
	err error
}

func (s *stubGate) Authorize(ctx context.Context, tool string, args map[string]any) error {
	return s.err
}

func invokeShell(t *testing.T, gate *stubGate, command string, timeoutSec int) (string, error) {
	t.Helper()
	sh := NewShell(gate)
	args := map[string]any{"command": command}
	if timeoutSec > 0 {
		args["timeoutSec"] = timeoutSec
	}
	argsJSON, _ := json.Marshal(args)
	return sh.InvokableRun(context.Background(), string(argsJSON))
}

func TestShell_AllowedEcho(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	result, err := invokeShell(t, &stubGate{err: nil}, "echo hello", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "exit=0") {
		t.Errorf("result should contain 'exit=0', got: %v", result)
	}
	if !strings.Contains(result, "hello") {
		t.Errorf("result should contain 'hello', got: %v", result)
	}
}

func TestShell_GateDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	result, err := invokeShell(t, &stubGate{err: errors.New("user denied")}, "echo hello", 0)
	if err != nil {
		t.Fatalf("expected no Go error, got: %v", err)
	}
	if !strings.Contains(result, "user denied") {
		t.Errorf("result should contain 'user denied', got: %v", result)
	}
}

func TestShell_GateDenylist(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	result, err := invokeShell(t, &stubGate{err: errors.New("blocked by denylist: rm -rf /")}, "rm -rf /", 0)
	if err != nil {
		t.Fatalf("expected no Go error, got: %v", err)
	}
	if !strings.Contains(result, "blocked") {
		t.Errorf("result should contain 'blocked', got: %v", result)
	}
}

func TestShell_NonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	result, err := invokeShell(t, &stubGate{err: nil}, "exit 7", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "exit=7") {
		t.Errorf("result should contain 'exit=7', got: %v", result)
	}
}

func TestShell_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	start := time.Now()
	result, err := invokeShell(t, &stubGate{err: nil}, "sleep 5", 1)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "exit=124") {
		t.Errorf("result should contain 'exit=124' for timeout, got: %v", result)
	}
	// Verify timeout fired: should be between ~500ms and ~3s, not 5s (which would mean no timeout)
	if elapsed < 500*time.Millisecond || elapsed > 3*time.Second {
		t.Errorf("timeout should fire ~1s, got %v", elapsed)
	}
}

func TestShell_OutputTruncation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	// Generate output larger than 64 KiB
	result, err := invokeShell(t, &stubGate{err: nil}, "yes A | head -c 100000", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "[truncated]") {
		t.Errorf("result should contain '[truncated]' marker, got: %v", result)
	}
	// Check that output is bounded near 64 KiB
	// The output line starts after "output:\n", so extract it
	idx := strings.Index(result, "output:\n")
	if idx < 0 {
		t.Fatal("result should contain 'output:\\n'")
	}
	outputBody := result[idx+8:] // Skip "output:\n"
	if len(outputBody) > 100*1024 {
		t.Errorf("output should be bounded near 64 KiB, got %d bytes", len(outputBody))
	}
}

func TestShell_ToolName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	sh := NewShell(&stubGate{err: nil})
	info, err := sh.Info(context.Background())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Name != "shell" {
		t.Errorf("want name %q, got %q", "shell", info.Name)
	}
}

func TestShell_DefaultTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	// Use fast command with default timeout (should not timeout)
	result, err := invokeShell(t, &stubGate{err: nil}, "echo fast", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "exit=0") {
		t.Errorf("result should contain 'exit=0', got: %v", result)
	}
}
