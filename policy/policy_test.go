package policy

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestTypeForPath(t *testing.T) {
	tests := []struct {
		path string
		kind string
		want bool
	}{
		{path: "AGENTS.md", kind: "agent_instructions", want: true},
		{path: "docs/CLAUDE.md", kind: "agent_instructions", want: true},
		{path: "GEMINI.md", kind: "agent_instructions", want: true},
		{path: ".github/copilot-instructions.md", kind: "agent_instructions", want: true},
		{path: ".github/instructions/review/go.instructions.md", kind: "agent_instructions", want: true},
		{path: ".cursor/rules/go.mdc", kind: "agent_instructions", want: true},
		{path: ".clinerules", kind: "agent_instructions", want: true},
		{path: ".clinerules/review.md", kind: "agent_instructions", want: true},
		{path: ".windsurf/rules/review.md", kind: "agent_instructions", want: true},
		{path: ".continue/rules/review.md", kind: "agent_instructions", want: true},
		{path: "docs/CODE_OF_CONDUCT.md", kind: "conduct", want: true},
		{path: "docs/code_of_conduct.md", kind: "conduct", want: true},
		{path: "CONTRIBUTING.md", kind: "contributing", want: true},
		{path: ".github/CONTRIBUTING.md", kind: "contributing", want: true},
		{path: "docs/CONTRIBUTING.md", kind: "contributing", want: true},
		{path: "SECURITY.md", kind: "security", want: true},
		{path: "docs/SECURITY.md", kind: "security", want: true},
		{path: "vendor/example/SECURITY.md", want: false},
		{path: "docs/subsystem/SECURITY.md", want: false},
		{path: ".github/actions/CONTRIBUTING.md", want: false},
		{path: "PULL_REQUEST_TEMPLATE.md", kind: "pull_request_template", want: true},
		{path: ".github/PULL_REQUEST_TEMPLATE.md", kind: "pull_request_template", want: true},
		{path: "docs/PULL_REQUEST_TEMPLATE.md", kind: "pull_request_template", want: true},
		{path: ".github/PULL_REQUEST_TEMPLATE/release.md", kind: "pull_request_template", want: true},
		{path: ".github/ISSUE_TEMPLATE.md", kind: "issue_template", want: true},
		{path: ".github/ISSUE_TEMPLATE/bug.yml", kind: "issue_template", want: true},
		{path: "docs/ISSUE_TEMPLATE/nested/bug.yml", want: false},
		{path: "README.md", want: false},
		{path: ".github/NOT_ISSUE_TEMPLATE.md", want: false},
		{path: ".github/instructions/review.md", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gotKind, got := typeForPath(tt.path)
			if got != tt.want {
				t.Fatalf("typeForPath(%q) matched = %v, want %v", tt.path, got, tt.want)
			}
			if got && gotKind != tt.kind {
				t.Errorf("typeForPath(%q) type = %q, want %q", tt.path, gotKind, tt.kind)
			}
		})
	}
}

func TestScanReportsCurrentCommittedFiles(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("get worktree: %v", err)
	}

	for _, name := range []string{
		"AGENTS.md",
		".github/ISSUE_TEMPLATE/bug.yml",
		".github/PULL_REQUEST_TEMPLATE.md",
		"README.md",
	} {
		fullPath := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("create parent directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("contents are not classified"), 0644); err != nil {
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

	if err := os.WriteFile(filepath.Join(dir, "SECURITY.md"), []byte("uncommitted"), 0644); err != nil {
		t.Fatalf("write untracked policy file: %v", err)
	}

	report, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if report.Repository != dir {
		t.Errorf("repository = %q, want %q", report.Repository, dir)
	}
	if report.ScannedCommit != commitHash.String() {
		t.Errorf("scanned commit = %q, want %q", report.ScannedCommit, commitHash.String())
	}
	want := []File{
		{Path: ".github/ISSUE_TEMPLATE/bug.yml", Type: "issue_template"},
		{Path: ".github/PULL_REQUEST_TEMPLATE.md", Type: "pull_request_template"},
		{Path: "AGENTS.md", Type: "agent_instructions"},
	}
	if !reflect.DeepEqual(report.Files, want) {
		t.Errorf("files = %#v, want %#v", report.Files, want)
	}
}

func TestScanDoesNotIncludeFilesOnlyPresentInHistory(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("get worktree: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("old"), 0644); err != nil {
		t.Fatalf("write old policy file: %v", err)
	}
	if _, err := worktree.Add("AGENTS.md"); err != nil {
		t.Fatalf("stage old policy file: %v", err)
	}
	if _, err := worktree.Commit("add old policy file", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	}); err != nil {
		t.Fatalf("commit old policy file: %v", err)
	}

	if err := os.Remove(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatalf("remove old policy file: %v", err)
	}
	if _, err := worktree.Remove("AGENTS.md"); err != nil {
		t.Fatalf("stage removal: %v", err)
	}
	if _, err := worktree.Commit("remove old policy file", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now().Add(time.Second)},
	}); err != nil {
		t.Fatalf("commit removal: %v", err)
	}

	report, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(report.Files) != 0 {
		t.Errorf("files = %#v, want no files from older commits", report.Files)
	}
}

func TestScanInvalidRepo(t *testing.T) {
	if _, err := Scan(t.TempDir()); err == nil {
		t.Fatal("expected an error for a directory that is not a git repository")
	}
}
