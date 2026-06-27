package cmd

import (
	"fmt"
	"os"

	"github.com/alexmx/skillman/internal/skill"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:     "rename <old-name> <new-name>",
	Short:   "Rename an installed skill",
	Aliases: []string{"mv"},
	Long: `Rename a skill that is already installed in the current workspace.

Moves .skillman/skills/<old-name> to <new-name>, rewrites the skill's declared
name to match, and re-points the agent symlinks. The skill's upstream source is
preserved, so 'skillman update' keeps tracking the original skill.`,
	Example: `  # Rename an installed skill
  skillman rename pdf acme-pdf`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName, newName := args[0], args[1]

		if oldName == newName {
			return fmt.Errorf("new name is the same as the current name")
		}
		if err := skill.ValidateName(newName); err != nil {
			return fmt.Errorf("invalid new name: %w", err)
		}

		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		if !workspace.SkillExistsInWorkspace(wd, oldName) {
			return fmt.Errorf("skill %q is not installed in this workspace", oldName)
		}
		if workspace.SkillExistsInWorkspace(wd, newName) {
			return fmt.Errorf("a skill named %q already exists in this workspace", newName)
		}

		agents, err := workspace.Rename(wd, oldName, newName)
		if err != nil {
			return err
		}

		if err := workspace.RenameSkillEntry(wd, oldName, newName); err != nil {
			return fmt.Errorf("updating config: %w", err)
		}

		fmt.Printf("Renamed %q to %q.\n", oldName, newName)
		for _, a := range agents {
			fmt.Printf("Re-linked for %s\n", a)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
