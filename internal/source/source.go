package source

import (
	"fmt"
	"strings"
)

// FetchResult represents a discovered skill from a source.
type FetchResult struct {
	Name      string // skill name from frontmatter
	SourceDir string // absolute path to the skill directory (temp or local)
	Source    string // source identifier (e.g. "github.com/org/repo/skill" or "local")
	Ref       string // git ref (tag, branch, commit) or empty for local
	CommitSHA string // resolved commit SHA or empty for local
	IsLocal   bool
}

// ParsedRef holds a parsed skill reference like "github.com/org/repo/skill@v1.0".
type ParsedRef struct {
	Raw     string
	Source  string // "github.com/org/repo" or "github.com/org/repo/subdir"
	Ref     string // "v1.0", "abc123", "" (latest)
	IsLocal bool
}

func ParseRef(raw string) ParsedRef {
	if isLocalRef(raw) {
		return ParsedRef{Raw: raw, Source: raw, IsLocal: true}
	}

	source := raw
	ref := ""
	if idx := strings.LastIndex(raw, "@"); idx != -1 {
		source = raw[:idx]
		ref = raw[idx+1:]
	}

	return ParsedRef{
		Raw:    raw,
		Source: source,
		Ref:    ref,
	}
}

func isLocalRef(s string) bool {
	return s == "." || s == ".." ||
		strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "../") ||
		strings.HasPrefix(s, "/")
}

// ParseGitHubSource extracts owner, repo, and optional subpath from a GitHub source.
// Example: "github.com/anthropics/skills/pdf" -> ("anthropics", "skills", "pdf")
func ParseGitHubSource(source string) (owner, repo, subpath string, err error) {
	source = strings.TrimPrefix(source, "github.com/")
	parts := strings.SplitN(source, "/", 3)
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid GitHub source: need at least owner/repo")
	}
	owner = parts[0]
	repo = parts[1]
	if len(parts) == 3 {
		subpath = parts[2]
	}
	return owner, repo, subpath, nil
}
