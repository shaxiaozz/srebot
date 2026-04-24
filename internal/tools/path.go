package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// resolveInsideWorkspace resolves rel relative to workspace and ensures the
// result stays inside workspace. workspace must be absolute.
func resolveInsideWorkspace(workspace, rel string) (string, error) {
	if rel == "" {
		return "", errors.New("tools: empty path")
	}
	if filepath.IsAbs(rel) {
		return "", errors.New("tools: absolute path not allowed")
	}
	if !filepath.IsAbs(workspace) {
		return "", errors.New("tools: workspace must be absolute")
	}

	// Canonicalize workspace (resolve symlinks); if ws doesn't exist yet, fall back.
	wsReal, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		wsReal = workspace
	}

	clean := filepath.Clean(rel)
	joined := filepath.Join(wsReal, clean)

	// Lexical check
	relLex, err := filepath.Rel(wsReal, joined)
	if err != nil || relLex == ".." || strings.HasPrefix(relLex, ".."+string(filepath.Separator)) {
		return "", errors.New("tools: path escapes workspace")
	}

	// Symlink check: walk up to the deepest existing ancestor and EvalSymlinks it.
	ancestor := joined
	for {
		if _, statErr := os.Lstat(ancestor); statErr == nil {
			break
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			break // reached root
		}
		ancestor = parent
	}
	realAncestor, err := filepath.EvalSymlinks(ancestor)
	if err == nil {
		relReal, err2 := filepath.Rel(wsReal, realAncestor)
		if err2 != nil || relReal == ".." || strings.HasPrefix(relReal, ".."+string(filepath.Separator)) {
			return "", errors.New("tools: path escapes workspace via symlink")
		}
	}

	return joined, nil
}
