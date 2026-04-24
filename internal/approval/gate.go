package approval

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

// ConfirmResult represents the user's response to an approval prompt.
type ConfirmResult int

const (
	ConfirmYes ConfirmResult = iota
	ConfirmNo
	ConfirmSessionAll
)

// Prompter asks the user for confirmation before executing a tool.
type Prompter interface {
	Confirm(msg string) (ConfirmResult, error)
}

// Gate controls authorization for tool invocations.
// It applies a policy and prompts the user when necessary.
type Gate interface {
	Authorize(ctx context.Context, toolName string, args map[string]any) error
}

// cliGate implements Gate with CLI prompting.
type cliGate struct {
	policy      Policy
	prompter    Prompter
	sessionAuto atomic.Bool
}

// NewCLIGate creates a new Gate with the given policy and prompter.
func NewCLIGate(p Policy, prompter Prompter) Gate {
	return &cliGate{
		policy:   p,
		prompter: prompter,
	}
}

// Authorize checks if a tool invocation is allowed.
// Returns nil if allowed, otherwise returns an error.
// Decision flow:
// 1. Check context cancellation
// 2. Apply policy
// 3. If Allow → return nil
// 4. If Deny → return error (distinguish denylist vs policy)
// 5. If AskUser → prompt user (unless session auto-approve is active)
func (g *cliGate) Authorize(ctx context.Context, toolName string, args map[string]any) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Apply policy
	decision := g.policy.Check(toolName, args)

	switch decision {
	case Allow:
		return nil

	case Deny:
		// Distinguish between denylist and policy denial
		if toolName == "shell" {
			if cmd, ok := args["command"].(string); ok {
				if matched, hit := HitDenylist(cmd); hit {
					return fmt.Errorf("blocked by denylist: %s", matched)
				}
			}
		}
		return errors.New("not allowed by policy")

	case AskUser:
		// Check if session auto-approve is active
		if g.sessionAuto.Load() {
			return nil
		}

		// Build and prompt
		msg := buildMessage(toolName, args)
		res, err := g.prompter.Confirm(msg)
		if err != nil {
			return err
		}

		switch res {
		case ConfirmYes:
			return nil
		case ConfirmSessionAll:
			g.sessionAuto.Store(true)
			return nil
		case ConfirmNo:
			return errors.New("user denied")
		default:
			return fmt.Errorf("unexpected confirm result: %v", res)
		}
	}

	return fmt.Errorf("unexpected policy decision: %v", decision)
}

// buildMessage constructs a user-friendly message for confirmation.
func buildMessage(toolName string, args map[string]any) string {
	return fmt.Sprintf("Allow tool %q with args %v?", toolName, args)
}
