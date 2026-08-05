package scan

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/chaoss/disclosure/detection"
	"github.com/chaoss/disclosure/detection/committer"
	"github.com/chaoss/disclosure/detection/gitnotes"
	"github.com/chaoss/disclosure/detection/toolmention"
	"github.com/chaoss/disclosure/detection/trailer"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func allDetectors() []detection.Detector {
	return []detection.Detector{
		&committer.Detector{},
		&gitnotes.Detector{},
		&trailer.Detector{},
		&toolmention.Detector{},
	}
}

func initTestRepo(t *testing.T) (string, []string) {
	t.Helper()
	const humanEmail = "human@example.com"

	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	commits := []struct {
		msg            string
		authorEmail    string
		committerEmail string
	}{
		{"initial commit", humanEmail, humanEmail},
		{"fix: update handler\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>", humanEmail, humanEmail},
		{"aider: refactor auth module", humanEmail, humanEmail},
		{`
this is a commit message

Co-Authored-By: Cursor <cursoragent@cursor.com>

Assisted-by: Claude 4.7 Opus
	(logic optimization and design fixes)
Assisted-by: Kimi K2.6 (unit tests, integration tests)
Assisted-by: ChatGPT (documentation review)
Assisted-by: Gemini (documentation)
`,
			humanEmail,
			humanEmail,
		},
		{
			"agent-authored change\n\nAssisted-By: Kimi K2.6",
			"198982749+copilot@users.noreply.github.com",
			humanEmail,
		},
	}

	var hashes []string
	for i, c := range commits {
		filename := filepath.Join(dir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(filename, []byte(c.msg), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if _, err := wt.Add(filepath.Base(filename)); err != nil {
			t.Fatalf("add: %v", err)
		}
		hash, err := wt.Commit(c.msg, &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test",
				Email: c.authorEmail,
				When:  time.Now().Add(time.Duration(i) * time.Second),
			},
			Committer: &object.Signature{
				Name:  "Test",
				Email: c.committerEmail,
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

func TestScanCommitRange(t *testing.T) {
	dir, hashes := initTestRepo(t)
	detectors := allDetectors()

	report, err := ScanCommitRange(dir, hashes[0]+".."+hashes[3], detectors)
	if err != nil {
		t.Fatalf("ScanCommitRange: %v", err)
	}

	if report.Summary.TotalCommits != 3 {
		t.Errorf("total commits = %d, want 3", report.Summary.TotalCommits)
	}

	if report.Summary.AICommits != 3 {
		t.Errorf("ai commits = %d, want 3", report.Summary.AICommits)
	}

	// Check that Claude Code was detected via co-author
	if count, ok := report.Summary.ToolCounts["Claude Code"]; !ok || count == 0 {
		t.Error("expected Claude Code in tool counts")
	}

	// Check that Aider was detected via message pattern
	if count, ok := report.Summary.ToolCounts["Aider"]; !ok || count == 0 {
		t.Error("expected Aider in tool counts")
	}

	// Check detection via assisted-by pattern 1
	if count, ok := report.Summary.ToolCounts["ChatGPT"]; !ok || count == 0 {
		t.Error("expected ChatGPT in tool counts")
	}

	// Check detection via assisted-by pattern 2
	if count, ok := report.Summary.ToolCounts["Claude 4.7 Opus"]; !ok || count == 0 {
		t.Error("expected Claude 4.7 Opus in tool counts")
	}

	// Check detection via assisted-by pattern 3
	if count, ok := report.Summary.ToolCounts["Kimi K2.6"]; !ok || count == 0 {
		t.Error("expected Kimi K2.6 Opus in tool counts")
	}
}

func TestScanCommitRangeAll(t *testing.T) {
	dir, _ := initTestRepo(t)
	detectors := allDetectors()

	report, err := ScanCommitRange(dir, "", detectors)
	if err != nil {
		t.Fatalf("ScanCommitRange: %v", err)
	}

	if report.Summary.TotalCommits != 5 {
		t.Errorf("total commits = %d, want 5", report.Summary.TotalCommits)
	}
}

func TestScanCommitDetectsAcrossDetectors(t *testing.T) {
	dir, hashes := initTestRepo(t)

	result, err := ScanCommit(dir, hashes[4], allDetectors())
	if err != nil {
		t.Fatalf("ScanCommit: %v", err)
	}

	wantToolsByDetector := map[string]string{
		(&committer.Detector{}).Name():   "GitHub Copilot (agent)",
		(&trailer.Detector{}).Name():     "Kimi K2.6",
		(&toolmention.Detector{}).Name(): "Kimi",
	}
	if len(result.Findings) != len(wantToolsByDetector) {
		t.Fatalf("got %d findings, want %d", len(result.Findings), len(wantToolsByDetector))
	}

	for _, finding := range result.Findings {
		wantTool, ok := wantToolsByDetector[finding.Detector]
		if !ok {
			t.Errorf("unexpected detector %q", finding.Detector)
			continue
		}
		if finding.Tool != wantTool {
			t.Errorf("%s tool = %q, want %q", finding.Detector, finding.Tool, wantTool)
		}
		delete(wantToolsByDetector, finding.Detector)
	}
}

func TestScanCommit(t *testing.T) {
	dir, hashes := initTestRepo(t)
	detectors := allDetectors()

	// Scan the commit with assisted-by and co-author trailers
	result, err := ScanCommit(dir, hashes[3], detectors)
	if err != nil {
		t.Fatalf("ScanCommit: %v", err)
	}

	if result.Hash != hashes[3] {
		t.Errorf("hash = %q, want %q", result.Hash, hashes[3])
	}

	if len(result.Findings) == 0 {
		t.Error("expected findings for assisted-by and co-author trailers")
	}

	foundCoauthor := false
	foundAssistedBy := false
	for _, f := range result.Findings {
		if f.Detector == "trailer" && f.Tool == "Cursor" {
			foundCoauthor = true
		} else if f.Detector == "trailer" && f.Tool == "Kimi K2.6" {
			foundAssistedBy = true
		}
	}
	if !foundCoauthor {
		t.Error("expected coauthor finding for Cursor")
	}
	if !foundAssistedBy {
		t.Error("expected assistedby finding for Kimi K2.6")
	}
}

func TestScanText(t *testing.T) {
	detectors := allDetectors()

	findings := ScanText("I used Claude to write this PR", detectors)
	if len(findings) == 0 {
		t.Error("expected findings for text mentioning Claude")
	}

	foundClaude := false
	for _, f := range findings {
		if f.Tool == "Claude" && f.Detector == "toolmention" {
			foundClaude = true
		}
	}
	if !foundClaude {
		t.Error("expected toolmention finding for Claude")
	}
}

func TestScanTextNoFindings(t *testing.T) {
	detectors := allDetectors()

	findings := ScanText("This is a normal PR description", detectors)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestScanCommitWithGitNotes(t *testing.T) {
	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	filename := filepath.Join(dir, "main.rs")
	if err := os.WriteFile(filename, []byte("fn main() {}"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := wt.Add("main.rs"); err != nil {
		t.Fatalf("add: %v", err)
	}

	hash, err := wt.Commit("add main", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "human@example.com",
			When:  time.Now(),
		},
		Committer: &object.Signature{
			Name:  "Test",
			Email: "human@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Attach a git-ai note using the git CLI
	noteContent := `src/main.rs
  abcd1234abcd1234 1
---
{
  "schema_version": "authorship/3.0.0",
  "base_commit_sha": "0000000000000000000000000000000000000000",
  "prompts": {
    "abcd1234abcd1234": {
      "agent_id": {
        "tool": "cursor",
        "model": "claude-4.5-opus"
      },
      "total_additions": 1,
      "total_deletions": 0,
      "accepted_lines": 1,
      "overriden_lines": 0
    }
  }
}`

	// Configure git identity for the notes commit (CI runners may not have one)
	for _, kv := range [][2]string{{"user.name", "Test"}, {"user.email", "test@test.com"}} {
		cfg := exec.Command("git", "config", kv[0], kv[1])
		cfg.Dir = dir
		if out, err := cfg.CombinedOutput(); err != nil {
			t.Fatalf("git config %s: %v\n%s", kv[0], err, out)
		}
	}

	cmd := exec.Command("git", "notes", "--ref=refs/notes/ai", "add", "-m", noteContent, hash.String())
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git notes add: %v\n%s", err, out)
	}

	detectors := allDetectors()
	result, err := ScanCommit(dir, hash.String(), detectors)
	if err != nil {
		t.Fatalf("ScanCommit: %v", err)
	}

	foundGitNotes := false
	for _, f := range result.Findings {
		if f.Detector == "gitnotes" && f.Tool == "cursor" {
			foundGitNotes = true
			if f.Confidence != detection.ConfidenceHigh {
				t.Errorf("confidence = %d, want high(%d)", f.Confidence, detection.ConfidenceHigh)
			}
		}
	}
	if !foundGitNotes {
		t.Errorf("expected gitnotes finding for cursor, got findings: %v", result.Findings)
	}
}

func TestReportSummaryByConfidence(t *testing.T) {
	dir, hashes := initTestRepo(t)
	detectors := allDetectors()

	report, err := ScanCommitRange(dir, hashes[0]+".."+hashes[2], detectors)
	if err != nil {
		t.Fatalf("ScanCommitRange: %v", err)
	}

	// Co-author and Assisted-By trailers should give high confidence
	if count, ok := report.Summary.ByConfidence["high"]; !ok || count == 0 {
		t.Error("expected high confidence findings")
	}

	// Message pattern should give medium confidence
	if count, ok := report.Summary.ByConfidence["medium"]; !ok || count == 0 {
		t.Error("expected medium confidence findings")
	}
}
