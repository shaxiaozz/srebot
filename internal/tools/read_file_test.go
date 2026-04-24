package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func invokeReadFile(t *testing.T, ws, path string) (string, error) {
	t.Helper()
	tr := NewReadFile(ws)
	args, _ := json.Marshal(map[string]string{"path": path})
	return tr.InvokableRun(context.Background(), string(args))
}

func TestReadFile_OK(t *testing.T) {
	ws := t.TempDir()
	content := "hello, world\n"
	if err := os.WriteFile(filepath.Join(ws, "hello.txt"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := invokeReadFile(t, ws, "hello.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != content {
		t.Errorf("want %q, got %q", content, got)
	}
}

func TestReadFile_NotFound(t *testing.T) {
	ws := t.TempDir()
	_, err := invokeReadFile(t, ws, "missing.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadFile_TooLarge(t *testing.T) {
	ws := t.TempDir()
	big := bytes.Repeat([]byte("x"), maxReadSize+1)
	if err := os.WriteFile(filepath.Join(ws, "big.txt"), big, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := invokeReadFile(t, ws, "big.txt")
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
}

func TestReadFile_NotUTF8(t *testing.T) {
	ws := t.TempDir()
	// Invalid UTF-8 sequence.
	if err := os.WriteFile(filepath.Join(ws, "bin.dat"), []byte{0xff, 0xfe, 0x00}, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := invokeReadFile(t, ws, "bin.dat")
	if err == nil {
		t.Fatal("expected error for non-UTF-8 file")
	}
}

func TestReadFile_PathEscapes(t *testing.T) {
	ws := t.TempDir()
	_, err := invokeReadFile(t, ws, "../outside.txt")
	if err == nil {
		t.Fatal("expected error for escaping path")
	}
}

func TestReadFile_ToolName(t *testing.T) {
	ws := t.TempDir()
	tr := NewReadFile(ws)
	info, err := tr.Info(context.Background())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Name != "read_file" {
		t.Errorf("want name %q, got %q", "read_file", info.Name)
	}
}
