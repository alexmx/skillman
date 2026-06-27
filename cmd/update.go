package cmd

import (
	"fmt"
	"os"

	"github.com/alexmx/skillman/internal/skill"
	"github.com/alexmx/skillman/internal/source"
	"github.com/alexmx/skillman/internal/tui"
	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [skill-name]",
	Short: "Update a skill to the latest version",
	Long: `Re-fetches the skill from its source and updates the workspace copy.

Without arguments, updates all remote skills declared in .skillman/config.yml.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		wc, err := workspace.LoadWorkspaceConfig(wd)
		if err != nil {
			return err
		}
		if wc == nil || len(wc.Skills) == 0 {
			return fmt.Errorf("no skills in this workspace")
		}

		// Determine which skills to update
		var entries []workspace.SkillEntry
		if len(args) == 1 {
			entry := wc.FindSkillEntry(args[0])
			if entry == nil {
				return fmt.Errorf("skill %q not found in workspace config", args[0])
			}
			entries = append(entries, *entry)
		} else {
			for _, e := range wc.Skills {
				if e.Source != "local" {
					entries = append(entries, e)
				}
			}
			if len(entries) == 0 {
				fmt.Println("No remote skills to update.")
				return nil
			}
		}

		// Update local skills individually
		var remoteEntries []workspace.SkillEntry
		for _, entry := range entries {
			if entry.Source == "local" {
				if err := updateLocalSkill(wd, entry); err != nil {
					return err
				}
			} else {
				remoteEntries = append(remoteEntries, entry)
			}
		}

		// Group remote skills by source to avoid cloning the same repo multiple times
		bySource := make(map[string][]workspace.SkillEntry)
		for _, e := range remoteEntries {
			bySource[e.Source] = append(bySource[e.Source], e)
		}

		for repoSource, repoEntries := range bySource {
			if err := updateFromRepo(wd, repoSource, repoEntries); err != nil {
				return err
			}
		}

		return nil
	},
}

func updateLocalSkill(wd string, entry workspace.SkillEntry) error {
	if entry.Path == "" {
		fmt.Printf("Skipping %q (no original path recorded)\n", entry.Name)
		return nil
	}
	fmt.Printf("Updating %q from %s...\n", entry.Name, entry.Path)

	result, err := source.FetchLocal(entry.Path)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", entry.Name, err)
	}

	if err := copyIntoWorkspace(wd, entry, result); err != nil {
		return err
	}

	if err := workspace.UpsertSkillEntry(wd, workspace.SkillEntry{
		Name:         entry.Name,
		OriginalName: entry.OriginalName,
		Source:       "local",
		Path:         entry.Path,
	}); err != nil {
		return fmt.Errorf("updating config: %w", err)
	}

	fmt.Printf("Updated %q from local path.\n", entry.Name)
	return nil
}

func updateFromRepo(wd string, repoSource string, entries []workspace.SkillEntry) error {
	sourceNames := make([]string, len(entries))
	for i, e := range entries {
		sourceNames[i] = e.SourceName()
	}

	fmt.Printf("Fetching %s...\n", repoSource)
	results, cleanup, err := source.FetchGitHubMultiple(repoSource, sourceNames, "")
	if err != nil {
		return fmt.Errorf("fetching %s: %w", repoSource, err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Index results by the skill's source name.
	resultBySource := make(map[string]source.FetchResult, len(results))
	for _, r := range results {
		resultBySource[r.Name] = r
	}

	// Pair each entry with its fetched result, keeping only those with updates.
	type pending struct {
		entry  workspace.SkillEntry
		result source.FetchResult
	}
	var updatable []pending
	var dispNames []string
	var updateDescs []string
	for _, entry := range entries {
		result, found := resultBySource[entry.SourceName()]
		if !found {
			fmt.Printf("Warning: skill %q not found in %s\n", entry.SourceName(), repoSource)
			continue
		}
		if result.CommitSHA == entry.Commit {
			fmt.Printf("Skill %q is already up to date (%s).\n", entry.Name, shortSHA(entry.Commit))
			continue
		}
		updatable = append(updatable, pending{entry, result})
		dispNames = append(dispNames, entry.Name)
		updateDescs = append(updateDescs, fmt.Sprintf("%s -> %s", shortSHA(entry.Commit), shortSHA(result.CommitSHA)))
	}

	if len(updatable) == 0 {
		return nil
	}

	// Let user pick which skills to update
	selected := updatable
	if len(updatable) > 1 {
		indices, err := tui.PickSkills("Select skills to update", dispNames, updateDescs)
		if err != nil {
			return err
		}
		if len(indices) == 0 {
			fmt.Println("No skills selected.")
			return nil
		}
		selected = nil
		for _, idx := range indices {
			selected = append(selected, updatable[idx])
		}
	}

	for _, p := range selected {
		if err := copyIntoWorkspace(wd, p.entry, &p.result); err != nil {
			return err
		}

		if err := workspace.UpsertSkillEntry(wd, workspace.SkillEntry{
			Name:         p.entry.Name,
			OriginalName: p.entry.OriginalName,
			Source:       p.entry.Source,
			Ref:          p.result.Ref,
			Commit:       p.result.CommitSHA,
		}); err != nil {
			return fmt.Errorf("updating config: %w", err)
		}

		fmt.Printf("Updated %q to %s.\n", p.entry.Name, shortSHA(p.result.CommitSHA))
	}

	return nil
}

// copyIntoWorkspace copies a freshly fetched skill into the workspace under the
// entry's (possibly aliased) name, rewriting the declared name when aliased.
func copyIntoWorkspace(wd string, entry workspace.SkillEntry, result *source.FetchResult) error {
	destPath := workspace.SkillmanSkillPath(wd, entry.Name)
	if err := workspace.CopyDir(result.SourceDir, destPath); err != nil {
		return fmt.Errorf("updating %s: %w", entry.Name, err)
	}
	if entry.Name != result.Name {
		if err := skill.SetName(destPath, entry.Name); err != nil {
			return fmt.Errorf("setting skill name: %w", err)
		}
	}
	return nil
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
