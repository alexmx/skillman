package agent

import (
	"testing"

	"github.com/alexmx/skillman/internal/config"
)

func makeConfig(agents map[string]config.AgentConfig) config.Config {
	return config.Config{
		Agents: agents,
	}
}

func TestFromConfig_ReturnsAllAgents(t *testing.T) {
	cfg := makeConfig(map[string]config.AgentConfig{
		"claude":  {Enabled: true, SkillPath: ".claude/skills"},
		"cursor":  {Enabled: false, SkillPath: ".cursor/skills"},
		"codex":   {Enabled: true, SkillPath: ".codex/skills"},
	})

	agents := FromConfig(cfg)

	if len(agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(agents))
	}

	byName := make(map[string]Agent, len(agents))
	for _, a := range agents {
		byName[a.Name] = a
	}

	cases := []struct {
		name      string
		skillPath string
		enabled   bool
	}{
		{"claude", ".claude/skills", true},
		{"cursor", ".cursor/skills", false},
		{"codex", ".codex/skills", true},
	}

	for _, tc := range cases {
		a, ok := byName[tc.name]
		if !ok {
			t.Errorf("expected agent %q not found", tc.name)
			continue
		}
		if a.SkillPath != tc.skillPath {
			t.Errorf("agent %q: expected SkillPath %q, got %q", tc.name, tc.skillPath, a.SkillPath)
		}
		if a.Enabled != tc.enabled {
			t.Errorf("agent %q: expected Enabled %v, got %v", tc.name, tc.enabled, a.Enabled)
		}
	}
}

func TestFromConfig_EmptyConfig(t *testing.T) {
	cfg := makeConfig(nil)

	agents := FromConfig(cfg)

	if len(agents) != 0 {
		t.Errorf("expected 0 agents for empty config, got %d", len(agents))
	}
}

func TestFromConfig_SortedByName(t *testing.T) {
	cfg := makeConfig(map[string]config.AgentConfig{
		"zebra":   {Enabled: true, SkillPath: "z/skills"},
		"alpha":   {Enabled: true, SkillPath: "a/skills"},
		"mango":   {Enabled: false, SkillPath: "m/skills"},
		"bravo":   {Enabled: true, SkillPath: "b/skills"},
	})

	agents := FromConfig(cfg)

	expected := []string{"alpha", "bravo", "mango", "zebra"}
	if len(agents) != len(expected) {
		t.Fatalf("expected %d agents, got %d", len(expected), len(agents))
	}

	for i, want := range expected {
		if agents[i].Name != want {
			t.Errorf("position %d: expected %q, got %q", i, want, agents[i].Name)
		}
	}
}

func TestFromConfig_DeterministicOrdering(t *testing.T) {
	cfg := makeConfig(map[string]config.AgentConfig{
		"delta":  {Enabled: true, SkillPath: "d/skills"},
		"alpha":  {Enabled: false, SkillPath: "a/skills"},
		"gamma":  {Enabled: true, SkillPath: "g/skills"},
		"beta":   {Enabled: false, SkillPath: "b/skills"},
	})

	const iterations = 10
	first := FromConfig(cfg)

	for i := 1; i < iterations; i++ {
		got := FromConfig(cfg)
		if len(got) != len(first) {
			t.Fatalf("iteration %d: length mismatch: expected %d, got %d", i, len(first), len(got))
		}
		for j := range first {
			if got[j].Name != first[j].Name {
				t.Errorf("iteration %d, position %d: expected %q, got %q", i, j, first[j].Name, got[j].Name)
			}
		}
	}
}

func TestEnabledAgents_FiltersToEnabled(t *testing.T) {
	cfg := makeConfig(map[string]config.AgentConfig{
		"claude":  {Enabled: true, SkillPath: ".claude/skills"},
		"cursor":  {Enabled: false, SkillPath: ".cursor/skills"},
		"codex":   {Enabled: true, SkillPath: ".codex/skills"},
		"copilot": {Enabled: false, SkillPath: ".github/skills"},
	})

	agents := EnabledAgents(cfg)

	if len(agents) != 2 {
		t.Fatalf("expected 2 enabled agents, got %d", len(agents))
	}

	for _, a := range agents {
		if !a.Enabled {
			t.Errorf("agent %q should be enabled but is not", a.Name)
		}
	}

	byName := make(map[string]struct{}, len(agents))
	for _, a := range agents {
		byName[a.Name] = struct{}{}
	}
	if _, ok := byName["claude"]; !ok {
		t.Error("expected agent \"claude\" in enabled agents")
	}
	if _, ok := byName["codex"]; !ok {
		t.Error("expected agent \"codex\" in enabled agents")
	}
}

func TestEnabledAgents_AllDisabled_ReturnsEmpty(t *testing.T) {
	cfg := makeConfig(map[string]config.AgentConfig{
		"claude":  {Enabled: false, SkillPath: ".claude/skills"},
		"cursor":  {Enabled: false, SkillPath: ".cursor/skills"},
	})

	agents := EnabledAgents(cfg)

	if agents != nil {
		t.Errorf("expected nil slice when all agents disabled, got %v", agents)
	}
}

func TestEnabledAgents_EmptyConfig_ReturnsEmpty(t *testing.T) {
	cfg := makeConfig(nil)

	agents := EnabledAgents(cfg)

	if len(agents) != 0 {
		t.Errorf("expected 0 enabled agents for empty config, got %d", len(agents))
	}
}

func TestEnabledAgents_SortedByName(t *testing.T) {
	cfg := makeConfig(map[string]config.AgentConfig{
		"zebra":   {Enabled: true, SkillPath: "z/skills"},
		"alpha":   {Enabled: true, SkillPath: "a/skills"},
		"mango":   {Enabled: false, SkillPath: "m/skills"},
		"bravo":   {Enabled: true, SkillPath: "b/skills"},
	})

	agents := EnabledAgents(cfg)

	expected := []string{"alpha", "bravo", "zebra"}
	if len(agents) != len(expected) {
		t.Fatalf("expected %d enabled agents, got %d", len(expected), len(agents))
	}

	for i, want := range expected {
		if agents[i].Name != want {
			t.Errorf("position %d: expected %q, got %q", i, want, agents[i].Name)
		}
	}
}
