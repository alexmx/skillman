package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexmx/skillman/internal/agent"
)

func testSetup(t *testing.T) (string, []agent.Agent) {
	t.Helper()
	workDir := t.TempDir()
	agents := agent.All()
	return workDir, agents
}

// createFakeSkill creates a skill directory with a SKILL.md file.
func createFakeSkill(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: test-skill
description: A test skill.
---
# Test
`), 0o644)
	return skillDir
}

func TestInstall(t *testing.T) {
	workDir, agents := testSetup(t)
	skillDir := createFakeSkill(t)

	installed, err := Install(workDir, "test-skill", skillDir, agents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(installed) != 4 {
		t.Fatalf("expected 4 installs, got %d", len(installed))
	}

	// Verify skill files are in .skillman/skills/
	skillmanPath := filepath.Join(workDir, ".skillman", "skills", "test-skill", "SKILL.md")
	if _, err := os.Stat(skillmanPath); err != nil {
		t.Errorf(".skillman/skills/test-skill/SKILL.md does not exist: %v", err)
	}

	// Verify agent symlinks are relative
	for _, ws := range installed {
		info, err := os.Lstat(ws.LinkPath)
		if err != nil {
			t.Errorf("symlink %s does not exist: %v", ws.LinkPath, err)
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s is not a symlink", ws.LinkPath)
			continue
		}
		target, _ := os.Readlink(ws.LinkPath)
		if filepath.IsAbs(target) {
			t.Errorf("symlink should be relative, got: %s", target)
		}
	}
}

func TestRemove(t *testing.T) {
	workDir, agents := testSetup(t)
	skillDir := createFakeSkill(t)

	Install(workDir, "test-skill", skillDir, agents)

	removed, err := Remove(workDir, "test-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(removed) != 4 {
		t.Fatalf("expected 4 agents removed, got %d", len(removed))
	}

	// Verify symlink removed
	if _, err := os.Lstat(filepath.Join(workDir, ".claude/skills/test-skill")); !os.IsNotExist(err) {
		t.Error("expected symlink to be removed")
	}

	// Verify .skillman/skills/test-skill removed
	if _, err := os.Stat(filepath.Join(workDir, ".skillman", "skills", "test-skill")); !os.IsNotExist(err) {
		t.Error("expected .skillman/skills/test-skill to be removed")
	}
}

func TestStatus(t *testing.T) {
	workDir, agents := testSetup(t)

	// Initially empty
	skills, err := Status(workDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}

	// Install and check
	skillDir := createFakeSkill(t)
	Install(workDir, "test-skill", skillDir, agents)

	skills, err = Status(workDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 4 {
		t.Errorf("expected 4 workspace skills, got %d", len(skills))
	}
}

func TestEnsureSymlinks(t *testing.T) {
	workDir, agents := testSetup(t)

	// Manually create .skillman/skills/test-skill (simulating git clone)
	skillDir := SkillmanSkillPath(workDir, "test-skill")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Test"), 0o644)

	linked, err := EnsureSymlinks(workDir, "test-skill", agents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(linked) != 4 {
		t.Fatalf("expected 4 links, got %d", len(linked))
	}

	// Verify status detects them
	skills, err := Status(workDir)
	if err != nil {
		t.Fatalf("status error: %v", err)
	}
	if len(skills) != 4 {
		t.Errorf("expected 4 skills in status, got %d", len(skills))
	}
}

func TestEnsureSymlinks_SkillNotInWorkspace(t *testing.T) {
	workDir, agents := testSetup(t)

	_, err := EnsureSymlinks(workDir, "nonexistent", agents)
	if err == nil {
		t.Error("expected error when skill not in .skillman/skills/")
	}
}

func TestDetectAgents(t *testing.T) {
	workDir, _ := testSetup(t)

	detected := DetectAgents(workDir)
	if len(detected) != 0 {
		t.Errorf("expected 0 detected agents, got %d", len(detected))
	}

	os.MkdirAll(filepath.Join(workDir, ".claude"), 0o755)
	detected = DetectAgents(workDir)
	if len(detected) != 1 {
		t.Fatalf("expected 1 detected agent, got %d", len(detected))
	}
	if detected[0].Name != "claude" {
		t.Errorf("expected claude, got %s", detected[0].Name)
	}

	os.MkdirAll(filepath.Join(workDir, ".cursor"), 0o755)
	detected = DetectAgents(workDir)
	if len(detected) != 2 {
		t.Errorf("expected 2 detected agents, got %d", len(detected))
	}
}

func TestWorkspaceConfig(t *testing.T) {
	dir := t.TempDir()

	// No config initially
	wc, err := LoadWorkspaceConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wc != nil {
		t.Error("expected nil config when file doesn't exist")
	}

	// Save and reload
	wc = &WorkspaceConfig{
		Skills: []SkillEntry{
			{Name: "pdf", Source: "github.com/anthropics/skills", Ref: "main", Commit: "abc123"},
			{Name: "commit", Source: "local", Path: "/tmp/commit-skill"},
		},
	}
	if err := SaveWorkspaceConfig(dir, wc); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Verify config is at .skillman/config.yml
	configFile := filepath.Join(dir, ".skillman", "config.yml")
	if _, err := os.Stat(configFile); err != nil {
		t.Errorf(".skillman/config.yml does not exist: %v", err)
	}

	loaded, err := LoadWorkspaceConfig(dir)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(loaded.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(loaded.Skills))
	}
	if loaded.Skills[0].Name != "pdf" || loaded.Skills[0].Source != "github.com/anthropics/skills" {
		t.Errorf("unexpected first entry: %+v", loaded.Skills[0])
	}
}

func TestUpsertSkillEntry(t *testing.T) {
	dir := t.TempDir()

	err := UpsertSkillEntry(dir, SkillEntry{Name: "pdf", Source: "github.com/org/repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Upsert same name (should update, not duplicate)
	err = UpsertSkillEntry(dir, SkillEntry{Name: "pdf", Source: "github.com/other/repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wc, _ := LoadWorkspaceConfig(dir)
	if len(wc.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(wc.Skills))
	}
	if wc.Skills[0].Source != "github.com/other/repo" {
		t.Errorf("expected updated source, got: %s", wc.Skills[0].Source)
	}
}

func TestRemoveSkillEntry(t *testing.T) {
	dir := t.TempDir()

	wc := &WorkspaceConfig{
		Skills: []SkillEntry{
			{Name: "skill-a", Source: "github.com/org/repo"},
			{Name: "skill-b", Source: "local"},
		},
	}
	SaveWorkspaceConfig(dir, wc)

	err := RemoveSkillEntry(dir, "skill-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, _ := LoadWorkspaceConfig(dir)
	if len(loaded.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(loaded.Skills))
	}
	if loaded.Skills[0].Name != "skill-b" {
		t.Errorf("wrong skill remaining: %s", loaded.Skills[0].Name)
	}
}

func TestFindSkillEntry(t *testing.T) {
	wc := &WorkspaceConfig{
		Skills: []SkillEntry{
			{Name: "pdf", Source: "github.com/org/repo"},
			{Name: "commit", Source: "local"},
		},
	}

	entry := wc.FindSkillEntry("pdf")
	if entry == nil {
		t.Fatal("expected to find pdf")
	}
	if entry.Source != "github.com/org/repo" {
		t.Errorf("unexpected source: %s", entry.Source)
	}

	entry = wc.FindSkillEntry("nonexistent")
	if entry != nil {
		t.Error("expected nil for nonexistent skill")
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "dest")

	os.WriteFile(filepath.Join(src, "file.txt"), []byte("hello"), 0o644)
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "sub", "nested.txt"), []byte("world"), 0o644)

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "file.txt"))
	if err != nil {
		t.Fatalf("reading file.txt: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("file.txt = %q, want %q", string(data), "hello")
	}

	data, err = os.ReadFile(filepath.Join(dst, "sub", "nested.txt"))
	if err != nil {
		t.Fatalf("reading sub/nested.txt: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("sub/nested.txt = %q, want %q", string(data), "world")
	}
}
