package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/shaxiaozz/srebot/internal/memory"
)

type rememberTool struct{ store memory.Store }

func (r *rememberTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "remember",
		Desc: "Append a long-term fact to the agent's memory history.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"fact": {Type: schema.String, Desc: "the fact to remember", Required: true},
		}),
	}, nil
}

func (r *rememberTool) InvokableRun(_ context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var a struct {
		Fact string `json:"fact"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &a); err != nil {
		return "", fmt.Errorf("remember: invalid args: %w", err)
	}
	if a.Fact == "" {
		return "", fmt.Errorf("remember: fact must not be empty")
	}
	cursor, err := r.store.AppendHistory("[MEMO] " + a.Fact)
	if err != nil {
		return "", fmt.Errorf("remember: %w", err)
	}
	return fmt.Sprintf("remembered (cursor=%d)", cursor), nil
}

// NewRemember returns a tool that appends facts to the agent's memory.
func NewRemember(store memory.Store) tool.InvokableTool {
	return &rememberTool{store: store}
}
