package gitops

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Commit holds the fields detectors care about from a git commit.
type Commit struct {
	Hash           string
	AuthorEmail    string
	CommitterEmail string
	Message        string
	Notes          string // Content from refs/notes/ai, if any
}

func commitFromObject(c *object.Commit, repo *git.Repository) Commit {
	commit := Commit{
		Hash:           c.Hash.String(),
		AuthorEmail:    c.Author.Email,
		CommitterEmail: c.Committer.Email,
		Message:        c.Message,
	}
	commit.Notes = readNote(repo, c.Hash)
	return commit
}

// GetCommit reads a single commit by hash from the repository at repoPath.
func GetCommit(repoPath string, hash string) (Commit, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return Commit{}, fmt.Errorf("opening repo: %w", err)
	}

	h := plumbing.NewHash(hash)
	c, err := repo.CommitObject(h)
	if err != nil {
		return Commit{}, fmt.Errorf("reading commit %s: %w", hash, err)
	}

	return commitFromObject(c, repo), nil
}

// ListCommits returns commits in the given range. The range format is "BASE..HEAD"
// where BASE and HEAD are commit hashes or ref names. If commitRange is empty,
// all commits reachable from HEAD are returned.
func ListCommits(repoPath string, commitRange string) ([]Commit, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("opening repo: %w", err)
	}

	if commitRange == "" {
		return listAllCommits(repo)
	}

	parts := strings.SplitN(commitRange, "..", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format %q, expected BASE..HEAD", commitRange)
	}

	baseName := strings.TrimSpace(parts[0])
	headName := strings.TrimSpace(parts[1])

	baseHash, err := resolveRef(repo, baseName)
	if err != nil {
		return nil, fmt.Errorf("resolving base %q: %w", baseName, err)
	}

	headHash, err := resolveRef(repo, headName)
	if err != nil {
		return nil, fmt.Errorf("resolving head %q: %w", headName, err)
	}

	return listCommitRange(repo, baseHash, headHash)
}

func resolveRef(repo *git.Repository, name string) (plumbing.Hash, error) {
	// Try as a full hash first
	if plumbing.IsHash(name) {
		return plumbing.NewHash(name), nil
	}

	// Try as a reference name
	ref, err := repo.Reference(plumbing.ReferenceName(name), true)
	if err == nil {
		return ref.Hash(), nil
	}

	// Try common ref prefixes
	for _, prefix := range []string{"refs/heads/", "refs/tags/", "refs/remotes/"} {
		ref, err = repo.Reference(plumbing.ReferenceName(prefix+name), true)
		if err == nil {
			return ref.Hash(), nil
		}
	}

	// Try as abbreviated hash by iterating objects (go-git doesn't support short hashes natively)
	return plumbing.Hash{}, fmt.Errorf("cannot resolve %q to a commit", name)
}

func listAllCommits(repo *git.Repository) ([]Commit, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD: %w", err)
	}

	iter, err := repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, fmt.Errorf("creating log iterator: %w", err)
	}

	var commits []Commit
	err = iter.ForEach(func(c *object.Commit) error {
		commits = append(commits, commitFromObject(c, repo))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating commits: %w", err)
	}

	return commits, nil
}

// notesRefs lists the git-notes namespaces we check for AI authorship logs,
// in priority order. refs/notes/ai is the git-ai standard namespace.
var notesRefs = []string{
	"refs/notes/ai",
}

// readNote reads the git note attached to commitHash under the AI notes refs.
// Returns empty string when no note exists.
func readNote(repo *git.Repository, commitHash plumbing.Hash) string {
	for _, refName := range notesRefs {
		ref, err := repo.Reference(plumbing.ReferenceName(refName), true)
		if err != nil {
			continue
		}

		notesCommit, err := repo.CommitObject(ref.Hash())
		if err != nil {
			continue
		}

		tree, err := notesCommit.Tree()
		if err != nil {
			continue
		}

		// Notes are stored as blobs named by the commit hash they annotate.
		// go-git uses the full hex hash as the path, but some implementations
		// split it as ab/cd1234... so try both.
		hashStr := commitHash.String()
		entry, err := tree.FindEntry(hashStr)
		if err != nil {
			// Try the split format: first 2 chars / remaining
			entry, err = tree.FindEntry(hashStr[:2] + "/" + hashStr[2:])
			if err != nil {
				continue
			}
		}

		blob, err := repo.BlobObject(entry.Hash)
		if err != nil {
			continue
		}

		reader, err := blob.Reader()
		if err != nil {
			continue
		}

		content, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			continue
		}
		if closeErr != nil {
			continue
		}

		return string(content)
	}

	return ""
}

func listCommitRange(repo *git.Repository, base, head plumbing.Hash) ([]Commit, error) {
	// Collect all commits reachable from head
	headCommit, err := repo.CommitObject(head)
	if err != nil {
		return nil, fmt.Errorf("reading head commit: %w", err)
	}

	// Collect ancestors of base to exclude
	baseExclude := map[plumbing.Hash]bool{base: true}
	baseCommit, err := repo.CommitObject(base)
	if err != nil {
		return nil, fmt.Errorf("reading base commit: %w", err)
	}

	baseIter := object.NewCommitIterCTime(baseCommit, nil, nil)
	_ = baseIter.ForEach(func(c *object.Commit) error {
		baseExclude[c.Hash] = true
		return nil
	})

	// Walk from head, collecting commits not in base's ancestry
	var commits []Commit
	headIter := object.NewCommitIterCTime(headCommit, nil, nil)
	err = headIter.ForEach(func(c *object.Commit) error {
		if baseExclude[c.Hash] {
			return nil
		}
		commits = append(commits, commitFromObject(c, repo))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating commits: %w", err)
	}

	return commits, nil
}
