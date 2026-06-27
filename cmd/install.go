package cmd

import (
	"fmt"
	"os"

	"github.com/alexmx/skillman/internal/agent"
	"github.com/alexmx/skillman/internal/skill"
	"github.com/alexmx/skillman/internal/source"
	"github.com/alexmx/skillman/internal/tui"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	installAs      string
	installNoTrack bool
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
  skillman install ./my-skill

  # Install under a different name (alias)
  skillman install github.com/anthropics/skills/pdf --as acme-pdf

  # Install without recording it in .skillman/config.yml
  skillman install github.com/anthropics/skills/pdf --no-track`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringVar(&installAs, "as", "", "Install the skill under a different name (alias)")
	installCmd.Flags().BoolVar(&installNoTrack, "no-track", false, "Install without recording the skill in .skillman/config.yml")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	ref := source.ParseRef(args[0])

	if installAs != "" {
		if err := skill.ValidateName(installAs); err != nil {
			return fmt.Errorf("invalid --as name: %w", err)
		}
	}

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

	name := installName(result.Name)

	fmt.Printf("Skill: %s\n", result.Name)
	if name != result.Name {
		fmt.Printf("Install as: %s\n", name)
	}
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

	entry := workspace.SkillEntry{
		Name:   name,
		Source: "local",
		Path:   result.SourceDir,
	}
	if name != result.Name {
		entry.OriginalName = result.Name
	}
	if err := installOne(wd, name, result.SourceDir, agents, entry); err != nil {
		return err
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

	if installAs != "" && len(results) > 1 {
		return fmt.Errorf("--as can only be used when installing a single skill (%d selected)", len(results))
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
		name := installName(result.Name)
		entry := workspace.SkillEntry{
			Name:   name,
			Source: result.Source,
			Ref:    result.Ref,
			Commit: result.CommitSHA,
		}
		if name != result.Name {
			entry.OriginalName = result.Name
		}
		if err := installOne(wd, name, result.SourceDir, agents, entry); err != nil {
			return err
		}
	}

	fmt.Println()
	printSecurityWarning()
	return nil
}

// installName returns the alias when --as was given, otherwise the source name.
func installName(sourceName string) string {
	if installAs != "" {
		return installAs
	}
	return sourceName
}

// installOne copies a skill into the workspace under name, links the agents,
// and rewrites the declared name when aliased. The config entry is recorded
// unless --no-track was passed ("install and forget").
func installOne(wd, name, sourceDir string, agents []agent.Agent, entry workspace.SkillEntry) error {
	if workspace.SkillExistsInWorkspace(wd, name) {
		fmt.Printf("Skill %q already installed, replacing.\n", name)
	}

	installed, err := workspace.Install(wd, name, sourceDir, agents)
	if err != nil {
		return fmt.Errorf("installing %s: %w", name, err)
	}

	if entry.OriginalName != "" {
		if err := skill.SetName(workspace.SkillmanSkillPath(wd, name), name); err != nil {
			return fmt.Errorf("setting skill name: %w", err)
		}
	}

	for _, ws := range installed {
		fmt.Printf("Installed %s for %s\n", ws.Name, ws.Agent)
	}

	if installNoTrack {
		fmt.Printf("Not tracking %q in config (--no-track).\n", name)
		return nil
	}

	if err := workspace.UpsertSkillEntry(wd, entry); err != nil {
		return fmt.Errorf("updating config: %w", err)
	}
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
