package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Tool is an alias for eino's InvokableTool so callers import one package.
type Tool = tool.InvokableTool

// Registry stores and retrieves tools by name.
type Registry interface {
	Register(t Tool) error
	Unregister(name string)
	Get(name string) (Tool, bool)
	Definitions(ctx context.Context) ([]*schema.ToolInfo, error)
}

type memRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry returns a new in-memory Registry.
func NewRegistry() Registry {
	return &memRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds t to the registry. Returns an error if the tool name is empty
// or already registered.
func (r *memRegistry) Register(t Tool) error {
	info, err := t.Info(context.Background())
	if err != nil {
		return fmt.Errorf("tools: failed to get tool info: %w", err)
	}
	if info.Name == "" {
		return fmt.Errorf("tools: tool name must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[info.Name]; exists {
		return fmt.Errorf("tools: tool %q is already registered", info.Name)
	}
	r.tools[info.Name] = t
	return nil
}

// Unregister removes the tool with the given name. No-op if not found.
func (r *memRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// Get returns the tool registered under name.
func (r *memRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Definitions returns ToolInfo for every registered tool, sorted by name.
func (r *memRegistry) Definitions(ctx context.Context) ([]*schema.ToolInfo, error) {
	r.mu.RLock()
	snapshot := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		snapshot = append(snapshot, t)
	}
	r.mu.RUnlock()

	infos := make([]*schema.ToolInfo, 0, len(snapshot))
	for _, t := range snapshot {
		info, err := t.Info(ctx)
		if err != nil {
			return nil, fmt.Errorf("tools: failed to get tool info: %w", err)
		}
		infos = append(infos, info)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos, nil
}
