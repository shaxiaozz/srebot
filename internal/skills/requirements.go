package skills

import (
	"os"
	"os/exec"
)

// Requirements expresses a skill's runtime dependencies.
// Bins are executables that must exist on PATH; Env are environment variables
// that must be set to a non-empty value.
type Requirements struct {
	Bins []string `yaml:"bins"`
	Env  []string `yaml:"env"`
}

// Check returns a list of missing requirements encoded as "bin:NAME" or
// "env:NAME". An empty (nil) slice means all requirements are satisfied.
func (r Requirements) Check() []string {
	var missing []string
	for _, b := range r.Bins {
		if _, err := exec.LookPath(b); err != nil {
			missing = append(missing, "bin:"+b)
		}
	}
	for _, e := range r.Env {
		if os.Getenv(e) == "" {
			missing = append(missing, "env:"+e)
		}
	}
	return missing
}
