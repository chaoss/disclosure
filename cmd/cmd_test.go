package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chaoss/disclosure/detection"
	"github.com/chaoss/disclosure/scan"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func initTestRepo(t *testing.T) string {
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

	commits := []struct {
		msg            string
		committerEmail string
	}{
		{"initial commit", "human@example.com"},
		{"fix: update handler\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>", "human@example.com"},
		{"aider: refactor auth module", "human@example.com"},
	}

	for i, c := range commits {
		filename := filepath.Join(dir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(filename, []byte(c.msg), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if _, err := wt.Add(filepath.Base(filename)); err != nil {
			t.Fatalf("add: %v", err)
		}
		_, err := wt.Commit(c.msg, &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test",
				Email: c.committerEmail,
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
	}

	return dir
}

func TestRunNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(nil, &stdout, &stderr)
	if code != ExitNoAI {
		t.Errorf("exit code = %d, want %d", code, ExitNoAI)
	}
	if !strings.Contains(stdout.String(), "disclosure") {
		t.Errorf("expected help output, got: %s", stdout.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"bogus"}, &stdout, &stderr)
	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
}

func TestRunVersion(t *testing.T) {
	originalVersion := Version
	Version = "1.2.3"
	t.Cleanup(func() {
		Version = originalVersion
	})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, &stdout, &stderr)
	if code != ExitNoAI {
		t.Errorf("exit code = %d, want %d", code, ExitNoAI)
	}
	if got, want := stdout.String(), "disclosure 1.2.3\n"; got != want {
		t.Errorf("version output = %q, want %q", got, want)
	}
}

func TestRunScanText(t *testing.T) {
	dir := initTestRepo(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--format=text", dir}, &stdout, &stderr)

	if code != ExitAI {
		t.Errorf("exit code = %d, want %d (stderr: %s)", code, ExitAI, stderr.String())
	}
	if !strings.Contains(stdout.String(), "AI signals") {
		t.Errorf("expected AI signals in output, got:\n%s", stdout.String())
	}
}

func TestRunScanJSON(t *testing.T) {
	dir := initTestRepo(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--format=json", dir}, &stdout, &stderr)

	if code != ExitAI {
		t.Errorf("exit code = %d, want %d (stderr: %s)", code, ExitAI, stderr.String())
	}

	var report scan.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal: %v (output: %s)", err, stdout.String())
	}
	if report.Summary.AICommits == 0 {
		t.Error("expected AI commits in report")
	}
}

func TestRunScanMinConfidence(t *testing.T) {
	dir := initTestRepo(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--format=json", "--min-confidence=high", dir}, &stdout, &stderr)

	var report scan.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Only high confidence findings should remain (co-author trailer)
	for _, cr := range report.Commits {
		for _, f := range cr.Findings {
			if f.Confidence < 3 {
				t.Errorf("found confidence %d below minimum high(3)", f.Confidence)
			}
		}
	}

	_ = code // exit code depends on whether high-confidence findings exist
}

func TestRunScanNoAI(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	filename := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(filename, []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := wt.Add("file.txt"); err != nil {
		t.Fatalf("add: %v", err)
	}
	_, err = wt.Commit("normal commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Human",
			Email: "human@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", dir}, &stdout, &stderr)
	if code != ExitNoAI {
		t.Errorf("exit code = %d, want %d (stderr: %s, stdout: %s)", code, ExitNoAI, stderr.String(), stdout.String())
	}
}

func TestRunTextCommandDetectsAI(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "input.txt")

	os.WriteFile(file, []byte(
		"I used Claude to write this",
	), 0644)

	var stdout, stderr bytes.Buffer

	code := Run([]string{
		"text",
		"--input=" + file,
	}, &stdout, &stderr)

	if code != ExitAI {
		t.Errorf("code=%d want AI", code)
	}
}

func TestRunTextCommandNoAI(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "input.txt")
	os.WriteFile(file, []byte("plain text"), 0644)

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"text",
		"--input=" + file,
	}, &stdout, &stderr)

	if code != ExitNoAI {
		t.Errorf("code=%d want no AI", code)
	}
}

func TestRunScanInvalidRepo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", t.TempDir()}, &stdout, &stderr)
	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
}

func TestRunScanInvalidFormat(t *testing.T) {
	dir := initTestRepo(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--format=xml", dir}, &stdout, &stderr)

	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}

	if !strings.Contains(stderr.String(), "unknown format: xml") {
		t.Errorf("expected invalid format error, got: %s", stderr.String())
	}
}

func TestFilterReport(t *testing.T) {
	report := scan.Report{
		Commits: []scan.CommitResult{
			{
				Hash: "abc123",
				Findings: []detection.Finding{
					{Detector: "toolmention", Tool: "Claude", Confidence: 1, Detail: "text"},
					{Detector: "trailer", Tool: "Claude Code", Confidence: 3, Detail: "trailer"},
				},
			},
		},
		Summary: scan.Summary{
			TotalCommits: 1,
			AICommits:    1,
			ToolCounts:   map[string]int{"Claude": 1, "Claude Code": 1},
			ByConfidence: map[string]int{"low": 1, "high": 1},
		},
	}

	filtered := filterReport(report, 3) // high only
	if len(filtered.Commits[0].Findings) != 1 {
		t.Fatalf("expected 1 finding after filter, got %d", len(filtered.Commits[0].Findings))
	}
	if filtered.Commits[0].Findings[0].Tool != "Claude Code" {
		t.Errorf("expected Claude Code, got %s", filtered.Commits[0].Findings[0].Tool)
	}
	if filtered.Summary.AICommits != 1 {
		t.Errorf("ai_commits = %d, want 1", filtered.Summary.AICommits)
	}
}

func TestFilterReportAllFiltered(t *testing.T) {
	report := scan.Report{
		Commits: []scan.CommitResult{
			{
				Hash: "abc123",
				Findings: []detection.Finding{
					{
						Detector:   "toolmention",
						Confidence: detection.ConfidenceLow,
					},
				},
			},
		},
		Summary: scan.Summary{
			TotalCommits: 1,
		},
	}

	filtered := filterReport(report, detection.ConfidenceHigh)

	if filtered.Summary.AICommits != 0 {
		t.Fatalf("AICommits=%d, want 0", filtered.Summary.AICommits)
	}

	if len(filtered.Commits[0].Findings) != 0 {
		t.Fatalf("expected no findings")
	}
}

func TestRunDocsMarkdownDefault(t *testing.T) {
	// Clean up default directory paths after the test finishes
	defer func() {
		_ = os.RemoveAll("./docs")
	}()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"docs"}, &stdout, &stderr)

	if code != ExitNoAI {
		t.Errorf("exit code = %d, want %d (stderr: %s)", code, ExitNoAI, stderr.String())
	}

	defaultDir := filepath.FromSlash("./docs/cli/markdown")
	if _, err := os.Stat(defaultDir); os.IsNotExist(err) {
		t.Fatalf("expected markdown output directory to exist: %s", defaultDir)
	}

	files, err := os.ReadDir(defaultDir)
	if err != nil || len(files) == 0 {
		t.Error("expected documentation files inside the markdown directory")
	}
}

func TestRunDocsFormats(t *testing.T) {
	tests := []struct {
		format     string
		expectFile string
	}{
		{format: "markdown", expectFile: ".md"},
		{format: "manpages", expectFile: "1"},
		{format: "rest", expectFile: ".rst"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			tmpDir := t.TempDir()

			var stdout, stderr bytes.Buffer
			code := Run([]string{"docs", "--format=" + tt.format, "--out=" + tmpDir}, &stdout, &stderr)

			if code != ExitNoAI {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, ExitNoAI, stderr.String())
			}

			docDir := filepath.Join(tmpDir, tt.format)
			if _, err := os.Stat(docDir); os.IsNotExist(err) {
				t.Fatalf("expected format directory to exist: %s", docDir)
			}

			files, err := os.ReadDir(docDir)
			if err != nil || len(files) == 0 {
				t.Fatalf("no files generated for format: %s", tt.format)
			}

			foundMatch := false
			for _, f := range files {
				if strings.Contains(strings.ToLower(f.Name()), tt.expectFile) {
					foundMatch = true
					break
				}
				fileInfo, err := f.Info()
				if err != nil {
					t.Fatalf("error getting info for file: %s", f.Name())
				}
				if fileInfo.Size() == 0 {
					t.Errorf("expected size to be non-zero for file: %s", f.Name())
				}
			}
			if !foundMatch {
				t.Errorf("could not find expected documentation artifact matching '%s' in output", tt.expectFile)
			}
		})
	}
}

func TestRunDocsInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"docs", "--format=html", "--out=" + tmpDir}, &stdout, &stderr)

	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
	if !strings.Contains(stderr.String(), "unknown format: html") {
		t.Errorf("expected unknown format error message, got: %s", stderr.String())
	}
}

func TestRunDocsInvalidArgument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"docs", "unexpected-argument"}, &stdout, &stderr)

	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
}

func TestRunDocsWriteError(t *testing.T) {
	tmpDir := t.TempDir()
	blockedPath := filepath.Join(tmpDir, "blocked_file")
	if err := os.WriteFile(blockedPath, []byte("this is a test file"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"docs", "--out=" + blockedPath}, &stdout, &stderr)

	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
}

func TestRunDocsEmptyFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{
		"docs",
		"--format=",
	}, &stdout, &stderr)

	if code != ExitError {
		t.Errorf("exit code=%d want error", code)
	}
}

func TestRunScanWithWeightsFlag(t *testing.T) {
	dir := initTestRepo(t)

	var stdout, stderr bytes.Buffer
	// Give trailer weight 0 and toolmention weight 1 so only toolmention contributes
	code := Run([]string{"scan", "--format=json", "--weights=trailer=0.55,toolmention=0.45", dir}, &stdout, &stderr)
	if code != ExitAI && code != ExitNoAI {
		t.Fatalf("unexpected exit code: %d (stderr: %s)", code, stderr.String())
	}

	var report scan.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal: %v (output: %s)", err, stdout.String())
	}

	var all []detection.Finding
	for _, cr := range report.Commits {
		all = append(all, cr.Findings...)
	}

	weights := map[string]float64{"trailer": 0.55, "toolmention": 0.45}
	expectedOverall, _ := detection.ConsolidateFindingScore(all, weights)
	if report.Summary.OverallScore != expectedOverall {
		t.Fatalf("overall score = %v, want %v (weights applied)", report.Summary.OverallScore, expectedOverall)
	}
}

func TestRunScanWithConfidenceScoresFlag(t *testing.T) {
	dir := initTestRepo(t)

	var stdout, stderr bytes.Buffer

	// override confidence scores: low=10, medium=50, high=90
	code := Run([]string{"scan", "--format=json", "--confidence-scores=low=10,medium=50,high=90", dir}, &stdout, &stderr)
	if code != ExitAI && code != ExitNoAI {
		t.Fatalf("unexpected exit code: %d (stderr: %s)", code, stderr.String())
	}

	var report scan.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal: %v (output: %s)", err, stdout.String())
	}

	var all []detection.Finding
	for _, cr := range report.Commits {
		all = append(all, cr.Findings...)
	}

	expectedOverall, _ := detection.ConsolidateFindingScore(all, nil)
	if report.Summary.OverallScore != expectedOverall {
		t.Fatalf("overall score = %v, want %v (conf scores applied)", report.Summary.OverallScore, expectedOverall)
	}
}

func TestRunScanInvalidMinConfidence(t *testing.T) {
	dir := initTestRepo(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"scan",
		"--min-confidence=invalid",
		dir,
	}, &stdout, &stderr)

	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}

	if !strings.Contains(stderr.String(), "invalid confidence") {
		t.Errorf("expected confidence error, got: %s", stderr.String())
	}
}

func TestRunScanInvalidWeightsFormat(t *testing.T) {
	dir := initTestRepo(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"scan",
		"--weights=trailer",
		dir,
	}, &stdout, &stderr)

	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
}

func TestRunScanInvalidConfidenceScoresFormat(t *testing.T) {
	dir := initTestRepo(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"scan",
		"--confidence-scores=low",
		dir,
	}, &stdout, &stderr)

	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
}

func TestFilterReportRecalculatesScore(t *testing.T) {
	report := scan.Report{
		Commits: []scan.CommitResult{
			{
				Hash: "abc123",
				Findings: []detection.Finding{
					{
						Detector:   "toolmention",
						Confidence: detection.ConfidenceLow,
						Score:      20,
					},
					{
						Detector:   "trailer",
						Confidence: detection.ConfidenceHigh,
						Score:      100,
					},
				},
			},
		},
	}

	filtered := filterReport(report, detection.ConfidenceHigh)

	if filtered.Commits[0].Score != 100 {
		t.Errorf(
			"score=%v want 100",
			filtered.Commits[0].Score,
		)
	}
}

func TestRunScanRejectsNaNWeights(t *testing.T) {
	dir := initTestRepo(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"scan",
		"--weights=trailer=NaN",
		dir,
	}, &stdout, &stderr)

	if code != ExitError {
		t.Errorf("expected error for NaN weight")
	}
}

func TestRunScanRejectsNaNConfidenceScore(t *testing.T) {
	dir := initTestRepo(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"scan",
		"--confidence-scores=high=NaN",
		dir,
	}, &stdout, &stderr)

	if code != ExitError {
		t.Errorf("expected error for NaN confidence score")
	}
}

func TestRunScanInvalidWeightsMissingValue(t *testing.T) {
	dir := initTestRepo(t)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--weights=trailer=,toolmention=0.5", dir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected ExitError for missing weight value, got %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid number in weights") {
		t.Fatalf("expected numeric parse error, got stderr: %s", stderr.String())
	}
}
