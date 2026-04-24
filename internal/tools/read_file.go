package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const maxReadSize = 1024 * 1024 // 1 MiB

type readFileTool struct{ workspace string }

func (r *readFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "read_file",
		Desc: "Read a UTF-8 text file from the agent workspace. Path must be relative to workspace root.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {Type: schema.String, Desc: "relative path inside workspace (must be relative to workspace root)", Required: true},
		}),
	}, nil
}

func (r *readFileTool) InvokableRun(_ context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &a); err != nil {
		return "", fmt.Errorf("read_file: invalid arguments: %w", err)
	}

	resolved, err := resolveInsideWorkspace(r.workspace, a.Path)
	if err != nil {
		return "", err
	}

	f, err := os.Open(resolved)
	if err != nil {
		return "", fmt.Errorf("read_file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxReadSize+1))
	if err != nil {
		return "", fmt.Errorf("read_file: %w", err)
	}
	if int64(len(data)) > maxReadSize {
		return "", fmt.Errorf("read_file: file too large (>1MiB)")
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("read_file: file is not valid UTF-8")
	}

	return string(data), nil
}

// NewReadFile returns a workspace-scoped read_file tool.
func NewReadFile(workspace string) tool.InvokableTool {
	return &readFileTool{workspace: workspace}
}
