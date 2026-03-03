package cmd

import (
	"fmt"
	"time"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/registry"
	"github.com/alexmx/skillman/internal/source"
	"github.com/alexmx/skillman/internal/store"
	"github.com/alexmx/skillman/internal/tui"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [skill-name]",
	Short: "Update a skill to the latest version",
	Long:  "Re-fetches the skill from its source at the latest ref (or a specified ref).",
	Args:  cobra.MaximumNArgs(1),
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

		// Determine which skills to update
		var entries []*registry.Entry
		if len(args) == 1 {
			entry := reg.Find(args[0])
			if entry == nil {
				return fmt.Errorf("skill %q is not installed", args[0])
			}
			entries = append(entries, entry)
		} else {
			for i := range reg.Skills {
				if !reg.Skills[i].Local {
					entries = append(entries, &reg.Skills[i])
				}
			}
			if len(entries) == 0 {
				fmt.Println("No remote skills to update.")
				return nil
			}
		}

		for _, entry := range entries {
			if entry.Local {
				fmt.Printf("Skipping %q (local skill)\n", entry.Name)
				continue
			}

			owner, repo, _, err := source.ParseGitHubSource(entry.Source)
			if err != nil {
				fmt.Printf("Skipping %q: %v\n", entry.Name, err)
				continue
			}

			fmt.Printf("Updating %q...\n", entry.Name)

			results, cleanup, err := source.FetchGitHub(
				fmt.Sprintf("github.com/%s/%s/%s", owner, repo, entry.Name),
				"", // latest
				true,
			)
			if err != nil {
				return fmt.Errorf("fetching %s: %w", entry.Name, err)
			}
			if cleanup != nil {
				defer cleanup()
			}

			if len(results) == 0 {
				fmt.Printf("No update found for %q.\n", entry.Name)
				continue
			}

			result := results[0]
			if result.CommitSHA == entry.CommitSHA {
				fmt.Printf("Skill %q is already up to date (%s).\n", entry.Name, entry.CommitSHA[:8])
				continue
			}

			yes, err := tui.Confirm(fmt.Sprintf("Update %s from %s to %s?", entry.Name, entry.CommitSHA[:8], result.CommitSHA[:8]))
			if err != nil {
				return err
			}
			if !yes {
				fmt.Println("Skipped.")
				continue
			}

			storePath := s.GitHubPath(owner, repo, result.Name)
			if err := store.CopyDir(result.SourceDir, storePath); err != nil {
				return fmt.Errorf("copying skill: %w", err)
			}

			entry.Ref = result.Ref
			entry.CommitSHA = result.CommitSHA
			entry.InstalledAt = time.Now()

			fmt.Printf("Updated %q to %s.\n", entry.Name, result.CommitSHA[:8])
		}

		return reg.Save(cfg)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
