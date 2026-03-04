package source

import (
	"testing"
)

func TestParseRef_Local(t *testing.T) {
	cases := []struct {
		input   string
		isLocal bool
	}{
		{"./my-skill", true},
		{"../my-skill", true},
		{"/absolute/path", true},
		{".", true},
		{"..", true},
		{"github.com/org/repo", false},
		{"github.com/org/repo@v1.0", false},
	}

	for _, tc := range cases {
		ref := ParseRef(tc.input)
		if ref.IsLocal != tc.isLocal {
			t.Errorf("ParseRef(%q).IsLocal = %v, want %v", tc.input, ref.IsLocal, tc.isLocal)
		}
	}
}

func TestParseRef_GitHubWithRef(t *testing.T) {
	ref := ParseRef("github.com/anthropics/skills/pdf@v1.0")
	if ref.Source != "github.com/anthropics/skills/pdf" {
		t.Errorf("source = %q, want %q", ref.Source, "github.com/anthropics/skills/pdf")
	}
	if ref.Ref != "v1.0" {
		t.Errorf("ref = %q, want %q", ref.Ref, "v1.0")
	}
}

func TestParseRef_GitHubNoRef(t *testing.T) {
	ref := ParseRef("github.com/anthropics/skills")
	if ref.Source != "github.com/anthropics/skills" {
		t.Errorf("source = %q", ref.Source)
	}
	if ref.Ref != "" {
		t.Errorf("ref = %q, want empty", ref.Ref)
	}
}

func TestParseRef_HTTPSPrefix(t *testing.T) {
	ref := ParseRef("https://github.com/anthropics/skills/pdf@v1.0")
	if ref.Source != "github.com/anthropics/skills/pdf" {
		t.Errorf("source = %q, want %q", ref.Source, "github.com/anthropics/skills/pdf")
	}
	if ref.Ref != "v1.0" {
		t.Errorf("ref = %q, want %q", ref.Ref, "v1.0")
	}
}

func TestParseRef_GitSuffix(t *testing.T) {
	ref := ParseRef("https://github.com/anthropics/skills.git")
	if ref.Source != "github.com/anthropics/skills" {
		t.Errorf("source = %q, want %q", ref.Source, "github.com/anthropics/skills")
	}
}

func TestParseGitHubSource(t *testing.T) {
	cases := []struct {
		input   string
		owner   string
		repo    string
		subpath string
		wantErr bool
	}{
		{"github.com/anthropics/skills", "anthropics", "skills", "", false},
		{"github.com/anthropics/skills/pdf", "anthropics", "skills", "pdf", false},
		{"github.com/org/repo/sub/path", "org", "repo", "sub/path", false},
		{"github.com/solo", "", "", "", true},
	}

	for _, tc := range cases {
		owner, repo, subpath, err := ParseGitHubSource(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseGitHubSource(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr {
			if owner != tc.owner {
				t.Errorf("owner = %q, want %q", owner, tc.owner)
			}
			if repo != tc.repo {
				t.Errorf("repo = %q, want %q", repo, tc.repo)
			}
			if subpath != tc.subpath {
				t.Errorf("subpath = %q, want %q", subpath, tc.subpath)
			}
		}
	}
}
