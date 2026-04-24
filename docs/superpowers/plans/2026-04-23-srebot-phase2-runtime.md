# srebot Runtime(第二阶段)实施计划

> **给执行方的说明:** 必需子技能:使用 superpowers:subagent-driven-development(推荐)或 superpowers:executing-plans 按任务逐个实施本计划。步骤用 checkbox(`- [ ]`)语法追踪进度,完成后改为 `- [x]` 并在任务标题追加 ✅。

**目标:** 在 Phase 1 的 I/O 基础层之上,装配"能力层":tools、approval、shell、mcp、provider、skills。本阶段结束后,所有能力模块均可单测通过,供 Phase 3 的 agent Graph 编排使用。

**架构:** 本阶段首次引入 eino 框架(`github.com/cloudwego/eino`)与 MCP 库(`github.com/mark3labs/mcp-go`)。tool 契约统一用 eino 原生类型(`tool.InvokableTool` / `schema.ToolInfo`),避免再包一层。Provider 基于 `github.com/cloudwego/eino-ext/components/model/openai`。Skills 自写 `Loader`,保留 nanobot 目录结构与 `requires: {bins, env}` 语义,**不使用** eino skill 中间件的依赖解析。

**技术栈:**
- Go 1.25.8
- eino:`github.com/cloudwego/eino`(tool / schema / compose / adk-skill 可选)
- eino openai:`github.com/cloudwego/eino-ext/components/model/openai`
- MCP:`github.com/mark3labs/mcp-go`
- 测试:标准 `testing`,原生 `t.Errorf`/`t.Fatal`,**禁止 testify**

**范围:** 3 个阶段中的第 2 个。实现顺序:tools(read/write)→ approval → shell(带 approval)→ mcp → provider → skills。每个模块独立单测,最小 80% 覆盖率(tool 简单模块可适度放宽到 70%)。

**Spec 引用:** §5.2、§5.4、§5.7、§5.8、§5.9,以及 §7(安全红线)、§8(Secret)。

---

## 文件结构

**本阶段新建文件:**

```
srebot/
├── internal/
│   ├── tools/
│   │   ├── registry.go               # Registry 接口 + 内存实现
│   │   ├── read_file.go              # NewReadFile(workspace) tool.InvokableTool
│   │   ├── write_file.go             # NewWriteFile(workspace)
│   │   ├── shell.go                  # NewShell(gate approval.Gate)
│   │   ├── remember.go               # NewRemember(memory.Store)
│   │   ├── registry_test.go
│   │   ├── read_file_test.go
│   │   ├── write_file_test.go
│   │   ├── shell_test.go
│   │   └── remember_test.go
│   ├── approval/
│   │   ├── denylist.go               # 硬编码红线
│   │   ├── policy.go                 # Policy + NewPolicy
│   │   ├── gate.go                   # Gate + CLIGate + Prompter
│   │   ├── denylist_test.go
│   │   ├── policy_test.go
│   │   └── gate_test.go
│   ├── mcp/
│   │   ├── server.go                 # Server 结构(url/command/args/env/toolTimeout)
│   │   ├── client.go                 # HTTP + stdio 两种 transport 的 Client 封装
│   │   ├── manager.go                # Manager:从 []Server 构造 + 并发启动 + 注册到 Registry
│   │   ├── server_test.go
│   │   ├── client_test.go
│   │   └── manager_test.go
│   ├── provider/
│   │   ├── provider.go               # ChatModel 接口(薄封装)
│   │   ├── openai.go                 # NewOpenAICompatible(基于 eino-ext openai)
│   │   └── openai_test.go
│   └── skills/
│       ├── skill.go                  # Skill 结构 + frontmatter 解析
│       ├── loader.go                 # Loader 接口 + FSLoader(两级合并)
│       ├── requirements.go           # Requirements + CheckRequirements
│       ├── skill_test.go
│       ├── loader_test.go
│       └── requirements_test.go
└── go.mod / go.sum                   # 增加 eino + mcp-go 依赖
```

**不在本阶段:** `internal/agent`(Phase 3)、`cmd/srebot`(Phase 3)、预设 `presets/default/`(Phase 3)、memory.Consolidator 的 LLM 实现(Phase 3,Phase 1 已留 Store 接口)。

---

## 前置准备

- [x] **步骤 0.1:引入 eino 与 mcp-go 依赖**

在 `/Users/fengjin/Desktop/GitHub/srebot` 执行:
```bash
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/openai@latest
go get github.com/mark3labs/mcp-go@latest
go mod tidy
```

预期:`go.mod` 新增三行 `require`,`go.sum` 写入对应哈希;无编译错误(此时仅下载,未使用)。

- [x] **步骤 0.2:确认 eino 接口版本**

通过 MCP `claude.ai Context7` 查询当前 eino 的 `tool.InvokableTool`、`schema.ToolInfo`、`schema.Message`、`compose.NewToolsNode` 签名,记下导入路径。若本地编辑器有 `gopls`,用 `go doc github.com/cloudwego/eino/components/tool` 验证。

---

## 任务 1:tools.Registry + 内存实现 ✅

**目标:** 定义统一的 tool 契约(直接复用 eino `tool.InvokableTool`),提供一个线程安全的 Registry。

**文件:**
- 新建:`internal/tools/registry.go`、`internal/tools/registry_test.go`

**依赖:** 仅 eino `components/tool` 与 `schema`。

- [x] **步骤 1.1:定义 Registry**

`internal/tools/registry.go`:
- 类型别名 `type Tool = tool.InvokableTool`(直接用 eino 的)
- 接口 `Registry`:`Register(Tool) error` / `Unregister(name string)` / `Get(name) (Tool, bool)` / `Definitions(ctx) ([]*schema.ToolInfo, error)`
- 实现 `memRegistry`:`sync.RWMutex` + `map[string]Tool`
- `NewRegistry() Registry`
- `Definitions` 按名字稳定排序后返回(builtin 不保证前,但同名注册两次报错)
- `Register` 空名或重复名返回 `error`

- [x] **步骤 1.2:写 Registry 单测**

`registry_test.go` 覆盖:
- 注册/获取/注销
- 重复名报错
- 空名报错
- 并发 Register/Get 无 data race(用 `-race` 跑)
- `Definitions` 稳定排序

- [x] **步骤 1.3:运行 + 验证**

```bash
go test -race -cover ./internal/tools/
```

预期 coverage ≥ 80%。

- [x] **步骤 1.4:提交**

```bash
git add internal/tools/registry.go internal/tools/registry_test.go go.mod go.sum
git commit -m "feat(tools): add Registry with eino InvokableTool contract"
```

---

## 任务 2:tools.read_file + write_file(workspace-scoped) ✅

**目标:** 实现两个无副作用外界的内置工具,路径严格限定 workspace,对齐 spec R4。

**文件:**
- 新建:`internal/tools/read_file.go`、`internal/tools/write_file.go` 及对应 `_test.go`

- [x] **步骤 2.1:共用 path guard**

在 `internal/tools/path.go`(新建)中实现 `resolveInsideWorkspace(workspace, rel string) (string, error)`:
- `filepath.Clean(rel)`
- 拒绝绝对路径
- `filepath.Join(workspace, clean)` 后再 `filepath.Rel(workspace, joined)`,若结果以 `..` 开头则返回 `path escapes workspace` 错误
- 返回绝对路径

- [x] **步骤 2.2:read_file**

`read_file.go`:
- 参数 schema:`{ "path": {type: string, required} }`
- 实现 `tool.InvokableTool`:`Info(ctx)` 返回 `*schema.ToolInfo`;`InvokableRun(ctx, argsJSON, opts...)` 解析 JSON → `resolveInsideWorkspace` → `os.ReadFile` → 返回字符串
- 文件不存在、超 1 MiB、非 UTF-8 有效检测 → 返回错误字符串(tool 不 panic)

- [x] **步骤 2.3:write_file**

`write_file.go`:
- 参数 schema:`{ "path": string, "content": string, "append": bool (optional, default false) }`
- 同样路径守卫
- 写入前 `os.MkdirAll(filepath.Dir(path), 0755)`
- `append==true` 用 `O_APPEND`,否则 truncate;文件权限 `0644`
- 返回 `"wrote N bytes to <relpath>"`

- [x] **步骤 2.4:单测**

覆盖:
- 正常读写
- 越界路径(`../`、绝对路径)被拒
- 读不存在文件报错
- write 自动建父目录
- append 模式

- [x] **步骤 2.5:验证 + 提交**

```bash
go test -race -cover ./internal/tools/
git add internal/tools/{path,read_file,write_file}*.go
git commit -m "feat(tools): add workspace-scoped read_file and write_file"
```

---

## 任务 3:approval 模块(denylist / policy / gate) ✅

**目标:** 独立实现审批三件套,尚未与 tools 装配。Shell tool 在任务 4 消费。

**文件:**
- 新建:`internal/approval/{denylist,policy,gate}.go` + 三个 `_test.go`

- [x] **步骤 3.1:denylist**

`denylist.go`:
- 导出 `HitDenylist(cmd string) (matched string, hit bool)`
- 规则用正则切片,匹配 spec §5.9:
  - `rm -rf /`(及 `rm\s+-[a-zA-Z]*r[a-zA-Z]*f?\s+/\s*$` 等变体)
  - `mkfs` / `dd\s+if=.*of=/dev/` / `>\s*/dev/sd`
  - `:\(\)\s*\{\s*:\|:&\s*\};\s*:`(fork bomb)
  - `shutdown` / `reboot` / `halt` / `poweroff`(独立命令)
  - `chmod\s+-R\s+777\s+/`
- 测试枚举正例/反例,覆盖边界(命令前有 sudo、有空格变体)

- [x] **步骤 3.2:Policy**

`policy.go`:
- `type Decision int` 常量 `Allow / Deny / AskUser`
- `Policy` 接口 `Check(tool string, args map[string]any) Decision`
- `NewPolicy(whitelist []string, readOnlyAutoApprove, yolo bool) Policy`
- 内部 read-only 白名单(写死):`read_file`
- **判定顺序严格对齐 spec §5.9:**
  1. tool==`shell` 且 args["command"] 命中 denylist → Deny
  2. yolo → Allow
  3. read-only 自动批准且 tool∈read-only 集 → Allow
  4. tool ∈ whitelist → Allow
  5. → AskUser
- `sessionAutoApprove` 不放在 Policy 内,由 Gate 持有(见 3.3)

- [x] **步骤 3.3:Gate + Prompter**

`gate.go`:
- `Prompter` 接口 `Confirm(msg string) (ConfirmResult, error)`;`ConfirmYes/No/SessionAll`
- `Gate` 接口 `Authorize(ctx, tool, args) error`(nil 表放行)
- `NewCLIGate(policy Policy, prompter Prompter) Gate`
- 内部持 `sessionAuto atomic.Bool`;`Authorize`:
  - 先让 Policy 决策
  - 若 `Deny` → `errors.New("blocked by denylist")` 或 `errors.New("not allowed by policy")`
  - 若 `Allow` → nil
  - 若 `AskUser` 且 `sessionAuto` 为 true → Allow
  - 否则调 `prompter.Confirm`,`SessionAll` 时设 `sessionAuto=true` 并放行,`No` → error `"user denied"`
- 提供 `stubPrompter` 用于测试(可预置返回值队列)

- [x] **步骤 3.4:单测**

覆盖 Policy 所有分支 + Gate 四条路径(Allow / Deny-denylist / Deny-policy / AskUser-yes/no/all)+ `sessionAuto` 粘性。

- [x] **步骤 3.5:验证 + 提交**

```bash
go test -race -cover ./internal/approval/
git add internal/approval/
git commit -m "feat(approval): add denylist, policy and CLI gate"
```

预期 coverage ≥ 85%。

---

## 任务 4:tools.shell(受 approval 保护) ✅

**目标:** 在 approval.Gate 之上包一个会调用 `/bin/sh -c` 的工具,spec R1/R2 的直接落地。

**文件:**
- 新建:`internal/tools/shell.go`、`shell_test.go`

- [x] **步骤 4.1:实现**

- 参数 schema:`{ "command": string, "timeoutSec": int (optional, default 30, max 300) }`
- `NewShell(gate approval.Gate) tool.InvokableTool`
- `InvokableRun`:
  1. 先调 `gate.Authorize(ctx, "shell", argsMap)`,拒绝则把错误文本回填("user denied" / "blocked by denylist")
  2. 通过后 `exec.CommandContext(ctxWithTimeout, "/bin/sh", "-c", cmd)`
  3. CombinedOutput,截断到 64 KiB
  4. 返回 `exit=N\nstdout/err:\n<body>`(非零 exit 不作为 error,交由 LLM 判断)

- [x] **步骤 4.2:单测**

用 stub Gate(总是放行 / 总是拒绝 / 总是命中 denylist):
- 放行 → 运行 `echo hello` 成功
- 拒绝 → 返回文本包含 `user denied`
- denylist → 返回文本包含 `blocked`
- 超时生效(`sleep 5` + timeout=1)
- 输出截断(产出 > 64 KiB)

注意:`exec.Command` 的测试在 CI 可能被禁,可用 build tag `//go:build !nosh` 或直接保留,必要时 skip。

- [x] **步骤 4.3:验证 + 提交**

```bash
go test -race -cover ./internal/tools/
git add internal/tools/shell*.go
git commit -m "feat(tools): add shell tool guarded by approval.Gate"
```

---

## 任务 5:tools.remember ✅

**目标:** LLM 可写 `MEMORY.md` 的"记住这件事"工具,直接转给 `memory.Store.AppendHistory` 或追加 MEMORY.md(MVP 选后者)。

**文件:**
- 新建:`internal/tools/remember.go`、`remember_test.go`

- [x] **步骤 5.1:实现**

- 参数 schema:`{ "fact": string }`
- `NewRemember(store memory.Store) tool.InvokableTool`
- Invoke 调 `store.AppendHistory("[MEMO] " + fact)` 并返回 `"remembered (cursor=N)"`

(写 MEMORY.md 的路径延后到 Phase 3 的 Consolidator;Phase 2 先复用 history 通道。)

- [x] **步骤 5.2:单测**

- 用真实 `memory.NewStore(tmpDir)`(Phase 1 已实现)
- 调用后读 history.jsonl,断言内容包含 `[MEMO]`

- [x] **步骤 5.3:验证 + 提交**

```bash
go test -race -cover ./internal/tools/
git add internal/tools/remember*.go
git commit -m "feat(tools): add remember tool backed by memory.Store"
```

---

## 任务 6:mcp client + manager(基于 mark3labs/mcp-go) ✅

**目标:** 从 `config.Resolved.MCPServers` 读取配置(顶层 `mcpServers`,与 `agents` 同级),为每个 server 启动 client(HTTP 或 stdio),把 list_tools 结果包成 `tool.InvokableTool` 注入 Registry,统一加 `mcp_<server>_<tool>` 前缀。

**文件:**
- 修改:`config/config.go` 增加 `MCPServers map[string]MCPServer` 字段 + 对应 Resolved 字段 + 默认值处理 + 测试
- 新建:`internal/mcp/{server,client,manager}.go` 及测试

- [x] **步骤 6.0:回补 config 的 MCPServers 字段**

在 `config/config.go`:
```go
type MCPServer struct {
    URL         string            `json:"url,omitempty"`
    Command     string            `json:"command,omitempty"`
    Args        []string          `json:"args,omitempty"`
    Env         map[string]string `json:"env,omitempty"`
    ToolTimeout int               `json:"toolTimeout,omitempty"`
}

type Root struct {
    // ... 现有字段
    MCPServers map[string]MCPServer `json:"mcpServers"`
}

type Resolved struct {
    // ... 现有字段
    MCPServers map[string]MCPServer
}
```

- `Resolve()` 把 `Root.MCPServers` 拷贝进 `Resolved.MCPServers`(nil 视作空 map)
- `Validate()` 对每个 server:
  - URL 与 Command 不能同时为空 → error
  - URL 必须 `http://` / `https://` 前缀(若非空)
  - ToolTimeout == 0 时填默认 60
- 补 `config_test.go`:涵盖 HTTP-only、stdio-only、两者都缺、URL 非法四种

- [x] **步骤 6.1:mcp.Server 适配**

`internal/mcp/server.go`:
- 类型别名 `type Server = config.MCPServer`(或重新声明,保持 internal/mcp 不反向 import config:**采用重新声明**,内部结构与 config 对齐,在 manager 层做字段拷贝)
- 判定函数 `(s Server) IsHTTP() bool { return s.URL != "" }`

**依赖方向:** `cmd/srebot` 在装配时把 `config.MCPServers` 转换成 `[]mcp.Server` 传给 `mcp.NewManager`,`internal/mcp` **不** import `config`。

- [x] **步骤 6.2:Client 封装**

`client.go`:
- `type Client struct { name string; timeout time.Duration; mc *mcpgo.Client ... }`
- 构造分两种:
  - `newHTTPClient(name, url string, timeout time.Duration)` → `mcpgo.NewClient` + Streamable HTTP transport
  - `newStdioClient(name, cmd string, args []string, env map[string]string, timeout time.Duration)` → stdio transport
- `Connect(ctx)`:`Initialize`
- `Tools() []tool.InvokableTool`:`ListTools` → 每个 remote 包成 `mcpTool`,名字 `fmt.Sprintf("mcp_%s_%s", clientName, remoteName)`
- `Close() error`

`mcpTool.InvokableRun`:用 `context.WithTimeout(ctx, client.timeout)` 调 `CallTool`,拼接 content blocks 为纯文本。

- [x] **步骤 6.3:Manager**

`manager.go`:
- `NewManager(servers map[string]Server) *Manager`
- `ConnectAll(ctx)` 并发启动;每个失败 log warn 并从 active 列表剔除(不中断其他)
- `RegisterAllInto(reg tools.Registry) error`
- `Close() error` 并发 Close

- [x] **步骤 6.4:单测**

- `server_test.go`:`IsHTTP` 判定
- `manager_test.go`:
  - 用 `httptest.Server` 模拟一个最小 MCP HTTP endpoint(实现 initialize / tools/list / tools/call 三个方法即可),验证 HTTP transport 路径能连上并注册一个带 `mcp_<name>_` 前缀的 tool
  - stdio 路径因依赖 subprocess,用 `testing.Short()` 跳过或单独 integration tag
  - "一个 server 坏、另一个 server 好" 的降级:给两个 server,一个 URL 不通,断言另一个仍注册成功
- 避免真实 stdio subprocess;复杂度留给后续 integration 测试

- [x] **步骤 6.5:验证 + 提交**

```bash
go test -race -cover ./internal/mcp/ ./config/
git add internal/mcp/ config/ go.mod go.sum
git commit -m "feat(mcp): load servers from config.mcpServers with HTTP+stdio transport"
```

coverage 目标:config ≥ 90%,mcp ≥ 70%(subprocess 路径难覆盖,可放宽)。

---

## 任务 7:provider(eino-ext openai) ✅

**目标:** 封装 `github.com/cloudwego/eino-ext/components/model/openai`,暴露给上层简单的 ChatModel 接口。同时支持 DeepSeek / Qwen / Ollama 等兼容端点。

**文件:**
- 新建:`internal/provider/{provider,openai}.go` + `openai_test.go`

- [x] **步骤 7.1:ChatModel 接口**

`provider.go`:
- 类型别名 `type ChatModel = model.ToolCallingChatModel`(直接用 eino 的)
- 这样 Phase 3 Graph 可以直接把 `ChatModel` 插入 `compose.NewChatModel`
- 留一个薄 wrapper `type Config struct { BaseURL, APIKey, Model string; Temperature float64; TimeoutSec int; MaxTokens int }`

- [x] **步骤 7.2:openai 实现**

`openai.go`:
- `func NewOpenAICompatible(cfg Config) (ChatModel, error)`
- 内部:`openai.NewChatModel(ctx, &openai.ChatModelConfig{...})`
- 超时通过 `http.Client{Timeout: ...}` 注入
- 参数校验:BaseURL/APIKey/Model 必填,否则返回错

- [x] **步骤 7.3:单测**

- 不做真实网络调用
- 启 `httptest.Server` mock `/chat/completions`,返回固定 JSON
- `NewOpenAICompatible(cfg)` 指向该 server,调 `Generate` 验证返回
- 参数缺失报错

- [x] **步骤 7.4:验证 + 提交**

```bash
go test -race -cover ./internal/provider/
git add internal/provider/ go.mod go.sum
git commit -m "feat(provider): add openai-compatible ChatModel via eino-ext"
```

coverage 目标 ≥ 75%。

---

## 任务 8:skills(自写 Loader,保留 nanobot requires)

**目标:** 两级目录(`workspace/skills/<name>/SKILL.md` 覆盖 `presetDir/skills/<name>/SKILL.md`),解析 frontmatter,检查 `requires: {bins, env}`,提供 summary/Load/Always 列表。

**文件:**
- 新建:`internal/skills/{skill,requirements,loader}.go` + 三个测试

- [ ] **步骤 8.1:Skill + frontmatter 解析**

`skill.go`:
- struct 对齐 spec §5.4 + 加字段:
  ```go
  type Skill struct {
      Name        string
      Description string
      Path        string // 绝对路径到 SKILL.md
      Dir         string // skill 目录
      Always      bool
      Requires    Requirements
      Available   bool
      Body        string // 正文(不含 frontmatter),懒加载
  }
  ```
- `parseSKILL(raw []byte) (frontmatter, body string, err error)`:
  - 要求首行为 `---`,到下一个 `---` 为 frontmatter
  - frontmatter 用 `gopkg.in/yaml.v3` 或手写 JSON(为减少依赖,优先 yaml,因为 mcp 已引入)
- frontmatter 字段:`name, description, always, requires`

- [ ] **步骤 8.2:Requirements**

`requirements.go`:
```go
type Requirements struct {
    Bins []string `yaml:"bins"`
    Env  []string `yaml:"env"`
}
func (r Requirements) Check() (missing []string)
```
- `bins`:用 `exec.LookPath` 检查
- `env`:用 `os.Getenv` 非空
- 返回缺失列表,空切片表示全部满足

测试用 `t.Setenv` / 临时 PATH。

- [ ] **步骤 8.3:Loader**

`loader.go`:
- `type Loader interface { List() []Skill; LoadContent(name) (string, error); Summary(exclude []string) string; AlwaysSkills() []Skill }`
- `NewLoader(workspaceDir, builtinDir string, disabled []string) Loader`
- 扫描:`<dir>/skills/*/SKILL.md` 两个根,workspace 同名覆盖 builtin
- `disabled` 名单中的 skill 直接不出现在 List
- `Available = Requires.Check() == nil`
- `Summary`:按名字排序输出 `- <name> (path): <description> [always|unavailable]`,`exclude` 中的跳过
- `LoadContent`:读正文(懒加载,缓存)

- [ ] **步骤 8.4:单测**

用 `t.TempDir()` 构造两级目录:
- builtin 里 `a/SKILL.md` always=false
- workspace 里 `a/SKILL.md` always=true(断言 workspace 覆盖)
- builtin 里 `b/SKILL.md` requires.bins=["definitely-missing-xyz"] → Available=false
- `disabled=["a"]` → List 不含 a
- Summary 排序、包含 `[unavailable]`
- LoadContent 返回正文且去掉 frontmatter

- [ ] **步骤 8.5:验证 + 提交**

```bash
go test -race -cover ./internal/skills/
git add internal/skills/
git commit -m "feat(skills): add two-tier loader with nanobot-style requires"
```

coverage 目标 ≥ 80%。

---

## 任务 9:Phase 2 总验收

- [ ] **步骤 9.1:全量单测**

```bash
go test -race -cover ./...
```

预期:全部 PASS;各模块 coverage 满足各自目标。

- [ ] **步骤 9.2:vet + fmt**

```bash
gofmt -l .
go vet ./...
```

预期:无输出、无 warning。

- [ ] **步骤 9.3:更新 plan**

本文件中所有任务标题追加 ✅,所有 `- [ ]` 改为 `- [x]`。提交:
```bash
git add docs/superpowers/plans/2026-04-23-srebot-phase2-runtime.md
git commit -m "docs(plan): mark phase 2 plan complete"
```

- [ ] **步骤 9.4:打 tag**

```bash
git tag v0.2.0-phase2
```

---

## 风险与降级

| 风险 | 降级方案 |
|---|---|
| eino 公开 API 与文档不一致 | 先用 Context7 验证签名;若签名不同,优先以 godoc 为准,并在 PR 说明中记录 |
| mcp-go 不支持进程内测试 | manager_test 聚焦 config + 前缀命名,subprocess 路径走 `testing.Short` 以外的 integration tag |
| eino-ext openai httptest 路径不走 `BaseURL` | 若 eino 内部硬编码 openai host,退一步用真实 `http.RoundTripper` 拦截 |
| yaml 依赖被引入后架构洁癖 | 接受:mcp.yaml + SKILL.md frontmatter 均为 yaml,收益足够 |
| shell 测试在 CI 环境无 /bin/sh | 用 `runtime.GOOS == "windows"` skip;Linux/macOS 必跑 |

---

## 与 Phase 3 的衔接

Phase 2 结束后,下列接口可被 Phase 3 直接消费:

- `tools.Registry`、`tools.NewReadFile/NewWriteFile/NewShell/NewRemember`
- `approval.NewPolicy/NewCLIGate`
- `mcp.NewManager + ConnectAll + RegisterAllInto`(从 `config.MCPServers` 转换传入)
- `provider.NewOpenAICompatible` → `ChatModel`(即 eino 的 `model.ToolCallingChatModel`)
- `skills.NewLoader`

Phase 3 任务:`internal/agent`(eino Graph:InputAdapter → SystemPromptBuilder → AgentLoop → OutputAdapter)、`memory.Consolidator`(LLM 压缩)、`cmd/srebot`(CLI + REPL + `--config/--resume/--yolo/--verbose/--session-id`)、`presets/default/`(SRE 发行版 MD + skills + mcp.yaml)、`test/e2e/`。
