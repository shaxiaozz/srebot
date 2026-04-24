package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Load 从 config.json 读取并返回合并了默认值、校验通过的 Resolved。
//
// 合并规则:先把 Defaults() 作为基线写入 Root,再用 JSON 覆盖 —— 因为
// json.Unmarshal 只覆盖 JSON 中出现的字段,这样天然实现"缺失字段沿用默认值"。
//
// Unix 下若文件权限宽于 0600,打印 warn 日志(不阻塞)。
func Load(path string) (*Resolved, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat config %s: %w", path, err)
	}
	if runtime.GOOS != "windows" {
		if mode := info.Mode().Perm(); mode&0o077 != 0 {
			slog.Warn("config file permissions too open; apiKey is plaintext",
				"path", path, "mode", fmt.Sprintf("%04o", mode))
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	root := Defaults()
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	res := root.Resolve()
	if res.Agent.Workspace, err = expandTilde(res.Agent.Workspace); err != nil {
		return nil, err
	}
	if err := res.Validate(); err != nil {
		return nil, err
	}
	return &res, nil
}

func expandTilde(p string) (string, error) {
	if !strings.HasPrefix(p, "~") {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home: %w", err)
	}
	if p == "~" {
		return home, nil
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}
