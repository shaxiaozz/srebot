package mcp

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/shaxiaozz/srebot/internal/tools"
)

// silentLogger discards logger output so test runs stay clean.
func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestManager_EmptyServers_NoPanic(t *testing.T) {
	m := NewManagerWithLogger(nil, silentLogger())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	m.ConnectAll(ctx)
	if len(m.clients) != 0 {
		t.Errorf("expected 0 clients, got %d", len(m.clients))
	}

	reg := tools.NewRegistry()
	if err := m.RegisterAllInto(reg); err != nil {
		t.Errorf("RegisterAllInto: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestManager_UnreachableHTTPServer_SkippedGracefully(t *testing.T) {
	// 127.0.0.1:1 is guaranteed-unreachable on macOS/Linux (port 1 is
	// privileged and nothing is bound in test envs). mcp-go's
	// streamable HTTP client will fail Start/Initialize quickly.
	servers := []Server{
		{Name: "dead", URL: "http://127.0.0.1:1/mcp", ToolTimeout: 1 * time.Second},
	}
	m := NewManagerWithLogger(servers, silentLogger())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	m.ConnectAll(ctx)
	if len(m.clients) != 0 {
		t.Errorf("expected unreachable server to be skipped, got %d clients", len(m.clients))
	}
}

func TestManager_InvalidStdioCommand_SkippedGracefully(t *testing.T) {
	servers := []Server{
		{Name: "bad", Command: "/does/not/exist/nope-binary-xyz", ToolTimeout: 1 * time.Second},
	}
	m := NewManagerWithLogger(servers, silentLogger())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	m.ConnectAll(ctx)
	if len(m.clients) != 0 {
		t.Errorf("expected bad stdio command to be skipped, got %d clients", len(m.clients))
	}
}

func TestNewManager_NilLoggerFallback(t *testing.T) {
	m := NewManagerWithLogger(nil, nil)
	if m.logger == nil {
		t.Error("expected default logger, got nil")
	}
}

func TestNewManager_DefaultConstructor(t *testing.T) {
	m := NewManager(nil)
	if m.logger == nil {
		t.Error("expected non-nil logger")
	}
}

// TODO: Add a full httptest.Server-based MCP Streamable HTTP integration
// test that exercises Connect + ListTools + CallTool + Close. Skipped for
// MVP because faithfully implementing the Streamable HTTP JSON-RPC
// handshake in-test is non-trivial. Coverage goal here is breadth
// (graceful failure, registry plumbing), not a full protocol mock.
