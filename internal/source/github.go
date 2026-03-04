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

type cloneResult struct {
	tmpDir    string
	commitSHA string
	ref       string
	owner     string
	repo      string
}

func cloneRepo(source, ref string) (*cloneResult, func(), error) {
	owner, repo, _, err := ParseGitHubSource(source)
	if err != nil {
		return nil, nil, err
	}

	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	tmpDir, err := os.MkdirTemp("", "skillman-*")
	if err != nil {
		return nil, nil, fmt.Errorf("creating temp dir: %w", err)
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

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
		// If tag clone fails, try as branch with a fresh tmpDir
		if ref != "" {
			os.RemoveAll(tmpDir)
			tmpDir, err = os.MkdirTemp("", "skillman-*")
			if err != nil {
				return nil, nil, fmt.Errorf("creating temp dir: %w", err)
			}
			cleanup = func() { os.RemoveAll(tmpDir) }
			cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(ref)
			r, err = git.PlainClone(tmpDir, false, cloneOpts)
		}
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("cloning repository: %w", err)
		}
	}

	head, err := r.Head()
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("getting HEAD: %w", err)
	}

	resolvedRef := ref
	if resolvedRef == "" {
		resolvedRef = head.Name().Short()
	}

	return &cloneResult{
		tmpDir:    tmpDir,
		commitSHA: head.Hash().String(),
		ref:       resolvedRef,
		owner:     owner,
		repo:      repo,
	}, cleanup, nil
}

// FetchGitHub clones a GitHub repo, discovers skills, and presents an interactive picker.
// The caller must call the returned cleanup function when done with the results.
func FetchGitHub(source, ref string) (results []FetchResult, cleanup func(), err error) {
	_, _, subpath, err := ParseGitHubSource(source)
	if err != nil {
		return nil, nil, err
	}

	cr, cleanup, err := cloneRepo(source, ref)
	if err != nil {
		return nil, nil, err
	}

	searchRoot := cr.tmpDir
	if subpath != "" {
		searchRoot = filepath.Join(cr.tmpDir, subpath)
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

	// Always let user pick which skills to install
	names := make([]string, len(skills))
	descs := make([]string, len(skills))
	dirs := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Frontmatter.Name
		descs[i] = s.Frontmatter.Description
		dirs[i] = s.Dir
	}
	repoSource := fmt.Sprintf("github.com/%s/%s", cr.owner, cr.repo)
	indices, err := tui.PickSkillsWithOptions("Select skills to install", repoSource, tui.SecurityWarning(), names, descs, nil, dirs)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	if len(indices) == 0 {
		fmt.Println("No skills selected.")
		cleanup()
		return nil, nil, nil
	}
	selected := make([]*skill.Skill, len(indices))
	for i, idx := range indices {
		selected[i] = skills[idx]
	}

	for _, s := range selected {
		results = append(results, FetchResult{
			Name:      s.Frontmatter.Name,
			SourceDir: s.Dir,
			Source:    fmt.Sprintf("github.com/%s/%s", cr.owner, cr.repo),
			Ref:       cr.ref,
			CommitSHA: cr.commitSHA,
		})
	}

	return results, cleanup, nil
}

// FetchGitHubDirect clones a repo and fetches a specific skill by name without
// showing a picker. Searches the entire repo for the skill. Used by update.
func FetchGitHubDirect(repoSource, skillName, ref string) (*FetchResult, func(), error) {
	cr, cleanup, err := cloneRepo(repoSource, ref)
	if err != nil {
		return nil, nil, err
	}

	// Search the entire repo for the skill
	skills, err := discoverSkills(cr.tmpDir)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("discovering skills: %w", err)
	}

	for _, s := range skills {
		if s.Frontmatter.Name == skillName {
			return &FetchResult{
				Name:      s.Frontmatter.Name,
				SourceDir: s.Dir,
				Source:    fmt.Sprintf("github.com/%s/%s", cr.owner, cr.repo),
				Ref:       cr.ref,
				CommitSHA: cr.commitSHA,
			}, cleanup, nil
		}
	}

	cleanup()
	return nil, nil, fmt.Errorf("skill %q not found in %s/%s", skillName, cr.owner, cr.repo)
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
