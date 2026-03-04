package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/registry"
	"github.com/alexmx/skillman/internal/store"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show skillman status for the current workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// Global info
		fmt.Println("Global:")
		fmt.Printf("  Config file:  %s\n", config.ConfigPath())
		fmt.Printf("  Store path:   %s\n", cfg.ResolvedStorePath())

		reg, err := registry.Load(cfg)
		if err == nil && len(reg.Skills) > 0 {
			fmt.Printf("  Skills:       %d installed (run 'skillman list' to see all)\n", len(reg.Skills))
		}

		// Workspace info
		wd, err := os.Getwd()
		if err != nil {
			return nil
		}

		s := store.New(cfg)

		fmt.Println("\nWorkspace:")
		fmt.Printf("  Path: %s\n", wd)

		// .skillman.yml
		wc, err := workspace.LoadWorkspaceConfig(wd)
		if err != nil {
			return nil
		}
		if wc != nil && len(wc.Skills) > 0 {
			fmt.Printf("  Declared: %d skills in .skillman.yml\n", len(wc.Skills))
			for _, sk := range wc.Skills {
				fmt.Printf("    - %s\n", sk)
			}
		}

		// Linked skills
		linked, err := workspace.Status(wd, cfg, s)
		if err != nil {
			return nil
		}

		if len(linked) == 0 {
			fmt.Println("  Linked: none")
			return nil
		}

		// Group by skill name
		bySkill := make(map[string][]workspace.LinkedSkill)
		for _, ls := range linked {
			bySkill[ls.Name] = append(bySkill[ls.Name], ls)
		}

		var names []string
		for name := range bySkill {
			names = append(names, name)
		}
		sort.Strings(names)

		fmt.Printf("  Linked: %d skills\n", len(names))
		for _, name := range names {
			skills := bySkill[name]
			var agents []string
			for _, l := range skills {
				a := l.Agent
				if l.IsBroken {
					a += " (broken)"
				}
				agents = append(agents, a)
			}

			source := ""
			if len(skills) > 0 && skills[0].StorePath != "" {
				source = fmt.Sprintf(" (%s)", skills[0].StorePath)
			}
			fmt.Printf("    %-20s -> %s%s\n", name, strings.Join(agents, ", "), source)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
