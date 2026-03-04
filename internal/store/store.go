package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexmx/skillman/internal/config"
)

type Store struct {
	Root string
}

func New(cfg config.Config) *Store {
	return &Store{Root: cfg.ResolvedStorePath()}
}

func (s *Store) Init() error {
	dirs := []string{
		s.Root,
		filepath.Join(s.Root, "github.com"),
		filepath.Join(s.Root, "local"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating store directory %s: %w", dir, err)
		}
	}
	return nil
}

func (s *Store) LocalPath(name string) string {
	return filepath.Join(s.Root, "local", name)
}

func (s *Store) GitHubPath(owner, repo, skill string) string {
	return filepath.Join(s.Root, "github.com", owner, repo, skill)
}

func (s *Store) List() ([]string, error) {
	var skills []string

	err := filepath.WalkDir(s.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == "SKILL.md" {
			rel, _ := filepath.Rel(s.Root, filepath.Dir(path))
			skills = append(skills, rel)
		}
		return nil
	})

	return skills, err
}

func (s *Store) Exists(storePath string) bool {
	full := filepath.Join(s.Root, storePath)
	_, err := os.Stat(full)
	return err == nil
}

// CopyDir recursively copies a directory tree.
func CopyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
