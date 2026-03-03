package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseContent(t *testing.T) {
	content := `---
name: test-skill
description: A test skill for unit testing.
license: MIT
metadata:
  author: test
  version: "1.0"
---

# Test Skill

Instructions here.
`
	s, err := ParseContent(content, "/tmp/test-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Frontmatter.Name != "test-skill" {
		t.Errorf("name = %q, want %q", s.Frontmatter.Name, "test-skill")
	}
	if s.Frontmatter.Description != "A test skill for unit testing." {
		t.Errorf("description = %q, want %q", s.Frontmatter.Description, "A test skill for unit testing.")
	}
	if s.Frontmatter.License != "MIT" {
		t.Errorf("license = %q, want %q", s.Frontmatter.License, "MIT")
	}
	if s.Frontmatter.Metadata["author"] != "test" {
		t.Errorf("metadata.author = %q, want %q", s.Frontmatter.Metadata["author"], "test")
	}
	if !strings.Contains(s.Body, "# Test Skill") {
		t.Errorf("body should contain '# Test Skill'")
	}
}

func TestParseContent_NoFrontmatter(t *testing.T) {
	_, err := ParseContent("# Just markdown", "/tmp/test")
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestParseContent_UnclosedFrontmatter(t *testing.T) {
	_, err := ParseContent("---\nname: test\n", "/tmp/test")
	if err == nil {
		t.Error("expected error for unclosed frontmatter")
	}
}

func TestValidate_Valid(t *testing.T) {
	s := &Skill{
		Frontmatter: Frontmatter{
			Name:        "my-skill",
			Description: "A valid skill for testing purposes.",
		},
		Dir: "/tmp/my-skill",
	}

	errs := Validate(s)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_NameRequired(t *testing.T) {
	s := &Skill{
		Frontmatter: Frontmatter{Description: "has description"},
	}
	errs := Validate(s)
	hasNameErr := false
	for _, e := range errs {
		if e.Field == "name" && strings.Contains(e.Message, "required") {
			hasNameErr = true
		}
	}
	if !hasNameErr {
		t.Error("expected name required error")
	}
}

func TestValidate_NameFormat(t *testing.T) {
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"valid-name", false},
		{"a", false},
		{"my-skill-123", false},
		{"PDF-Processing", true},  // uppercase
		{"-starts-hyphen", true},  // starts with hyphen
		{"ends-hyphen-", true},    // ends with hyphen
		{"double--hyphen", true},  // consecutive hyphens
		{strings.Repeat("a", 65), true}, // too long
	}

	for _, tc := range cases {
		s := &Skill{
			Frontmatter: Frontmatter{
				Name:        tc.name,
				Description: "test",
			},
			Dir: "/tmp/" + tc.name,
		}
		errs := Validate(s)
		hasErr := false
		for _, e := range errs {
			if e.Field == "name" {
				hasErr = true
			}
		}
		if hasErr != tc.wantErr {
			t.Errorf("name %q: hasErr=%v, wantErr=%v", tc.name, hasErr, tc.wantErr)
		}
	}
}

func TestValidate_DescriptionRequired(t *testing.T) {
	s := &Skill{
		Frontmatter: Frontmatter{Name: "test"},
		Dir:         "/tmp/test",
	}
	errs := Validate(s)
	hasDescErr := false
	for _, e := range errs {
		if e.Field == "description" {
			hasDescErr = true
		}
	}
	if !hasDescErr {
		t.Error("expected description required error")
	}
}

func TestValidate_DescriptionTooLong(t *testing.T) {
	s := &Skill{
		Frontmatter: Frontmatter{
			Name:        "test",
			Description: strings.Repeat("a", 1025),
		},
		Dir: "/tmp/test",
	}
	errs := Validate(s)
	hasDescErr := false
	for _, e := range errs {
		if e.Field == "description" && strings.Contains(e.Message, "1024") {
			hasDescErr = true
		}
	}
	if !hasDescErr {
		t.Error("expected description too long error")
	}
}

func TestValidate_DirectoryNameMismatch(t *testing.T) {
	s := &Skill{
		Frontmatter: Frontmatter{
			Name:        "my-skill",
			Description: "test",
		},
		Dir: "/tmp/wrong-name",
	}
	errs := Validate(s)
	hasDirErr := false
	for _, e := range errs {
		if strings.Contains(e.Message, "directory") {
			hasDirErr = true
		}
	}
	if !hasDirErr {
		t.Error("expected directory name mismatch error")
	}
}

func TestValidate_CompatibilityTooLong(t *testing.T) {
	s := &Skill{
		Frontmatter: Frontmatter{
			Name:          "test",
			Description:   "test",
			Compatibility: strings.Repeat("a", 501),
		},
		Dir: "/tmp/test",
	}
	errs := Validate(s)
	hasErr := false
	for _, e := range errs {
		if e.Field == "compatibility" {
			hasErr = true
		}
	}
	if !hasErr {
		t.Error("expected compatibility too long error")
	}
}

func TestParse_FromFile(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")
	os.MkdirAll(skillDir, 0o755)

	content := `---
name: my-skill
description: Test skill from file.
---

# My Skill
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644)

	s, err := Parse(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Frontmatter.Name != "my-skill" {
		t.Errorf("name = %q, want %q", s.Frontmatter.Name, "my-skill")
	}
}
