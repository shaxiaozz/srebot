package skills

import (
	"strings"
	"testing"
)

func TestParseSKILL_OK(t *testing.T) {
	raw := []byte("---\nname: a\ndescription: x\nalways: true\n---\nhello\n")
	fm, body, err := parseSKILL(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Name != "a" || fm.Description != "x" || !fm.Always {
		t.Fatalf("bad frontmatter: %+v", fm)
	}
	if string(body) != "hello\n" {
		t.Fatalf("bad body: %q", string(body))
	}
}

func TestParseSKILL_NoFrontmatter(t *testing.T) {
	_, _, err := parseSKILL([]byte("name: a\nbody\n"))
	if err == nil || !strings.Contains(err.Error(), "opening") {
		t.Fatalf("expected opening-delimiter error, got %v", err)
	}
}

func TestParseSKILL_MissingClosing(t *testing.T) {
	_, _, err := parseSKILL([]byte("---\nname: a\n"))
	if err == nil || !strings.Contains(err.Error(), "closing") {
		t.Fatalf("expected closing-delimiter error, got %v", err)
	}
}

func TestParseSKILL_Empty(t *testing.T) {
	_, _, err := parseSKILL([]byte(""))
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty error, got %v", err)
	}
}

func TestParseSKILL_RequiresBlock(t *testing.T) {
	raw := []byte("---\nname: a\ndescription: x\nrequires:\n  bins: [echo]\n  env: [HOME]\n---\nbody\n")
	fm, _, err := parseSKILL(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fm.Requires.Bins) != 1 || fm.Requires.Bins[0] != "echo" {
		t.Fatalf("bad bins: %+v", fm.Requires.Bins)
	}
	if len(fm.Requires.Env) != 1 || fm.Requires.Env[0] != "HOME" {
		t.Fatalf("bad env: %+v", fm.Requires.Env)
	}
}
