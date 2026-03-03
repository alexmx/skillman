package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/store"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show linked skills in the current workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		s := store.New(cfg)

		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		skills, err := workspace.Status(wd, cfg, s)
		if err != nil {
			return err
		}

		if len(skills) == 0 {
			fmt.Println("No skills linked in this workspace.")
			return nil
		}

		// Group by skill name
		bySkill := make(map[string][]workspace.LinkedSkill)
		for _, ls := range skills {
			bySkill[ls.Name] = append(bySkill[ls.Name], ls)
		}

		var names []string
		for name := range bySkill {
			names = append(names, name)
		}
		sort.Strings(names)

		fmt.Printf("Workspace: %s\n\n", wd)
		for _, name := range names {
			linked := bySkill[name]
			var agents []string
			for _, l := range linked {
				status := l.Agent
				if l.IsBroken {
					status += " (broken)"
				}
				agents = append(agents, status)
			}

			source := ""
			if len(linked) > 0 && linked[0].StorePath != "" {
				source = fmt.Sprintf(" (%s)", linked[0].StorePath)
			}
			fmt.Printf("  %-20s -> %s%s\n", name, strings.Join(agents, ", "), source)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
