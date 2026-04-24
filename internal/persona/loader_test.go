package persona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_AllFourPresent(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "SOUL.md", "soul-body")
	writeFile(t, ws, "AGENTS.md", "agents-body")
	writeFile(t, ws, "USER.md", "user-body")
	writeFile(t, ws, "TOOLS.md", "tools-body")

	p, err := NewFSLoader(ws).Load()
	if err != nil {
		t.Fatal(err)
	}
	if p.Soul != "soul-body" {
		t.Errorf("Soul = %q", p.Soul)
	}
	if p.Agents != "agents-body" {
		t.Errorf("Agents = %q", p.Agents)
	}
	if p.User != "user-body" {
		t.Errorf("User = %q", p.User)
	}
	if p.Tools != "tools-body" {
		t.Errorf("Tools = %q", p.Tools)
	}
}

func TestLoad_MissingFilesReturnEmpty(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "SOUL.md", "only-soul")

	p, err := NewFSLoader(ws).Load()
	if err != nil {
		t.Fatal(err)
	}
	if p.Soul != "only-soul" {
		t.Errorf("Soul = %q", p.Soul)
	}
	if p.Agents != "" || p.User != "" || p.Tools != "" {
		t.Errorf("others not empty: %+v", p)
	}
}

func TestBootstrap_ConcatenatesInOrder(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "AGENTS.md", "A")
	writeFile(t, ws, "SOUL.md", "S")
	writeFile(t, ws, "USER.md", "U")
	writeFile(t, ws, "TOOLS.md", "T")

	p, _ := NewFSLoader(ws).Load()
	boot := p.Bootstrap()
	iA := strings.Index(boot, "## AGENTS.md")
	iS := strings.Index(boot, "## SOUL.md")
	iU := strings.Index(boot, "## USER.md")
	iT := strings.Index(boot, "## TOOLS.md")
	if !(iA < iS && iS < iU && iU < iT) {
		t.Errorf("expected order AGENTS<SOUL<USER<TOOLS, got %d %d %d %d", iA, iS, iU, iT)
	}
}

func TestBootstrap_SkipsEmptyFiles(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "SOUL.md", "has-content")

	p, _ := NewFSLoader(ws).Load()
	boot := p.Bootstrap()
	if !strings.Contains(boot, "## SOUL.md") {
		t.Error("missing SOUL section")
	}
	if strings.Contains(boot, "## AGENTS.md") {
		t.Error("should not contain AGENTS section")
	}
}
