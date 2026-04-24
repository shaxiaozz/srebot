package skills

import (
	"strings"
	"testing"
)

func TestRequirements_AllSatisfied(t *testing.T) {
	t.Setenv("FOO_REQ_TEST", "bar")
	r := Requirements{Bins: []string{"echo", "ls"}, Env: []string{"FOO_REQ_TEST"}}
	missing := r.Check()
	if len(missing) != 0 {
		t.Fatalf("expected no missing, got %v", missing)
	}
}

func TestRequirements_MissingBin(t *testing.T) {
	r := Requirements{Bins: []string{"definitely-not-a-real-binary-xyz123"}}
	missing := r.Check()
	if len(missing) != 1 || !strings.HasPrefix(missing[0], "bin:definitely-not-a-real-binary-xyz123") {
		t.Fatalf("unexpected missing: %v", missing)
	}
}

func TestRequirements_MissingEnv(t *testing.T) {
	r := Requirements{Env: []string{"NEVER_SET_VAR_XYZ_SREBOT"}}
	missing := r.Check()
	if len(missing) != 1 || missing[0] != "env:NEVER_SET_VAR_XYZ_SREBOT" {
		t.Fatalf("unexpected missing: %v", missing)
	}
}

func TestRequirements_Mixed(t *testing.T) {
	t.Setenv("PRESENT_VAR_SREBOT", "x")
	r := Requirements{
		Bins: []string{"echo", "definitely-not-a-real-binary-xyz123"},
		Env:  []string{"PRESENT_VAR_SREBOT", "NEVER_SET_VAR_XYZ_SREBOT"},
	}
	missing := r.Check()
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing, got %v", missing)
	}
	joined := strings.Join(missing, ",")
	if !strings.Contains(joined, "bin:definitely") || !strings.Contains(joined, "env:NEVER_SET_VAR_XYZ_SREBOT") {
		t.Fatalf("unexpected contents: %v", missing)
	}
}
