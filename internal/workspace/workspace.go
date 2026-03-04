package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexmx/skillman/internal/agent"
	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/store"
	"gopkg.in/yaml.v3"
)

type WorkspaceConfig struct {
	Skills []string `yaml:"skills"`
}

type LinkedSkill struct {
	Name      string
	Agent     string
	LinkPath  string
	StorePath string
	IsBroken  bool
}

// DetectAgents returns enabled agents that have a config directory in the workspace.
// For example, if .claude/ exists, the claude agent is detected.
func DetectAgents(workspaceRoot string, cfg config.Config) []agent.Agent {
	var detected []agent.Agent
	for _, a := range agent.EnabledAgents(cfg) {
		// Check for the agent's parent directory (e.g. ".claude" from ".claude/skills")
		parts := strings.SplitN(a.SkillPath, "/", 2)
		agentDir := filepath.Join(workspaceRoot, parts[0])
		if _, err := os.Stat(agentDir); err == nil {
			detected = append(detected, a)
		}
	}
	return detected
}

// Link creates symlinks for a skill from the store into the specified agent directories.
func Link(workspaceRoot string, skillName string, agents []agent.Agent, s *store.Store) ([]LinkedSkill, error) {
	// Find the skill in the store
	storePath := findSkillInStore(skillName, s)
	if storePath == "" {
		return nil, fmt.Errorf("skill %q not found in store", skillName)
	}

	fullStorePath := filepath.Join(s.Root, storePath)
	var linked []LinkedSkill

	for _, a := range agents {
		agentSkillDir := filepath.Join(workspaceRoot, a.SkillPath)
		if err := os.MkdirAll(agentSkillDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating agent skill dir %s: %w", agentSkillDir, err)
		}

		linkPath := filepath.Join(agentSkillDir, skillName)

		// Remove existing link if present
		if _, err := os.Lstat(linkPath); err == nil {
			os.Remove(linkPath)
		}

		if err := os.Symlink(fullStorePath, linkPath); err != nil {
			return nil, fmt.Errorf("creating symlink for agent %s: %w", a.Name, err)
		}

		linked = append(linked, LinkedSkill{
			Name:      skillName,
			Agent:     a.Name,
			LinkPath:  linkPath,
			StorePath: storePath,
		})
	}

	return linked, nil
}

// Unlink removes symlinks for a skill from all agent directories in the workspace.
func Unlink(workspaceRoot string, skillName string, cfg config.Config) ([]string, error) {
	agents := agent.EnabledAgents(cfg)
	var unlinked []string

	for _, a := range agents {
		linkPath := filepath.Join(workspaceRoot, a.SkillPath, skillName)
		info, err := os.Lstat(linkPath)
		if err != nil {
			continue // not linked for this agent
		}

		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(linkPath); err != nil {
				return nil, fmt.Errorf("removing symlink %s: %w", linkPath, err)
			}
			unlinked = append(unlinked, a.Name)
		}
	}

	return unlinked, nil
}

// Status returns all linked skills for each agent in the workspace.
func Status(workspaceRoot string, cfg config.Config, s *store.Store) ([]LinkedSkill, error) {
	agents := agent.EnabledAgents(cfg)
	var skills []LinkedSkill

	for _, a := range agents {
		agentDir := filepath.Join(workspaceRoot, a.SkillPath)
		entries, err := os.ReadDir(agentDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, entry := range entries {
			linkPath := filepath.Join(agentDir, entry.Name())
			info, err := os.Lstat(linkPath)
			if err != nil {
				continue
			}

			ls := LinkedSkill{
				Name:     entry.Name(),
				Agent:    a.Name,
				LinkPath: linkPath,
			}

			if info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(linkPath)
				if err == nil {
					rel, err := filepath.Rel(s.Root, target)
					if err == nil && !strings.HasPrefix(rel, "..") {
						ls.StorePath = rel
					}
					// Check if target exists
					if _, err := os.Stat(target); err != nil {
						ls.IsBroken = true
					}
				}
			}

			skills = append(skills, ls)
		}
	}

	return skills, nil
}

func findSkillInStore(name string, s *store.Store) string {
	// Check local first
	localPath := "local/" + name
	if s.Exists(localPath) {
		return localPath
	}

	// Search github.com subdirectories
	storeSkills, _ := s.List()
	for _, sp := range storeSkills {
		if filepath.Base(sp) == name {
			return sp
		}
	}

	return ""
}

// LoadWorkspaceConfig reads a .skillman.yml file.
func LoadWorkspaceConfig(workspaceRoot string) (*WorkspaceConfig, error) {
	path := filepath.Join(workspaceRoot, ".skillman.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var wc WorkspaceConfig
	if err := yaml.Unmarshal(data, &wc); err != nil {
		return nil, fmt.Errorf("parsing .skillman.yml: %w", err)
	}
	return &wc, nil
}

// SaveWorkspaceConfig writes a .skillman.yml file.
func SaveWorkspaceConfig(workspaceRoot string, wc *WorkspaceConfig) error {
	path := filepath.Join(workspaceRoot, ".skillman.yml")
	data, err := yaml.Marshal(wc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// AddToWorkspaceConfig adds a skill ref to .skillman.yml if not already present.
func AddToWorkspaceConfig(workspaceRoot, skillRef string) error {
	wc, err := LoadWorkspaceConfig(workspaceRoot)
	if err != nil {
		return err
	}
	if wc == nil {
		wc = &WorkspaceConfig{}
	}

	for _, s := range wc.Skills {
		if s == skillRef {
			return nil // already present
		}
	}

	wc.Skills = append(wc.Skills, skillRef)
	return SaveWorkspaceConfig(workspaceRoot, wc)
}

// RemoveFromWorkspaceConfig removes a skill from .skillman.yml by name.
func RemoveFromWorkspaceConfig(workspaceRoot, skillName string) error {
	wc, err := LoadWorkspaceConfig(workspaceRoot)
	if err != nil {
		return err
	}
	if wc == nil {
		return nil
	}

	var filtered []string
	for _, s := range wc.Skills {
		if s != skillName {
			filtered = append(filtered, s)
		}
	}

	wc.Skills = filtered
	return SaveWorkspaceConfig(workspaceRoot, wc)
}
