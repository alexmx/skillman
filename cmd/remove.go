package cmd

import (
	"fmt"
	"os"

	"github.com/alexmx/skillman/internal/tui"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove [skill-names...]",
	Short:   "Remove skills from the current workspace",
	Aliases: []string{"rm"},
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		names := args
		if len(names) == 0 {
			wc, err := workspace.LoadWorkspaceConfig(wd)
			if err != nil {
				return err
			}
			if wc == nil || len(wc.Skills) == 0 {
				fmt.Println("No skills in this workspace.")
				return nil
			}

			skillNames := make([]string, len(wc.Skills))
			skillDescs := make([]string, len(wc.Skills))
			for i, e := range wc.Skills {
				skillNames[i] = e.Name
				skillDescs[i] = e.Source
			}

			indices, err := tui.PickSkills("Select skills to remove", skillNames, skillDescs)
			if err != nil {
				return err
			}
			if len(indices) == 0 {
				fmt.Println("No skills selected.")
				return nil
			}
			for _, idx := range indices {
				names = append(names, wc.Skills[idx].Name)
			}
		}

		for _, name := range names {
			agents, err := workspace.Remove(wd, name)
			if err != nil {
				return fmt.Errorf("removing %s: %w", name, err)
			}

			if len(agents) == 0 {
				fmt.Printf("Skill %q was not in this workspace.\n", name)
			} else {
				for _, a := range agents {
					fmt.Printf("Removed %s from %s\n", name, a)
				}
			}

			if err := workspace.RemoveSkillEntry(wd, name); err != nil {
				return fmt.Errorf("updating config: %w", err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
