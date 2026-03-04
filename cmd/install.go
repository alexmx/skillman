package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/registry"
	"github.com/alexmx/skillman/internal/source"
	"github.com/alexmx/skillman/internal/store"
	"github.com/alexmx/skillman/internal/tui"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <source>",
	Short: "Install a skill from a local path or GitHub",
	Long: `Install a skill into the central store.

Sources:
  ./path/to/skill              Local skill directory
  github.com/org/repo          GitHub repository (discovers all skills)
  github.com/org/repo/skill    Specific skill from a GitHub repository
  github.com/org/repo@v1.0     Pin to a specific tag or ref`,
	Example: `  # Install skills from a GitHub repository
  skillman install github.com/anthropics/skills

  # Install a specific skill from a repository
  skillman install github.com/anthropics/skills/pdf

  # Pin to a specific version
  skillman install github.com/anthropics/skills@v1.0

  # Install from a local directory
  skillman install ./my-skill`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	ref := source.ParseRef(args[0])

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	s := store.New(cfg)
	if err := s.Init(); err != nil {
		return fmt.Errorf("initializing store: %w", err)
	}

	reg, err := registry.Load(cfg)
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	if ref.IsLocal {
		return installLocal(ref, s, reg, cfg)
	}
	return installGitHub(ref, s, reg, cfg)
}

func installLocal(ref source.ParsedRef, s *store.Store, reg *registry.Registry, cfg config.Config) error {
	result, err := source.FetchLocal(ref.Raw)
	if err != nil {
		return err
	}

	// Check if already installed
	if existing := reg.Find(result.Name); existing != nil && existing.Local {
		fmt.Printf("Skill %q is already installed (local).\n", result.Name)
		return nil
	}

	printSecurityWarning()

	fmt.Printf("\nSkill: %s\n", result.Name)
	fmt.Printf("Source: %s\n\n", result.SourceDir)

	yes, err := tui.Confirm("Install this skill?")
	if err != nil {
		return err
	}
	if !yes {
		fmt.Println("Cancelled.")
		return nil
	}

	// Create symlink in store/local/
	storePath := s.LocalPath(result.Name)
	if err := os.MkdirAll(s.LocalPath(""), 0o755); err != nil {
		return err
	}

	// Remove existing if present
	os.Remove(storePath)

	if err := os.Symlink(result.SourceDir, storePath); err != nil {
		return fmt.Errorf("creating symlink: %w", err)
	}

	reg.Add(registry.Entry{
		Name:        result.Name,
		Source:      result.Source,
		StorePath:   "local/" + result.Name,
		Local:       true,
		InstalledAt: time.Now(),
	})

	if err := reg.Save(cfg); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	fmt.Printf("Installed %q from local path.\n", result.Name)
	fmt.Println()
	printSecurityWarning()
	return nil
}

func installGitHub(ref source.ParsedRef, s *store.Store, reg *registry.Registry, cfg config.Config) error {
	results, cleanup, err := source.FetchGitHub(ref.Source, ref.Ref)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if len(results) == 0 {
		return nil
	}

	for _, result := range results {
		// Warn if replacing a skill from a different source
		if existing := reg.Find(result.Name); existing != nil && existing.Source != result.Source {
			yes, err := tui.Confirm(fmt.Sprintf("Skill %q is already installed from %s. Replace with %s?", result.Name, existing.Source, result.Source))
			if err != nil {
				return err
			}
			if !yes {
				fmt.Printf("Skipping %q.\n", result.Name)
				continue
			}
		}

		owner, repo, _, err := source.ParseGitHubSource(result.Source)
		if err != nil {
			return fmt.Errorf("parsing source for %q: %w", result.Name, err)
		}
		storePath := s.GitHubPath(owner, repo, result.Name)

		if err := store.CopyDir(result.SourceDir, storePath); err != nil {
			return fmt.Errorf("copying skill %q to store: %w", result.Name, err)
		}

		reg.Add(registry.Entry{
			Name:        result.Name,
			Source:      result.Source,
			Ref:         result.Ref,
			CommitSHA:   result.CommitSHA,
			StorePath:   fmt.Sprintf("github.com/%s/%s/%s", owner, repo, result.Name),
			InstalledAt: time.Now(),
		})

		fmt.Printf("Installed %q from %s\n", result.Name, result.Source)
	}

	if err := reg.Save(cfg); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	fmt.Println()
	printSecurityWarning()
	return nil
}

func printSecurityWarning() {
	fmt.Println(tui.SecurityWarning())
}
