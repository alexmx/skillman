package agent

import "github.com/alexmx/skillman/internal/config"

type Agent struct {
	Name      string
	SkillPath string // relative to workspace root
	Enabled   bool
}

func FromConfig(cfg config.Config) []Agent {
	var agents []Agent
	for name, ac := range cfg.Agents {
		agents = append(agents, Agent{
			Name:      name,
			SkillPath: ac.SkillPath,
			Enabled:   ac.Enabled,
		})
	}
	return agents
}

func EnabledAgents(cfg config.Config) []Agent {
	var agents []Agent
	for _, a := range FromConfig(cfg) {
		if a.Enabled {
			agents = append(agents, a)
		}
	}
	return agents
}
