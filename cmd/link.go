package cmd

import (
	"fmt"
	"os"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/registry"
	"github.com/alexmx/skillman/internal/store"
	"github.com/alexmx/skillman/internal/tui"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var linkSave bool

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

		names := args
		if len(names) == 0 {
			// Show picker with installed skills
			if len(reg.Skills) == 0 {
				return fmt.Errorf("no skills installed. Run 'skillman install' first")
			}

			skillNames := make([]string, len(reg.Skills))
			skillDescs := make([]string, len(reg.Skills))
			for i, e := range reg.Skills {
				skillNames[i] = e.Name
				skillDescs[i] = e.Source
			}

			indices, err := tui.PickSkills(skillNames, skillDescs)
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

		for _, name := range names {
			linked, err := workspace.Link(wd, name, cfg, s)
			if err != nil {
				return fmt.Errorf("linking %s: %w", name, err)
			}

			for _, l := range linked {
				fmt.Printf("Linked %s -> %s (%s)\n", l.Name, l.Agent, l.LinkPath)
			}

			if linkSave {
				entry := reg.Find(name)
				ref := name
				if entry != nil && entry.Source != "local" {
					ref = entry.Source
					if entry.Ref != "" {
						ref += "@" + entry.Ref
					}
				}
				if err := workspace.AddToWorkspaceConfig(wd, ref); err != nil {
					return fmt.Errorf("updating .skillman.yml: %w", err)
				}
			}
		}

		return nil
	},
}

func init() {
	linkCmd.Flags().BoolVar(&linkSave, "save", false, "add to .skillman.yml")
	rootCmd.AddCommand(linkCmd)
}
