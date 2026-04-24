package approval

import (
	"testing"
)

func TestHitDenylist(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		shouldHit   bool
		matchedRule string
	}{
		// rm -rf / variants (positive)
		{
			name:        "rm -rf /",
			cmd:         "rm -rf /",
			shouldHit:   true,
			matchedRule: "rm -rf /",
		},
		{
			name:        "rm -rf /*",
			cmd:         "rm -rf /*",
			shouldHit:   true,
			matchedRule: "rm -rf /",
		},
		{
			name:        "sudo rm -rf /",
			cmd:         "sudo rm -rf /",
			shouldHit:   true,
			matchedRule: "rm -rf /",
		},
		{
			name:        "rm -fr /",
			cmd:         "rm -fr /",
			shouldHit:   true,
			matchedRule: "rm -rf /",
		},
		{
			name:        "rm -rfi /",
			cmd:         "rm -rfi /",
			shouldHit:   true,
			matchedRule: "rm -rf /",
		},
		// mkfs (positive)
		{
			name:        "mkfs.ext4",
			cmd:         "mkfs.ext4 /dev/sdb",
			shouldHit:   true,
			matchedRule: "mkfs",
		},
		{
			name:        "mkfs space",
			cmd:         "mkfs /dev/sda1",
			shouldHit:   true,
			matchedRule: "mkfs",
		},
		// dd of=/dev/ (positive)
		{
			name:        "dd of=/dev/sda",
			cmd:         "dd if=/dev/zero of=/dev/sda",
			shouldHit:   true,
			matchedRule: "dd of=/dev/",
		},
		// > /dev/sd* (positive)
		{
			name:        "redirect to /dev/sda",
			cmd:         "echo x > /dev/sda",
			shouldHit:   true,
			matchedRule: "> /dev/sd*",
		},
		// fork bomb (positive)
		{
			name:        "fork bomb",
			cmd:         ":() { :| :& }; :",
			shouldHit:   true,
			matchedRule: "fork bomb",
		},
		// shutdown/reboot/halt/poweroff (positive)
		{
			name:        "shutdown",
			cmd:         "shutdown -h now",
			shouldHit:   true,
			matchedRule: "shutdown/reboot/halt/poweroff",
		},
		{
			name:        "reboot",
			cmd:         "reboot",
			shouldHit:   true,
			matchedRule: "shutdown/reboot/halt/poweroff",
		},
		{
			name:        "halt",
			cmd:         "halt",
			shouldHit:   true,
			matchedRule: "shutdown/reboot/halt/poweroff",
		},
		{
			name:        "poweroff",
			cmd:         "poweroff",
			shouldHit:   true,
			matchedRule: "shutdown/reboot/halt/poweroff",
		},
		{
			name:        "sudo shutdown",
			cmd:         "sudo shutdown",
			shouldHit:   true,
			matchedRule: "shutdown/reboot/halt/poweroff",
		},
		// chmod -R 777 / (positive)
		{
			name:        "chmod -R 777 /",
			cmd:         "chmod -R 777 /",
			shouldHit:   true,
			matchedRule: "chmod -R 777 /",
		},
		// Negatives (should NOT hit)
		{
			name:      "rm regular file",
			cmd:       "rm file.txt",
			shouldHit: false,
		},
		{
			name:      "rm -rf /tmp/foo",
			cmd:       "rm -rf /tmp/foo",
			shouldHit: false,
		},
		{
			name:      "echo /dev/sda (just printing path)",
			cmd:       "echo /dev/sda",
			shouldHit: false,
		},
		{
			name:      "rebooted (word boundary protects against substring)",
			cmd:       "rebooted yesterday",
			shouldHit: false,
		},
		{
			name:      "chmod -R 777 /tmp",
			cmd:       "chmod -R 777 /tmp",
			shouldHit: false,
		},
		{
			name:      "empty string",
			cmd:       "",
			shouldHit: false,
		},
		{
			name:      "safe ls",
			cmd:       "ls -la",
			shouldHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, hit := HitDenylist(tt.cmd)
			if hit != tt.shouldHit {
				t.Errorf("HitDenylist(%q): hit=%v, want %v", tt.cmd, hit, tt.shouldHit)
			}
			if tt.shouldHit && matched != tt.matchedRule {
				t.Errorf("HitDenylist(%q): matched=%q, want %q", tt.cmd, matched, tt.matchedRule)
			}
			if !tt.shouldHit && matched != "" {
				t.Errorf("HitDenylist(%q): matched=%q, want empty", tt.cmd, matched)
			}
		})
	}
}
