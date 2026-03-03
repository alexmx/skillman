package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var nameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
var consecutiveHyphens = regexp.MustCompile(`--`)

type Frontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty"`
}

type Skill struct {
	Frontmatter Frontmatter
	Body        string
	Dir         string
}

func Parse(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading SKILL.md: %w", err)
	}
	return ParseContent(string(data), filepath.Dir(path))
}

func ParseContent(content, dir string) (*Skill, error) {
	fm, body, err := extractFrontmatter(content)
	if err != nil {
		return nil, err
	}

	var frontmatter Frontmatter
	if err := yaml.Unmarshal([]byte(fm), &frontmatter); err != nil {
		return nil, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}

	return &Skill{
		Frontmatter: frontmatter,
		Body:        body,
		Dir:         dir,
	}, nil
}

func extractFrontmatter(content string) (frontmatter, body string, err error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return "", "", fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
	}

	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		return "", "", fmt.Errorf("SKILL.md frontmatter is not closed (missing closing ---)")
	}

	frontmatter = strings.TrimSpace(rest[:idx])
	body = strings.TrimSpace(rest[idx+4:])
	return frontmatter, body, nil
}

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func Validate(s *Skill) []ValidationError {
	var errs []ValidationError

	// Name validation
	if s.Frontmatter.Name == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "is required"})
	} else {
		name := s.Frontmatter.Name
		if len(name) > 64 {
			errs = append(errs, ValidationError{Field: "name", Message: "must be at most 64 characters"})
		}
		if !nameRegex.MatchString(name) {
			errs = append(errs, ValidationError{Field: "name", Message: "must contain only lowercase letters, numbers, and hyphens, and must not start or end with a hyphen"})
		}
		if consecutiveHyphens.MatchString(name) {
			errs = append(errs, ValidationError{Field: "name", Message: "must not contain consecutive hyphens"})
		}
	}

	// Description validation
	if s.Frontmatter.Description == "" {
		errs = append(errs, ValidationError{Field: "description", Message: "is required"})
	} else if len(s.Frontmatter.Description) > 1024 {
		errs = append(errs, ValidationError{Field: "description", Message: "must be at most 1024 characters"})
	}

	// Compatibility validation
	if s.Frontmatter.Compatibility != "" && len(s.Frontmatter.Compatibility) > 500 {
		errs = append(errs, ValidationError{Field: "compatibility", Message: "must be at most 500 characters"})
	}

	// Directory name must match skill name
	if s.Dir != "" && s.Frontmatter.Name != "" {
		dirName := filepath.Base(s.Dir)
		if dirName != s.Frontmatter.Name {
			errs = append(errs, ValidationError{
				Field:   "name",
				Message: fmt.Sprintf("must match directory name (skill name %q != directory %q)", s.Frontmatter.Name, dirName),
			})
		}
	}

	return errs
}

func LoadFromDir(dir string) (*Skill, error) {
	skillFile := filepath.Join(dir, "SKILL.md")
	return Parse(skillFile)
}
