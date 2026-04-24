package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkill(t *testing.T, root, name, fm, body string) {
	t.Helper()
	dir := filepath.Join(root, "skills", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\n" + fm + "---\n" + body
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestNewLoader_BuiltinOnly(t *testing.T) {
	builtin := t.TempDir()
	writeSkill(t, builtin, "a", "name: a\ndescription: first\n", "body a\n")
	l, err := NewLoader("", builtin, nil)
	if err != nil {
		t.Fatal(err)
	}
	sks := l.List()
	if len(sks) != 1 || sks[0].Name != "a" {
		t.Fatalf("unexpected list: %+v", sks)
	}
}

func TestNewLoader_WorkspaceOverridesBuiltin(t *testing.T) {
	builtin := t.TempDir()
	workspace := t.TempDir()
	writeSkill(t, builtin, "a", "name: a\ndescription: builtin\nalways: false\n", "b\n")
	writeSkill(t, workspace, "a", "name: a\ndescription: workspace\nalways: true\n", "w\n")
	l, err := NewLoader(workspace, builtin, nil)
	if err != nil {
		t.Fatal(err)
	}
	sks := l.List()
	if len(sks) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(sks))
	}
	if sks[0].Description != "workspace" || !sks[0].Always {
		t.Fatalf("workspace should win: %+v", sks[0])
	}
	if !strings.Contains(sks[0].Path, workspace) {
		t.Fatalf("path should be from workspace: %s", sks[0].Path)
	}
}

func TestNewLoader_Disabled(t *testing.T) {
	builtin := t.TempDir()
	writeSkill(t, builtin, "a", "name: a\ndescription: aa\n", "x\n")
	writeSkill(t, builtin, "b", "name: b\ndescription: bb\n", "x\n")
	l, err := NewLoader("", builtin, []string{"a"})
	if err != nil {
		t.Fatal(err)
	}
	sks := l.List()
	if len(sks) != 1 || sks[0].Name != "b" {
		t.Fatalf("expected only b: %+v", sks)
	}
}

func TestNewLoader_AvailabilityFromRequires(t *testing.T) {
	builtin := t.TempDir()
	writeSkill(t, builtin, "x", "name: x\ndescription: x\nrequires:\n  bins: [definitely-not-real-xyz]\n", "body\n")
	writeSkill(t, builtin, "y", "name: y\ndescription: y\n", "body\n")
	l, err := NewLoader("", builtin, nil)
	if err != nil {
		t.Fatal(err)
	}
	var x, y *Skill
	for _, s := range l.List() {
		s := s
		switch s.Name {
		case "x":
			x = &s
		case "y":
			y = &s
		}
	}
	if x == nil || y == nil {
		t.Fatalf("missing skills")
	}
	if x.Available {
		t.Fatalf("x should be unavailable")
	}
	if !y.Available {
		t.Fatalf("y should be available")
	}
}

func TestNewLoader_NonExistentDirs(t *testing.T) {
	l, err := NewLoader("/no/such/ws/path/xyz", "/no/such/bi/path/xyz", nil)
	if err != nil {
		t.Fatalf("should not error: %v", err)
	}
	if len(l.List()) != 0 {
		t.Fatalf("expected empty list")
	}
}

func TestSummary_ExcludeAndAnnotations(t *testing.T) {
	builtin := t.TempDir()
	writeSkill(t, builtin, "always1", "name: always1\ndescription: A\nalways: true\n", "x\n")
	writeSkill(t, builtin, "unavail1", "name: unavail1\ndescription: U\nrequires:\n  bins: [definitely-not-real-xyz]\n", "x\n")
	writeSkill(t, builtin, "normal1", "name: normal1\ndescription: N\n", "x\n")
	l, err := NewLoader("", builtin, nil)
	if err != nil {
		t.Fatal(err)
	}
	sum := l.Summary([]string{"normal1"})
	if strings.Contains(sum, "normal1") {
		t.Fatalf("summary should exclude normal1: %s", sum)
	}
	if !strings.Contains(sum, "always1") || !strings.Contains(sum, "[always]") {
		t.Fatalf("missing always annotation: %s", sum)
	}
	if !strings.Contains(sum, "unavail1") || !strings.Contains(sum, "[unavailable]") {
		t.Fatalf("missing unavailable annotation: %s", sum)
	}
}

func TestLoadContent_LazyAndCached(t *testing.T) {
	builtin := t.TempDir()
	writeSkill(t, builtin, "c", "name: c\ndescription: c\n", "hello world\n")
	l, err := NewLoader("", builtin, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := l.LoadContent("c")
	if err != nil {
		t.Fatal(err)
	}
	if first != "hello world\n" {
		t.Fatalf("bad body: %q", first)
	}
	// Mutate file on disk.
	path := filepath.Join(builtin, "skills", "c", "SKILL.md")
	if err := os.WriteFile(path, []byte("---\nname: c\ndescription: c\n---\nCHANGED\n"), 0644); err != nil {
		t.Fatal(err)
	}
	second, err := l.LoadContent("c")
	if err != nil {
		t.Fatal(err)
	}
	if second != "hello world\n" {
		t.Fatalf("expected cached content, got %q", second)
	}
}

func TestLoadContent_UnknownSkill(t *testing.T) {
	l, err := NewLoader("", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.LoadContent("nope")
	if err == nil || !strings.Contains(err.Error(), "unknown skill") {
		t.Fatalf("expected unknown skill error, got %v", err)
	}
}

func TestAlwaysSkills_FilterByAvailable(t *testing.T) {
	builtin := t.TempDir()
	writeSkill(t, builtin, "ok", "name: ok\ndescription: ok\nalways: true\n", "x\n")
	writeSkill(t, builtin, "broken", "name: broken\ndescription: b\nalways: true\nrequires:\n  bins: [definitely-not-real-xyz]\n", "x\n")
	writeSkill(t, builtin, "nonalways", "name: nonalways\ndescription: n\n", "x\n")
	l, err := NewLoader("", builtin, nil)
	if err != nil {
		t.Fatal(err)
	}
	as := l.AlwaysSkills()
	if len(as) != 1 || as[0].Name != "ok" {
		t.Fatalf("expected only 'ok' in always skills, got %+v", as)
	}
}
