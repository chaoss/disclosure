package gitops

import (
	"fmt"
	"sort"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TreeSnapshot contains the paths and commit hash from the repository's current HEAD.
type TreeSnapshot struct {
	Commit string
	Paths  []string
}

// GetHeadTree lists file paths from the committed tree at HEAD without reading file contents.
func GetHeadTree(repoPath string) (TreeSnapshot, error) {
	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{
		EnableDotGitCommonDir: true,
	})
	if err != nil {
		return TreeSnapshot{}, fmt.Errorf("opening repo: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return TreeSnapshot{}, fmt.Errorf("getting HEAD: %w", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return TreeSnapshot{}, fmt.Errorf("reading HEAD commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return TreeSnapshot{}, fmt.Errorf("reading HEAD tree: %w", err)
	}

	snapshot := TreeSnapshot{
		Commit: head.Hash().String(),
		Paths:  make([]string, 0),
	}
	if err := tree.Files().ForEach(func(file *object.File) error {
		snapshot.Paths = append(snapshot.Paths, file.Name)
		return nil
	}); err != nil {
		return TreeSnapshot{}, fmt.Errorf("listing files in HEAD tree: %w", err)
	}

	sort.Strings(snapshot.Paths)
	return snapshot, nil
}
