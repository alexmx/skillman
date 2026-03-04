package store

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/alexmx/skillman/internal/config"
)

// newTestStore creates a Store whose root is a fresh temp directory.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	return &Store{Root: root}
}

// TestNew verifies that New sets Root from the config's resolved store path.
func TestNew(t *testing.T) {
	t.Run("uses explicit StorePath from config", func(t *testing.T) {
		dir := t.TempDir()
		cfg := config.Config{StorePath: dir}
		s := New(cfg)
		if s.Root != dir {
			t.Errorf("Root = %q, want %q", s.Root, dir)
		}
	})

	t.Run("falls back to default store path when StorePath is empty", func(t *testing.T) {
		cfg := config.Config{}
		s := New(cfg)
		want := config.DefaultStorePath()
		if s.Root != want {
			t.Errorf("Root = %q, want %q", s.Root, want)
		}
	})
}

// TestInit verifies that Init creates the expected directory structure.
func TestInit(t *testing.T) {
	s := newTestStore(t)

	if err := s.Init(); err != nil {
		t.Fatalf("Init() returned unexpected error: %v", err)
	}

	expectedDirs := []string{
		s.Root,
		filepath.Join(s.Root, "github.com"),
		filepath.Join(s.Root, "local"),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("expected directory %q to exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q exists but is not a directory", dir)
		}
	}
}

// TestInit_Idempotent verifies that calling Init twice does not fail.
func TestInit_Idempotent(t *testing.T) {
	s := newTestStore(t)

	if err := s.Init(); err != nil {
		t.Fatalf("first Init() failed: %v", err)
	}
	if err := s.Init(); err != nil {
		t.Fatalf("second Init() failed: %v", err)
	}
}

// TestLocalPath verifies that LocalPath returns the correct path.
func TestLocalPath(t *testing.T) {
	s := newTestStore(t)

	got := s.LocalPath("my-skill")
	want := filepath.Join(s.Root, "local", "my-skill")
	if got != want {
		t.Errorf("LocalPath(%q) = %q, want %q", "my-skill", got, want)
	}
}

// TestLocalPath_Nested verifies path construction when the name contains slashes.
func TestLocalPath_Nested(t *testing.T) {
	s := newTestStore(t)

	got := s.LocalPath("group/my-skill")
	want := filepath.Join(s.Root, "local", "group", "my-skill")
	if got != want {
		t.Errorf("LocalPath(%q) = %q, want %q", "group/my-skill", got, want)
	}
}

// TestGitHubPath verifies that GitHubPath returns the correct path.
func TestGitHubPath(t *testing.T) {
	s := newTestStore(t)

	got := s.GitHubPath("alexmx", "skills", "pdf")
	want := filepath.Join(s.Root, "github.com", "alexmx", "skills", "pdf")
	if got != want {
		t.Errorf("GitHubPath(%q, %q, %q) = %q, want %q", "alexmx", "skills", "pdf", got, want)
	}
}

// TestList verifies that List discovers SKILL.md files at multiple depths.
func TestList(t *testing.T) {
	s := newTestStore(t)

	// Create a variety of skills at different paths.
	skills := []struct {
		relDir string
	}{
		{filepath.Join("local", "skill-a")},
		{filepath.Join("local", "skill-b")},
		{filepath.Join("github.com", "owner", "repo", "skill-c")},
	}

	for _, sk := range skills {
		dir := filepath.Join(s.Root, sk.relDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("creating directory %q: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# skill"), 0o644); err != nil {
			t.Fatalf("writing SKILL.md: %v", err)
		}
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}

	if len(got) != len(skills) {
		t.Fatalf("List() returned %d results, want %d: %v", len(got), len(skills), got)
	}

	want := []string{
		filepath.Join("github.com", "owner", "repo", "skill-c"),
		filepath.Join("local", "skill-a"),
		filepath.Join("local", "skill-b"),
	}
	sort.Strings(got)
	sort.Strings(want)
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("List()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestList_Empty verifies that List returns an empty (not nil) slice for an empty store.
func TestList_Empty(t *testing.T) {
	s := newTestStore(t)

	got, err := s.List()
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List() returned %v, want empty", got)
	}
}

// TestList_IgnoresNonSkillFiles verifies that plain files are not returned.
func TestList_IgnoresNonSkillFiles(t *testing.T) {
	s := newTestStore(t)

	dir := filepath.Join(s.Root, "local", "some-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating directory: %v", err)
	}
	// Write a file that is NOT named SKILL.md.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0o644); err != nil {
		t.Fatalf("writing README.md: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List() returned %v, want empty", got)
	}
}

// TestExists verifies that Exists returns true for present paths and false for absent ones.
func TestExists(t *testing.T) {
	s := newTestStore(t)

	skillDir := filepath.Join(s.Root, "local", "present-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("creating skill directory: %v", err)
	}

	t.Run("returns true for existing path", func(t *testing.T) {
		if !s.Exists(filepath.Join("local", "present-skill")) {
			t.Error("Exists() = false, want true for existing path")
		}
	})

	t.Run("returns false for missing path", func(t *testing.T) {
		if s.Exists(filepath.Join("local", "missing-skill")) {
			t.Error("Exists() = true, want false for missing path")
		}
	})
}

// TestExists_File verifies that Exists works for individual files, not only directories.
func TestExists_File(t *testing.T) {
	s := newTestStore(t)

	skillDir := filepath.Join(s.Root, "local", "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("creating directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# skill"), 0o644); err != nil {
		t.Fatalf("writing SKILL.md: %v", err)
	}

	if !s.Exists(filepath.Join("local", "my-skill", "SKILL.md")) {
		t.Error("Exists() = false for existing file, want true")
	}
}

// TestCopyDir verifies that CopyDir copies files and subdirectories correctly.
func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Build source tree:
	//   src/
	//     file.txt
	//     sub/
	//       nested.txt
	if err := os.WriteFile(filepath.Join(src, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("creating file.txt: %v", err)
	}
	subDir := filepath.Join(src, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("creating sub/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("world"), 0o644); err != nil {
		t.Fatalf("creating sub/nested.txt: %v", err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir() returned unexpected error: %v", err)
	}

	// Verify top-level file.
	data, err := os.ReadFile(filepath.Join(dst, "file.txt"))
	if err != nil {
		t.Fatalf("reading dst/file.txt: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("dst/file.txt = %q, want %q", string(data), "hello")
	}

	// Verify nested file.
	data, err = os.ReadFile(filepath.Join(dst, "sub", "nested.txt"))
	if err != nil {
		t.Fatalf("reading dst/sub/nested.txt: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("dst/sub/nested.txt = %q, want %q", string(data), "world")
	}
}

// TestCopyDir_CreatesDestination verifies that CopyDir creates the destination when absent.
func TestCopyDir_CreatesDestination(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "new-dir")

	if err := os.WriteFile(filepath.Join(src, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatalf("creating file.txt: %v", err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir() returned unexpected error: %v", err)
	}

	if _, err := os.Stat(dst); err != nil {
		t.Errorf("destination directory was not created: %v", err)
	}
}

// TestCopyDir_OverwritesExistingFiles verifies that CopyDir into an existing destination
// overwrites files that already exist there.
func TestCopyDir_OverwritesExistingFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Write the initial file in the destination.
	if err := os.WriteFile(filepath.Join(dst, "file.txt"), []byte("old content"), 0o644); err != nil {
		t.Fatalf("creating pre-existing file.txt: %v", err)
	}

	// Write the replacement file in the source.
	if err := os.WriteFile(filepath.Join(src, "file.txt"), []byte("new content"), 0o644); err != nil {
		t.Fatalf("creating src/file.txt: %v", err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir() returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "file.txt"))
	if err != nil {
		t.Fatalf("reading dst/file.txt: %v", err)
	}
	if string(data) != "new content" {
		t.Errorf("dst/file.txt = %q, want %q", string(data), "new content")
	}
}

// TestCopyDir_OverwritesNestedFiles verifies that CopyDir overwrites files in existing
// sub-directories of the destination.
func TestCopyDir_OverwritesNestedFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Pre-populate destination with a nested file.
	dstSub := filepath.Join(dst, "sub")
	if err := os.MkdirAll(dstSub, 0o755); err != nil {
		t.Fatalf("creating dst/sub/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dstSub, "nested.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("creating dst/sub/nested.txt: %v", err)
	}

	// Source has updated content in the same nested location.
	srcSub := filepath.Join(src, "sub")
	if err := os.MkdirAll(srcSub, 0o755); err != nil {
		t.Fatalf("creating src/sub/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcSub, "nested.txt"), []byte("fresh"), 0o644); err != nil {
		t.Fatalf("creating src/sub/nested.txt: %v", err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir() returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "sub", "nested.txt"))
	if err != nil {
		t.Fatalf("reading dst/sub/nested.txt: %v", err)
	}
	if string(data) != "fresh" {
		t.Errorf("dst/sub/nested.txt = %q, want %q", string(data), "fresh")
	}
}
