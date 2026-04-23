# srebot Foundation (Phase 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scaffold the Go module and implement the four foundational storage/IO modules (`config`, `memory.Store`, `session`, `persona`) with full unit tests.

**Architecture:** Pure I/O / parsing layer. No LLM, no eino, no network. Each module is an independent, testable unit that will be composed by later phases.

**Tech Stack:** Go 1.22+, `gopkg.in/yaml.v3`, Go standard library (`log/slog`, `testing`), `github.com/stretchr/testify`.

**Scope:** Phase 1 of 3. Phase 2 (skills + approval + tools + mcp + provider) and Phase 3 (Consolidator + agent + CLI + E2E) follow.

**Spec reference:** `docs/superpowers/specs/2026-04-23-srebot-mvp-design.md` sections §5.1, §5.3, §5.5 (Store only), §5.6.

---

## File Structure

**New files (create in this phase):**

```
srebot/
├── go.mod                              # module init
├── go.sum                              # deps lock
├── config/
│   ├── config.go                       # structs + Validate()
│   ├── loader.go                       # Load(): flag > env > file > defaults + ${VAR} interpolation
│   ├── config.example.yaml             # user-facing sample
│   ├── config_test.go
│   └── loader_test.go
├── internal/
│   ├── memory/
│   │   ├── store.go                    # MemoryStore: MEMORY.md + history.jsonl + cursors
│   │   └── store_test.go
│   ├── session/
│   │   ├── session.go                  # session.jsonl append/load
│   │   └── session_test.go
│   └── persona/
│       ├── loader.go                   # 4 MD loader
│       └── loader_test.go
└── .gitignore
```

---

## Task 1: Module Scaffolding

**Files:**
- Create: `go.mod`, `.gitignore`

- [ ] **Step 1: Initialize Go module**

Run (cwd=`/Users/fengjin/Desktop/GitHub/srebot`):
```bash
go mod init github.com/fengjinjin/srebot
```
Expected: creates `go.mod` with `module github.com/fengjinjin/srebot` and `go 1.22`.

- [ ] **Step 2: Add core deps**

Run:
```bash
go get gopkg.in/yaml.v3@latest
go get github.com/stretchr/testify@latest
```
Expected: `go.sum` created, `go.mod` gains `require` block.

- [ ] **Step 3: Create `.gitignore`**

Create `/Users/fengjin/Desktop/GitHub/srebot/.gitignore`:
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

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum .gitignore
git commit -m "chore: init go module and gitignore"
```

---

## Task 2: config package — structs and defaults

**Files:**
- Create: `config/config.go`
- Test: `config/config_test.go`

- [ ] **Step 1: Write failing test** — create `config/config_test.go`:

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

- [ ] **Step 2: Run test, expect failure**

```bash
go test ./config/...
```
Expected: `undefined: Defaults` / `undefined: Config`.

- [ ] **Step 3: Write `config/config.go`**

```go
// Package config defines and loads srebot runtime configuration.
package config

import (
	"errors"
	"fmt"
	"strings"
)

// Config is the fully-resolved runtime configuration.
type Config struct {
	Provider  ProviderConfig  `yaml:"provider"`
	Workspace string          `yaml:"workspace"`
	PresetDir string          `yaml:"preset_dir"`
	Memory    MemoryConfig    `yaml:"memory"`
	Approval  ApprovalConfig  `yaml:"approval"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// ProviderConfig is the OpenAI-compatible endpoint config.
type ProviderConfig struct {
	BaseURL    string  `yaml:"base_url"`
	APIKey     string  `yaml:"api_key"`
	Model      string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
	TimeoutSec int     `yaml:"timeout_sec"`
}

// MemoryConfig tunes the memory subsystem.
type MemoryConfig struct {
	ContextWindowTokens int `yaml:"context_window_tokens"`
	MaxHistoryEntries   int `yaml:"max_history_entries"`
}

// ApprovalConfig tunes tool-call approval behavior.
type ApprovalConfig struct {
	YOLO                bool     `yaml:"yolo"`
	ReadOnlyAutoApprove bool     `yaml:"read_only_auto_approve"`
	Whitelist           []string `yaml:"whitelist"`
}

// LoggingConfig tunes logger level/format.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Defaults returns a Config populated with safe defaults.
func Defaults() Config {
	return Config{
		Provider: ProviderConfig{
			BaseURL:    "https://api.openai.com/v1",
			Model:      "gpt-4o-mini",
			Temperature: 0.2,
			TimeoutSec: 120,
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

// Validate returns an error if required fields are missing or out of range.
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

// ErrMissingEnv is returned when ${VAR} interpolation cannot resolve.
var ErrMissingEnv = errors.New("missing environment variable")
```

- [ ] **Step 4: Run test, expect pass**

```bash
go test ./config/... -v
```
Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add config/
git commit -m "feat(config): Config structs, Defaults, Validate"
```

---

## Task 3: config package — Loader with priority merge

**Files:**
- Create: `config/loader.go`, `config/config.example.yaml`
- Test: `config/loader_test.go`

- [ ] **Step 1: Write failing test** — create `config/loader_test.go`:

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

- [ ] **Step 2: Run test, expect failure**

```bash
go test ./config/... -run Load
```
Expected: `undefined: Load` / `undefined: Flags`.

- [ ] **Step 3: Write `config/loader.go`**

```go
package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Flags is the CLI-provided override set. Empty strings mean "not set".
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

// Load resolves the config with priority:
//   flag > env (SREBOT_*) > file > defaults.
//
// ${VAR} interpolation runs on the raw YAML before parsing; any missing
// variable fails fast with ErrMissingEnv.
func Load(flags *Flags) (*Config, error) {
	c := Defaults()

	// 1. file
	if flags != nil && flags.ConfigPath != "" {
		if err := mergeFile(&c, flags.ConfigPath); err != nil {
			return nil, err
		}
	}

	// 2. env (selected overrides)
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

- [ ] **Step 4: Create `config/config.example.yaml`**

```yaml
# srebot configuration. Copy to ~/.srebot/config.yaml or pass --config.
# ${VAR} placeholders are resolved from environment; missing vars fail fast.

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

- [ ] **Step 5: Run tests, expect pass**

```bash
go test ./config/... -v
```
Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add config/
git commit -m "feat(config): Load with priority flag>env>file>defaults + \${VAR} interpolation"
```

---

## Task 4: memory.Store — MEMORY.md read

**Files:**
- Create: `internal/memory/store.go`
- Test: `internal/memory/store_test.go`

- [ ] **Step 1: Write failing test** — create `internal/memory/store_test.go`:

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

- [ ] **Step 2: Run, expect failure**

```bash
go test ./internal/memory/...
```
Expected: `undefined: NewStore`.

- [ ] **Step 3: Write `internal/memory/store.go`**

```go
// Package memory provides the MEMORY.md long-term facts store and the
// history.jsonl append-only compressed-entry log with monotonic cursors.
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

// HistoryEntry is a single compressed memory record appended by the Consolidator.
type HistoryEntry struct {
	Cursor    int    `json:"cursor"`
	Timestamp string `json:"timestamp"`
	Content   string `json:"content"`
}

// Store is the pure file-IO layer of the memory subsystem.
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
	workspace    string
	memoryDir    string
	memoryFile   string
	historyFile  string
	cursorFile   string
	dreamCursor  string
}

// NewStore creates (or opens) a Store backed by ``<workspace>/memory/``.
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

// ReadMemory returns the MEMORY.md body. isTemplate is reserved for a future
// "is this the unmodified template" check — MVP always returns false.
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

// (other methods added in subsequent tasks)

// --- unused helpers (silence linter until later tasks wire them in) ---
var _ = io.EOF
var _ = json.Marshal
var _ = strconv.Itoa
var _ = time.Now
```

- [ ] **Step 4: Run, expect pass**

```bash
go test ./internal/memory/... -v
```
Expected: 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/memory/
git commit -m "feat(memory): Store scaffold + ReadMemory"
```

---

## Task 5: memory.Store — AppendHistory + cursor

**Files:**
- Modify: `internal/memory/store.go`
- Modify: `internal/memory/store_test.go`

- [ ] **Step 1: Append failing tests** — append to `internal/memory/store_test.go`:

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

	s2, _ := NewStore(ws) // reopen
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

- [ ] **Step 2: Run, expect failure**

```bash
go test ./internal/memory/...
```
Expected: `*fsStore does not implement Store (missing method AppendHistory)`.

- [ ] **Step 3: Extend `internal/memory/store.go`** — replace the "unused helpers" stub block at the end with:

```go
// AppendHistory appends entry to history.jsonl and returns its new cursor.
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
	// Fallback: scan file for max cursor.
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

// ReadUnprocessedHistory returns all entries with Cursor > sinceCursor.
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
			// tolerate a single malformed line by skipping to newline
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// LastDreamCursor returns the last dream watermark (0 if never set).
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

// SetLastDreamCursor persists the dream watermark.
func (s *fsStore) SetLastDreamCursor(c int) error {
	return os.WriteFile(s.dreamCursor, []byte(strconv.Itoa(c)), 0o644)
}

// CompactHistory truncates history.jsonl to the newest maxEntries records.
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

// RawArchive appends a raw (unsummarized) record with a [RAW] marker as a
// fallback when Consolidator LLM calls fail.
func (s *fsStore) RawArchive(raw string) error {
	_, err := s.AppendHistory("[RAW] " + raw)
	return err
}
```

- [ ] **Step 4: Run tests, expect pass**

```bash
go test ./internal/memory/... -v
```
Expected: all memory tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/memory/
git commit -m "feat(memory): AppendHistory, cursor, ReadUnprocessedHistory, dream cursor, compact, RawArchive"
```

---

## Task 6: memory.Store — CompactHistory + RawArchive tests

**Files:**
- Modify: `internal/memory/store_test.go`

- [ ] **Step 1: Append tests** to `internal/memory/store_test.go`:

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

- [ ] **Step 2: Run, expect pass**

```bash
go test ./internal/memory/... -v
```
Expected: all PASS (implementation already present from Task 5).

- [ ] **Step 3: Commit**

```bash
git add internal/memory/
git commit -m "test(memory): CompactHistory and RawArchive coverage"
```

---

## Task 7: session — append + load

**Files:**
- Create: `internal/session/session.go`
- Test: `internal/session/session_test.go`

- [ ] **Step 1: Write failing test** — create `internal/session/session_test.go`:

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

	// simulate a partial write
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

Add helper at the bottom of the same file:

```go
func appendRaw(t *testing.T, path, data string) {
	t.Helper()
	f, err := osOpenFileAppend(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(data)
	require.NoError(t, err)
}
```

...then create `internal/session/testhelpers_test.go`:

```go
package session

import "os"

func osOpenFileAppend(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
}
```

- [ ] **Step 2: Run, expect failure**

```bash
go test ./internal/session/...
```
Expected: `undefined: Open` / `undefined: Message`.

- [ ] **Step 3: Write `internal/session/session.go`**

```go
// Package session provides append-only storage for raw agent message history
// (session.jsonl).  It is the source-of-truth log (unlike memory.history.jsonl
// which holds compressed summaries).
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

// Message is the minimal shape persisted per turn.  The agent layer may
// enrich this later by embedding richer fields; forward-compat is by JSON.
type Message struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
	Timestamp  string          `json:"timestamp,omitempty"`
}

// Session persists and restores an ordered message stream under a session id.
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

// Open returns a Session for the given id; if id is "", a new random id is
// generated.  The session.jsonl file is created on first Append if absent.
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

// Load reads all complete JSON lines. A truncated trailing line is silently
// dropped (crash-recovery behavior).
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
			// skip malformed / partial line (last-line truncation case)
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

- [ ] **Step 4: Run, expect pass**

```bash
go test ./internal/session/... -v
```
Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/session/
git commit -m "feat(session): append-only session.jsonl with crash-recovery Load"
```

---

## Task 8: persona — 4 MD loader

**Files:**
- Create: `internal/persona/loader.go`
- Test: `internal/persona/loader_test.go`

- [ ] **Step 1: Write failing test** — create `internal/persona/loader_test.go`:

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
	// order: AGENTS, SOUL, USER, TOOLS  (matches nanobot BOOTSTRAP_FILES)
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
	// others missing

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

- [ ] **Step 2: Run, expect failure**

```bash
go test ./internal/persona/...
```
Expected: `undefined: NewFSLoader`.

- [ ] **Step 3: Write `internal/persona/loader.go`**

```go
// Package persona loads the four Markdown files that define a srebot agent's
// identity (SOUL), instructions (AGENTS), user profile (USER), and tool
// guidance (TOOLS).  Every turn rebuilds the system prompt from these files
// so edits are picked up immediately.
package persona

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Persona holds the four bootstrap file contents.  Empty string = file absent.
type Persona struct {
	Soul   string
	Agents string
	User   string
	Tools  string
}

// Loader is the persona reader abstraction.
type Loader interface {
	Load() (*Persona, error)
}

// BootstrapOrder matches nanobot's BOOTSTRAP_FILES ordering:
// AGENTS first (instructions), then SOUL (personality), USER, TOOLS.
var bootstrapOrder = []string{"AGENTS.md", "SOUL.md", "USER.md", "TOOLS.md"}

type fsLoader struct {
	dir string
}

// NewFSLoader returns a Loader reading MD files from the given directory.
func NewFSLoader(workspaceDir string) Loader {
	return &fsLoader{dir: workspaceDir}
}

// Load reads all four files; missing files map to empty strings, not errors.
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

// Bootstrap concatenates all present files as Markdown sections, ordered
// per bootstrapOrder.  Empty files are skipped.  Returns "" when no files
// are present.
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

- [ ] **Step 4: Run, expect pass**

```bash
go test ./internal/persona/... -v
```
Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/persona/
git commit -m "feat(persona): 4 MD loader + Bootstrap concatenation"
```

---

## Task 9: Coverage check + phase commit

- [ ] **Step 1: Run full test suite with race + coverage**

```bash
go test -race -cover ./...
```
Expected: all PASS. Coverage targets:
- `config/` ≥ 90%
- `internal/memory/` ≥ 80%
- `internal/session/` ≥ 80%
- `internal/persona/` ≥ 80%

If any module falls below, add the missing test case before continuing.

- [ ] **Step 2: Install golangci-lint if missing**

```bash
which golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

- [ ] **Step 3: Lint**

```bash
golangci-lint run ./...
```
Expected: no issues. Fix any reported.

- [ ] **Step 4: Tag phase milestone**

```bash
git tag -a v0.1.0-phase1 -m "Phase 1 foundation: config, memory.Store, session, persona"
```

---

## Out of Phase 1 scope (next phases)

**Phase 2:** `skills`, `approval`, `tools` (builtin + Registry), `mcp`, `provider` (ChatModel abstraction + openai impl).

**Phase 3:** `memory.Consolidator`, `internal/agent` (eino Graph), `cmd/srebot` CLI + REPL, E2E tests.
