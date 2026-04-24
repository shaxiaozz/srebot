// Package memory 提供 MEMORY.md 长期事实存储、history.jsonl 压缩条目
// append-only 日志以及单调自增的游标管理。
package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// HistoryEntry 是 Consolidator 追加的单条压缩记忆记录。
type HistoryEntry struct {
	Cursor    int    `json:"cursor"`
	Timestamp string `json:"timestamp"`
	Content   string `json:"content"`
}

// Store 是 memory 子系统的纯文件 I/O 层。
type Store interface {
	ReadMemory() (body string, isTemplate bool, err error)
	AppendHistory(entry string) (cursor int, err error)
	ReadUnprocessedHistory(sinceCursor int) ([]HistoryEntry, error)
	LastDreamCursor() int
	SetLastDreamCursor(c int) error
	CompactHistory(maxEntries int) error
	RawArchive(raw string) error
}

type fsStore struct {
	workspace   string
	memoryDir   string
	memoryFile  string
	historyFile string
	cursorFile  string
	dreamCursor string
}

// NewStore 在 “<workspace>/memory/“ 下创建(或打开)Store。
func NewStore(workspace string) (Store, error) {
	dir := filepath.Join(workspace, "memory")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir memory: %w", err)
	}
	return &fsStore{
		workspace:   workspace,
		memoryDir:   dir,
		memoryFile:  filepath.Join(dir, "MEMORY.md"),
		historyFile: filepath.Join(dir, "history.jsonl"),
		cursorFile:  filepath.Join(dir, ".cursor"),
		dreamCursor: filepath.Join(dir, ".dream_cursor"),
	}, nil
}

// ReadMemory 返回 MEMORY.md 正文。isTemplate 预留给未来"是否为未改动模板"的判断,
// MVP 始终返回 false。
func (s *fsStore) ReadMemory() (string, bool, error) {
	b, err := os.ReadFile(s.memoryFile)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read MEMORY.md: %w", err)
	}
	return string(b), false, nil
}

// AppendHistory 将 entry 追加到 history.jsonl 并返回新分配的游标值。
func (s *fsStore) AppendHistory(entry string) (int, error) {
	c, err := s.nextCursor()
	if err != nil {
		return 0, err
	}
	record := HistoryEntry{
		Cursor:    c,
		Timestamp: time.Now().UTC().Format("2006-01-02 15:04"),
		Content:   entry,
	}
	b, err := json.Marshal(record)
	if err != nil {
		return 0, fmt.Errorf("marshal history entry: %w", err)
	}
	f, err := os.OpenFile(s.historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open history.jsonl: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return 0, fmt.Errorf("write history.jsonl: %w", err)
	}
	if err := os.WriteFile(s.cursorFile, []byte(strconv.Itoa(c)), 0o644); err != nil {
		return 0, fmt.Errorf("write .cursor: %w", err)
	}
	return c, nil
}

func (s *fsStore) nextCursor() (int, error) {
	if b, err := os.ReadFile(s.cursorFile); err == nil {
		if n, perr := strconv.Atoi(string(b)); perr == nil {
			return n + 1, nil
		}
	}
	// 兜底:扫描文件取最大游标。
	entries, err := s.readAllEntries()
	if err != nil {
		return 0, err
	}
	max := 0
	for _, e := range entries {
		if e.Cursor > max {
			max = e.Cursor
		}
	}
	return max + 1, nil
}

// ReadUnprocessedHistory 返回 Cursor > sinceCursor 的所有条目。
func (s *fsStore) ReadUnprocessedHistory(sinceCursor int) ([]HistoryEntry, error) {
	entries, err := s.readAllEntries()
	if err != nil {
		return nil, err
	}
	out := make([]HistoryEntry, 0, len(entries))
	for _, e := range entries {
		if e.Cursor > sinceCursor {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *fsStore) readAllEntries() ([]HistoryEntry, error) {
	f, err := os.Open(s.historyFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open history.jsonl: %w", err)
	}
	defer f.Close()

	var out []HistoryEntry
	dec := json.NewDecoder(f)
	for {
		var e HistoryEntry
		if err := dec.Decode(&e); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// LastDreamCursor 返回上次 Dream 水位(未设置时为 0)。
func (s *fsStore) LastDreamCursor() int {
	b, err := os.ReadFile(s.dreamCursor)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(string(b))
	if err != nil {
		return 0
	}
	return n
}

// SetLastDreamCursor 持久化 Dream 水位。
func (s *fsStore) SetLastDreamCursor(c int) error {
	return os.WriteFile(s.dreamCursor, []byte(strconv.Itoa(c)), 0o644)
}

// CompactHistory 截断 history.jsonl 到最新 maxEntries 条。
func (s *fsStore) CompactHistory(maxEntries int) error {
	if maxEntries <= 0 {
		return nil
	}
	entries, err := s.readAllEntries()
	if err != nil {
		return err
	}
	if len(entries) <= maxEntries {
		return nil
	}
	keep := entries[len(entries)-maxEntries:]
	tmp := s.historyFile + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	for _, e := range keep {
		b, _ := json.Marshal(e)
		if _, err := f.Write(append(b, '\n')); err != nil {
			f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, s.historyFile)
}

// RawArchive 在 Consolidator LLM 调用失败时,将原始消息以 [RAW] 前缀追加为兜底。
func (s *fsStore) RawArchive(raw string) error {
	_, err := s.AppendHistory("[RAW] " + raw)
	return err
}
