package gitops

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func initTestRepo(t *testing.T) (string, []string) {
	t.Helper()
	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	var hashes []string
	for i, msg := range []string{"first commit", "second commit", "third commit"} {
		filename := filepath.Join(dir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(filename, []byte(msg), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if _, err := wt.Add(filepath.Base(filename)); err != nil {
			t.Fatalf("add: %v", err)
		}
		hash, err := wt.Commit(msg, &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test",
				Email: "test@example.com",
				When:  time.Now().Add(time.Duration(i) * time.Second),
			},
		})
		if err != nil {
			t.Fatalf("commit: %v", err)
		}
		hashes = append(hashes, hash.String())
	}

	return dir, hashes
}

func TestGetCommit(t *testing.T) {
	dir, hashes := initTestRepo(t)

	c, err := GetCommit(dir, hashes[0])
	if err != nil {
		t.Fatalf("GetCommit: %v", err)
	}

	if c.Hash != hashes[0] {
		t.Errorf("hash = %q, want %q", c.Hash, hashes[0])
	}
	if c.Message != "first commit" {
		t.Errorf("message = %q, want %q", c.Message, "first commit")
	}
	if c.AuthorEmail != "test@example.com" {
		t.Errorf("author email = %q, want %q", c.AuthorEmail, "test@example.com")
	}
}

func TestGetCommitNotFound(t *testing.T) {
	dir, _ := initTestRepo(t)

	_, err := GetCommit(dir, "0000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for missing commit")
	}
}

func TestListCommitsAll(t *testing.T) {
	dir, hashes := initTestRepo(t)

	commits, err := ListCommits(dir, "")
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}

	if len(commits) != len(hashes) {
		t.Fatalf("got %d commits, want %d", len(commits), len(hashes))
	}

	// Commits should be in reverse chronological order (newest first)
	if commits[0].Hash != hashes[2] {
		t.Errorf("first commit hash = %q, want %q (newest)", commits[0].Hash, hashes[2])
	}
}

func TestListCommitsRange(t *testing.T) {
	dir, hashes := initTestRepo(t)

	// Range from first commit to third: should return second and third
	commits, err := ListCommits(dir, hashes[0]+".."+hashes[2])
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2", len(commits))
	}

	commitHashes := map[string]bool{}
	for _, c := range commits {
		commitHashes[c.Hash] = true
	}

	if !commitHashes[hashes[1]] {
		t.Error("expected second commit in range")
	}
	if !commitHashes[hashes[2]] {
		t.Error("expected third commit in range")
	}
	if commitHashes[hashes[0]] {
		t.Error("base commit should be excluded from range")
	}
}

func TestListCommitsInvalidRange(t *testing.T) {
	dir, _ := initTestRepo(t)

	_, err := ListCommits(dir, "bad-range-format")
	if err == nil {
		t.Error("expected error for invalid range format")
	}
}

func TestListCommitsInvalidRepo(t *testing.T) {
	_, err := ListCommits(t.TempDir(), "")
	if err == nil {
		t.Error("expected error for non-repo directory")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	dir, _ := initTestRepo(t)

	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if err := wt.Checkout(&git.CheckoutOptions{
		Branch: "refs/heads/codex/fix-bug",
		Create: true,
	}); err != nil {
		t.Fatalf("checkout: %v", err)
	}

	branch, err := GetCurrentBranch(dir)
	if err != nil {
		t.Fatalf("GetCurrentBranch: unexpected error: %v", err)
	}
	if branch != "codex/fix-bug" {
		t.Errorf("GetCurrentBranch: got %q, want %q", branch, "codex/fix-bug")
	}
}

func TestGetCurrentBranchDetachedHead(t *testing.T) {
	dir, hashes := initTestRepo(t)

	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if err := wt.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(hashes[0]),
	}); err != nil {
		t.Fatalf("checkout: %v", err)
	}

	branch, err := GetCurrentBranch(dir)
	if err != nil {
		t.Fatalf("GetCurrentBranch: unexpected error: %v", err)
	}
	if branch != "" {
		t.Errorf("GetCurrentBranch: got %q, want empty string for detached HEAD", branch)
	}
}

func TestGetCurrentBranchInvalidRepo(t *testing.T) {
	_, err := GetCurrentBranch(t.TempDir())
	if err == nil {
		t.Error("expected error for non-repo directory")
	}
}
