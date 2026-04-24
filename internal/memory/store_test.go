package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore_CreatesMemoryDir(t *testing.T) {
	ws := t.TempDir()
	if _, err := NewStore(ws); err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws, "memory")); err != nil {
		t.Fatalf("memory dir not created: %v", err)
	}
}

func TestReadMemory_MissingFileReturnsEmpty(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body, isTemplate, err := s.ReadMemory()
	if err != nil {
		t.Fatalf("ReadMemory: %v", err)
	}
	if body != "" {
		t.Errorf("body = %q, want empty", body)
	}
	if isTemplate {
		t.Error("isTemplate should be false")
	}
}

func TestReadMemory_ExistingFileReturnsContent(t *testing.T) {
	ws := t.TempDir()
	s, err := NewStore(ws)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(ws, "memory", "MEMORY.md")
	if err := os.WriteFile(path, []byte("# Facts\n- foo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	body, isTemplate, err := s.ReadMemory()
	if err != nil {
		t.Fatalf("ReadMemory: %v", err)
	}
	if !contains(body, "foo") {
		t.Errorf("body = %q should contain 'foo'", body)
	}
	if isTemplate {
		t.Error("isTemplate should be false")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAppendHistory_CursorMonotonic(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c1, err := s.AppendHistory("first")
	if err != nil {
		t.Fatal(err)
	}
	c2, err := s.AppendHistory("second")
	if err != nil {
		t.Fatal(err)
	}
	c3, err := s.AppendHistory("third")
	if err != nil {
		t.Fatal(err)
	}
	if c1 != 1 || c2 != 2 || c3 != 3 {
		t.Errorf("cursors = %d %d %d, want 1 2 3", c1, c2, c3)
	}
}

func TestAppendHistory_PersistsCursor(t *testing.T) {
	ws := t.TempDir()
	s1, _ := NewStore(ws)
	_, _ = s1.AppendHistory("a")
	_, _ = s1.AppendHistory("b")

	s2, _ := NewStore(ws)
	c, err := s2.AppendHistory("c")
	if err != nil {
		t.Fatal(err)
	}
	if c != 3 {
		t.Errorf("cursor = %d, want 3", c)
	}
}

func TestReadUnprocessedHistory_FiltersByCursor(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	_, _ = s.AppendHistory("one")
	_, _ = s.AppendHistory("two")
	_, _ = s.AppendHistory("three")

	entries, err := s.ReadUnprocessedHistory(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].Cursor != 2 || entries[0].Content != "two" {
		t.Errorf("entries[0] = %+v", entries[0])
	}
	if entries[1].Cursor != 3 {
		t.Errorf("entries[1].Cursor = %d, want 3", entries[1].Cursor)
	}
}

func TestDreamCursor_ZeroByDefault(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	if s.LastDreamCursor() != 0 {
		t.Errorf("LastDreamCursor = %d, want 0", s.LastDreamCursor())
	}
}

func TestDreamCursor_SetAndRead(t *testing.T) {
	ws := t.TempDir()
	s, _ := NewStore(ws)
	if err := s.SetLastDreamCursor(42); err != nil {
		t.Fatal(err)
	}
	s2, _ := NewStore(ws)
	if s2.LastDreamCursor() != 42 {
		t.Errorf("LastDreamCursor = %d, want 42", s2.LastDreamCursor())
	}
}

func TestCompactHistory_KeepsNewest(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	for i := 0; i < 5; i++ {
		_, _ = s.AppendHistory("entry")
	}
	if err := s.CompactHistory(2); err != nil {
		t.Fatal(err)
	}
	entries, err := s.ReadUnprocessedHistory(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].Cursor != 4 || entries[1].Cursor != 5 {
		t.Errorf("cursors = %d %d, want 4 5", entries[0].Cursor, entries[1].Cursor)
	}
}

func TestRawArchive_PrefixesMarker(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	if err := s.RawArchive("dumped messages"); err != nil {
		t.Fatal(err)
	}
	entries, _ := s.ReadUnprocessedHistory(0)
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d", len(entries))
	}
	if !contains(entries[0].Content, "[RAW]") || !contains(entries[0].Content, "dumped messages") {
		t.Errorf("content = %q", entries[0].Content)
	}
}

func TestCompactHistory_ZeroOrNegativeNoOp(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	for i := 0; i < 3; i++ {
		_, _ = s.AppendHistory("x")
	}
	if err := s.CompactHistory(0); err != nil {
		t.Fatal(err)
	}
	if err := s.CompactHistory(-1); err != nil {
		t.Fatal(err)
	}
	entries, _ := s.ReadUnprocessedHistory(0)
	if len(entries) != 3 {
		t.Errorf("entries = %d, want 3", len(entries))
	}
}

func TestCompactHistory_UnderThresholdNoOp(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	for i := 0; i < 2; i++ {
		_, _ = s.AppendHistory("x")
	}
	if err := s.CompactHistory(5); err != nil {
		t.Fatal(err)
	}
	entries, _ := s.ReadUnprocessedHistory(0)
	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}
}

func TestNextCursor_FallbackWhenCursorFileCorrupt(t *testing.T) {
	ws := t.TempDir()
	s, _ := NewStore(ws)
	_, _ = s.AppendHistory("first")
	_, _ = s.AppendHistory("second")
	// 破坏 .cursor 文件,迫使走兜底扫描路径
	if err := os.WriteFile(filepath.Join(ws, "memory", ".cursor"), []byte("not-a-number"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := s.AppendHistory("third")
	if err != nil {
		t.Fatal(err)
	}
	if c != 3 {
		t.Errorf("cursor = %d, want 3 (fallback scan)", c)
	}
}

func TestLastDreamCursor_CorruptFileReturnsZero(t *testing.T) {
	ws := t.TempDir()
	s, _ := NewStore(ws)
	if err := os.WriteFile(filepath.Join(ws, "memory", ".dream_cursor"), []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := s.LastDreamCursor(); got != 0 {
		t.Errorf("LastDreamCursor = %d, want 0", got)
	}
}
