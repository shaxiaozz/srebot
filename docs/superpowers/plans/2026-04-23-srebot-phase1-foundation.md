# srebot Foundation(第一阶段)实施计划

> **给执行方的说明:** 必需子技能:使用 superpowers:subagent-driven-development(推荐)或 superpowers:executing-plans 按任务逐个实施本计划。步骤用 checkbox(`- [ ]`)语法追踪进度。

**目标:** 初始化 Go module,并实现四个基础的存储/IO 模块(`config`、`memory.Store`、`session`、`persona`),含完整单元测试。

**架构:** 纯 I/O / 解析层。不引入 LLM、不依赖 eino、无网络调用。每个模块独立、可测试,供后续阶段组装。

**技术栈:** Go 1.25.8、仅 Go 标准库(`encoding/json`、`log/slog`、`testing`)。**所有测试用原生 `t.Errorf`/`t.Fatal` 断言,不使用 testify**。

**范围:** 3 个阶段中的第 1 个。第 2 阶段(skills + approval + tools + mcp + provider)与第 3 阶段(Consolidator + agent + CLI + E2E)随后。

**Spec 引用:** `docs/superpowers/specs/2026-04-23-srebot-mvp-design.md` 的 §5.1、§5.3、§5.5(仅 Store)、§5.6。

---

## 文件结构

**本阶段新建文件:**

```
srebot/
├── go.mod                              # module init
├── go.sum                              # 依赖 lock
├── config/
│   ├── config.go                       # 结构体 + Defaults + Validate
│   ├── loader.go                       # Load(path):读 JSON → 填 Defaults → Validate
│   ├── config.example.json             # 用户样例
│   ├── config_test.go
│   └── loader_test.go
├── internal/
│   ├── memory/
│   │   ├── store.go                    # MemoryStore:MEMORY.md + history.jsonl + 游标
│   │   └── store_test.go
│   ├── session/
│   │   ├── session.go                  # session.jsonl append/load
│   │   └── session_test.go
│   └── persona/
│       ├── loader.go                   # 四 MD 加载器
│       └── loader_test.go
└── .gitignore
```

---

## 任务 1:Module 初始化 ✅

**文件:**
- 新建:`go.mod`、`.gitignore`

- [x] **步骤 1:初始化 Go module**

执行(cwd=`/Users/fengjin/Desktop/GitHub/srebot`):
```bash
go mod init github.com/shaxiaozz/srebot
```
预期:生成 `go.mod`,内容为 `module github.com/shaxiaozz/srebot` 及 `go 1.25.8`。

- [x] **步骤 2:创建 `.gitignore`**

新建 `/Users/fengjin/Desktop/GitHub/srebot/.gitignore`:
```
# build
/srebot
/bin/
/dist/

# test
coverage.out
*.test

# editor
.vscode/
.idea/
*.swp
.DS_Store

# runtime
/tmp/
~/.srebot/
```

- [x] **步骤 3:提交**

```bash
git add go.mod go.sum .gitignore
git commit -m "chore: init go module and gitignore"
```

---

## 任务 2:config 包 — 结构体、默认值与 Validate ✅

**文件:**
- 新建:`config/config.go`
- 测试:`config/config_test.go`

**结构约定:**
- `Root` = JSON 顶层;子字段 `agents.defaults` 只放 agent/LLM 参数,其余 `approval`/`logging`/`memory` 是全局运行时配置
- `Resolved` = 供下游组件使用的平面结构(去掉 `agents.defaults` 嵌套)

- [x] **步骤 1:写失败测试** — 新建 `config/config_test.go`:

```go
package config

import "testing"

func TestDefaults_HasSaneValues(t *testing.T) {
	r := Defaults()
	if r.Agents.Defaults.Provider != "openai" {
		t.Errorf("Provider = %q, want openai", r.Agents.Defaults.Provider)
	}
	if r.Agents.Defaults.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %q", r.Agents.Defaults.BaseURL)
	}
	if r.Agents.Defaults.Model != "gpt-4o-mini" {
		t.Errorf("Model = %q", r.Agents.Defaults.Model)
	}
	if r.Agents.Defaults.TimeoutSec != 120 {
		t.Errorf("TimeoutSec = %d", r.Agents.Defaults.TimeoutSec)
	}
	if r.Agents.Defaults.ContextWindowTokens != 65536 {
		t.Errorf("ContextWindowTokens = %d", r.Agents.Defaults.ContextWindowTokens)
	}
	if r.Memory.MaxHistoryEntries != 1000 {
		t.Errorf("MaxHistoryEntries = %d", r.Memory.MaxHistoryEntries)
	}
	if r.Approval.YOLO {
		t.Error("YOLO should default false")
	}
	if !r.Approval.ReadOnlyAutoApprove {
		t.Error("ReadOnlyAutoApprove should default true")
	}
	if r.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q", r.Logging.Level)
	}
}

func TestResolve_FlattensRoot(t *testing.T) {
	r := Defaults()
	r.Agents.Defaults.APIKey = "sk-x"
	res := r.Resolve()
	if res.Agent.APIKey != "sk-x" {
		t.Errorf("Agent.APIKey = %q", res.Agent.APIKey)
	}
	if res.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q", res.Logging.Level)
	}
	if res.Memory.MaxHistoryEntries != 1000 {
		t.Errorf("Memory.MaxHistoryEntries = %d", res.Memory.MaxHistoryEntries)
	}
}

func TestValidate_RejectsEmptyAPIKey(t *testing.T) {
	res := Defaults().Resolve()
	res.Agent.APIKey = ""
	err := res.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "apiKey") {
		t.Errorf("error %q should mention apiKey", err)
	}
}

func TestValidate_RejectsZeroContextWindow(t *testing.T) {
	res := Defaults().Resolve()
	res.Agent.APIKey = "sk-test"
	res.Agent.ContextWindowTokens = 0
	err := res.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "contextWindowTokens") {
		t.Errorf("error %q should mention contextWindowTokens", err)
	}
}

func TestValidate_OKWhenMinimalFieldsSet(t *testing.T) {
	res := Defaults().Resolve()
	res.Agent.APIKey = "sk-test"
	if err := res.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// containsStr reports whether substr is within s.
func containsStr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

- [x] **步骤 2:运行测试,确认失败**

```bash
go test ./config/...
```
预期:`undefined: Defaults` / `undefined: Root` / `undefined: Resolved`。

- [x] **步骤 3:编写 `config/config.go`**

```go
// Package config 定义并加载 srebot 的 config.json 运行时配置。
//
// 结构:
//   Root = JSON 顶层
//     agents.defaults: 每个 agent 可独立覆盖的 LLM/工作区参数
//     approval/logging/memory: 全局运行时配置(不随 agent 切换)
//   Resolved = 供下游组件使用的平面视图(MVP 只用 agents.defaults)
package config

import (
	"fmt"
	"strings"
)

// Root 映射 config.json 顶层结构。
type Root struct {
	Agents   AgentsSection  `json:"agents"`
	Approval ApprovalConfig `json:"approval"`
	Logging  LoggingConfig  `json:"logging"`
	Memory   MemoryConfig   `json:"memory"`
}

// AgentsSection 对齐 nanobot 的 agents.{defaults,<name>} 嵌套。
// MVP 只使用 defaults。
type AgentsSection struct {
	Defaults AgentConfig `json:"defaults"`
}

// AgentConfig 是单个 agent 的 LLM / 工作区参数。
type AgentConfig struct {
	Workspace           string   `json:"workspace"`
	PresetDir           string   `json:"presetDir"`
	Provider            string   `json:"provider"`
	BaseURL             string   `json:"baseUrl"`
	APIKey              string   `json:"apiKey"`
	Model               string   `json:"model"`
	Temperature         float64  `json:"temperature"`
	TimeoutSec          int      `json:"timeoutSec"`
	MaxTokens           int      `json:"maxTokens"`
	ContextWindowTokens int      `json:"contextWindowTokens"`
	DisabledSkills      []string `json:"disabledSkills"`
}

// ApprovalConfig 全局工具审批配置。
type ApprovalConfig struct {
	YOLO                bool     `json:"yolo"`
	ReadOnlyAutoApprove bool     `json:"readOnlyAutoApprove"`
	Whitelist           []string `json:"whitelist"`
}

// LoggingConfig 全局日志配置。
type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

// MemoryConfig 全局 memory 参数。
type MemoryConfig struct {
	MaxHistoryEntries int `json:"maxHistoryEntries"`
}

// Resolved 是合并后供下游组件消费的平面结构。
type Resolved struct {
	Agent    AgentConfig
	Approval ApprovalConfig
	Logging  LoggingConfig
	Memory   MemoryConfig
}

// Defaults 返回带安全默认值的 Root。用于填充 JSON 缺失字段。
func Defaults() Root {
	return Root{
		Agents: AgentsSection{
			Defaults: AgentConfig{
				Workspace:           "~/.srebot/workspace",
				PresetDir:           "presets/default",
				Provider:            "openai",
				BaseURL:             "https://api.openai.com/v1",
				Model:               "gpt-4o-mini",
				Temperature:         0.2,
				TimeoutSec:          120,
				MaxTokens:           8192,
				ContextWindowTokens: 65536,
				DisabledSkills:      nil,
			},
		},
		Approval: ApprovalConfig{
			YOLO:                false,
			ReadOnlyAutoApprove: true,
			Whitelist:           nil,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		Memory: MemoryConfig{
			MaxHistoryEntries: 1000,
		},
	}
}

// Resolve 把 Root 展平成 Resolved。MVP 只使用 agents.defaults。
func (r Root) Resolve() Resolved {
	return Resolved{
		Agent:    r.Agents.Defaults,
		Approval: r.Approval,
		Logging:  r.Logging,
		Memory:   r.Memory,
	}
}

// Validate 在必填字段缺失或越界时返回 error。
func (r *Resolved) Validate() error {
	var errs []string
	a := r.Agent
	if strings.TrimSpace(a.APIKey) == "" {
		errs = append(errs, "agents.defaults.apiKey is required")
	}
	if strings.TrimSpace(a.BaseURL) == "" {
		errs = append(errs, "agents.defaults.baseUrl is required")
	}
	if strings.TrimSpace(a.Model) == "" {
		errs = append(errs, "agents.defaults.model is required")
	}
	if strings.TrimSpace(a.Provider) == "" {
		errs = append(errs, "agents.defaults.provider is required")
	}
	if a.TimeoutSec <= 0 {
		errs = append(errs, "agents.defaults.timeoutSec must be > 0")
	}
	if a.ContextWindowTokens <= 0 {
		errs = append(errs, "agents.defaults.contextWindowTokens must be > 0")
	}
	if r.Memory.MaxHistoryEntries <= 0 {
		errs = append(errs, "memory.maxHistoryEntries must be > 0")
	}
	if len(errs) > 0 {
		return fmt.Errorf("config invalid: %s", strings.Join(errs, "; "))
	}
	return nil
}
```

- [x] **步骤 4:运行测试,确认通过**

```bash
go test ./config/... -v
```
预期:5 个测试全部 PASS。

- [x] **步骤 5:提交**

```bash
git add config/
git commit -m "feat(config): Root/AgentConfig/Resolved structs + Defaults + Resolve + Validate"
```

---

## 任务 3:config 包 — Load(从 config.json 读取) ✅

**文件:**
- 新建:`config/loader.go`、`config/config.example.json`
- 测试:`config/loader_test.go`

**说明:** 不支持 env 插值、不支持优先级合并、不自动寻址默认路径。调用方必须显式传入路径。文件缺失字段由 `Defaults()` 填充。Unix 下 warn 权限宽于 0600。

- [x] **步骤 1:写失败测试** — 新建 `config/loader_test.go`:

```go
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
```

- [x] **步骤 2:运行,确认失败**

```bash
go test ./config/... -run Load
```
预期:`undefined: Load`。

- [x] **步骤 3:编写 `config/loader.go`**

```go
package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Load 从 config.json 读取并返回合并了默认值、校验通过的 Resolved。
//
// 合并规则:先把 Defaults() 作为基线写入 Root,再用 JSON 覆盖 —— 因为
// json.Unmarshal 只覆盖 JSON 中出现的字段,这样天然实现"缺失字段沿用默认值"。
//
// Unix 下若文件权限宽于 0600,打印 warn 日志(不阻塞)。
func Load(path string) (*Resolved, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat config %s: %w", path, err)
	}
	if runtime.GOOS != "windows" {
		if mode := info.Mode().Perm(); mode&0o077 != 0 {
			slog.Warn("config file permissions too open; apiKey is plaintext",
				"path", path, "mode", fmt.Sprintf("%04o", mode))
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	root := Defaults()
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	res := root.Resolve()
	if res.Agent.Workspace, err = expandTilde(res.Agent.Workspace); err != nil {
		return nil, err
	}
	if err := res.Validate(); err != nil {
		return nil, err
	}
	return &res, nil
}

func expandTilde(p string) (string, error) {
	if !strings.HasPrefix(p, "~") {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home: %w", err)
	}
	if p == "~" {
		return home, nil
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}
```

- [x] **步骤 4:创建 `config/config.example.json`**

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.srebot/workspace",
      "presetDir": "presets/default",
      "provider": "openai",
      "baseUrl": "https://api.openai.com/v1",
      "apiKey": "sk-REPLACE-ME",
      "model": "gpt-4o-mini",
      "temperature": 0.2,
      "timeoutSec": 120,
      "maxTokens": 8192,
      "contextWindowTokens": 65536,
      "disabledSkills": []
    }
  },
  "approval": {
    "yolo": false,
    "readOnlyAutoApprove": true,
    "whitelist": []
  },
  "logging": {
    "level": "info",
    "format": "text"
  },
  "memory": {
    "maxHistoryEntries": 1000
  }
}
```

- [x] **步骤 5:运行测试,确认通过**

```bash
go test ./config/... -v
```
预期:全部测试 PASS。

- [x] **步骤 6:提交**

```bash
git add config/
git commit -m "feat(config): Load from config.json with Resolve + defaults merge + tilde expansion"
```

## 任务 4-6:memory.Store 完整实现 ✅

**文件:**
- 新建:`internal/memory/store.go`
- 测试:`internal/memory/store_test.go`

**说明:** 该任务包含 ReadMemory / AppendHistory / 游标 / ReadUnprocessedHistory / Dream cursor / CompactHistory / RawArchive 全部功能,一次提交。测试共 10 个,全部用标准库 `testing`(不使用 testify)。

- [x] **步骤 1:写完整测试文件** — 新建 `internal/memory/store_test.go`:

```go
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
```

- [x] **步骤 2:运行,确认失败**

```bash
go test ./internal/memory/...
```
预期:`undefined: NewStore`。

- [x] **步骤 3:编写 `internal/memory/store.go`** — 包含 `HistoryEntry`、`Store` 接口和 `fsStore` 实现,覆盖 `ReadMemory` / `AppendHistory`(含 `nextCursor` 兜底)/ `ReadUnprocessedHistory` / `LastDreamCursor` / `SetLastDreamCursor` / `CompactHistory` / `RawArchive`。具体实现见仓库 `internal/memory/store.go`。

- [x] **步骤 4:运行测试,确认全部通过**

```bash
go test ./internal/memory/... -v
```
预期:10 个测试全部 PASS。

- [x] **步骤 5:提交**

```bash
git add internal/memory/
git commit -m "feat(memory): Store with MEMORY.md, history.jsonl, cursor, dream cursor, compact, raw archive"
```

---

## 任务 7:session — append + load ✅

**文件:**
- 新建:`internal/session/session.go`
- 测试:`internal/session/session_test.go`

- [x] **步骤 1:写失败测试** — 新建 `internal/session/session_test.go`:

```go
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
```

- [x] **步骤 2:运行,确认失败**

```bash
go test ./internal/session/...
```
预期:`undefined: Open` / `undefined: Message`。

- [x] **步骤 3:编写 `internal/session/session.go`** — 定义 `Message` 结构体、`Session` 接口、`fsSession` 实现(`Open`/`Append`/`Load`/`Close`),`Load` 容忍最后一行截断(`json.Unmarshal` 失败时 `continue`)。完整代码见仓库 `internal/session/session.go`。

- [x] **步骤 4:运行,确认通过**

```bash
go test ./internal/session/... -v
```
预期:4 个测试 PASS。

- [x] **步骤 5:提交**

```bash
git add internal/session/
git commit -m "feat(session): append-only session.jsonl + 崩溃恢复 Load"
```

---

## 任务 8:persona — 四 MD 加载器 ✅

**文件:**
- 新建:`internal/persona/loader.go`
- 测试:`internal/persona/loader_test.go`

- [x] **步骤 1:写失败测试** — 新建 `internal/persona/loader_test.go`:

```go
package persona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_AllFourPresent(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "SOUL.md", "soul-body")
	writeFile(t, ws, "AGENTS.md", "agents-body")
	writeFile(t, ws, "USER.md", "user-body")
	writeFile(t, ws, "TOOLS.md", "tools-body")

	p, err := NewFSLoader(ws).Load()
	if err != nil {
		t.Fatal(err)
	}
	if p.Soul != "soul-body" {
		t.Errorf("Soul = %q", p.Soul)
	}
	if p.Agents != "agents-body" {
		t.Errorf("Agents = %q", p.Agents)
	}
	if p.User != "user-body" {
		t.Errorf("User = %q", p.User)
	}
	if p.Tools != "tools-body" {
		t.Errorf("Tools = %q", p.Tools)
	}
}

func TestLoad_MissingFilesReturnEmpty(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "SOUL.md", "only-soul")

	p, err := NewFSLoader(ws).Load()
	if err != nil {
		t.Fatal(err)
	}
	if p.Soul != "only-soul" {
		t.Errorf("Soul = %q", p.Soul)
	}
	if p.Agents != "" || p.User != "" || p.Tools != "" {
		t.Errorf("others not empty: %+v", p)
	}
}

func TestBootstrap_ConcatenatesInOrder(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "AGENTS.md", "A")
	writeFile(t, ws, "SOUL.md", "S")
	writeFile(t, ws, "USER.md", "U")
	writeFile(t, ws, "TOOLS.md", "T")

	p, _ := NewFSLoader(ws).Load()
	boot := p.Bootstrap()
	// 顺序:AGENTS、SOUL、USER、TOOLS
	iA := strings.Index(boot, "## AGENTS.md")
	iS := strings.Index(boot, "## SOUL.md")
	iU := strings.Index(boot, "## USER.md")
	iT := strings.Index(boot, "## TOOLS.md")
	if !(iA < iS && iS < iU && iU < iT) {
		t.Errorf("expected order AGENTS<SOUL<USER<TOOLS, got %d %d %d %d", iA, iS, iU, iT)
	}
}

func TestBootstrap_SkipsEmptyFiles(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "SOUL.md", "has-content")

	p, _ := NewFSLoader(ws).Load()
	boot := p.Bootstrap()
	if !strings.Contains(boot, "## SOUL.md") {
		t.Error("missing SOUL section")
	}
	if strings.Contains(boot, "## AGENTS.md") {
		t.Error("should not contain AGENTS section")
	}
}
```

- [x] **步骤 2:运行,确认失败**

```bash
go test ./internal/persona/...
```
预期:`undefined: NewFSLoader`。

- [x] **步骤 3:编写 `internal/persona/loader.go`** — 定义 `Persona` 结构体(Soul/Agents/User/Tools)、`Loader` 接口、`fsLoader` 实现。`bootstrapOrder = [AGENTS, SOUL, USER, TOOLS]`。`Bootstrap()` 按序拼接存在的文件,空文件跳过。完整代码见仓库 `internal/persona/loader.go`。

- [x] **步骤 4:运行,确认通过**

```bash
go test ./internal/persona/... -v
```
预期:4 个测试 PASS。

- [x] **步骤 5:提交**

```bash
git add internal/persona/
git commit -m "feat(persona): 四 MD 加载器 + Bootstrap 拼接"
```

---

## 任务 9:覆盖率检查 + 阶段提交 ✅

- [x] **步骤 1:带 race + coverage 跑全量测试**

```bash
go test -race -cover ./...
```
预期:全部 PASS。覆盖率目标:
- `config/` ≥ 90%
- `internal/memory/` ≥ 80%
- `internal/session/` ≥ 80%
- `internal/persona/` ≥ 80%

任一模块不达标,补充缺失测试用例后再继续。

- [x] **步骤 2:安装 golangci-lint(若未安装)**

```bash
which golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

- [x] **步骤 3:Lint**

```bash
golangci-lint run ./...
```
预期:无 issue。若有报错按提示修复。

- [x] **步骤 4:打阶段里程碑 tag**

```bash
git tag -a v0.1.0-phase1 -m "Phase 1 foundation: config, memory.Store, session, persona"
```

---

## 超出第一阶段范围(下阶段再做)

**第二阶段:** `skills`、`approval`、`tools`(builtin + Registry)、`mcp`、`provider`(ChatModel 抽象 + openai 实现)。

**第三阶段:** `memory.Consolidator`、`internal/agent`(eino Graph)、`cmd/srebot` CLI + REPL、E2E 测试。
