package mcp

import (
	"testing"
	"time"
)

// TestNewClient_HTTPSelected verifies HTTP transport is picked when
// URL is set. We don't actually Connect() — that would hit the network.
func TestNewClient_HTTPSelected(t *testing.T) {
	c, err := newClient(Server{
		Name:        "remote",
		URL:         "https://example.com/mcp",
		ToolTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	if c.mc == nil {
		t.Fatal("expected underlying client to be set")
	}
	if c.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", c.timeout)
	}
	_ = c.Close()
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	c, err := newClient(Server{Name: "x", URL: "https://example.com/mcp"})
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	if c.timeout != defaultToolTimeout {
		t.Errorf("timeout = %v, want %v", c.timeout, defaultToolTimeout)
	}
	_ = c.Close()
}
