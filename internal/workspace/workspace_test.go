package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexmx/skillman/internal/agent"
	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/store"
)

func testSetup(t *testing.T) (string, config.Config, *store.Store, []agent.Agent) {
	t.Helper()

	workDir := t.TempDir()
	storeDir := t.TempDir()

	cfg := config.Config{
		StorePath: storeDir,
		Agents: map[string]config.AgentConfig{
			"claude": {Enabled: true, SkillPath: ".claude/skills"},
			"cursor": {Enabled: true, SkillPath: ".cursor/skills"},
		},
	}

	s := store.New(cfg)
	s.Init()

	// Create a fake skill in the store
	skillDir := filepath.Join(storeDir, "local", "test-skill")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: test-skill
description: A test skill.
---
# Test
`), 0o644)

	agents := agent.EnabledAgents(cfg)
	return workDir, cfg, s, agents
}

func TestLink(t *testing.T) {
	workDir, _, s, agents := testSetup(t)

	linked, err := Link(workDir, "test-skill", agents, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(linked) != 2 {
		t.Fatalf("expected 2 links, got %d", len(linked))
	}

	// Verify symlinks exist
	for _, l := range linked {
		info, err := os.Lstat(l.LinkPath)
		if err != nil {
			t.Errorf("symlink %s does not exist: %v", l.LinkPath, err)
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s is not a symlink", l.LinkPath)
		}
	}
}

func TestUnlink(t *testing.T) {
	workDir, cfg, s, agents := testSetup(t)

	// Link first
	Link(workDir, "test-skill", agents, s)

	// Unlink
	unlinked, err := Unlink(workDir, "test-skill", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(unlinked) != 2 {
		t.Fatalf("expected 2 agents unlinked, got %d", len(unlinked))
	}

	// Verify symlinks are removed
	claudeLink := filepath.Join(workDir, ".claude/skills/test-skill")
	if _, err := os.Lstat(claudeLink); !os.IsNotExist(err) {
		t.Error("expected symlink to be removed")
	}
}

func TestStatus(t *testing.T) {
	workDir, cfg, s, agents := testSetup(t)

	// Initially empty
	skills, err := Status(workDir, cfg, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}

	// Link and check
	Link(workDir, "test-skill", agents, s)
	skills, err = Status(workDir, cfg, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Errorf("expected 2 linked skills, got %d", len(skills))
	}
}

func TestDetectAgents(t *testing.T) {
	workDir, cfg, _, _ := testSetup(t)

	// No agent dirs exist yet
	detected := DetectAgents(workDir, cfg)
	if len(detected) != 0 {
		t.Errorf("expected 0 detected agents, got %d", len(detected))
	}

	// Create .claude/ directory
	os.MkdirAll(filepath.Join(workDir, ".claude"), 0o755)
	detected = DetectAgents(workDir, cfg)
	if len(detected) != 1 {
		t.Fatalf("expected 1 detected agent, got %d", len(detected))
	}
	if detected[0].Name != "claude" {
		t.Errorf("expected claude, got %s", detected[0].Name)
	}

	// Create .cursor/ directory
	os.MkdirAll(filepath.Join(workDir, ".cursor"), 0o755)
	detected = DetectAgents(workDir, cfg)
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
		Skills: []string{
			"github.com/anthropics/skills/pdf@v1.0",
			"github.com/anthropics/skills/commit@v1.0",
		},
	}
	if err := SaveWorkspaceConfig(dir, wc); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadWorkspaceConfig(dir)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(loaded.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(loaded.Skills))
	}
}

func TestAddToWorkspaceConfig(t *testing.T) {
	dir := t.TempDir()

	err := AddToWorkspaceConfig(dir, "github.com/org/repo/skill@v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Add same again (should not duplicate)
	err = AddToWorkspaceConfig(dir, "github.com/org/repo/skill@v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wc, _ := LoadWorkspaceConfig(dir)
	if len(wc.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(wc.Skills))
	}
}

func TestRemoveFromWorkspaceConfig(t *testing.T) {
	dir := t.TempDir()

	wc := &WorkspaceConfig{
		Skills: []string{
			"skill-a",
			"skill-b",
		},
	}
	SaveWorkspaceConfig(dir, wc)

	err := RemoveFromWorkspaceConfig(dir, "skill-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, _ := LoadWorkspaceConfig(dir)
	if len(loaded.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(loaded.Skills))
	}
	if loaded.Skills[0] != "skill-b" {
		t.Errorf("wrong skill remaining: %s", loaded.Skills[0])
	}
}

func TestLink_SkillNotInStore(t *testing.T) {
	workDir, _, s, agents := testSetup(t)

	_, err := Link(workDir, "nonexistent", agents, s)
	if err == nil {
		t.Error("expected error when linking nonexistent skill")
	}
}
