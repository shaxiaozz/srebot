# srebot MVP 设计文档

- **日期:** 2026-04-23
- **状态:** Draft,待实现
- **目标版本:** v0.1 (MVP)

## 1. 背景与定位

srebot 是基于 [eino](https://github.com/cloudwego/eino) 框架,复刻 [nanobot](https://github.com/HKUDS/nanobot) 核心设计理念的 Go 版超轻量 AI 代理,专注 SRE 运维场景。

**定位:** 框架级复刻 + SRE 预设。核心 runtime 通用,SRE 只是内置的一套 `SOUL/AGENTS/USER/TOOLS.md` + skills + MCP 预设。未来可扩展 DBA、DevOps 等其他发行版。

**核心哲学:** 四个 Markdown 文件定义一个 Agent 的完整"灵魂",配置即人格,而非写代码。

| 文件 | 作用 |
|---|---|
| `SOUL.md` | 性格、价值观、沟通风格 |
| `AGENTS.md` | 行为指令、工作流程规则 |
| `USER.md` | 用户信息、偏好 |
| `TOOLS.md` | 工具使用注意事项 |

## 2. MVP 决策汇总

| 维度 | 决策 |
|---|---|
| 定位 | 框架级复刻 + SRE 预设 |
| MVP 形态 | CLI + 完整 SRE 发行版示范 |
| 安全策略 | 白名单 + 人工确认(`y`/`n`/`a`) + `--yolo` flag + denylist 红线 |
| Memory | L1:MEMORY.md + history.jsonl + Consolidator(Dream 留接口不实现) |
| Provider | openai-compatible 一个(覆盖 DeepSeek / Qwen / Ollama / vLLM) |
| Skills | 目录形态 `skills/<name>/SKILL.md`;workspace 覆盖 builtin;靠 `read_file` 懒加载;`always` 全量注入 |
| Persona | SOUL/AGENTS/USER/TOOLS.md 每轮重读 |
| 工具层 | builtin(read_file, write_file, shell, remember)+ MCP 动态注册;approval 装饰器 |
| Session | 默认新建;`--resume` 恢复上次 |
| 编排 | eino Graph:InputAdapter → SystemPromptBuilder → AgentLoop → OutputAdapter |

## 3. 项目目录结构

```
srebot/
├── cmd/srebot/              # CLI 入口
├── config/                  # 配置加载 + 样例(顶层,可被外部 import)
│   ├── config.go            # 结构体、解析、校验、默认值
│   ├── loader.go            # 优先级合并
│   └── config.example.yaml
├── internal/
│   ├── agent/               # eino Graph 编排
│   ├── provider/            # ChatModel 抽象 + openai 实现
│   ├── persona/             # 四件套加载
│   ├── skills/              # Skills 索引 + 加载器
│   ├── memory/              # Store + Consolidator + Dream 接口
│   ├── session/             # 原始消息 session.jsonl
│   ├── tools/               # builtin + Registry
│   ├── mcp/                 # MCP client
│   └── approval/            # Policy + Gate + denylist
├── presets/default/         # SRE 发行版预设
│   ├── SOUL.md
│   ├── AGENTS.md
│   ├── USER.md
│   ├── TOOLS.md
│   ├── mcp.yaml
│   └── skills/
│       ├── k8s-oom-troubleshoot/SKILL.md
│       ├── prom-query-cheatsheet/SKILL.md
│       └── incident-response-principles/SKILL.md
├── test/e2e/
├── examples/
├── go.mod
└── README.md
```

**配置加载优先级:** CLI flag > 环境变量 > `~/.srebot/config.yaml` > `config/config.example.yaml` 默认值。

## 4. 整体架构

### 4.1 分层视图

```
┌──────────────────────────────────────────────────────────┐
│ Entry Layer      cmd/srebot (CLI, REPL)                   │
├──────────────────────────────────────────────────────────┤
│ Runtime Layer    internal/agent (eino Graph)              │
├──────────────────────────────────────────────────────────┤
│ Capability Layer provider / skills / tools / mcp /        │
│                  memory / persona / approval / session    │
├──────────────────────────────────────────────────────────┤
│ Asset Layer      presets/default/  +  ~/.srebot/workspace │
└──────────────────────────────────────────────────────────┘
```

### 4.2 eino Graph 节点

```
[User Input]
     │
     ▼
┌──────────────┐
│ InputAdapter │  追加 session message
└──────┬───────┘
       ▼
┌────────────────────────┐
│ SystemPromptBuilder    │  每轮重拼:
│                        │  identity + persona (4 MD)
│                        │  + # Memory (MEMORY.md)
│                        │  + # Active Skills (always 全量)
│                        │  + # Available Skills (summary + 路径)
│                        │  + # Recent History (history.jsonl tail)
└──────┬─────────────────┘
       ▼
┌────────────────────────┐
│ AgentLoop              │  eino ChatModel ↔ ToolsNode 循环
│                        │  tools = builtin + MCP(带 approval 装饰)
└──────┬─────────────────┘
       ▼
┌──────────────┐
│ OutputAdapter│  写 session + 触发 Consolidator 评估
└──────────────┘
```

### 4.3 启动装配(bootstrap)

CLI 启动一次性装配,运行时不再改:

1. `config.Load()` — 按优先级合并
2. `persona.NewFSLoader(workspace)` — 准备 MD 加载器
3. `skills.NewLoader(workspace/skills, presets/default/skills, disabled)` — 构建两级索引
4. `mcp.Manager.LoadFromConfig(mcp.yaml)` + 启动 MCP clients
5. `tools.Registry` 注册 builtin(read_file / write_file / shell / remember)+ MCP tools
6. `provider.NewOpenAICompatible(cfg)` — 实例化 ChatModel
7. `memory.NewStore(workspace)` + `memory.NewConsolidator(store, provider, budget)`
8. `session.Open(workspace, id)` — 默认新 id,`--resume` 加载上次
9. `agent.Build(Deps{...})` — 组装 eino Graph
10. 进入 REPL

## 5. 核心组件接口

### 5.1 config

```go
package config

type Config struct {
    Provider  ProviderConfig
    Workspace string
    PresetDir string
    Memory    MemoryConfig
    Approval  ApprovalConfig
    Logging   LoggingConfig
}
func Load(flags *Flags) (*Config, error)
func (c *Config) Validate() error
```

### 5.2 provider

```go
package provider

type ChatModel interface {
    Generate(ctx context.Context, msgs []*schema.Message, opts ...Option) (*schema.Message, error)
    Stream(ctx context.Context, msgs []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error)
    BindTools(tools []*schema.ToolInfo) error
}
func NewOpenAICompatible(cfg ProviderConfig) (ChatModel, error)
```

### 5.3 persona

```go
package persona

type Persona struct { Soul, Agents, User, Tools string }
type Loader interface { Load() (*Persona, error) }
func NewFSLoader(workspaceDir string) Loader
```

### 5.4 skills

```go
package skills

type Skill struct {
    Name, Description, Path string
    Always                  bool
    Requires                Requirements
    Available               bool
}
type Loader interface {
    List() []Skill
    LoadContent(name string) (string, error)
    Summary(exclude []string) string
    AlwaysSkills() []Skill
}
func NewLoader(workspaceDir, builtinDir string, disabled []string) Loader
```

Skills 以目录形式组织(`skills/<name>/SKILL.md`),允许 skill 带附属资产。Workspace 同名覆盖 builtin。LLM 通过 summary 里暴露的路径 + `read_file` tool 懒加载正文。

### 5.5 memory

```go
package memory

type Store interface {
    ReadMemory() (string, bool, error)
    AppendHistory(entry string) (cursor int, err error)
    ReadUnprocessedHistory(sinceCursor int) ([]HistoryEntry, error)
    LastDreamCursor() int
    SetLastDreamCursor(c int) error
    CompactHistory() error
}
type HistoryEntry struct {
    Cursor    int
    Timestamp time.Time
    Content   string
}
type Consolidator interface {
    MaybeConsolidate(ctx context.Context, active []*schema.Message) (kept []*schema.Message, err error)
}
func NewStore(workspaceDir string) Store
func NewConsolidator(store Store, model provider.ChatModel, budgetTokens int) Consolidator
```

**文件布局:**
```
~/.srebot/workspace/memory/
├── MEMORY.md          # 长期事实,remember 工具 + 人工编辑
├── history.jsonl      # Consolidator 产出的压缩条目
├── .cursor            # 自增游标
└── .dream_cursor      # MVP 恒为 0
```

**Consolidator 触发:** 每轮 turn 结束,若估算 `tokens(active) > context_window * 0.8`,调 LLM 压缩最老一批消息 → `AppendHistory(summary)` → 返回裁剪后列表。LLM 失败时 fallback `raw_archive`(带 `[RAW]` 前缀)。

### 5.6 session

```go
package session

type Session interface {
    ID() string
    Append(msg *schema.Message) error
    Load() ([]*schema.Message, error)
    Close() error
}
func Open(workspaceDir, id string) (Session, error)   // id 空则生成新
```

**与 memory 的边界:** session 存"原始发生了什么",memory 存"浓缩后要长期记住什么",两条独立数据流。

### 5.7 tools

```go
package tools

type Tool interface {
    Info() *schema.ToolInfo
    Invoke(ctx context.Context, argsJSON string) (string, error)
}
type Registry interface {
    Register(t Tool) error
    Unregister(name string)
    Get(name string) (Tool, bool)
    Definitions() []*schema.ToolInfo   // 稳定排序,builtin 前 MCP 后
}

func NewReadFile(workspace string) Tool
func NewWriteFile(workspace string) Tool
func NewShell(gate approval.Gate) Tool
func NewRemember(store memory.Store) Tool
```

### 5.8 mcp

```go
package mcp

type Client interface {
    Connect(ctx context.Context) error
    Tools() []tools.Tool
    Close() error
}
type Manager interface {
    LoadFromConfig(path string) error
    RegisterAllInto(reg tools.Registry) error
    Close() error
}
```

MCP tools 命名统一加 `mcp_` 前缀。单个 server 启动失败不影响其他。

### 5.9 approval

```go
package approval

type Decision int
const ( Allow Decision = iota; Deny; AskUser )

type Policy interface { Check(toolName string, args map[string]any) Decision }
type Gate   interface { Authorize(ctx context.Context, toolName string, args map[string]any) error }
type Prompter interface { Confirm(msg string) (ConfirmResult, error) }

type ConfirmResult int
const ( ConfirmYes ConfirmResult = iota; ConfirmNo; ConfirmSessionAll )

func NewPolicy(whitelist []string, readOnlyAutoApprove, yolo bool) Policy
func NewCLIGate(policy Policy, prompter Prompter) Gate
```

**Decision 优先级:**
```
1. denylist 命中             → Deny   (最高,yolo/a/whitelist 都不能绕)
2. yolo == true              → Allow
3. sessionAutoApprove == true→ Allow  (用户选过 [a])
4. read-only 内置白名单       → Allow
5. 用户配置 whitelist         → Allow
6. 其他                      → AskUser
```

**CLI 交互:** `[y] 允许 / [n] 拒绝 / [a] 本会话全部允许`,Ctrl+C 视为 Deny。

**Denylist(写死在 `internal/approval/denylist.go`):**
- `rm -rf /` 及变体
- `mkfs` / `dd if=... of=/dev/...` / `> /dev/sd*`
- `:(){ :|:& };:` (fork bomb)
- `shutdown` / `reboot` / `halt` / `poweroff`
- `chmod -R 777 /`

### 5.10 agent

```go
package agent

type Agent interface {
    Run(ctx context.Context, userInput string) (reply string, err error)
}
type Deps struct {
    Config       *config.Config
    Provider     provider.ChatModel
    Persona      persona.Loader
    Skills       skills.Loader
    MemoryStore  memory.Store
    Consolidator memory.Consolidator
    Session      session.Session
    Tools        tools.Registry
}
func Build(d Deps) (Agent, error)
```

Agent 只做编排,所有业务在下层模块。依赖注入通过 `Deps` 结构体,不用全局单例。

### 5.11 依赖图

```
cmd/srebot
   └─► internal/agent
         ├─► internal/provider
         ├─► internal/persona
         ├─► internal/skills
         ├─► internal/memory ───► internal/provider
         ├─► internal/session
         ├─► internal/tools ────► internal/approval
         └─► internal/mcp ──────► internal/tools
```

**铁律:** `internal/agent` 只 import interface,具体实现在 `cmd/srebot/main.go` 装配注入。

## 6. 运行时数据流

### 6.1 一次 turn 的完整步骤

1. CLI 读一行用户输入
2. `session.Append(userMsg)` — 先落盘再进入 Graph(崩溃不丢)
3. `SystemPromptBuilder` 每轮重新组装:
   - identity(runtime metadata:workspace、OS、时间)
   - persona(4 MD 拼接)
   - `# Memory` — MEMORY.md 正文(模板未修改则跳过)
   - `# Active Skills` — always=true 的 skill 全量正文
   - `# Available Skills` — 其余 skill 的 name + description + 绝对路径
   - `# Recent History` — `history.jsonl` 中 cursor > dream_cursor 的 tail 50 条
4. `AgentLoop` 进入 eino 的 ChatModel ↔ ToolsNode 循环:
   - ChatModel 生成 message(带 tool_calls 或 finish)
   - ToolsNode 对每个 tool_call:经 `approval.Gate.Authorize` → denylist/yolo/session-auto/whitelist 判定 → 可能弹 prompt → `tool.Invoke`
   - tool result 回填,回到 ChatModel
   - 直到 finish
5. `OutputAdapter`:
   - `session.Append(assistantMsg)`
   - `Consolidator.MaybeConsolidate(active)` — 超预算则压缩
   - 返回 reply 到 CLI

### 6.2 三条不变式(Invariants)

- **I1.** Session 是 source of truth:`session.jsonl` 任何时刻完整;`history.jsonl` 和 `MEMORY.md` 丢失只损失"已压缩记忆",不丢对话
- **I2.** Approval 不可绕过:所有执行/写类 tool 必须经 `approval.Gate` 包装;denylist 永远生效
- **I3.** System prompt 构造是纯函数:输入 persona/memory/skills/history 快照,输出字符串,无副作用

### 6.3 降级路径

| 故障点 | 降级 | 用户感知 |
|---|---|---|
| Consolidator LLM 失败 | `Store.raw_archive` 带 `[RAW]` 前缀 | log warn,对话继续 |
| MCP server 启动失败 | 跳过,其他继续 | 启动日志 warn,tool 少一块 |
| Provider 超时 | 返回 error,本轮失败 | CLI 显示 "provider timeout" |
| `session.jsonl` 写失败 | **fail-fast** | 程序退出(I1 不容违反) |
| `MEMORY.md` / `history.jsonl` 写失败 | log error,继续 | 下轮 prompt 可能少点上下文 |
| Skill `requires` 不满足 | summary 标 `(unavailable)` | LLM 可见 |
| Tool approval 被拒 | tool result = "user denied" | LLM 告知用户 |
| Denylist 命中 | tool result = "blocked by safety denylist" | LLM 告知用户 |

## 7. 安全红线(硬约束)

- **R1.** 任何执行/写类 tool 未经 `approval.Gate` 包装禁止注册到 Registry(单测扫描保证)
- **R2.** `shell` tool denylist 最高优先级拦截(列表见 5.9)
- **R3.** MCP server 的 env 必须经 config 层 `${VAR}` 插值,不得从原始 yaml 透传
- **R4.** `read_file` / `write_file` 的 path 限定在 workspace 下,`filepath.Clean` + 前缀检查

## 8. Secret / 日志 / 超时

**Secret:**
- config 文件只存 `${VAR}` 引用,`config.Load` 做插值,缺失 fail-fast
- logger 统一 `redactSecrets` middleware(api_key / token / password / Authorization 替换 `***`)
- tool args 入 session 前过一遍 redact

**超时:**
- Provider 调用默认 120s(可配置)
- shell tool 默认 60s
- MCP tool 默认 30s
- Ctrl+C 取消顶层 ctx,eino Graph 退出当前 iteration,已写 session 不回滚
- Approval prompt 处 Ctrl+C 视为 Deny,流程继续

**Panic:**
- `agent.Run` 顶层 `defer recover()`:log error + stack,session 追加 `[system] internal error`,REPL 继续
- 单个 tool 不 recover,让 panic 炸到 agent 层统一处理

**日志分级:**
- `DEBUG` — 完整 prompt、tool args、token 估算
- `INFO` — turn 开始/结束、tool call 名字、consolidation 触发
- `WARN` — 降级(MCP skip、raw_archive、skill unavailable)
- `ERROR` — panic 恢复、fail-fast
- 默认 INFO,`--verbose` 开 DEBUG,使用 Go 1.21+ 标准库 `slog`

## 9. 测试策略

### 9.1 单元测试

| 模块 | 重点 |
|---|---|
| config | 优先级合并、`${VAR}` 插值、缺失 fail-fast、yaml 错误 |
| persona | 4 MD 缺失/存在组合、拼接顺序、空文件 |
| skills | workspace 覆盖 builtin、frontmatter 解析、always 识别、requires 检查、summary 排序 |
| memory.Store | 游标单调、corruption 容错、next_cursor 兜底、raw_archive |
| memory.Consolidator | 未超预算 no-op、超预算压缩、provider 失败 fallback |
| session | append/load 往返、崩溃恢复、id 唯一 |
| approval | denylist 最高优先、yolo/a/whitelist 放行、Ctrl+C→Deny |
| tools builtin | read_file 路径穿越拦截、write_file 同上、shell 超时、remember 追加 |
| mcp.Manager | 单个 server 失败不影响其他、命名前缀 |

**Mock:** `fakeChatModel` 按预设队列返回 message/tool_calls;`t.TempDir()`;in-proc fake MCP server。

### 9.2 集成测试(`internal/agent/integration_test.go`)

1. 完整一轮 turn
2. 多轮 tool 循环(3 轮后 finish)
3. approval 拦截后 loop 继续
4. denylist 拦截 yolo 也失效
5. `--resume` 恢复 session
6. Consolidator 触发
7. Skills 两级目录 workspace 胜出
8. MCP 失败降级

### 9.3 E2E 测试(`test/e2e/`)

本地 HTTP mock server 实现 OpenAI `/v1/chat/completions` 最小子集,`os/exec` 驱动 CLI:

1. 冷启动 + 单轮问答
2. `--resume` 恢复
3. `--yolo` 无 prompt 跑通
4. CLI approval 交互(go-expect 按 `y`)

### 9.4 覆盖率

- `internal/**` ≥ 80%,`config` ≥ 90%,`cmd/srebot/main.go` 豁免
- CI:GitHub Actions,push/PR 触发,`golangci-lint` + `go test -race -cover`

### 9.5 TDD 执行顺序

```
1. config
2. memory.Store
3. session
4. persona
5. skills
6. approval
7. tools builtin (deps approval)
8. mcp
9. provider (fake + 真实 thin wrapper)
10. memory.Consolidator
11. agent (集成)
12. cmd/srebot + E2E
```

## 10. 超出 MVP 范围(v0.2+)

为避免 scope creep,以下显式排除:

- IM channel 适配(飞书/Slack/DingTalk)
- Web UI
- Dream 机制(history.jsonl → MEMORY.md 二次提炼)
- anthropic / ark 等其他 provider
- 向量检索 memory
- 多 session 并发
- Cron 调度
- Auto-compact 进度条等 UX 优化

## 11. 开放问题

无(所有已知问题已在决策汇总中锁定)。
