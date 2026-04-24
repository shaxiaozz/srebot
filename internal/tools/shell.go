package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/shaxiaozz/srebot/internal/approval"
)

type shellTool struct {
	gate approval.Gate
}

func (s *shellTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "shell",
		Desc: "Execute a shell command via /bin/sh -c. Subject to approval gate and denylist.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command":    {Type: schema.String, Desc: "shell command to run", Required: true},
			"timeoutSec": {Type: schema.Integer, Desc: "timeout in seconds, max 300", Required: false},
		}),
	}, nil
}

func (s *shellTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var a struct {
		Command    string `json:"command"`
		TimeoutSec int    `json:"timeoutSec"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &a); err != nil {
		return "", fmt.Errorf("shell: invalid arguments: %w", err)
	}

	// Build args map for gate authorization
	args := map[string]any{
		"command":    a.Command,
		"timeoutSec": a.TimeoutSec,
	}

	// Check authorization
	if err := s.gate.Authorize(ctx, "shell", args); err != nil {
		// Return error message as output so LLM can see the rejection
		return err.Error(), nil
	}

	// Apply timeout
	timeoutSec := a.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	if timeoutSec > 300 {
		timeoutSec = 300
	}

	ctxT, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Execute command
	cmd := exec.CommandContext(ctxT, "/bin/sh", "-c", a.Command)

	// Run and capture output
	output, err := cmd.CombinedOutput()

	// Determine exit code
	exitCode := 0
	// Check for timeout first (context.DeadlineExceeded or err wrapper)
	if errors.Is(err, context.DeadlineExceeded) {
		exitCode = 124 // Standard timeout exit code
	} else if ctxT.Err() == context.DeadlineExceeded {
		exitCode = 124 // Command was killed by timeout
	} else if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	// Truncate output to 64 KiB
	const maxOutputSize = 64 * 1024
	body := string(output)
	if len(body) > maxOutputSize {
		body = body[:maxOutputSize] + "\n[truncated]\n"
	}

	// Format result
	result := fmt.Sprintf("exit=%d\noutput:\n%s", exitCode, body)
	return result, nil
}

// NewShell returns a shell tool guarded by the provided approval gate.
func NewShell(gate approval.Gate) tool.InvokableTool {
	return &shellTool{gate: gate}
}
