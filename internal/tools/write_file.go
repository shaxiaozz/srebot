package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type writeFileTool struct{ workspace string }

func (w *writeFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "write_file",
		Desc: "Write content to a file inside the agent workspace. Creates parent directories.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path":    {Type: schema.String, Desc: "relative path inside workspace", Required: true},
			"content": {Type: schema.String, Desc: "content to write", Required: true},
			"append":  {Type: schema.Boolean, Desc: "if true, append to file instead of overwriting"},
		}),
	}, nil
}

func (w *writeFileTool) InvokableRun(_ context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var a struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Append  bool   `json:"append"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &a); err != nil {
		return "", fmt.Errorf("write_file: invalid arguments: %w", err)
	}

	resolved, err := resolveInsideWorkspace(w.workspace, a.Path)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return "", fmt.Errorf("write_file: failed to create directories: %w", err)
	}

	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if a.Append {
		flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	}

	f, err := os.OpenFile(resolved, flags, 0644)
	if err != nil {
		return "", fmt.Errorf("write_file: failed to open file: %w", err)
	}
	defer f.Close()

	n, err := f.WriteString(a.Content)
	if err != nil {
		return "", fmt.Errorf("write_file: failed to write: %w", err)
	}

	return fmt.Sprintf("wrote %d bytes to %s", n, a.Path), nil
}

// NewWriteFile returns a workspace-scoped write_file tool.
func NewWriteFile(workspace string) tool.InvokableTool {
	return &writeFileTool{workspace: workspace}
}
