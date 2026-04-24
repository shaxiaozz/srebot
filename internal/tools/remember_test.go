package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shaxiaozz/srebot/internal/memory"
)

func invokeRemember(t *testing.T, store memory.Store, fact string) (string, error) {
	t.Helper()
	tool := NewRemember(store)
	args, _ := json.Marshal(map[string]string{"fact": fact})
	return tool.InvokableRun(context.Background(), string(args))
}

func TestRemember_AppendsToStore(t *testing.T) {
	store, err := memory.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	const fact = "production runs us-east-1 only"
	res, err := invokeRemember(t, store, fact)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(res, "remembered") {
		t.Errorf("want 'remembered' in response, got %q", res)
	}
	if !strings.Contains(res, "cursor=1") {
		t.Errorf("want 'cursor=1' in response, got %q", res)
	}

	// Verify via store.ReadUnprocessedHistory
	entries, err := store.ReadUnprocessedHistory(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Content, "[MEMO]") {
		t.Errorf("want '[MEMO]' in content, got %q", entries[0].Content)
	}
	if !strings.Contains(entries[0].Content, fact) {
		t.Errorf("want %q in content, got %q", fact, entries[0].Content)
	}
}

func TestRemember_EmptyFact(t *testing.T) {
	store, err := memory.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_, err = invokeRemember(t, store, "")
	if err == nil {
		t.Fatal("expected error for empty fact")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("want 'must not be empty' in error, got %q", err.Error())
	}
}

func TestRemember_InvalidJSON(t *testing.T) {
	store, err := memory.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	tool := NewRemember(store)
	_, err = tool.InvokableRun(context.Background(), "not-json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid args") {
		t.Errorf("want 'invalid args' in error, got %q", err.Error())
	}
}

func TestRemember_ToolName(t *testing.T) {
	store, err := memory.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	tool := NewRemember(store)
	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Name != "remember" {
		t.Errorf("want name %q, got %q", "remember", info.Name)
	}
}

func TestRemember_MultipleFacts(t *testing.T) {
	store, err := memory.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	fact1 := "database is in us-west-2"
	fact2 := "queue timeout is 30 seconds"

	res1, err := invokeRemember(t, store, fact1)
	if err != nil {
		t.Fatalf("first remember: %v", err)
	}
	if !strings.Contains(res1, "cursor=1") {
		t.Errorf("want 'cursor=1' in first response, got %q", res1)
	}

	res2, err := invokeRemember(t, store, fact2)
	if err != nil {
		t.Fatalf("second remember: %v", err)
	}
	if !strings.Contains(res2, "cursor=2") {
		t.Errorf("want 'cursor=2' in second response, got %q", res2)
	}

	// Verify both entries are in store
	entries, err := store.ReadUnprocessedHistory(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}

	// Check both facts are present with [MEMO] prefix
	if !strings.Contains(entries[0].Content, fact1) {
		t.Errorf("want %q in first entry content", fact1)
	}
	if !strings.Contains(entries[1].Content, fact2) {
		t.Errorf("want %q in second entry content", fact2)
	}
	for i, e := range entries {
		if !strings.Contains(e.Content, "[MEMO]") {
			t.Errorf("entry %d missing '[MEMO]' prefix", i)
		}
	}
}
