package agent

import (
	"testing"
)

func TestAll_ReturnsFourAgents(t *testing.T) {
	agents := All()
	if len(agents) != 4 {
		t.Fatalf("expected 4 agents, got %d", len(agents))
	}
}

func TestAll_SortedByName(t *testing.T) {
	agents := All()
	for i := 1; i < len(agents); i++ {
		if agents[i].Name <= agents[i-1].Name {
			t.Errorf("agents not sorted: %q after %q", agents[i].Name, agents[i-1].Name)
		}
	}
}

func TestAll_ExpectedAgents(t *testing.T) {
	agents := All()
	expected := map[string]string{
		"claude":  ".claude/skills",
		"codex":   ".codex/skills",
		"copilot": ".github/skills",
		"cursor":  ".cursor/skills",
	}

	for _, a := range agents {
		want, ok := expected[a.Name]
		if !ok {
			t.Errorf("unexpected agent %q", a.Name)
			continue
		}
		if a.SkillPath != want {
			t.Errorf("agent %q: SkillPath = %q, want %q", a.Name, a.SkillPath, want)
		}
	}
}
