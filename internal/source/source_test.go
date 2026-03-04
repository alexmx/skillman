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

// ---------------------------------------------------------------------------
// normalizeSource
// ---------------------------------------------------------------------------

func TestNormalizeSource(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		// https:// prefix stripped
		{"https://github.com/org/repo", "github.com/org/repo"},
		// http:// prefix stripped
		{"http://github.com/org/repo", "github.com/org/repo"},
		// .git suffix stripped
		{"github.com/org/repo.git", "github.com/org/repo"},
		// https:// + .git combination
		{"https://github.com/org/repo.git", "github.com/org/repo"},
		// http:// + .git combination
		{"http://github.com/org/repo.git", "github.com/org/repo"},
		// no prefix or suffix — unchanged
		{"github.com/org/repo", "github.com/org/repo"},
		// bare string with no recognised prefix — unchanged
		{"org/repo", "org/repo"},
		// .git in the middle of a path is not stripped
		{"github.com/org/repo.git/subdir", "github.com/org/repo.git/subdir"},
		// empty string stays empty
		{"", ""},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeSource(tc.input)
			if got != tc.want {
				t.Errorf("normalizeSource(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isLocalRef
// ---------------------------------------------------------------------------

func TestIsLocalRef(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		// canonical local indicators
		{".", true},
		{"..", true},
		// dot-slash prefix
		{"./skill", true},
		{"./", true},
		// dot-dot-slash prefix
		{"../skill", true},
		{"../../skill", true},
		// absolute paths
		{"/absolute/path", true},
		{"/", true},
		// non-local — bare names and GitHub-style refs
		{"github.com/org/repo", false},
		{"org/repo", false},
		{"myskill", false},
		// a string starting with a dot but not ./ or .. is not local
		{".hidden", false},
		// http/https URLs are not local
		{"https://github.com/org/repo", false},
		{"http://github.com/org/repo", false},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := isLocalRef(tc.input)
			if got != tc.want {
				t.Errorf("isLocalRef(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ParseRef — additional cases not covered by the existing tests
// ---------------------------------------------------------------------------

func TestParseRef_RawFieldPreserved(t *testing.T) {
	cases := []string{
		"./local-skill",
		"github.com/org/repo@v2.0",
		"https://github.com/org/repo.git",
		".",
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got := ParseRef(input)
			if got.Raw != input {
				t.Errorf("ParseRef(%q).Raw = %q, want %q", input, got.Raw, input)
			}
		})
	}
}

func TestParseRef_LocalRefs(t *testing.T) {
	cases := []struct {
		input      string
		wantSource string
	}{
		{".", "."},
		{"..", ".."},
		{"./my-skill", "./my-skill"},
		{"../other-skill", "../other-skill"},
		{"/absolute/path/to/skill", "/absolute/path/to/skill"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := ParseRef(tc.input)
			if !got.IsLocal {
				t.Errorf("ParseRef(%q).IsLocal = false, want true", tc.input)
			}
			if got.Source != tc.wantSource {
				t.Errorf("ParseRef(%q).Source = %q, want %q", tc.input, got.Source, tc.wantSource)
			}
			if got.Ref != "" {
				t.Errorf("ParseRef(%q).Ref = %q, want empty", tc.input, got.Ref)
			}
		})
	}
}

func TestParseRef_HTTPPrefix(t *testing.T) {
	got := ParseRef("http://github.com/org/repo@main")
	if got.Source != "github.com/org/repo" {
		t.Errorf("source = %q, want %q", got.Source, "github.com/org/repo")
	}
	if got.Ref != "main" {
		t.Errorf("ref = %q, want %q", got.Ref, "main")
	}
	if got.IsLocal {
		t.Errorf("IsLocal = true, want false")
	}
}

func TestParseRef_GitSuffixNoHTTPS(t *testing.T) {
	got := ParseRef("github.com/org/repo.git")
	if got.Source != "github.com/org/repo" {
		t.Errorf("source = %q, want %q", got.Source, "github.com/org/repo")
	}
	if got.Ref != "" {
		t.Errorf("ref = %q, want empty", got.Ref)
	}
}

func TestParseRef_HTTPSAndGitSuffix(t *testing.T) {
	// When a @ref is present, normalizeSource runs on the full string before
	// splitting on '@'. TrimSuffix(".git") only matches when ".git" is at the
	// very end, so "repo.git@v3.1" retains the ".git" in the source segment.
	// Only a bare ".git" tail (no @ref) gets stripped.
	got := ParseRef("https://github.com/org/repo.git@v3.1")
	if got.Source != "github.com/org/repo.git" {
		t.Errorf("source = %q, want %q", got.Source, "github.com/org/repo.git")
	}
	if got.Ref != "v3.1" {
		t.Errorf("ref = %q, want %q", got.Ref, "v3.1")
	}
}

func TestParseRef_HTTPSAndGitSuffixNoRef(t *testing.T) {
	// Without a @ref, the ".git" suffix IS at the end and gets stripped.
	got := ParseRef("https://github.com/org/repo.git")
	if got.Source != "github.com/org/repo" {
		t.Errorf("source = %q, want %q", got.Source, "github.com/org/repo")
	}
	if got.Ref != "" {
		t.Errorf("ref = %q, want empty", got.Ref)
	}
}

func TestParseRef_CommitSHARef(t *testing.T) {
	sha := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	input := "github.com/org/repo@" + sha
	got := ParseRef(input)
	if got.Source != "github.com/org/repo" {
		t.Errorf("source = %q, want %q", got.Source, "github.com/org/repo")
	}
	if got.Ref != sha {
		t.Errorf("ref = %q, want %q", got.Ref, sha)
	}
}

func TestParseRef_TrailingAtSign(t *testing.T) {
	// A trailing '@' means the ref portion is an empty string.
	got := ParseRef("github.com/org/repo@")
	if got.Source != "github.com/org/repo" {
		t.Errorf("source = %q, want %q", got.Source, "github.com/org/repo")
	}
	if got.Ref != "" {
		t.Errorf("ref = %q, want empty string", got.Ref)
	}
}

func TestParseRef_MultipleAtSigns(t *testing.T) {
	// LastIndex behaviour: split at the last '@'.
	got := ParseRef("github.com/org/repo@some@ref")
	if got.Source != "github.com/org/repo@some" {
		t.Errorf("source = %q, want %q", got.Source, "github.com/org/repo@some")
	}
	if got.Ref != "ref" {
		t.Errorf("ref = %q, want %q", got.Ref, "ref")
	}
}

func TestParseRef_SubpathWithRef(t *testing.T) {
	got := ParseRef("github.com/org/repo/subdir@feature-branch")
	if got.Source != "github.com/org/repo/subdir" {
		t.Errorf("source = %q, want %q", got.Source, "github.com/org/repo/subdir")
	}
	if got.Ref != "feature-branch" {
		t.Errorf("ref = %q, want %q", got.Ref, "feature-branch")
	}
	if got.IsLocal {
		t.Errorf("IsLocal = true, want false")
	}
}

// ---------------------------------------------------------------------------
// ParseGitHubSource — additional edge cases
// ---------------------------------------------------------------------------

func TestParseGitHubSource_BareOwnerRepo(t *testing.T) {
	// Input without the "github.com/" prefix — treated as bare owner/repo.
	owner, repo, subpath, err := ParseGitHubSource("myorg/myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "myorg" {
		t.Errorf("owner = %q, want %q", owner, "myorg")
	}
	if repo != "myrepo" {
		t.Errorf("repo = %q, want %q", repo, "myrepo")
	}
	if subpath != "" {
		t.Errorf("subpath = %q, want empty", subpath)
	}
}

func TestParseGitHubSource_EmptyString(t *testing.T) {
	_, _, _, err := ParseGitHubSource("")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestParseGitHubSource_OnlyOwner(t *testing.T) {
	_, _, _, err := ParseGitHubSource("github.com/onlyowner")
	if err == nil {
		t.Error("expected error when repo is missing, got nil")
	}
}

func TestParseGitHubSource_DeepSubpath(t *testing.T) {
	owner, repo, subpath, err := ParseGitHubSource("github.com/org/repo/a/b/c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "org" {
		t.Errorf("owner = %q, want %q", owner, "org")
	}
	if repo != "repo" {
		t.Errorf("repo = %q, want %q", repo, "repo")
	}
	// SplitN(..., 3) collapses everything after the second slash into subpath.
	if subpath != "a/b/c" {
		t.Errorf("subpath = %q, want %q", subpath, "a/b/c")
	}
}

func TestParseGitHubSource_ErrorMessage(t *testing.T) {
	_, _, _, err := ParseGitHubSource("github.com/solo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := "invalid GitHub source: need at least owner/repo"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
