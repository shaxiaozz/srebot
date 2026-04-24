package skills

import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Skill is a loaded skill descriptor. The body is lazy-loaded on demand via
// Loader.LoadContent; Skill values exposed via Loader.List do not carry body.
type Skill struct {
	Name        string
	Description string
	Path        string // absolute path to SKILL.md
	Dir         string // absolute path to the skill directory
	Always      bool
	Requires    Requirements
	Available   bool
}

type frontmatter struct {
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Always      bool         `yaml:"always"`
	Requires    Requirements `yaml:"requires"`
}

// parseSKILL splits a SKILL.md into frontmatter and body. The file MUST start
// with a "---" line, followed by YAML, a closing "---" line, and the body.
func parseSKILL(raw []byte) (frontmatter, []byte, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return frontmatter{}, nil, errors.New("empty skill file")
	}

	// Normalize line endings to simplify delimiter detection.
	normalized := bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))

	lines := bytes.SplitN(normalized, []byte("\n"), 2)
	if len(lines) < 2 || string(bytes.TrimRight(lines[0], " \t")) != "---" {
		return frontmatter{}, nil, errors.New("missing opening frontmatter delimiter")
	}

	rest := lines[1]
	// Find the next "---" on its own line.
	var fmBuf bytes.Buffer
	var body []byte
	found := false
	remaining := rest
	for {
		nl := bytes.IndexByte(remaining, '\n')
		if nl == -1 {
			// Last line without trailing newline.
			if string(bytes.TrimRight(remaining, " \t")) == "---" {
				found = true
				body = nil
			} else {
				fmBuf.Write(remaining)
			}
			break
		}
		line := remaining[:nl]
		if string(bytes.TrimRight(line, " \t")) == "---" {
			found = true
			body = remaining[nl+1:]
			break
		}
		fmBuf.Write(line)
		fmBuf.WriteByte('\n')
		remaining = remaining[nl+1:]
	}

	if !found {
		return frontmatter{}, nil, errors.New("missing closing frontmatter delimiter")
	}

	var fm frontmatter
	if err := yaml.Unmarshal(fmBuf.Bytes(), &fm); err != nil {
		return frontmatter{}, nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	return fm, body, nil
}
