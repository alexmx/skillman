package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/registry"
	"github.com/alexmx/skillman/internal/source"
	"github.com/alexmx/skillman/internal/store"
	"github.com/spf13/cobra"
)

var installAll bool

var installCmd = &cobra.Command{
	Use:   "install <source>",
	Short: "Install a skill from a local path or GitHub",
	Long: `Install a skill into the central store.

Sources:
  ./path/to/skill              Local skill directory
  github.com/org/repo          GitHub repository (discovers all skills)
  github.com/org/repo/skill    Specific skill from a GitHub repository
  github.com/org/repo@v1.0     Pin to a specific tag or ref`,
	Example: `  # Install all skills from a GitHub repository (interactive picker)
  skillman install github.com/anthropics/skills

  # Install a specific skill from a repository
  skillman install github.com/anthropics/skills/pdf

  # Pin to a specific version
  skillman install github.com/anthropics/skills@v1.0

  # Install all skills without prompting
  skillman install github.com/anthropics/skills --all

  # Install from a local directory
  skillman install ./my-skill`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	installCmd.Flags().BoolVar(&installAll, "all", false, "install all discovered skills without prompting")
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
	return nil
}

func installGitHub(ref source.ParsedRef, s *store.Store, reg *registry.Registry, cfg config.Config) error {
	results, cleanup, err := source.FetchGitHub(ref.Source, ref.Ref, installAll)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	for _, result := range results {
		owner, repo, _, _ := source.ParseGitHubSource(result.Source)
		storePath := s.GitHubPath(owner, repo, result.Name)

		if err := store.CopyDir(result.SourceDir, storePath); err != nil {
			return fmt.Errorf("copying skill %q to store: %w", result.Name, err)
		}

		sourceID := fmt.Sprintf("github.com/%s/%s/%s", owner, repo, result.Name)
		reg.Add(registry.Entry{
			Name:        result.Name,
			Source:      sourceID,
			Ref:         result.Ref,
			CommitSHA:   result.CommitSHA,
			StorePath:   fmt.Sprintf("github.com/%s/%s/%s", owner, repo, result.Name),
			InstalledAt: time.Now(),
		})

		fmt.Printf("Installed %q from %s\n", result.Name, sourceID)
	}

	if err := reg.Save(cfg); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	return nil
}
