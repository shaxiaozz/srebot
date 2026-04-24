package tools

import (
	"path/filepath"
	"testing"
)

func TestResolveInsideWorkspace_OK(t *testing.T) {
	ws := t.TempDir()
	got, err := resolveInsideWorkspace(ws, "foo/bar.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wsReal, _ := filepath.EvalSymlinks(ws)
	want := filepath.Join(wsReal, "foo/bar.txt")
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestResolveInsideWorkspace_AbsoluteRelRejected(t *testing.T) {
	ws := t.TempDir()
	if _, err := resolveInsideWorkspace(ws, "/etc/passwd"); err == nil {
		t.Fatal("expected error for absolute rel path")
	}
}

func TestResolveInsideWorkspace_DotDotDirect(t *testing.T) {
	ws := t.TempDir()
	if _, err := resolveInsideWorkspace(ws, "../foo"); err == nil {
		t.Fatal("expected error for ../foo")
	}
}

func TestResolveInsideWorkspace_DotDotEscapes(t *testing.T) {
	ws := t.TempDir()
	if _, err := resolveInsideWorkspace(ws, "a/../../b"); err == nil {
		t.Fatal("expected error for a/../../b")
	}
}

func TestResolveInsideWorkspace_EmptyRel(t *testing.T) {
	ws := t.TempDir()
	if _, err := resolveInsideWorkspace(ws, ""); err == nil {
		t.Fatal("expected error for empty rel")
	}
}

func TestResolveInsideWorkspace_NonAbsoluteWorkspace(t *testing.T) {
	if _, err := resolveInsideWorkspace("relative/workspace", "file.txt"); err == nil {
		t.Fatal("expected error for non-absolute workspace")
	}
}
