package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexmx/skillman/internal/config"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	dir := t.TempDir()
	return config.Config{
		StorePath: filepath.Join(dir, "store"),
	}
}

func TestRegistry_AddAndFind(t *testing.T) {
	reg := &Registry{}

	entry := Entry{
		Name:        "test-skill",
		Source:      "local",
		StorePath:   "local/test-skill",
		Local:       true,
		InstalledAt: time.Now(),
	}

	reg.Add(entry)

	found := reg.Find("test-skill")
	if found == nil {
		t.Fatal("expected to find skill")
	}
	if found.Name != "test-skill" {
		t.Errorf("name = %q, want %q", found.Name, "test-skill")
	}
}

func TestRegistry_AddUpdatesExisting(t *testing.T) {
	reg := &Registry{}

	reg.Add(Entry{Name: "skill", Source: "local", Ref: "v1"})
	reg.Add(Entry{Name: "skill", Source: "local", Ref: "v2"})

	if len(reg.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(reg.Skills))
	}
	if reg.Skills[0].Ref != "v2" {
		t.Errorf("ref = %q, want %q", reg.Skills[0].Ref, "v2")
	}
}

func TestRegistry_Remove(t *testing.T) {
	reg := &Registry{
		Skills: []Entry{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
		},
	}

	ok := reg.Remove("b")
	if !ok {
		t.Error("expected Remove to return true")
	}
	if len(reg.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(reg.Skills))
	}
	if reg.Find("b") != nil {
		t.Error("expected b to be removed")
	}
}

func TestRegistry_RemoveNotFound(t *testing.T) {
	reg := &Registry{Skills: []Entry{{Name: "a"}}}
	ok := reg.Remove("nonexistent")
	if ok {
		t.Error("expected Remove to return false")
	}
}

func TestRegistry_FindBySource(t *testing.T) {
	reg := &Registry{
		Skills: []Entry{
			{Name: "a", Source: "github.com/org/repo/a"},
			{Name: "b", Source: "local"},
		},
	}

	found := reg.FindBySource("local")
	if found == nil || found.Name != "b" {
		t.Errorf("expected to find skill b by source")
	}
}

func TestRegistry_SaveAndLoad(t *testing.T) {
	cfg := testConfig(t)
	os.MkdirAll(cfg.StorePath, 0o755)

	reg := &Registry{
		Skills: []Entry{
			{Name: "test", Source: "local", InstalledAt: time.Now()},
		},
	}

	if err := reg.Save(cfg); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := Load(cfg)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if len(loaded.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(loaded.Skills))
	}
	if loaded.Skills[0].Name != "test" {
		t.Errorf("name = %q, want %q", loaded.Skills[0].Name, "test")
	}
}

func TestRegistry_LoadNotExist(t *testing.T) {
	cfg := testConfig(t)

	reg, err := Load(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reg.Skills) != 0 {
		t.Errorf("expected empty registry, got %d skills", len(reg.Skills))
	}
}
