package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func TestLoad_MissingPathReturnsError(t *testing.T) {
	if _, err := Load("/nonexistent/config.json"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_ValidConfigParsesFields(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.json", `{
  "agents": {
    "defaults": {
      "apiKey": "sk-abc",
      "baseUrl": "https://api.deepseek.com/v1",
      "model": "deepseek-chat",
      "provider": "openai"
    }
  }
}`)
	res, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Agent.APIKey != "sk-abc" {
		t.Errorf("APIKey = %q", res.Agent.APIKey)
	}
	if res.Agent.BaseURL != "https://api.deepseek.com/v1" {
		t.Errorf("BaseURL = %q", res.Agent.BaseURL)
	}
	if res.Agent.Model != "deepseek-chat" {
		t.Errorf("Model = %q", res.Agent.Model)
	}
}

func TestLoad_MissingFieldsFilledByDefaults(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.json", `{
  "agents": { "defaults": { "apiKey": "sk-x" } }
}`)
	res, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Agent.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %q", res.Agent.BaseURL)
	}
	if res.Agent.Model != "gpt-4o-mini" {
		t.Errorf("Model = %q", res.Agent.Model)
	}
	if res.Agent.TimeoutSec != 120 {
		t.Errorf("TimeoutSec = %d", res.Agent.TimeoutSec)
	}
	if res.Memory.MaxHistoryEntries != 1000 {
		t.Errorf("MaxHistoryEntries = %d", res.Memory.MaxHistoryEntries)
	}
	if !res.Approval.ReadOnlyAutoApprove {
		t.Error("ReadOnlyAutoApprove should default true")
	}
	if res.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q", res.Logging.Level)
	}
}

func TestLoad_TopLevelOverrides(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.json", `{
  "agents": { "defaults": { "apiKey": "sk-x" } },
  "approval": { "yolo": true, "readOnlyAutoApprove": false },
  "logging":  { "level": "debug" },
  "memory":   { "maxHistoryEntries": 2000 }
}`)
	res, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !res.Approval.YOLO {
		t.Error("YOLO should be true")
	}
	if res.Approval.ReadOnlyAutoApprove {
		t.Error("ReadOnlyAutoApprove should be false")
	}
	if res.Logging.Level != "debug" {
		t.Errorf("Level = %q", res.Logging.Level)
	}
	if res.Memory.MaxHistoryEntries != 2000 {
		t.Errorf("MaxHistoryEntries = %d", res.Memory.MaxHistoryEntries)
	}
}

func TestLoad_EmptyAPIKeyFailsValidation(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.json", `{
  "agents": { "defaults": { "model": "gpt-4o" } }
}`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "apiKey") {
		t.Errorf("error %q should mention apiKey", err)
	}
}

func TestLoad_BadJSONReturnsError(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.json", `{ this is not: valid json`)
	if _, err := Load(p); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_ExpandsTildeInWorkspace(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.json", `{
  "agents": { "defaults": { "apiKey": "sk-x", "workspace": "~/srebot-test-ws" } }
}`)
	res, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "srebot-test-ws")
	if res.Agent.Workspace != want {
		t.Errorf("Workspace = %q, want %q", res.Agent.Workspace, want)
	}
}

func TestLoad_ExpandsBareTilde(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.json", `{
  "agents": { "defaults": { "apiKey": "sk-x", "workspace": "~" } }
}`)
	res, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	home, _ := os.UserHomeDir()
	if res.Agent.Workspace != home {
		t.Errorf("Workspace = %q, want %q", res.Agent.Workspace, home)
	}
}

func TestLoad_NonTildePathUnchanged(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.json", `{
  "agents": { "defaults": { "apiKey": "sk-x", "workspace": "/tmp/ws" } }
}`)
	res, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Agent.Workspace != "/tmp/ws" {
		t.Errorf("Workspace = %q", res.Agent.Workspace)
	}
}

func TestLoad_TildePrefixWithoutSlashUnchanged(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.json", `{
  "agents": { "defaults": { "apiKey": "sk-x", "workspace": "~alice/ws" } }
}`)
	res, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// "~alice/ws" 不匹配 "~" 或 "~/" 前缀,保持原样
	if res.Agent.Workspace != "~alice/ws" {
		t.Errorf("Workspace = %q", res.Agent.Workspace)
	}
}

func TestLoad_WithMCPServers(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.json", `{
  "agents": { "defaults": { "apiKey": "sk-x" } },
  "mcpServers": {
    "remote": { "url": "https://example.com/mcp", "toolTimeout": 15 },
    "local":  { "command": "mcp-bin", "args": ["--x"], "env": { "K": "V" } }
  }
}`)
	res, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(res.MCPServers) != 2 {
		t.Fatalf("expected 2 mcp servers, got %d", len(res.MCPServers))
	}
	remote := res.MCPServers["remote"]
	if remote.URL != "https://example.com/mcp" {
		t.Errorf("remote.URL = %q", remote.URL)
	}
	if remote.ToolTimeout != 15 {
		t.Errorf("remote.ToolTimeout = %d", remote.ToolTimeout)
	}
	local := res.MCPServers["local"]
	if local.Command != "mcp-bin" {
		t.Errorf("local.Command = %q", local.Command)
	}
	if len(local.Args) != 1 || local.Args[0] != "--x" {
		t.Errorf("local.Args = %v", local.Args)
	}
	if local.Env["K"] != "V" {
		t.Errorf("local.Env[K] = %q", local.Env["K"])
	}
	// default timeout for local since it wasn't specified
	if local.ToolTimeout != 60 {
		t.Errorf("local.ToolTimeout = %d, want 60", local.ToolTimeout)
	}
}
