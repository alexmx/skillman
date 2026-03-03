package source

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alexmx/skillman/internal/skill"
	"github.com/alexmx/skillman/internal/tui"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// FetchGitHub clones a GitHub repo, discovers skills, and returns results.
// The caller must call the returned cleanup function when done with the results.
func FetchGitHub(source, ref string, all bool) (results []FetchResult, cleanup func(), err error) {
	owner, repo, subpath, err := ParseGitHubSource(source)
	if err != nil {
		return nil, nil, err
	}

	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	tmpDir, err := os.MkdirTemp("", "skillman-*")
	if err != nil {
		return nil, nil, fmt.Errorf("creating temp dir: %w", err)
	}
	cleanup = func() { os.RemoveAll(tmpDir) }

	fmt.Printf("Cloning %s/%s...\n", owner, repo)

	cloneOpts := &git.CloneOptions{
		URL:      cloneURL,
		Depth:    1,
		Progress: nil,
	}

	if ref != "" {
		cloneOpts.ReferenceName = plumbing.NewTagReferenceName(ref)
		cloneOpts.SingleBranch = true
	}

	r, err := git.PlainClone(tmpDir, false, cloneOpts)
	if err != nil {
		// If tag clone fails, try as branch
		if ref != "" {
			cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(ref)
			r, err = git.PlainClone(tmpDir, false, cloneOpts)
		}
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("cloning repository: %w", err)
		}
	}

	// Get commit SHA
	head, err := r.Head()
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("getting HEAD: %w", err)
	}
	commitSHA := head.Hash().String()

	// Resolve the ref name
	resolvedRef := ref
	if resolvedRef == "" {
		resolvedRef = head.Name().Short()
	}

	// Discover skills
	searchRoot := tmpDir
	if subpath != "" {
		searchRoot = filepath.Join(tmpDir, subpath)
	}

	skills, err := discoverSkills(searchRoot)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("discovering skills: %w", err)
	}

	if len(skills) == 0 {
		cleanup()
		return nil, nil, fmt.Errorf("no skills found in %s", source)
	}

	// If subpath specified a specific skill, return just that one
	if subpath != "" && len(skills) == 1 {
		all = true
	}

	// Let user pick if not --all
	selected := skills
	if !all && len(skills) > 1 {
		names := make([]string, len(skills))
		descs := make([]string, len(skills))
		for i, s := range skills {
			names[i] = s.Frontmatter.Name
			descs[i] = s.Frontmatter.Description
		}
		indices, err := tui.PickSkills(names, descs)
		if err != nil {
			cleanup()
			return nil, nil, err
		}
		if len(indices) == 0 {
			fmt.Println("No skills selected.")
			cleanup()
			return nil, cleanup, nil
		}
		selected = make([]*skill.Skill, len(indices))
		for i, idx := range indices {
			selected[i] = skills[idx]
		}
	}

	for _, s := range selected {
		results = append(results, FetchResult{
			Name:      s.Frontmatter.Name,
			SourceDir: s.Dir,
			Source:    fmt.Sprintf("github.com/%s/%s/%s", owner, repo, s.Frontmatter.Name),
			Ref:       resolvedRef,
			CommitSHA: commitSHA,
		})
	}

	return results, cleanup, nil
}

func discoverSkills(root string) ([]*skill.Skill, error) {
	var skills []*skill.Skill

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}

		// Skip hidden directories and common non-skill dirs
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
		}

		if d.Name() == "SKILL.md" {
			s, err := skill.Parse(path)
			if err != nil {
				return nil // skip invalid skills
			}
			errs := skill.Validate(s)
			if len(errs) == 0 {
				skills = append(skills, s)
			}
		}
		return nil
	})

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Frontmatter.Name < skills[j].Frontmatter.Name
	})

	return skills, err
}
