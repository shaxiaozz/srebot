package tools

import (
	"errors"
	"path/filepath"
	"strings"
)

// resolveInsideWorkspace resolves rel relative to workspace and ensures the
// result stays inside workspace. workspace must be absolute.
func resolveInsideWorkspace(workspace, rel string) (string, error) {
	if !filepath.IsAbs(workspace) {
		return "", errors.New("tools: workspace must be an absolute path")
	}
	if rel == "" {
		return "", errors.New("tools: path must not be empty")
	}
	if filepath.IsAbs(rel) {
		return "", errors.New("tools: path must be relative, not absolute")
	}

	clean := filepath.Clean(rel)
	joined := filepath.Join(workspace, clean)

	rel2, err := filepath.Rel(workspace, joined)
	if err != nil || strings.HasPrefix(rel2, "..") {
		return "", errors.New("tools: path escapes workspace")
	}

	return joined, nil
}
