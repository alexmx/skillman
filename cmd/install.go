package cmd

import (
	"fmt"
	"os"

	"github.com/alexmx/skillman/internal/agent"
	"github.com/alexmx/skillman/internal/source"
	"github.com/alexmx/skillman/internal/tui"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <source>",
	Short: "Install a skill into the current workspace",
	Long: `Fetch a skill and install it into the current workspace's .skillman/skills/ directory.

Sources:
  ./path/to/skill              Local skill directory
  github.com/org/repo          GitHub repository (discovers all skills)
  github.com/org/repo/skill    Specific skill from a GitHub repository
  github.com/org/repo@v1.0     Pin to a specific tag or ref`,
	Example: `  # Install skills from a GitHub repository
  skillman install github.com/anthropics/skills

  # Install a specific skill from a repository
  skillman install github.com/anthropics/skills/pdf

  # Pin to a specific version
  skillman install github.com/anthropics/skills@v1.0

  # Install from a local directory
  skillman install ./my-skill`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	ref := source.ParseRef(args[0])

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if ref.IsLocal {
		return installLocal(ref, wd)
	}
	return installGitHub(ref, wd)
}

func installLocal(ref source.ParsedRef, wd string) error {
	result, err := source.FetchLocal(ref.Raw)
	if err != nil {
		return err
	}

	fmt.Printf("Skill: %s\n", result.Name)
	fmt.Printf("Source: %s\n\n", result.SourceDir)

	yes, err := tui.Confirm("Install this skill?")
	if err != nil {
		return err
	}
	if !yes {
		fmt.Println("Cancelled.")
		return nil
	}

	agents, err := pickAgents(wd)
	if err != nil {
		return err
	}
	if len(agents) == 0 {
		fmt.Println("No agents selected.")
		return nil
	}

	if workspace.SkillExistsInWorkspace(wd, result.Name) {
		fmt.Printf("Skill %q already installed, replacing.\n", result.Name)
	}

	installed, err := workspace.Install(wd, result.Name, result.SourceDir, agents)
	if err != nil {
		return fmt.Errorf("installing %s: %w", result.Name, err)
	}
	for _, ws := range installed {
		fmt.Printf("Installed %s for %s\n", ws.Name, ws.Agent)
	}

	if err := workspace.UpsertSkillEntry(wd, workspace.SkillEntry{
		Name:   result.Name,
		Source: "local",
		Path:   result.SourceDir,
	}); err != nil {
		return fmt.Errorf("updating config: %w", err)
	}

	fmt.Println()
	printSecurityWarning()
	return nil
}

func installGitHub(ref source.ParsedRef, wd string) error {
	// Fetch and pick skills first
	results, cleanup, err := source.FetchGitHub(ref.Source, ref.Ref)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if len(results) == 0 {
		return nil
	}

	// Then pick agents
	agents, err := pickAgents(wd)
	if err != nil {
		return err
	}
	if len(agents) == 0 {
		fmt.Println("No agents selected.")
		return nil
	}

	for _, result := range results {
		if workspace.SkillExistsInWorkspace(wd, result.Name) {
			fmt.Printf("Skill %q already installed, replacing.\n", result.Name)
		}

		installed, err := workspace.Install(wd, result.Name, result.SourceDir, agents)
		if err != nil {
			return fmt.Errorf("installing %s: %w", result.Name, err)
		}
		for _, ws := range installed {
			fmt.Printf("Installed %s for %s\n", ws.Name, ws.Agent)
		}

		if err := workspace.UpsertSkillEntry(wd, workspace.SkillEntry{
			Name:   result.Name,
			Source: result.Source,
			Ref:    result.Ref,
			Commit: result.CommitSHA,
		}); err != nil {
			return fmt.Errorf("updating config: %w", err)
		}
	}

	fmt.Println()
	printSecurityWarning()
	return nil
}

func pickAgents(workspaceRoot string) ([]agent.Agent, error) {
	allAgents := agent.All()
	detected := workspace.DetectAgents(workspaceRoot)

	agentNames := make([]string, len(allAgents))
	agentDescs := make([]string, len(allAgents))
	for i, a := range allAgents {
		agentNames[i] = a.Name
		agentDescs[i] = a.SkillPath
	}

	// Pre-select detected agents
	preselected := make(map[int]bool)
	for i, a := range allAgents {
		for _, d := range detected {
			if a.Name == d.Name {
				preselected[i] = true
			}
		}
	}

	// If all agents are detected, skip the picker
	if len(detected) == len(allAgents) {
		return allAgents, nil
	}

	indices, err := tui.PickSkillsWithPreselection(
		"Select agents to install for",
		agentNames,
		agentDescs,
		preselected,
	)
	if err != nil {
		return nil, err
	}

	var selected []agent.Agent
	for _, idx := range indices {
		selected = append(selected, allAgents[idx])
	}
	return selected, nil
}

func printSecurityWarning() {
	fmt.Println(tui.SecurityWarning())
}
