package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_NewIDGeneratesUniqueFile(t *testing.T) {
	ws := t.TempDir()
	s1, err := Open(ws, "")
	if err != nil {
		t.Fatal(err)
	}
	defer s1.Close()
	s2, err := Open(ws, "")
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if s1.ID() == s2.ID() {
		t.Errorf("ids should differ: %q", s1.ID())
	}
}

func TestAppendAndLoad_RoundTrip(t *testing.T) {
	ws := t.TempDir()
	s, err := Open(ws, "test-sid")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Append(Message{Role: "user", Content: "hello"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Append(Message{Role: "assistant", Content: "world"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s2, err := Open(ws, "test-sid")
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := s2.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("len(msgs) = %d, want 2", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("msgs[0] = %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("msgs[1].Role = %q", msgs[1].Role)
	}
}

func TestLoad_MissingSessionReturnsEmpty(t *testing.T) {
	s, err := Open(t.TempDir(), "never-written")
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Errorf("len(msgs) = %d, want 0", len(msgs))
	}
}

func TestLoad_TruncatedLastLineDropped(t *testing.T) {
	ws := t.TempDir()
	s, err := Open(ws, "sid")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Append(Message{Role: "user", Content: "ok"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(ws, "session", "sid.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"role":"assistant","content":"partial`); err != nil {
		t.Fatal(err)
	}
	f.Close()

	s2, err := Open(ws, "sid")
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := s2.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	if msgs[0].Content != "ok" {
		t.Errorf("msgs[0].Content = %q", msgs[0].Content)
	}
}
