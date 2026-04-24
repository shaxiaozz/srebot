package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Loader provides access to the two-tier skills index.
type Loader interface {
	List() []Skill
	LoadContent(name string) (string, error)
	Summary(exclude []string) string
	AlwaysSkills() []Skill
}

type fsLoader struct {
	mu       sync.RWMutex
	skills   map[string]*Skill
	ordered  []string
	contents map[string]string
}

// NewLoader scans both directories at construction time. workspaceDir entries
// override builtinDir entries with the same name. disabled names are excluded
// from the index entirely. Either dir may be empty or non-existent.
func NewLoader(workspaceDir, builtinDir string, disabled []string) (Loader, error) {
	l := &fsLoader{
		skills:   make(map[string]*Skill),
		contents: make(map[string]string),
	}

	disabledSet := make(map[string]struct{}, len(disabled))
	for _, n := range disabled {
		disabledSet[n] = struct{}{}
	}

	// Builtin first, workspace second so workspace overwrites.
	for _, dir := range []string{builtinDir, workspaceDir} {
		if dir == "" {
			continue
		}
		if err := l.scanTier(dir, disabledSet); err != nil {
			return nil, err
		}
	}

	// Compute availability.
	for _, s := range l.skills {
		s.Available = len(s.Requires.Check()) == 0
	}

	// Build sorted order for stable iteration.
	l.ordered = make([]string, 0, len(l.skills))
	for n := range l.skills {
		l.ordered = append(l.ordered, n)
	}
	sort.Strings(l.ordered)

	return l, nil
}

func (l *fsLoader) scanTier(dir string, disabled map[string]struct{}) error {
	pattern := filepath.Join(dir, "skills", "*", "SKILL.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("skills: glob %q: %w", pattern, err)
	}
	for _, path := range matches {
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skills: abs %q: %v\n", path, err)
			continue
		}
		raw, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skills: read %q: %v\n", absPath, err)
			continue
		}
		fm, _, err := parseSKILL(raw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skills: parse %q: %v\n", absPath, err)
			continue
		}
		if fm.Name == "" {
			fmt.Fprintf(os.Stderr, "skills: %q missing name\n", absPath)
			continue
		}
		if _, disabledHit := disabled[fm.Name]; disabledHit {
			continue
		}
		l.skills[fm.Name] = &Skill{
			Name:        fm.Name,
			Description: fm.Description,
			Path:        absPath,
			Dir:         filepath.Dir(absPath),
			Always:      fm.Always,
			Requires:    fm.Requires,
		}
	}
	return nil
}

func (l *fsLoader) List() []Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Skill, 0, len(l.ordered))
	for _, n := range l.ordered {
		out = append(out, *l.skills[n])
	}
	return out
}

func (l *fsLoader) AlwaysSkills() []Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var out []Skill
	for _, n := range l.ordered {
		s := l.skills[n]
		if s.Always && s.Available {
			out = append(out, *s)
		}
	}
	return out
}

func (l *fsLoader) LoadContent(name string) (string, error) {
	l.mu.RLock()
	if c, ok := l.contents[name]; ok {
		l.mu.RUnlock()
		return c, nil
	}
	s, ok := l.skills[name]
	l.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("skills: unknown skill %q", name)
	}
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		return "", fmt.Errorf("skills: read %q: %w", name, err)
	}
	_, body, err := parseSKILL(raw)
	if err != nil {
		return "", fmt.Errorf("skills: parse %q: %w", name, err)
	}
	bs := string(body)
	l.mu.Lock()
	l.contents[name] = bs
	l.mu.Unlock()
	return bs, nil
}

func (l *fsLoader) Summary(exclude []string) string {
	skip := make(map[string]struct{}, len(exclude))
	for _, n := range exclude {
		skip[n] = struct{}{}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	var b strings.Builder
	for _, n := range l.ordered {
		if _, ex := skip[n]; ex {
			continue
		}
		s := l.skills[n]
		fmt.Fprintf(&b, "- %s (%s): %s", s.Name, s.Path, s.Description)
		if s.Always {
			b.WriteString(" [always]")
		}
		if !s.Available {
			b.WriteString(" [unavailable]")
		}
		b.WriteByte('\n')
	}
	return b.String()
}
