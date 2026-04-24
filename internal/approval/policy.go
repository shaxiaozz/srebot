package approval

// Decision represents the outcome of a policy check.
type Decision int

const (
	Allow Decision = iota
	Deny
	AskUser
)

// Policy determines whether a tool invocation should be allowed, denied, or require user confirmation.
type Policy interface {
	Check(toolName string, args map[string]any) Decision
}

// policy implements Policy with whitelist, read-only auto-approval, and yolo mode.
type policy struct {
	whitelist           map[string]struct{}
	readOnlyAutoApprove bool
	yolo                bool
	readOnlyToolset     map[string]struct{}
}

// NewPolicy creates a new Policy with the given configuration.
// whitelist: set of tool names that are always allowed.
// readOnlyAutoApprove: if true, read-only tools are automatically approved.
// yolo: if true, all non-denylist tools are allowed (yolo mode).
func NewPolicy(whitelist []string, readOnlyAutoApprove, yolo bool) Policy {
	// Build whitelist map for O(1) lookup
	wl := make(map[string]struct{})
	for _, name := range whitelist {
		wl[name] = struct{}{}
	}

	// Hard-coded read-only toolset for Phase 2
	readOnlySet := map[string]struct{}{
		"read_file": {},
	}

	return &policy{
		whitelist:           wl,
		readOnlyAutoApprove: readOnlyAutoApprove,
		yolo:                yolo,
		readOnlyToolset:     readOnlySet,
	}
}

// Check evaluates a tool invocation against the policy.
// Decision order (highest to lowest priority):
// 1. If tool is "shell" and command hits denylist → Deny
// 2. If yolo mode is enabled → Allow
// 3. If readOnlyAutoApprove and tool is read-only → Allow
// 4. If tool is in whitelist → Allow
// 5. Otherwise → AskUser
func (p *policy) Check(toolName string, args map[string]any) Decision {
	// Denylist check for shell commands (highest priority)
	if toolName == "shell" {
		if cmd, ok := args["command"].(string); ok {
			if _, hit := HitDenylist(cmd); hit {
				return Deny
			}
		}
	}

	// Yolo mode (but NOT for denylist-blocked commands)
	if p.yolo {
		return Allow
	}

	// Read-only auto-approve
	if p.readOnlyAutoApprove {
		if _, ok := p.readOnlyToolset[toolName]; ok {
			return Allow
		}
	}

	// Whitelist check
	if _, ok := p.whitelist[toolName]; ok {
		return Allow
	}

	// Default to asking user
	return AskUser
}
