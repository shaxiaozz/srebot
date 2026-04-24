// Package session 为 agent 原始消息历史(session.jsonl)提供 append-only 存储。
// 它是"source-of-truth"日志(与 memory.history.jsonl 不同,后者存压缩摘要)。
package session

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Message 是每轮持久化的最小结构体。
// 后续 agent 层可通过 JSON 向前兼容地扩展字段。
type Message struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
	Timestamp  string          `json:"timestamp,omitempty"`
}

// Session 按 session id 持久化并恢复有序消息流。
type Session interface {
	ID() string
	Append(msg Message) error
	Load() ([]Message, error)
	Close() error
}

type fsSession struct {
	id   string
	path string
	f    *os.File
}

// Open 返回给定 id 的 Session;id 为 "" 时生成随机 id。
// session.jsonl 文件在第一次 Append 时按需创建。
func Open(workspace, id string) (Session, error) {
	dir := filepath.Join(workspace, "session")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir session: %w", err)
	}
	if id == "" {
		b := make([]byte, 6)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("gen session id: %w", err)
		}
		id = hex.EncodeToString(b)
	}
	return &fsSession{
		id:   id,
		path: filepath.Join(dir, id+".jsonl"),
	}, nil
}

func (s *fsSession) ID() string { return s.id }

func (s *fsSession) Append(msg Message) error {
	if s.f == nil {
		f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open session file: %w", err)
		}
		s.f = f
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	if _, err := s.f.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return nil
}

// Load 读取所有完整 JSON 行。最后一行若被截断则静默丢弃(崩溃恢复)。
func (s *fsSession) Load() ([]Message, error) {
	f, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open session: %w", err)
	}
	defer f.Close()

	var out []Message
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var m Message
		if err := json.Unmarshal(line, &m); err != nil {
			continue
		}
		out = append(out, m)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}
	return out, nil
}

func (s *fsSession) Close() error {
	if s.f == nil {
		return nil
	}
	err := s.f.Close()
	s.f = nil
	return err
}
