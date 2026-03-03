package cmd

import (
	"fmt"
	"os"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink <skill-names...>",
	Short: "Unlink skills from the current workspace",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		for _, name := range args {
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
