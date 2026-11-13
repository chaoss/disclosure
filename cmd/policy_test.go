package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chaoss/disclosure/policy"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func initPolicyCommandRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("get worktree: %v", err)
	}

	for name, content := range files {
		fullPath := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("create parent directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		if _, err := worktree.Add(name); err != nil {
			t.Fatalf("stage %s: %v", name, err)
		}
	}
	if _, err := worktree.Commit("add test files", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	}); err != nil {
		t.Fatalf("commit files: %v", err)
	}
	return dir
}

func TestRunPolicyJSON(t *testing.T) {
	dir := initPolicyCommandRepo(t, map[string]string{
		"AGENTS.md":                      "not interpreted",
		".github/ISSUE_TEMPLATE/bug.yml": "not interpreted",
		"README.md":                      "not a policy path",
	})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"policy", "--format=json", dir}, &stdout, &stderr)
	if code != ExitNoAI {
		t.Fatalf("exit code = %d, want success; stderr: %s", code, stderr.String())
	}

	var report policy.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode JSON output: %v (output: %s)", err, stdout.String())
	}
	if report.Repository != dir || report.ScannedCommit == "" {
		t.Errorf("report metadata = %#v, expected repository %q and a scanned commit", report, dir)
	}
	if len(report.Files) != 2 {
		t.Fatalf("found %d files, want 2: %#v", len(report.Files), report.Files)
	}
}

func TestRunPolicyNoMatchesSucceeds(t *testing.T) {
	dir := initPolicyCommandRepo(t, map[string]string{"README.md": "plain project readme"})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"policy", dir}, &stdout, &stderr)
	if code != ExitNoAI {
		t.Fatalf("exit code = %d, want success; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No policy-related files found.") {
		t.Errorf("output = %q, want no-files message", stdout.String())
	}
}

func TestRunPolicyInvalidRepo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"policy", t.TempDir()}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("exit code = %d, want error", code)
	}
}

func TestRunPolicyInvalidFormat(t *testing.T) {
	dir := initPolicyCommandRepo(t, map[string]string{"README.md": "plain project readme"})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"policy", "--format=xml", dir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("exit code = %d, want error", code)
	}
	if !strings.Contains(stderr.String(), "unknown format: xml") {
		t.Errorf("stderr = %q, want unknown-format message", stderr.String())
	}
}
