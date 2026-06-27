package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexmx/skillman/internal/agent"
	"github.com/alexmx/skillman/internal/source"
	"github.com/alexmx/skillman/internal/workspace"
)

func writeFakeSkill(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: A fake skill.\n---\n# "+name+"\n"), 0o644)
	return dir
}

func TestInstallOne_Alias(t *testing.T) {
	wd := t.TempDir()
	src := writeFakeSkill(t, "use-something")
	agents := agent.All()

	entry := workspace.SkillEntry{
		Name:         "prefix-use-something",
		OriginalName: "use-something",
		Source:       "local",
		Path:         src,
	}
	if err := installOne(wd, entry.Name, src, agents, entry); err != nil {
		t.Fatalf("installOne error: %v", err)
	}

	// Installed under the alias, with the declared name rewritten to match.
	data, err := os.ReadFile(filepath.Join(workspace.SkillmanSkillPath(wd, "prefix-use-something"), "SKILL.md"))
	if err != nil {
		t.Fatalf("reading installed SKILL.md: %v", err)
	}
	if !strings.Contains(string(data), "name: prefix-use-something") {
		t.Errorf("declared name not rewritten to alias: %s", data)
	}

	// Config records the alias and the upstream source name.
	wc, _ := workspace.LoadWorkspaceConfig(wd)
	e := wc.FindSkillEntry("prefix-use-something")
	if e == nil {
		t.Fatal("expected aliased entry in config")
	}
	if e.SourceName() != "use-something" {
		t.Errorf("SourceName() = %q, want %q", e.SourceName(), "use-something")
	}

	// A subsequent update copy preserves the alias rewrite.
	if err := copyIntoWorkspace(wd, *e, &source.FetchResult{Name: "use-something", SourceDir: src}); err != nil {
		t.Fatalf("copyIntoWorkspace error: %v", err)
	}
	data, _ = os.ReadFile(filepath.Join(workspace.SkillmanSkillPath(wd, "prefix-use-something"), "SKILL.md"))
	if !strings.Contains(string(data), "name: prefix-use-something") {
		t.Errorf("update did not preserve alias name: %s", data)
	}
}
