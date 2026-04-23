# srebot Foundation(第一阶段)实施计划

> **给执行方的说明:** 必需子技能:使用 superpowers:subagent-driven-development(推荐)或 superpowers:executing-plans 按任务逐个实施本计划。步骤用 checkbox(`- [ ]`)语法追踪进度。

**目标:** 初始化 Go module,并实现四个基础的存储/IO 模块(`config`、`memory.Store`、`session`、`persona`),含完整单元测试。

**架构:** 纯 I/O / 解析层。不引入 LLM、不依赖 eino、无网络调用。每个模块独立、可测试,供后续阶段组装。

**技术栈:** Go 1.22+、`gopkg.in/yaml.v3`、Go 标准库(`log/slog`、`testing`)、`github.com/stretchr/testify`。

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
│   ├── config.go                       # 结构体 + Validate()
│   ├── loader.go                       # Load():flag > env > file > defaults + ${VAR} 插值
│   ├── config.example.yaml             # 用户样例
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

## 任务 1:Module 初始化

**文件:**
- 新建:`go.mod`、`.gitignore`

- [ ] **步骤 1:初始化 Go module**

执行(cwd=`/Users/fengjin/Desktop/GitHub/srebot`):
```bash
go mod init github.com/fengjinjin/srebot
```
预期:生成 `go.mod`,内容为 `module github.com/fengjinjin/srebot` 及 `go 1.22`。

- [ ] **步骤 2:添加核心依赖**

执行:
```bash
go get gopkg.in/yaml.v3@latest
go get github.com/stretchr/testify@latest
```
预期:生成 `go.sum`,`go.mod` 增加 `require` 段。

- [ ] **步骤 3:创建 `.gitignore`**

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

- [ ] **步骤 4:提交**

```bash
git add go.mod go.sum .gitignore
git commit -m "chore: init go module and gitignore"
```

---

## 任务 2:config 包 — 结构体与默认值

**文件:**
- 新建:`config/config.go`
- 测试:`config/config_test.go`

- [ ] **步骤 1:写失败测试** — 新建 `config/config_test.go`:

```go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaults_HasSaneValues(t *testing.T) {
	c := Defaults()
	assert.Equal(t, "https://api.openai.com/v1", c.Provider.BaseURL)
	assert.Equal(t, "gpt-4o-mini", c.Provider.Model)
	assert.Equal(t, 120, c.Provider.TimeoutSec)
	assert.Equal(t, 8192, c.Memory.ContextWindowTokens)
	assert.Equal(t, 1000, c.Memory.MaxHistoryEntries)
	assert.False(t, c.Approval.YOLO)
	assert.True(t, c.Approval.ReadOnlyAutoApprove)
	assert.Equal(t, "info", c.Logging.Level)
}

func TestValidate_RejectsEmptyProviderFields(t *testing.T) {
	c := Defaults()
	c.Provider.APIKey = ""
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider.api_key")
}

func TestValidate_RejectsZeroContextWindow(t *testing.T) {
	c := Defaults()
	c.Provider.APIKey = "sk-test"
	c.Memory.ContextWindowTokens = 0
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "memory.context_window_tokens")
}

func TestValidate_OKWhenMinimalFieldsSet(t *testing.T) {
	c := Defaults()
	c.Provider.APIKey = "sk-test"
	assert.NoError(t, c.Validate())
}
```

- [ ] **步骤 2:运行测试,确认失败**

```bash
go test ./config/...
```
预期:`undefined: Defaults` / `undefined: Config`。

- [ ] **步骤 3:编写 `config/config.go`**

```go
// Package config 定义并加载 srebot 运行时配置。
package config

import (
	"errors"
	"fmt"
	"strings"
)

// Config 是合并后的完整运行时配置。
type Config struct {
	Provider  ProviderConfig  `yaml:"provider"`
	Workspace string          `yaml:"workspace"`
	PresetDir string          `yaml:"preset_dir"`
	Memory    MemoryConfig    `yaml:"memory"`
	Approval  ApprovalConfig  `yaml:"approval"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// ProviderConfig 是 OpenAI-compatible endpoint 配置。
type ProviderConfig struct {
	BaseURL     string  `yaml:"base_url"`
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
	TimeoutSec  int     `yaml:"timeout_sec"`
}

// MemoryConfig 调整 memory 子系统参数。
type MemoryConfig struct {
	ContextWindowTokens int `yaml:"context_window_tokens"`
	MaxHistoryEntries   int `yaml:"max_history_entries"`
}

// ApprovalConfig 调整工具调用审批行为。
type ApprovalConfig struct {
	YOLO                bool     `yaml:"yolo"`
	ReadOnlyAutoApprove bool     `yaml:"read_only_auto_approve"`
	Whitelist           []string `yaml:"whitelist"`
}

// LoggingConfig 调整日志级别/格式。
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Defaults 返回带安全默认值的 Config。
func Defaults() Config {
	return Config{
		Provider: ProviderConfig{
			BaseURL:     "https://api.openai.com/v1",
			Model:       "gpt-4o-mini",
			Temperature: 0.2,
			TimeoutSec:  120,
		},
		Workspace: "~/.srebot/workspace",
		PresetDir: "presets/default",
		Memory: MemoryConfig{
			ContextWindowTokens: 8192,
			MaxHistoryEntries:   1000,
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
	}
}

// Validate 在必填字段缺失或越界时返回 error。
func (c *Config) Validate() error {
	var errs []string
	if strings.TrimSpace(c.Provider.APIKey) == "" {
		errs = append(errs, "provider.api_key is required")
	}
	if strings.TrimSpace(c.Provider.BaseURL) == "" {
		errs = append(errs, "provider.base_url is required")
	}
	if strings.TrimSpace(c.Provider.Model) == "" {
		errs = append(errs, "provider.model is required")
	}
	if c.Provider.TimeoutSec <= 0 {
		errs = append(errs, "provider.timeout_sec must be > 0")
	}
	if c.Memory.ContextWindowTokens <= 0 {
		errs = append(errs, "memory.context_window_tokens must be > 0")
	}
	if c.Memory.MaxHistoryEntries <= 0 {
		errs = append(errs, "memory.max_history_entries must be > 0")
	}
	if len(errs) > 0 {
		return fmt.Errorf("config invalid: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ErrMissingEnv 在 ${VAR} 插值找不到环境变量时返回。
var ErrMissingEnv = errors.New("missing environment variable")
```

- [ ] **步骤 4:运行测试,确认通过**

```bash
go test ./config/... -v
```
预期:4 个测试全部 PASS。

- [ ] **步骤 5:提交**

```bash
git add config/
git commit -m "feat(config): Config structs, Defaults, Validate"
```

---

## 任务 3:config 包 — Loader 与优先级合并

**文件:**
- 新建:`config/loader.go`、`config/config.example.yaml`
- 测试:`config/loader_test.go`

- [ ] **步骤 1:写失败测试** — 新建 `config/loader_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(body), 0o600))
	return p
}

func TestLoad_DefaultsOnly_FailsValidationWithoutAPIKey(t *testing.T) {
	_, err := Load(&Flags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider.api_key")
}

func TestLoad_EnvInterpolation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MY_KEY", "sk-abc")
	p := writeFile(t, dir, "c.yaml", `
provider:
  api_key: ${MY_KEY}
  base_url: https://api.deepseek.com/v1
  model: deepseek-chat
  timeout_sec: 60
`)
	c, err := Load(&Flags{ConfigPath: p})
	require.NoError(t, err)
	assert.Equal(t, "sk-abc", c.Provider.APIKey)
	assert.Equal(t, "https://api.deepseek.com/v1", c.Provider.BaseURL)
}

func TestLoad_MissingEnvVarFailsFast(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.yaml", `
provider:
  api_key: ${DEFINITELY_NOT_SET_XYZ}
`)
	_, err := Load(&Flags{ConfigPath: p})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingEnv)
}

func TestLoad_PriorityFlagOverridesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("K", "from-env")
	p := writeFile(t, dir, "c.yaml", `
provider:
  api_key: ${K}
  model: file-model
`)
	c, err := Load(&Flags{ConfigPath: p, Model: "flag-model"})
	require.NoError(t, err)
	assert.Equal(t, "flag-model", c.Provider.Model)
	assert.Equal(t, "from-env", c.Provider.APIKey)
}

func TestLoad_BadYAMLReturnsError(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.yaml", "provider: [this is: not valid")
	_, err := Load(&Flags{ConfigPath: p})
	require.Error(t, err)
}
```

- [ ] **步骤 2:运行测试,确认失败**

```bash
go test ./config/... -run Load
```
预期:`undefined: Load` / `undefined: Flags`。

- [ ] **步骤 3:编写 `config/loader.go`**

```go
package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Flags 是 CLI 传入的覆盖项。空字符串视为"未设置"。
type Flags struct {
	ConfigPath string
	Workspace  string
	PresetDir  string
	Model      string
	BaseURL    string
	YOLO       bool
	Verbose    bool
	Resume     bool
}

// Load 按优先级合并配置:
//   flag > env (SREBOT_*) > file > defaults。
//
// ${VAR} 插值在解析 YAML 前作用于原文;任何缺失变量会 fail-fast 返回 ErrMissingEnv。
func Load(flags *Flags) (*Config, error) {
	c := Defaults()

	// 1. file
	if flags != nil && flags.ConfigPath != "" {
		if err := mergeFile(&c, flags.ConfigPath); err != nil {
			return nil, err
		}
	}

	// 2. env(选定的覆盖)
	if v := os.Getenv("SREBOT_API_KEY"); v != "" {
		c.Provider.APIKey = v
	}
	if v := os.Getenv("SREBOT_MODEL"); v != "" {
		c.Provider.Model = v
	}
	if v := os.Getenv("SREBOT_BASE_URL"); v != "" {
		c.Provider.BaseURL = v
	}
	if v := os.Getenv("SREBOT_WORKSPACE"); v != "" {
		c.Workspace = v
	}

	// 3. flag
	if flags != nil {
		if flags.Model != "" {
			c.Provider.Model = flags.Model
		}
		if flags.BaseURL != "" {
			c.Provider.BaseURL = flags.BaseURL
		}
		if flags.Workspace != "" {
			c.Workspace = flags.Workspace
		}
		if flags.PresetDir != "" {
			c.PresetDir = flags.PresetDir
		}
		if flags.YOLO {
			c.Approval.YOLO = true
		}
		if flags.Verbose {
			c.Logging.Level = "debug"
		}
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func mergeFile(c *Config, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %s: %w", path, err)
	}
	interpolated, err := interpolateEnv(string(raw))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal([]byte(interpolated), c); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	return nil
}

var envRefRe = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

func interpolateEnv(s string) (string, error) {
	var firstErr error
	out := envRefRe.ReplaceAllStringFunc(s, func(m string) string {
		name := envRefRe.FindStringSubmatch(m)[1]
		v, ok := os.LookupEnv(name)
		if !ok {
			if firstErr == nil {
				firstErr = fmt.Errorf("%w: %s", ErrMissingEnv, name)
			}
			return m
		}
		return v
	})
	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}
```

- [ ] **步骤 4:创建 `config/config.example.yaml`**

```yaml
# srebot 配置文件样例。复制到 ~/.srebot/config.yaml 或使用 --config 指定。
# ${VAR} 占位符从环境变量解析;缺失变量会在启动时 fail-fast。

provider:
  base_url: https://api.openai.com/v1
  api_key: ${OPENAI_API_KEY}
  model: gpt-4o-mini
  temperature: 0.2
  timeout_sec: 120

workspace: ~/.srebot/workspace
preset_dir: presets/default

memory:
  context_window_tokens: 8192
  max_history_entries: 1000

approval:
  yolo: false
  read_only_auto_approve: true
  whitelist: []

logging:
  level: info     # debug|info|warn|error
  format: text    # text|json
```

- [ ] **步骤 5:运行测试,确认通过**

```bash
go test ./config/... -v
```
预期:全部测试 PASS。

- [ ] **步骤 6:提交**

```bash
git add config/
git commit -m "feat(config): Load with priority flag>env>file>defaults + \${VAR} interpolation"
```

---

## 任务 4:memory.Store — ReadMemory

**文件:**
- 新建:`internal/memory/store.go`
- 测试:`internal/memory/store_test.go`

- [ ] **步骤 1:写失败测试** — 新建 `internal/memory/store_test.go`:

```go
package memory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore_CreatesMemoryDir(t *testing.T) {
	ws := t.TempDir()
	s, err := NewStore(ws)
	require.NoError(t, err)
	_ = s
	_, err = os.Stat(filepath.Join(ws, "memory"))
	assert.NoError(t, err)
}

func TestReadMemory_MissingFileReturnsEmpty(t *testing.T) {
	s, err := NewStore(t.TempDir())
	require.NoError(t, err)
	body, isTemplate, err := s.ReadMemory()
	require.NoError(t, err)
	assert.Empty(t, body)
	assert.False(t, isTemplate)
}

func TestReadMemory_ExistingFileReturnsContent(t *testing.T) {
	ws := t.TempDir()
	s, err := NewStore(ws)
	require.NoError(t, err)
	path := filepath.Join(ws, "memory", "MEMORY.md")
	require.NoError(t, os.WriteFile(path, []byte("# Facts\n- foo\n"), 0o600))

	body, isTemplate, err := s.ReadMemory()
	require.NoError(t, err)
	assert.Contains(t, body, "foo")
	assert.False(t, isTemplate)
}
```

- [ ] **步骤 2:运行,确认失败**

```bash
go test ./internal/memory/...
```
预期:`undefined: NewStore`。

- [ ] **步骤 3:编写 `internal/memory/store.go`**

```go
// Package memory 提供 MEMORY.md 长期事实存储、history.jsonl 压缩条目 append-only 日志
// 以及单调自增的游标管理。
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

// NewStore 在 ``<workspace>/memory/`` 下创建(或打开)Store。
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

// (其余方法在后续任务补充)

// --- 占位引用(待后续任务接入时移除) ---
var _ = io.EOF
var _ = json.Marshal
var _ = strconv.Itoa
var _ = time.Now
```

- [ ] **步骤 4:运行,确认通过**

```bash
go test ./internal/memory/... -v
```
预期:3 个测试 PASS。

- [ ] **步骤 5:提交**

```bash
git add internal/memory/
git commit -m "feat(memory): Store 脚手架 + ReadMemory"
```

---

## 任务 5:memory.Store — AppendHistory 与游标

**文件:**
- 修改:`internal/memory/store.go`
- 修改:`internal/memory/store_test.go`

- [ ] **步骤 1:追加失败测试** — 向 `internal/memory/store_test.go` 尾部追加:

```go
func TestAppendHistory_CursorMonotonic(t *testing.T) {
	s, err := NewStore(t.TempDir())
	require.NoError(t, err)

	c1, err := s.AppendHistory("first")
	require.NoError(t, err)
	c2, err := s.AppendHistory("second")
	require.NoError(t, err)
	c3, err := s.AppendHistory("third")
	require.NoError(t, err)

	assert.Equal(t, 1, c1)
	assert.Equal(t, 2, c2)
	assert.Equal(t, 3, c3)
}

func TestAppendHistory_PersistsCursor(t *testing.T) {
	ws := t.TempDir()
	s1, _ := NewStore(ws)
	_, _ = s1.AppendHistory("a")
	_, _ = s1.AppendHistory("b")

	s2, _ := NewStore(ws) // 重新打开
	c, err := s2.AppendHistory("c")
	require.NoError(t, err)
	assert.Equal(t, 3, c)
}

func TestReadUnprocessedHistory_FiltersByCursor(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	_, _ = s.AppendHistory("one")
	_, _ = s.AppendHistory("two")
	_, _ = s.AppendHistory("three")

	entries, err := s.ReadUnprocessedHistory(1)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, 2, entries[0].Cursor)
	assert.Equal(t, "two", entries[0].Content)
	assert.Equal(t, 3, entries[1].Cursor)
}

func TestDreamCursor_ZeroByDefault(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	assert.Equal(t, 0, s.LastDreamCursor())
}

func TestDreamCursor_SetAndRead(t *testing.T) {
	ws := t.TempDir()
	s, _ := NewStore(ws)
	require.NoError(t, s.SetLastDreamCursor(42))

	s2, _ := NewStore(ws)
	assert.Equal(t, 42, s2.LastDreamCursor())
}
```

- [ ] **步骤 2:运行,确认失败**

```bash
go test ./internal/memory/...
```
预期:`*fsStore does not implement Store (missing method AppendHistory)`。

- [ ] **步骤 3:扩展 `internal/memory/store.go`** — 用以下代码替换文件末尾的"占位引用"段:

```go
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
			// 容忍单条破损,跳到下一行
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
```

- [ ] **步骤 4:运行测试,确认通过**

```bash
go test ./internal/memory/... -v
```
预期:所有 memory 测试 PASS。

- [ ] **步骤 5:提交**

```bash
git add internal/memory/
git commit -m "feat(memory): AppendHistory, cursor, ReadUnprocessedHistory, dream cursor, compact, RawArchive"
```

---

## 任务 6:memory.Store — CompactHistory 与 RawArchive 测试

**文件:**
- 修改:`internal/memory/store_test.go`

- [ ] **步骤 1:追加测试** 到 `internal/memory/store_test.go`:

```go
func TestCompactHistory_KeepsNewest(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	for i := 0; i < 5; i++ {
		_, _ = s.AppendHistory("entry")
	}
	require.NoError(t, s.CompactHistory(2))

	entries, err := s.ReadUnprocessedHistory(0)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, 4, entries[0].Cursor)
	assert.Equal(t, 5, entries[1].Cursor)
}

func TestRawArchive_PrefixesMarker(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	require.NoError(t, s.RawArchive("dumped messages"))
	entries, _ := s.ReadUnprocessedHistory(0)
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0].Content, "[RAW]")
	assert.Contains(t, entries[0].Content, "dumped messages")
}
```

- [ ] **步骤 2:运行,确认通过**

```bash
go test ./internal/memory/... -v
```
预期:全部 PASS(实现已在任务 5 完成)。

- [ ] **步骤 3:提交**

```bash
git add internal/memory/
git commit -m "test(memory): CompactHistory 与 RawArchive 覆盖"
```

---

## 任务 7:session — append + load

**文件:**
- 新建:`internal/session/session.go`
- 测试:`internal/session/session_test.go`

- [ ] **步骤 1:写失败测试** — 新建 `internal/session/session_test.go`:

```go
package session

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_NewIDGeneratesUniqueFile(t *testing.T) {
	ws := t.TempDir()
	s1, err := Open(ws, "")
	require.NoError(t, err)
	defer s1.Close()
	s2, err := Open(ws, "")
	require.NoError(t, err)
	defer s2.Close()
	assert.NotEqual(t, s1.ID(), s2.ID())
}

func TestAppendAndLoad_RoundTrip(t *testing.T) {
	ws := t.TempDir()
	s, err := Open(ws, "test-sid")
	require.NoError(t, err)

	require.NoError(t, s.Append(Message{Role: "user", Content: "hello"}))
	require.NoError(t, s.Append(Message{Role: "assistant", Content: "world"}))
	require.NoError(t, s.Close())

	s2, err := Open(ws, "test-sid")
	require.NoError(t, err)
	msgs, err := s2.Load()
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, "user", msgs[0].Role)
	assert.Equal(t, "hello", msgs[0].Content)
	assert.Equal(t, "assistant", msgs[1].Role)
}

func TestLoad_MissingSessionReturnsEmpty(t *testing.T) {
	s, err := Open(t.TempDir(), "never-written")
	require.NoError(t, err)
	msgs, err := s.Load()
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestLoad_TruncatedLastLineDropped(t *testing.T) {
	ws := t.TempDir()
	s, err := Open(ws, "sid")
	require.NoError(t, err)
	require.NoError(t, s.Append(Message{Role: "user", Content: "ok"}))
	require.NoError(t, s.Close())

	// 模拟部分写入(断电/崩溃)
	path := filepath.Join(ws, "session", "sid.jsonl")
	appendRaw(t, path, `{"role":"assistant","content":"partial`)

	s2, err := Open(ws, "sid")
	require.NoError(t, err)
	msgs, err := s2.Load()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "ok", msgs[0].Content)
}
```

再在同包下新建 `internal/session/testhelpers_test.go`:

```go
package session

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func appendRaw(t *testing.T, path, data string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(data)
	require.NoError(t, err)
}
```

- [ ] **步骤 2:运行,确认失败**

```bash
go test ./internal/session/...
```
预期:`undefined: Open` / `undefined: Message`。

- [ ] **步骤 3:编写 `internal/session/session.go`**

```go
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
			// 跳过畸形/部分写入行(最后一行被截断的情况)
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
```

- [ ] **步骤 4:运行,确认通过**

```bash
go test ./internal/session/... -v
```
预期:4 个测试 PASS。

- [ ] **步骤 5:提交**

```bash
git add internal/session/
git commit -m "feat(session): append-only session.jsonl + 崩溃恢复 Load"
```

---

## 任务 8:persona — 四 MD 加载器

**文件:**
- 新建:`internal/persona/loader.go`
- 测试:`internal/persona/loader_test.go`

- [ ] **步骤 1:写失败测试** — 新建 `internal/persona/loader_test.go`:

```go
package persona

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600))
}

func TestLoad_AllFourPresent(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "SOUL.md", "soul-body")
	writeFile(t, ws, "AGENTS.md", "agents-body")
	writeFile(t, ws, "USER.md", "user-body")
	writeFile(t, ws, "TOOLS.md", "tools-body")

	p, err := NewFSLoader(ws).Load()
	require.NoError(t, err)
	assert.Equal(t, "soul-body", p.Soul)
	assert.Equal(t, "agents-body", p.Agents)
	assert.Equal(t, "user-body", p.User)
	assert.Equal(t, "tools-body", p.Tools)
}

func TestLoad_MissingFilesReturnEmpty(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "SOUL.md", "only-soul")

	p, err := NewFSLoader(ws).Load()
	require.NoError(t, err)
	assert.Equal(t, "only-soul", p.Soul)
	assert.Empty(t, p.Agents)
	assert.Empty(t, p.User)
	assert.Empty(t, p.Tools)
}

func TestBootstrap_ConcatenatesInOrder(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "AGENTS.md", "A")
	writeFile(t, ws, "SOUL.md", "S")
	writeFile(t, ws, "USER.md", "U")
	writeFile(t, ws, "TOOLS.md", "T")

	p, _ := NewFSLoader(ws).Load()
	boot := p.Bootstrap()
	// 顺序:AGENTS、SOUL、USER、TOOLS(对齐 nanobot BOOTSTRAP_FILES)
	iA := indexOf(boot, "## AGENTS.md")
	iS := indexOf(boot, "## SOUL.md")
	iU := indexOf(boot, "## USER.md")
	iT := indexOf(boot, "## TOOLS.md")
	assert.True(t, iA < iS && iS < iU && iU < iT,
		"expected order AGENTS<SOUL<USER<TOOLS, got %d %d %d %d", iA, iS, iU, iT)
}

func TestBootstrap_SkipsEmptyFiles(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "SOUL.md", "has-content")
	// 其他缺失

	p, _ := NewFSLoader(ws).Load()
	boot := p.Bootstrap()
	assert.Contains(t, boot, "## SOUL.md")
	assert.NotContains(t, boot, "## AGENTS.md")
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

- [ ] **步骤 2:运行,确认失败**

```bash
go test ./internal/persona/...
```
预期:`undefined: NewFSLoader`。

- [ ] **步骤 3:编写 `internal/persona/loader.go`**

```go
// Package persona 加载四个定义 srebot 人格的 Markdown 文件:
// SOUL(个性)、AGENTS(行为指令)、USER(用户信息)、TOOLS(工具手册)。
// 每轮对话都会重新读取这些文件以组装 system prompt,以便用户的改动立即生效。
package persona

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Persona 承载四个 bootstrap 文件的内容。空字符串 = 文件不存在。
type Persona struct {
	Soul   string
	Agents string
	User   string
	Tools  string
}

// Loader 是 persona 读取抽象。
type Loader interface {
	Load() (*Persona, error)
}

// bootstrapOrder 与 nanobot 的 BOOTSTRAP_FILES 对齐:
// AGENTS 优先(指令),随后 SOUL(人格)、USER、TOOLS。
var bootstrapOrder = []string{"AGENTS.md", "SOUL.md", "USER.md", "TOOLS.md"}

type fsLoader struct {
	dir string
}

// NewFSLoader 返回从给定目录读取 MD 文件的 Loader。
func NewFSLoader(workspaceDir string) Loader {
	return &fsLoader{dir: workspaceDir}
}

// Load 读取全部四个文件;缺失文件映射为空字符串而非 error。
func (l *fsLoader) Load() (*Persona, error) {
	p := &Persona{}
	m := map[string]*string{
		"SOUL.md":   &p.Soul,
		"AGENTS.md": &p.Agents,
		"USER.md":   &p.User,
		"TOOLS.md":  &p.Tools,
	}
	for name, dst := range m {
		b, err := os.ReadFile(filepath.Join(l.dir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		*dst = string(b)
	}
	return p, nil
}

// Bootstrap 将存在的文件按 bootstrapOrder 顺序拼成 Markdown section。
// 空文件被跳过;全部缺失时返回 ""。
func (p *Persona) Bootstrap() string {
	get := map[string]string{
		"AGENTS.md": p.Agents,
		"SOUL.md":   p.Soul,
		"USER.md":   p.User,
		"TOOLS.md":  p.Tools,
	}
	var parts []string
	for _, name := range bootstrapOrder {
		if body := get[name]; body != "" {
			parts = append(parts, fmt.Sprintf("## %s\n\n%s", name, body))
		}
	}
	return strings.Join(parts, "\n\n")
}
```

- [ ] **步骤 4:运行,确认通过**

```bash
go test ./internal/persona/... -v
```
预期:4 个测试 PASS。

- [ ] **步骤 5:提交**

```bash
git add internal/persona/
git commit -m "feat(persona): 四 MD 加载器 + Bootstrap 拼接"
```

---

## 任务 9:覆盖率检查 + 阶段提交

- [ ] **步骤 1:带 race + coverage 跑全量测试**

```bash
go test -race -cover ./...
```
预期:全部 PASS。覆盖率目标:
- `config/` ≥ 90%
- `internal/memory/` ≥ 80%
- `internal/session/` ≥ 80%
- `internal/persona/` ≥ 80%

任一模块不达标,补充缺失测试用例后再继续。

- [ ] **步骤 2:安装 golangci-lint(若未安装)**

```bash
which golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

- [ ] **步骤 3:Lint**

```bash
golangci-lint run ./...
```
预期:无 issue。若有报错按提示修复。

- [ ] **步骤 4:打阶段里程碑 tag**

```bash
git tag -a v0.1.0-phase1 -m "Phase 1 foundation: config, memory.Store, session, persona"
```

---

## 超出第一阶段范围(下阶段再做)

**第二阶段:** `skills`、`approval`、`tools`(builtin + Registry)、`mcp`、`provider`(ChatModel 抽象 + openai 实现)。

**第三阶段:** `memory.Consolidator`、`internal/agent`(eino Graph)、`cmd/srebot` CLI + REPL、E2E 测试。
