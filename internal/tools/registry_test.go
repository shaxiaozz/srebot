package tools

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// stubTool satisfies tool.InvokableTool for testing purposes.
type stubTool struct{ name string }

func (s *stubTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: s.name, Desc: "stub"}, nil
}

func (s *stubTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return "ok", nil
}

func newStub(name string) Tool { return &stubTool{name: name} }

// TestRegisterGetUnregister verifies the happy path.
func TestRegisterGetUnregister(t *testing.T) {
	r := NewRegistry()

	if err := r.Register(newStub("alpha")); err != nil {
		t.Fatalf("Register: unexpected error: %v", err)
	}

	got, ok := r.Get("alpha")
	if !ok {
		t.Fatal("Get: expected tool to exist")
	}
	info, _ := got.Info(context.Background())
	if info.Name != "alpha" {
		t.Errorf("Get: want name %q, got %q", "alpha", info.Name)
	}

	r.Unregister("alpha")
	if _, ok := r.Get("alpha"); ok {
		t.Error("Unregister: tool still present after removal")
	}
}

// TestRegisterDuplicate verifies that registering the same name twice returns an error.
func TestRegisterDuplicate(t *testing.T) {
	r := NewRegistry()

	if err := r.Register(newStub("dup")); err != nil {
		t.Fatalf("first Register: unexpected error: %v", err)
	}
	if err := r.Register(newStub("dup")); err == nil {
		t.Fatal("second Register: expected error for duplicate, got nil")
	}
}

// TestRegisterEmptyName verifies that a tool with an empty name is rejected.
func TestRegisterEmptyName(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(newStub("")); err == nil {
		t.Fatal("Register: expected error for empty name, got nil")
	}
}

// TestDefinitionsOrder verifies that Definitions returns infos sorted alphabetically.
func TestDefinitionsOrder(t *testing.T) {
	r := NewRegistry()
	names := []string{"zeta", "alpha", "mango", "beta"}
	for _, n := range names {
		if err := r.Register(newStub(n)); err != nil {
			t.Fatalf("Register %q: %v", n, err)
		}
	}

	infos, err := r.Definitions(context.Background())
	if err != nil {
		t.Fatalf("Definitions: %v", err)
	}
	if len(infos) != len(names) {
		t.Fatalf("Definitions: want %d infos, got %d", len(names), len(infos))
	}

	want := []string{"alpha", "beta", "mango", "zeta"}
	for i, info := range infos {
		if info.Name != want[i] {
			t.Errorf("Definitions[%d]: want %q, got %q", i, want[i], info.Name)
		}
	}
}

// TestConcurrentRegisterGet verifies there are no data races under concurrent access.
func TestConcurrentRegisterGet(t *testing.T) {
	r := NewRegistry()

	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := range workers {
		go func(i int) {
			defer wg.Done()
			name := string(rune('a' + i))
			_ = r.Register(newStub(name))
			_, _ = r.Get(name)
		}(i)
	}
	wg.Wait()
}

// flakyStub succeeds on the first Info call (used by Register) and returns an
// error on every subsequent call (used by Definitions).
type flakyStub struct {
	name      string
	infoCalls int
}

func (f *flakyStub) Info(_ context.Context) (*schema.ToolInfo, error) {
	f.infoCalls++
	if f.infoCalls > 1 {
		return nil, errors.New("info failed")
	}
	return &schema.ToolInfo{Name: f.name, Desc: "flaky"}, nil
}

func (f *flakyStub) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return "", nil
}

// TestDefinitions_InfoError verifies that Definitions propagates an Info error.
func TestDefinitions_InfoError(t *testing.T) {
	r := NewRegistry()

	if err := r.Register(&flakyStub{name: "x"}); err != nil {
		t.Fatalf("Register: unexpected error: %v", err)
	}

	infos, err := r.Definitions(context.Background())
	if err == nil {
		t.Fatal("Definitions: expected error, got nil")
	}
	if infos != nil {
		t.Errorf("Definitions: expected nil slice on error, got %v", infos)
	}
}

// TestUnregister_NonExistent verifies that Unregister is a no-op for unknown names.
func TestUnregister_NonExistent(t *testing.T) {
	r := NewRegistry()

	r.Unregister("does-not-exist") // must not panic

	if _, ok := r.Get("does-not-exist"); ok {
		t.Error("Get: expected false for never-registered tool")
	}
}
