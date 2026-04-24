package approval

import (
	"regexp"
	"sync"
)

// denyRule pairs a human-readable name with a compiled regex pattern.
type denyRule struct {
	name string
	re   *regexp.Regexp
}

// denyRules holds compiled patterns, initialized once at package init.
var (
	denyRules []denyRule
	initOnce  sync.Once
)

// initDenylist compiles all denylist patterns.
func initDenylist() {
	initOnce.Do(func() {
		denyRules = []denyRule{
			// rm -rf / and variants
			{
				name: "rm -rf /",
				re:   regexp.MustCompile(`(?i)\brm\s+-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+/(?:\s|$|\*)`),
			},
			{
				name: "rm -rf /",
				re:   regexp.MustCompile(`(?i)\brm\s+-[a-zA-Z]*f[a-zA-Z]*r[a-zA-Z]*\s+/(?:\s|$|\*)`),
			},
			// mkfs
			{
				name: "mkfs",
				re:   regexp.MustCompile(`(?i)\bmkfs(?:\.|\s)`),
			},
			// dd if=... of=/dev/...
			{
				name: "dd of=/dev/",
				re:   regexp.MustCompile(`(?i)\bdd\s+.*\bof=/dev/`),
			},
			// > /dev/sd*
			{
				name: "> /dev/sd*",
				re:   regexp.MustCompile(`(?i)>\s*/dev/sd[a-z]`),
			},
			// fork bomb
			{
				name: "fork bomb",
				re:   regexp.MustCompile(`:\(\)\s*\{\s*:\|\s*:\s*&\s*\}\s*;\s*:`),
			},
			// shutdown/reboot/halt/poweroff
			{
				name: "shutdown/reboot/halt/poweroff",
				re:   regexp.MustCompile(`(?i)\b(sudo\s+)?(shutdown|reboot|halt|poweroff)\b`),
			},
			// chmod -R 777 /
			{
				name: "chmod -R 777 /",
				re:   regexp.MustCompile(`(?i)\bchmod\s+-R\s+777\s+/(?:\s|$)`),
			},
		}
	})
}

// HitDenylist checks if cmd matches any denylist pattern.
// Returns the matched rule name and a boolean indicating if a match was found.
// Empty cmd returns ("", false).
func HitDenylist(cmd string) (matched string, hit bool) {
	initDenylist()

	if cmd == "" {
		return "", false
	}

	for _, rule := range denyRules {
		if rule.re.MatchString(cmd) {
			return rule.name, true
		}
	}
	return "", false
}
