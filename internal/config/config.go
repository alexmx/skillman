package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const appName = "skillman"

type AgentConfig struct {
	Enabled   bool   `toml:"enabled"`
	SkillPath string `toml:"skill_path"`
}

type Config struct {
	StorePath string                 `toml:"store_path,omitempty"`
	Agents    map[string]AgentConfig `toml:"agents"`
}

func DefaultAgents() map[string]AgentConfig {
	return map[string]AgentConfig{
		"claude":  {Enabled: true, SkillPath: ".claude/skills"},
		"cursor":  {Enabled: true, SkillPath: ".cursor/skills"},
		"codex":   {Enabled: true, SkillPath: ".codex/skills"},
		"copilot": {Enabled: true, SkillPath: ".github/skills"},
	}
}

func DefaultConfig() Config {
	return Config{
		Agents: DefaultAgents(),
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return h
}

func configHome() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	return filepath.Join(homeDir(), ".config")
}

func dataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v
	}
	return filepath.Join(homeDir(), ".local", "share")
}

func ConfigDir() string {
	return filepath.Join(configHome(), appName)
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.toml")
}

func DefaultStorePath() string {
	return filepath.Join(dataHome(), appName, "store")
}

func Load() (Config, error) {
	cfg := DefaultConfig()

	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	var fileCfg Config
	if err := toml.Unmarshal(data, &fileCfg); err != nil {
		return cfg, err
	}

	if fileCfg.StorePath != "" {
		cfg.StorePath = fileCfg.StorePath
	}

	for name, agentCfg := range fileCfg.Agents {
		existing, ok := cfg.Agents[name]
		if !ok {
			cfg.Agents[name] = agentCfg
			continue
		}
		existing.Enabled = agentCfg.Enabled
		if agentCfg.SkillPath != "" {
			existing.SkillPath = agentCfg.SkillPath
		}
		cfg.Agents[name] = existing
	}

	return cfg, nil
}

func (c Config) ResolvedStorePath() string {
	if c.StorePath != "" {
		return c.StorePath
	}
	return DefaultStorePath()
}
