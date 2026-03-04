package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/alexmx/skillman/internal/config"
)

type Entry struct {
	Name        string    `json:"name"`
	Source      string    `json:"source"`
	Ref         string    `json:"ref"`
	CommitSHA   string    `json:"commit_sha"`
	InstalledAt time.Time `json:"installed_at"`
	StorePath   string    `json:"store_path"`
	Local       bool      `json:"local"`
}

type Registry struct {
	Skills []Entry `json:"skills"`
}

func Path(cfg config.Config) string {
	return filepath.Join(cfg.ResolvedStorePath(), "registry.json")
}

func Load(cfg config.Config) (*Registry, error) {
	path := Path(cfg)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{}, nil
		}
		return nil, err
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *Registry) Save(cfg config.Config) error {
	path := Path(cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (r *Registry) Add(entry Entry) {
	for i, e := range r.Skills {
		if e.Name == entry.Name {
			r.Skills[i] = entry
			return
		}
	}
	r.Skills = append(r.Skills, entry)
}

func (r *Registry) Remove(name string) bool {
	for i, e := range r.Skills {
		if e.Name == name {
			r.Skills = append(r.Skills[:i], r.Skills[i+1:]...)
			return true
		}
	}
	return false
}

func (r *Registry) Find(name string) *Entry {
	for i, e := range r.Skills {
		if e.Name == name {
			return &r.Skills[i]
		}
	}
	return nil
}

func (r *Registry) FindBySource(source string) *Entry {
	for i, e := range r.Skills {
		if e.Source == source {
			return &r.Skills[i]
		}
	}
	return nil
}
