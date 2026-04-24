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
