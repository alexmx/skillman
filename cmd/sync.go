package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/registry"
	"github.com/alexmx/skillman/internal/source"
	"github.com/alexmx/skillman/internal/store"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync workspace symlinks with .skillman.yml",
	Long:  "Reads .skillman.yml and ensures the workspace symlinks match the declared skills. Installs missing skills from their sources.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		s := store.New(cfg)
		if err := s.Init(); err != nil {
			return err
		}

		reg, err := registry.Load(cfg)
		if err != nil {
			return err
		}

		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		wc, err := workspace.LoadWorkspaceConfig(wd)
		if err != nil {
			return err
		}
		if wc == nil {
			return fmt.Errorf("no .skillman.yml found in current directory")
		}

		if len(wc.Skills) == 0 {
			fmt.Println("No skills declared in .skillman.yml")
			return nil
		}

		// Resolve desired skill names
		desiredNames := make(map[string]bool)
		for _, raw := range wc.Skills {
			ref := source.ParseRef(raw)
			// Extract skill name from the ref
			name := extractSkillName(ref)

			// Check if the skill is installed
			entry := reg.Find(name)
			if entry == nil {
				fmt.Printf("Skill %q not in store, installing from %s...\n", name, raw)
				installArgs := []string{raw}
				if err := runInstall(cmd, installArgs); err != nil {
					return fmt.Errorf("installing %s: %w", raw, err)
				}
				// Reload registry after install
				reg, err = registry.Load(cfg)
				if err != nil {
					return err
				}
			}

			desiredNames[name] = true
		}

		// Detect agents present in the workspace, prompt if not all found
		agents, err := pickAgents(wd, cfg)
		if err != nil {
			return err
		}
		if len(agents) == 0 {
			fmt.Println("No agents selected.")
			return nil
		}

		// Link all desired skills
		for name := range desiredNames {
			linked, err := workspace.Link(wd, name, agents, s)
			if err != nil {
				fmt.Printf("Warning: could not link %s: %v\n", name, err)
				continue
			}
			for _, l := range linked {
				fmt.Printf("Linked %s -> %s\n", l.Name, l.Agent)
			}
		}

		// Remove stale links (linked but not in .skillman.yml)
		currentSkills, err := workspace.Status(wd, cfg, s)
		if err != nil {
			return err
		}

		seen := make(map[string]bool)
		for _, ls := range currentSkills {
			if seen[ls.Name] {
				continue
			}
			seen[ls.Name] = true

			if !desiredNames[ls.Name] {
				agents, err := workspace.Unlink(wd, ls.Name, cfg)
				if err != nil {
					fmt.Printf("Warning: could not unlink %s: %v\n", ls.Name, err)
					continue
				}
				for _, a := range agents {
					fmt.Printf("Unlinked %s from %s (not in .skillman.yml)\n", ls.Name, a)
				}
			}
		}

		fmt.Println("Sync complete.")
		return nil
	},
}

func extractSkillName(ref source.ParsedRef) string {
	if ref.IsLocal {
		// For local paths, use directory name
		parts := strings.Split(strings.TrimRight(ref.Source, "/"), "/")
		return parts[len(parts)-1]
	}

	// For remote refs, use the last path component
	src := ref.Source
	parts := strings.Split(src, "/")
	return parts[len(parts)-1]
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
