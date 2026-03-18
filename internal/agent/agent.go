package agent

type Agent struct {
	Name      string
	SkillPath string // relative to workspace root
}

// All returns all supported agents in sorted order.
func All() []Agent {
	return []Agent{
		{Name: "claude", SkillPath: ".claude/skills"},
		{Name: "codex", SkillPath: ".codex/skills"},
		{Name: "copilot", SkillPath: ".github/skills"},
		{Name: "cursor", SkillPath: ".cursor/skills"},
	}
}
