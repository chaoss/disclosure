package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGetHeadTree(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("get worktree: %v", err)
	}

	for _, name := range []string{"AGENTS.md", ".github/SECURITY.md"} {
		fullPath := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("create parent directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("not read by the tree inventory"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		if _, err := worktree.Add(name); err != nil {
			t.Fatalf("stage %s: %v", name, err)
		}
	}

	commitHash, err := worktree.Commit("add policy files", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit files: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "untracked.md"), []byte("untracked"), 0644); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}

	snapshot, err := GetHeadTree(dir)
	if err != nil {
		t.Fatalf("GetHeadTree: %v", err)
	}
	if snapshot.Commit != commitHash.String() {
		t.Errorf("commit = %q, want %q", snapshot.Commit, commitHash.String())
	}
	if want := []string{".github/SECURITY.md", "AGENTS.md"}; !reflect.DeepEqual(snapshot.Paths, want) {
		t.Errorf("paths = %v, want %v", snapshot.Paths, want)
	}
}

func TestGetHeadTreeInvalidRepo(t *testing.T) {
	if _, err := GetHeadTree(t.TempDir()); err == nil {
		t.Fatal("expected an error for a directory that is not a git repository")
	}
}

func TestGetHeadTreeLinkedWorktree(t *testing.T) {
	gitBinary, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git executable is not available")
	}

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("get worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("instructions"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	if _, err := worktree.Add("AGENTS.md"); err != nil {
		t.Fatalf("stage test file: %v", err)
	}
	commitHash, err := worktree.Commit("add instructions", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit test file: %v", err)
	}

	linkedDir := filepath.Join(t.TempDir(), "linked-worktree")
	command := exec.Command(gitBinary, "-C", dir, "worktree", "add", "--detach", linkedDir, commitHash.String())
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("create linked worktree: %v\n%s", err, output)
	}

	snapshot, err := GetHeadTree(linkedDir)
	if err != nil {
		t.Fatalf("GetHeadTree linked worktree: %v", err)
	}
	if snapshot.Commit != commitHash.String() {
		t.Errorf("commit = %q, want %q", snapshot.Commit, commitHash.String())
	}
	if want := []string{"AGENTS.md"}; !reflect.DeepEqual(snapshot.Paths, want) {
		t.Errorf("paths = %v, want %v", snapshot.Paths, want)
	}
}
