package cmd

import (
	"fmt"
	"os"

	"github.com/alexmx/skillman/internal/agent"
	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/registry"
	"github.com/alexmx/skillman/internal/store"
	"github.com/alexmx/skillman/internal/tui"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link [skill-names...]",
	Short: "Link skills from the store into the current workspace",
	Long:  "Creates symlinks from the central store into agent skill directories for the current workspace.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		s := store.New(cfg)
		reg, err := registry.Load(cfg)
		if err != nil {
			return err
		}

		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		// Pick skills
		names := args
		if len(names) == 0 {
			if len(reg.Skills) == 0 {
				return fmt.Errorf("no skills installed. Run 'skillman install' first")
			}

			skillNames := make([]string, len(reg.Skills))
			skillDescs := make([]string, len(reg.Skills))
			for i, e := range reg.Skills {
				skillNames[i] = e.Name
				skillDescs[i] = e.Source
			}

			indices, err := tui.PickSkills("Select skills to link from store", skillNames, skillDescs)
			if err != nil {
				return err
			}
			if len(indices) == 0 {
				fmt.Println("No skills selected.")
				return nil
			}
			for _, idx := range indices {
				names = append(names, reg.Skills[idx].Name)
			}
		}

		// Pick agents — detect which ones exist in the workspace, prompt to confirm
		agents, err := pickAgents(wd, cfg)
		if err != nil {
			return err
		}
		if len(agents) == 0 {
			fmt.Println("No agents selected.")
			return nil
		}

		for _, name := range names {
			linked, err := workspace.Link(wd, name, agents, s)
			if err != nil {
				return fmt.Errorf("linking %s: %w", name, err)
			}

			for _, l := range linked {
				fmt.Printf("Linked %s -> %s (%s)\n", l.Name, l.Agent, l.LinkPath)
			}

			if err := workspace.AddToWorkspaceConfig(wd, name); err != nil {
				return fmt.Errorf("updating .skillman.yml: %w", err)
			}
		}

		return nil
	},
}

func pickAgents(workspaceRoot string, cfg config.Config) ([]agent.Agent, error) {
	allAgents := agent.EnabledAgents(cfg)
	detected := workspace.DetectAgents(workspaceRoot, cfg)

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
		"Select agents to link for",
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

func init() {
	rootCmd.AddCommand(linkCmd)
}
