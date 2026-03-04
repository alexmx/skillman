package cmd

import (
	"fmt"
	"os"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/store"
	"github.com/alexmx/skillman/internal/tui"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink [skill-names...]",
	Short: "Unlink skills from the current workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		names := args
		if len(names) == 0 {
			s := store.New(cfg)
			linked, err := workspace.Status(wd, cfg, s)
			if err != nil {
				return err
			}
			if len(linked) == 0 {
				fmt.Println("No skills linked in this workspace.")
				return nil
			}

			// Deduplicate by skill name
			seen := make(map[string]bool)
			var skillNames []string
			var skillDescs []string
			for _, l := range linked {
				if !seen[l.Name] {
					seen[l.Name] = true
					skillNames = append(skillNames, l.Name)
					skillDescs = append(skillDescs, l.StorePath)
				}
			}

			indices, err := tui.PickSkills("Select skills to unlink", skillNames, skillDescs)
			if err != nil {
				return err
			}
			if len(indices) == 0 {
				fmt.Println("No skills selected.")
				return nil
			}
			for _, idx := range indices {
				names = append(names, skillNames[idx])
			}
		}

		for _, name := range names {
			agents, err := workspace.Unlink(wd, name, cfg)
			if err != nil {
				return fmt.Errorf("unlinking %s: %w", name, err)
			}

			if len(agents) == 0 {
				fmt.Printf("Skill %q was not linked in any agent directory.\n", name)
			} else {
				for _, a := range agents {
					fmt.Printf("Unlinked %s from %s\n", name, a)
				}
			}

			if err := workspace.RemoveFromWorkspaceConfig(wd, name); err != nil {
				return fmt.Errorf("updating .skillman.yml: %w", err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(unlinkCmd)
}
