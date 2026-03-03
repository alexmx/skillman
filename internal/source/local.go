package source

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexmx/skillman/internal/skill"
)

// FetchLocal validates a local skill directory and returns a FetchResult.
func FetchLocal(path string) (*FetchResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", absPath)
	}

	s, err := skill.LoadFromDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("loading skill: %w", err)
	}

	errs := skill.Validate(s)
	if len(errs) > 0 {
		msg := fmt.Sprintf("skill %q has validation errors:\n", s.Frontmatter.Name)
		for _, e := range errs {
			msg += fmt.Sprintf("  - %s\n", e)
		}
		return nil, fmt.Errorf("%s", msg)
	}

	return &FetchResult{
		Name:      s.Frontmatter.Name,
		SourceDir: absPath,
		Source:    "local",
		IsLocal:   true,
	}, nil
}
