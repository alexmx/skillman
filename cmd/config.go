package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexmx/skillman/internal/agent"
	"github.com/alexmx/skillman/internal/tui"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and configure the current workspace",
	Long:  "Shows workspace skills and agent status, then lets you toggle which agents have symlinks.",
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		wc, err := workspace.LoadWorkspaceConfig(wd)
		if err != nil {
			return err
		}

		if wc == nil || len(wc.Skills) == 0 {
			fmt.Printf("Workspace: %s\n", wd)
			fmt.Println("\nNo skills installed. Run 'skillman install' first.")
			return nil
		}

		allAgents := agent.All()

		// Collect current symlink state per agent
		skills, _ := workspace.Status(wd)
		agentHasLinks := make(map[string]bool)
		for _, ws := range skills {
			agentHasLinks[ws.Agent] = true
		}

		// --- Interactive agent toggle ---

		agentNames := make([]string, len(allAgents))
		agentDescs := make([]string, len(allAgents))
		preselected := make(map[int]bool)
		for i, a := range allAgents {
			agentNames[i] = a.Name
			agentDescs[i] = a.SkillPath
			if agentHasLinks[a.Name] {
				preselected[i] = true
			}
		}

		indices, err := tui.PickSkillsWithPreselection(
			"Toggle agents",
			agentNames,
			agentDescs,
			preselected,
		)
		if err != nil {
			return err
		}

		selectedAgents := make(map[string]bool)
		for _, idx := range indices {
			selectedAgents[allAgents[idx].Name] = true
		}

		// Apply changes
		for _, a := range allAgents {
			wasLinked := agentHasLinks[a.Name]
			wantLinked := selectedAgents[a.Name]

			if wantLinked && !wasLinked {
				for _, e := range wc.Skills {
					if workspace.SkillExistsInWorkspace(wd, e.Name) {
						_, err := workspace.EnsureSymlinks(wd, e.Name, []agent.Agent{a})
						if err != nil {
							fmt.Printf("Warning: could not link %s for %s: %v\n", e.Name, a.Name, err)
						}
					}
				}
			} else if !wantLinked && wasLinked {
				for _, e := range wc.Skills {
					linkPath := filepath.Join(wd, a.SkillPath, e.Name)
					if _, err := os.Lstat(linkPath); err == nil {
						os.Remove(linkPath)
					}
				}
			}
		}

		// --- Print final state ---

		fmt.Printf("\nWorkspace: %s\n", wd)

		// Re-read symlink state after changes
		skills, _ = workspace.Status(wd)

		// Group agents by skill
		skillAgents := make(map[string][]string)
		for _, ws := range skills {
			skillAgents[ws.Name] = append(skillAgents[ws.Name], ws.Agent)
		}

		fmt.Printf("\nSkills (%d):\n", len(wc.Skills))
		for _, e := range wc.Skills {
			src := e.Source
			if e.Path != "" {
				src = e.Path
			}
			if e.Commit != "" {
				src += "@" + shortSHA(e.Commit)
			}
			agents := skillAgents[e.Name]
			if len(agents) > 0 {
				fmt.Printf("  %-20s -> %s  (%s)\n", e.Name, strings.Join(agents, ", "), src)
			} else {
				fmt.Printf("  %-20s    no symlinks  (%s)\n", e.Name, src)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
