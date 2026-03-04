package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfigFile creates a config.toml at ConfigPath() inside a temporary
// directory tree rooted at dir.  It returns the path to the file.
func writeConfigFile(t *testing.T, dir, content string) {
	t.Helper()
	cfgDir := filepath.Join(dir, appName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	path := filepath.Join(cfgDir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
}

// setXDGConfigHome redirects XDG_CONFIG_HOME to dir for the duration of the
// test, ensuring Load() reads from the temp directory.
func setXDGConfigHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", dir)
}

// TestDefaultConfig verifies that DefaultConfig returns the four built-in
// agents, all enabled, with the correct skill paths and an empty StorePath.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.StorePath != "" {
		t.Errorf("StorePath = %q, want empty string", cfg.StorePath)
	}

	want := map[string]AgentConfig{
		"claude":  {Enabled: true, SkillPath: ".claude/skills"},
		"cursor":  {Enabled: true, SkillPath: ".cursor/skills"},
		"codex":   {Enabled: true, SkillPath: ".codex/skills"},
		"copilot": {Enabled: true, SkillPath: ".github/skills"},
	}

	if len(cfg.Agents) != len(want) {
		t.Fatalf("len(Agents) = %d, want %d", len(cfg.Agents), len(want))
	}

	for name, wantAgent := range want {
		got, ok := cfg.Agents[name]
		if !ok {
			t.Errorf("agent %q not found in DefaultConfig", name)
			continue
		}
		if got.Enabled != wantAgent.Enabled {
			t.Errorf("agent %q: Enabled = %v, want %v", name, got.Enabled, wantAgent.Enabled)
		}
		if got.SkillPath != wantAgent.SkillPath {
			t.Errorf("agent %q: SkillPath = %q, want %q", name, got.SkillPath, wantAgent.SkillPath)
		}
	}
}

// TestLoad_NoConfigFile verifies that Load returns defaults when no config file
// exists — it must not return an error.
func TestLoad_NoConfigFile(t *testing.T) {
	dir := t.TempDir()
	setXDGConfigHome(t, dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	defaults := DefaultConfig()

	if cfg.StorePath != defaults.StorePath {
		t.Errorf("StorePath = %q, want %q", cfg.StorePath, defaults.StorePath)
	}
	if len(cfg.Agents) != len(defaults.Agents) {
		t.Errorf("len(Agents) = %d, want %d", len(cfg.Agents), len(defaults.Agents))
	}
}

// TestLoad_OverridesStorePath verifies that a config file with store_path
// replaces the empty default.
func TestLoad_OverridesStorePath(t *testing.T) {
	dir := t.TempDir()
	setXDGConfigHome(t, dir)

	customStore := filepath.Join(dir, "custom", "store")
	writeConfigFile(t, dir, `store_path = "`+customStore+`"`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.StorePath != customStore {
		t.Errorf("StorePath = %q, want %q", cfg.StorePath, customStore)
	}
}

// TestLoad_DisablesAgent verifies field-by-field merge: when a config file sets
// enabled = false for an existing agent, SkillPath from the default is preserved.
func TestLoad_DisablesAgent(t *testing.T) {
	dir := t.TempDir()
	setXDGConfigHome(t, dir)

	writeConfigFile(t, dir, `
[agents.claude]
enabled = false
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	claude, ok := cfg.Agents["claude"]
	if !ok {
		t.Fatal("agent 'claude' not found after merge")
	}

	if claude.Enabled {
		t.Error("claude.Enabled = true, want false")
	}

	// SkillPath must be preserved from the default because the config file
	// did not supply one.
	wantSkillPath := DefaultAgents()["claude"].SkillPath
	if claude.SkillPath != wantSkillPath {
		t.Errorf("claude.SkillPath = %q, want %q", claude.SkillPath, wantSkillPath)
	}

	// All other agents must remain unchanged.
	for _, name := range []string{"cursor", "codex", "copilot"} {
		agent, ok := cfg.Agents[name]
		if !ok {
			t.Errorf("agent %q missing after merge", name)
			continue
		}
		if !agent.Enabled {
			t.Errorf("agent %q: Enabled = false, want true", name)
		}
	}
}

// TestLoad_AddsCustomAgent verifies that an agent present in the config file
// but not in the defaults is added to the final map.
func TestLoad_AddsCustomAgent(t *testing.T) {
	dir := t.TempDir()
	setXDGConfigHome(t, dir)

	writeConfigFile(t, dir, `
[agents.myagent]
enabled = true
skill_path = ".myagent/skills"
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	custom, ok := cfg.Agents["myagent"]
	if !ok {
		t.Fatal("custom agent 'myagent' not found after load")
	}
	if !custom.Enabled {
		t.Error("myagent.Enabled = false, want true")
	}
	if custom.SkillPath != ".myagent/skills" {
		t.Errorf("myagent.SkillPath = %q, want %q", custom.SkillPath, ".myagent/skills")
	}

	// Built-in agents must still be present.
	if len(cfg.Agents) != len(DefaultAgents())+1 {
		t.Errorf("len(Agents) = %d, want %d", len(cfg.Agents), len(DefaultAgents())+1)
	}
}

// TestResolvedStorePath_WithCustomPath verifies that a non-empty StorePath is
// returned as-is.
func TestResolvedStorePath_WithCustomPath(t *testing.T) {
	custom := "/some/custom/store"
	cfg := Config{StorePath: custom}

	got := cfg.ResolvedStorePath()
	if got != custom {
		t.Errorf("ResolvedStorePath() = %q, want %q", got, custom)
	}
}

// TestResolvedStorePath_WithoutCustomPath verifies that an empty StorePath falls
// back to DefaultStorePath().
func TestResolvedStorePath_WithoutCustomPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	cfg := Config{}

	got := cfg.ResolvedStorePath()
	want := DefaultStorePath()
	if got != want {
		t.Errorf("ResolvedStorePath() = %q, want %q", got, want)
	}
}

// TestConfigPath_RespectsXDGConfigHome verifies that ConfigPath() uses
// XDG_CONFIG_HOME when the variable is set.
func TestConfigPath_RespectsXDGConfigHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	got := ConfigPath()
	want := filepath.Join(dir, appName, "config.toml")
	if got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}

// TestConfigPath_DefaultsToHomeDotConfig verifies that ConfigPath() falls back to
// ~/.config/skillman/config.toml when XDG_CONFIG_HOME is not set.
func TestConfigPath_DefaultsToHomeDotConfig(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("XDG_CONFIG_HOME", "") // ensure the variable is unset

	got := ConfigPath()
	want := filepath.Join(fakeHome, ".config", appName, "config.toml")
	if got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}

// TestDefaultStorePath_RespectsXDGDataHome verifies that DefaultStorePath() uses
// XDG_DATA_HOME when the variable is set.
func TestDefaultStorePath_RespectsXDGDataHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	got := DefaultStorePath()
	want := filepath.Join(dir, appName, "store")
	if got != want {
		t.Errorf("DefaultStorePath() = %q, want %q", got, want)
	}
}

// TestDefaultStorePath_DefaultsToHomeDotLocalShare verifies that
// DefaultStorePath() falls back to ~/.local/share/skillman/store when
// XDG_DATA_HOME is not set.
func TestDefaultStorePath_DefaultsToHomeDotLocalShare(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("XDG_DATA_HOME", "") // ensure the variable is unset

	got := DefaultStorePath()
	want := filepath.Join(fakeHome, ".local", "share", appName, "store")
	if got != want {
		t.Errorf("DefaultStorePath() = %q, want %q", got, want)
	}
}
