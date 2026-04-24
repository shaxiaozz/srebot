package mcp

import (
	"context"
	"log/slog"
	"sync"

	"github.com/shaxiaozz/srebot/internal/tools"
)

// Manager owns a set of MCP servers, connects to them in parallel and
// registers their tools into a tools.Registry. Failed connections are
// logged and skipped — they never abort the others.
type Manager struct {
	servers []Server
	clients []*Client // only successfully connected
	logger  *slog.Logger
}

// NewManager returns a Manager using slog.Default() for logging.
func NewManager(servers []Server) *Manager {
	return NewManagerWithLogger(servers, slog.Default())
}

// NewManagerWithLogger lets callers inject a slog.Logger. A nil logger
// falls back to slog.Default().
func NewManagerWithLogger(servers []Server, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{servers: servers, logger: logger}
}

// ConnectAll connects to every configured server in parallel. Failures
// are logged at warn level and the offending client is skipped — the
// method never returns an error so a single broken MCP server cannot
// take down the agent.
func (m *Manager) ConnectAll(ctx context.Context) {
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	for _, s := range m.servers {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := newClient(s)
			if err != nil {
				m.logger.Warn("mcp: create client failed", "server", s.Name, "err", err)
				return
			}
			if err := c.Connect(ctx); err != nil {
				m.logger.Warn("mcp: connect failed", "server", s.Name, "err", err)
				_ = c.Close()
				return
			}
			mu.Lock()
			m.clients = append(m.clients, c)
			mu.Unlock()
		}()
	}
	wg.Wait()
}

// RegisterAllInto registers every discovered tool from every connected
// client into reg. Returns the first registration error encountered.
func (m *Manager) RegisterAllInto(reg tools.Registry) error {
	for _, c := range m.clients {
		for _, t := range c.Tools() {
			if err := reg.Register(t); err != nil {
				return err
			}
		}
	}
	return nil
}

// Close closes every connected client. Returns the first error it sees,
// but always attempts to close all clients.
func (m *Manager) Close() error {
	var firstErr error
	for _, c := range m.clients {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
