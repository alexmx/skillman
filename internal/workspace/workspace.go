package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexmx/skillman/internal/agent"
	"gopkg.in/yaml.v3"
)

const (
	skillmanDir        = ".skillman"
	skillmanSkillsDir  = "skills"
	skillmanConfigFile = "config.yml"
)

// SkillEntry tracks a skill's source information in the workspace config.
type SkillEntry struct {
	Name   string `yaml:"name"`
	Source string `yaml:"source"`           // "github.com/org/repo" or "local"
	Ref    string `yaml:"ref,omitempty"`    // git ref (tag, branch)
	Commit string `yaml:"commit,omitempty"` // resolved commit SHA
	Path   string `yaml:"path,omitempty"`   // original path for local skills
}

// WorkspaceConfig is the .skillman/config.yml file.
type WorkspaceConfig struct {
	Skills []SkillEntry `yaml:"skills"`
}

// WorkspaceSkill represents a skill in the workspace with its agent symlink state.
type WorkspaceSkill struct {
	Name     string
	Agent    string
	LinkPath string
	IsBroken bool
}

// SkillmanSkillPath returns the absolute path to a skill inside .skillman/skills/.
func SkillmanSkillPath(workspaceRoot, skillName string) string {
	return filepath.Join(workspaceRoot, skillmanDir, skillmanSkillsDir, skillName)
}

// DetectAgents returns agents that have a config directory in the workspace.
func DetectAgents(workspaceRoot string) []agent.Agent {
	var detected []agent.Agent
	for _, a := range agent.All() {
		parts := strings.SplitN(a.SkillPath, "/", 2)
		agentDir := filepath.Join(workspaceRoot, parts[0])
		if _, err := os.Stat(agentDir); err == nil {
			detected = append(detected, a)
		}
	}
	return detected
}

// Install copies skill files into .skillman/skills/{name} and creates agent symlinks.
func Install(workspaceRoot string, skillName string, sourceDir string, agents []agent.Agent) ([]WorkspaceSkill, error) {
	destPath := SkillmanSkillPath(workspaceRoot, skillName)

	if err := CopyDir(sourceDir, destPath); err != nil {
		return nil, fmt.Errorf("copying skill to workspace: %w", err)
	}

	return EnsureSymlinks(workspaceRoot, skillName, agents)
}

// EnsureSymlinks creates relative symlinks from each agent's skill directory to
// .skillman/skills/{name}. The skill must already exist in .skillman/skills/.
func EnsureSymlinks(workspaceRoot string, skillName string, agents []agent.Agent) ([]WorkspaceSkill, error) {
	destPath := SkillmanSkillPath(workspaceRoot, skillName)
	if _, err := os.Stat(destPath); err != nil {
		return nil, fmt.Errorf("skill %q not found in .skillman/skills/", skillName)
	}

	var results []WorkspaceSkill

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

		relTarget, err := filepath.Rel(agentSkillDir, destPath)
		if err != nil {
			return nil, fmt.Errorf("computing relative path for agent %s: %w", a.Name, err)
		}

		if err := os.Symlink(relTarget, linkPath); err != nil {
			return nil, fmt.Errorf("creating symlink for agent %s: %w", a.Name, err)
		}

		results = append(results, WorkspaceSkill{
			Name:     skillName,
			Agent:    a.Name,
			LinkPath: linkPath,
		})
	}

	return results, nil
}

// SkillExistsInWorkspace returns true if .skillman/skills/{name} exists.
func SkillExistsInWorkspace(workspaceRoot, skillName string) bool {
	_, err := os.Stat(SkillmanSkillPath(workspaceRoot, skillName))
	return err == nil
}

// Remove removes agent symlinks and the .skillman/skills/{name} directory for a skill.
func Remove(workspaceRoot string, skillName string) ([]string, error) {
	var removed []string

	for _, a := range agent.All() {
		linkPath := filepath.Join(workspaceRoot, a.SkillPath, skillName)
		info, err := os.Lstat(linkPath)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(linkPath); err != nil {
				return nil, fmt.Errorf("removing symlink %s: %w", linkPath, err)
			}
			removed = append(removed, a.Name)
		}
	}

	// Remove the .skillman/skills/{name} directory
	destPath := SkillmanSkillPath(workspaceRoot, skillName)
	if _, err := os.Stat(destPath); err == nil {
		if err := os.RemoveAll(destPath); err != nil {
			return nil, fmt.Errorf("removing %s: %w", destPath, err)
		}
	}

	return removed, nil
}

// Status returns all workspace skills by scanning .skillman/skills/ and checking agent symlinks.
func Status(workspaceRoot string) ([]WorkspaceSkill, error) {
	var skills []WorkspaceSkill

	skillsDir := filepath.Join(workspaceRoot, skillmanDir, skillmanSkillsDir)
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()

		for _, a := range agent.All() {
			linkPath := filepath.Join(workspaceRoot, a.SkillPath, skillName)
			info, err := os.Lstat(linkPath)
			if err != nil {
				continue
			}

			ws := WorkspaceSkill{
				Name:     skillName,
				Agent:    a.Name,
				LinkPath: linkPath,
			}

			if info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(linkPath)
				if err == nil {
					absTarget := target
					if !filepath.IsAbs(target) {
						absTarget = filepath.Join(filepath.Dir(linkPath), target)
					}
					if _, err := os.Stat(absTarget); err != nil {
						ws.IsBroken = true
					}
				}
			}

			skills = append(skills, ws)
		}
	}

	return skills, nil
}

// --- Config operations ---

// configPath returns the path to .skillman/config.yml.
func configPath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, skillmanDir, skillmanConfigFile)
}

// LoadWorkspaceConfig reads the .skillman/config.yml file.
func LoadWorkspaceConfig(workspaceRoot string) (*WorkspaceConfig, error) {
	path := configPath(workspaceRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var wc WorkspaceConfig
	if err := yaml.Unmarshal(data, &wc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &wc, nil
}

// SaveWorkspaceConfig writes the .skillman/config.yml file, creating .skillman/ if needed.
func SaveWorkspaceConfig(workspaceRoot string, wc *WorkspaceConfig) error {
	dir := filepath.Join(workspaceRoot, skillmanDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	path := configPath(workspaceRoot)
	data, err := yaml.Marshal(wc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// FindSkillEntry finds a skill entry in the config by name.
func (wc *WorkspaceConfig) FindSkillEntry(name string) *SkillEntry {
	if wc == nil {
		return nil
	}
	for i, e := range wc.Skills {
		if e.Name == name {
			return &wc.Skills[i]
		}
	}
	return nil
}

// UpsertSkillEntry adds or updates a skill entry in the config.
func UpsertSkillEntry(workspaceRoot string, entry SkillEntry) error {
	wc, err := LoadWorkspaceConfig(workspaceRoot)
	if err != nil {
		return err
	}
	if wc == nil {
		wc = &WorkspaceConfig{}
	}

	for i, e := range wc.Skills {
		if e.Name == entry.Name {
			wc.Skills[i] = entry
			return SaveWorkspaceConfig(workspaceRoot, wc)
		}
	}

	wc.Skills = append(wc.Skills, entry)
	return SaveWorkspaceConfig(workspaceRoot, wc)
}

// RemoveSkillEntry removes a skill entry from the config by name.
func RemoveSkillEntry(workspaceRoot, skillName string) error {
	wc, err := LoadWorkspaceConfig(workspaceRoot)
	if err != nil {
		return err
	}
	if wc == nil {
		return nil
	}

	var filtered []SkillEntry
	for _, e := range wc.Skills {
		if e.Name != skillName {
			filtered = append(filtered, e)
		}
	}

	wc.Skills = filtered
	return SaveWorkspaceConfig(workspaceRoot, wc)
}

// --- File utilities ---

// CopyDir recursively copies a directory tree.
func CopyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
