package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/skill"
	"github.com/alexmx/skillman/internal/store"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <path-or-name>",
	Short: "Validate a skill against the Agent Skills spec",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// Resolve path: either a direct path or a skill name in the store
		dir := target
		if !filepath.IsAbs(dir) && !isLocalPath(dir) {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			s := store.New(cfg)
			dir = filepath.Join(s.Root, target)
		}

		s, err := skill.LoadFromDir(dir)
		if err != nil {
			return fmt.Errorf("loading skill: %w", err)
		}

		errs := skill.Validate(s)
		if len(errs) == 0 {
			fmt.Printf("Skill %q is valid.\n", s.Frontmatter.Name)
			return nil
		}

		fmt.Printf("Skill %q has %d validation error(s):\n", s.Frontmatter.Name, len(errs))
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("validation failed")
	},
}

func isLocalPath(s string) bool {
	return s == "." || s == ".." ||
		len(s) >= 2 && (s[:2] == "./" || s[:2] == "..") ||
		filepath.IsAbs(s)
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
