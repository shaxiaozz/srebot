package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func invokeWriteFile(t *testing.T, ws, path, content string, appendMode bool) (string, error) {
	t.Helper()
	tw := NewWriteFile(ws)
	args, _ := json.Marshal(map[string]interface{}{
		"path":    path,
		"content": content,
		"append":  appendMode,
	})
	return tw.InvokableRun(context.Background(), string(args))
}

func TestWriteFile_Basic(t *testing.T) {
	ws := t.TempDir()
	msg, err := invokeWriteFile(t, ws, "out.txt", "hello", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "out.txt") {
		t.Errorf("expected path in return message, got %q", msg)
	}

	data, err := os.ReadFile(filepath.Join(ws, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("want %q, got %q", "hello", string(data))
	}
}

func TestWriteFile_Append(t *testing.T) {
	ws := t.TempDir()
	if _, err := invokeWriteFile(t, ws, "log.txt", "first\n", false); err != nil {
		t.Fatal(err)
	}
	if _, err := invokeWriteFile(t, ws, "log.txt", "second\n", true); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(ws, "log.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := "first\nsecond\n"
	if string(data) != want {
		t.Errorf("want %q, got %q", want, string(data))
	}
}

func TestWriteFile_CreatesParentDirs(t *testing.T) {
	ws := t.TempDir()
	if _, err := invokeWriteFile(t, ws, "a/b/c/deep.txt", "content", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(ws, "a/b/c/deep.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "content" {
		t.Errorf("want %q, got %q", "content", string(data))
	}
}

func TestWriteFile_PathEscapeRejected(t *testing.T) {
	ws := t.TempDir()
	_, err := invokeWriteFile(t, ws, "../escape.txt", "bad", false)
	if err == nil {
		t.Fatal("expected error for escaping path")
	}
}

func TestWriteFile_MessageFormat(t *testing.T) {
	ws := t.TempDir()
	msg, err := invokeWriteFile(t, ws, "msg.txt", "abc", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(msg, "wrote ") {
		t.Errorf("unexpected message format: %q", msg)
	}
	if !strings.Contains(msg, "msg.txt") {
		t.Errorf("expected path in message: %q", msg)
	}
}

func TestWriteFile_ToolName(t *testing.T) {
	ws := t.TempDir()
	tw := NewWriteFile(ws)
	info, err := tw.Info(context.Background())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Name != "write_file" {
		t.Errorf("want name %q, got %q", "write_file", info.Name)
	}
}

func TestWriteFile_Overwrite(t *testing.T) {
	ws := t.TempDir()
	if _, err := invokeWriteFile(t, ws, "over.txt", "original", false); err != nil {
		t.Fatal(err)
	}
	if _, err := invokeWriteFile(t, ws, "over.txt", "new", false); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(ws, "over.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Errorf("want %q, got %q", "new", string(data))
	}
}
