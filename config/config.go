// Package config 定义并加载 srebot 的 config.json 运行时配置。
//
// 结构:
//
//	Root = JSON 顶层
//	  agents.defaults: 每个 agent 可独立覆盖的 LLM/工作区参数
//	  approval/logging/memory: 全局运行时配置(不随 agent 切换)
//	Resolved = 供下游组件使用的平面视图(MVP 只用 agents.defaults)
package config

import (
	"fmt"
	"strings"
)

// Root 映射 config.json 顶层结构。
type Root struct {
	Agents     AgentsSection        `json:"agents"`
	Approval   ApprovalConfig       `json:"approval"`
	Logging    LoggingConfig        `json:"logging"`
	Memory     MemoryConfig         `json:"memory"`
	MCPServers map[string]MCPServer `json:"mcpServers,omitempty"`
}

// MCPServer 描述单个 MCP server 的连接配置。url 与 command 二选一:
// 设置 url 走 Streamable HTTP transport;设置 command/args/env 走 stdio transport。
type MCPServer struct {
	URL         string            `json:"url,omitempty"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	ToolTimeout int               `json:"toolTimeout,omitempty"`
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
	Agent      AgentConfig
	Approval   ApprovalConfig
	Logging    LoggingConfig
	Memory     MemoryConfig
	MCPServers map[string]MCPServer
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
		Agent:      r.Agents.Defaults,
		Approval:   r.Approval,
		Logging:    r.Logging,
		Memory:     r.Memory,
		MCPServers: r.MCPServers,
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
	for name, srv := range r.MCPServers {
		hasURL := strings.TrimSpace(srv.URL) != ""
		hasCmd := strings.TrimSpace(srv.Command) != ""
		switch {
		case !hasURL && !hasCmd:
			errs = append(errs, fmt.Sprintf("mcp server %q: must set either url or command", name))
		case hasURL && hasCmd:
			errs = append(errs, fmt.Sprintf("mcp server %q: url and command are mutually exclusive", name))
		case hasURL:
			if !strings.HasPrefix(srv.URL, "http://") && !strings.HasPrefix(srv.URL, "https://") {
				errs = append(errs, fmt.Sprintf("mcp server %q: url must start with http:// or https://", name))
			}
		}
		if srv.ToolTimeout < 0 {
			errs = append(errs, fmt.Sprintf("mcp server %q: toolTimeout must be >= 0", name))
		} else if srv.ToolTimeout == 0 {
			srv.ToolTimeout = 60
			r.MCPServers[name] = srv
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("config invalid: %s", strings.Join(errs, "; "))
	}
	return nil
}
