package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func TestRunTextCommandMinConfidence(t *testing.T) {
	tmp := t.TempDir()

	// Case 1: Plain text tool mention without checkbox has score 20 (low confidence).
	toolMentionFile := filepath.Join(tmp, "tool_mention.txt")
	os.WriteFile(toolMentionFile, []byte("I used Claude to write this"), 0644)

	// Low min-confidence detects low-confidence tool mention.
	{
		var stdout, stderr bytes.Buffer
		code := Run([]string{"text", "--input=" + toolMentionFile, "--min-confidence=low"}, &stdout, &stderr)
		if code != ExitAI {
			t.Errorf("tool mention min-confidence=low: code = %d, want %d", code, ExitAI)
		}
	}

	// Medium and high min-confidence filter out low-confidence tool mention.
	{
		var stdout, stderr bytes.Buffer
		code := Run([]string{"text", "--input=" + toolMentionFile, "--min-confidence=medium"}, &stdout, &stderr)
		if code != ExitNoAI {
			t.Errorf("tool mention min-confidence=medium: code = %d, want %d", code, ExitNoAI)
		}
		if !strings.Contains(stdout.String(), "No AI involvement detected.") {
			t.Errorf("tool mention min-confidence=medium: expected 'No AI involvement detected.', got: %s", stdout.String())
		}
	}
	{
		var stdout, stderr bytes.Buffer
		code := Run([]string{"text", "--input=" + toolMentionFile, "--min-confidence=high"}, &stdout, &stderr)
		if code != ExitNoAI {
			t.Errorf("tool mention min-confidence=high: code = %d, want %d", code, ExitNoAI)
		}
	}
	{
		var stdout, stderr bytes.Buffer
		code := Run([]string{"text", "--input=" + toolMentionFile, "--format=json", "--min-confidence=medium"}, &stdout, &stderr)
		if code != ExitNoAI {
			t.Errorf("tool mention JSON min-confidence=medium: code = %d, want %d", code, ExitNoAI)
		}
		var result struct {
			Findings []detection.Finding `json:"findings"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal json: %v", err)
		}
		if len(result.Findings) != 0 {
			t.Errorf("tool mention JSON min-confidence=medium: got %d findings, want 0", len(result.Findings))
		}
	}

	// Case 2: Ticked AI used checkbox with tool mention has score 95 (high confidence).
	// When --enable-checkbox-detection is passed, min-confidence=high, medium, and low all retain it.
	checkboxWithToolFile := filepath.Join(tmp, "checkbox_with_tool.txt")
	os.WriteFile(checkboxWithToolFile, []byte("[x] This contribution was assisted or created by Generative AI tools.\n\nClaude was used for drafting."), 0644)
	{
		for _, minConf := range []string{"low", "medium", "high"} {
			var stdout, stderr bytes.Buffer
			code := Run([]string{
				"text",
				"--enable-checkbox-detection",
				"--input=" + checkboxWithToolFile,
				"--min-confidence=" + minConf,
			}, &stdout, &stderr)
			if code != ExitAI {
				t.Errorf("ticked checkbox with tool min-confidence=%s: code = %d, want %d (stderr=%s)", minConf, code, ExitAI, stderr.String())
			}
		}

		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"text",
			"--enable-checkbox-detection",
			"--input=" + checkboxWithToolFile,
			"--format=json",
			"--min-confidence=high",
		}, &stdout, &stderr)
		if code != ExitAI {
			t.Errorf("ticked checkbox with tool JSON min-confidence=high: code = %d, want %d", code, ExitAI)
		}
		var result struct {
			Findings []detection.Finding `json:"findings"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal json: %v", err)
		}
		if len(result.Findings) != 1 {
			t.Fatalf("ticked checkbox with tool: got %d findings, want 1", len(result.Findings))
		}
		if result.Findings[0].Confidence != detection.ConfidenceHigh || result.Findings[0].Score != 95 {
			t.Errorf("ticked checkbox with tool: score=%f, confidence=%v, want 95.0, high", result.Findings[0].Score, result.Findings[0].Confidence)
		}
	}

	// Case 3: Ticked AI used checkbox without tool mention has score 75 (high confidence).
	// Retained at min-confidence=high.
	checkboxOnlyFile := filepath.Join(tmp, "checkbox_only.txt")
	os.WriteFile(checkboxOnlyFile, []byte("[x] This contribution was assisted or created by Generative AI tools."), 0644)
	{
		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"text",
			"--enable-checkbox-detection",
			"--input=" + checkboxOnlyFile,
			"--min-confidence=high",
		}, &stdout, &stderr)
		if code != ExitAI {
			t.Errorf("ticked checkbox only min-confidence=high: code = %d, want %d", code, ExitAI)
		}
	}

	// Case 4: Ticked NOT-used checkbox with tool mention has score 20 (low confidence).
	// Filtered out at min-confidence=medium and min-confidence=high.
	checkboxNotUsedFile := filepath.Join(tmp, "checkbox_not_used.txt")
	os.WriteFile(checkboxNotUsedFile, []byte("[x] This contribution was NOT assisted or created by Generative AI tools.\n\nClaude was mentioned in comments."), 0644)
	{
		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"text",
			"--enable-checkbox-detection",
			"--input=" + checkboxNotUsedFile,
			"--min-confidence=medium",
		}, &stdout, &stderr)
		if code != ExitNoAI {
			t.Errorf("ticked NOT-used checkbox min-confidence=medium: code = %d, want %d", code, ExitNoAI)
		}
	}

	// Case 5: Invalid min-confidence should return ExitError.
	{
		var stdout, stderr bytes.Buffer
		code := Run([]string{"text", "--input=" + toolMentionFile, "--min-confidence=bogus"}, &stdout, &stderr)
		if code != ExitError {
			t.Errorf("min-confidence=bogus: code = %d, want %d", code, ExitError)
		}
	}
}

func TestRunTextCommandCustomConfidenceLevels(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "input.txt")
	os.WriteFile(file, []byte("I used Claude to write this"), 0644)

	// Tool mention score is 20. Overriding low=10, medium=25 promotes score 20 to medium confidence (10 < 20 <= 25).
	// Aggregate confidence should also be reported as medium.
	{
		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"text",
			"--input=" + file,
			"--confidence-levels=low=10,medium=25,high=50",
			"--min-confidence=medium",
		}, &stdout, &stderr)
		if code != ExitAI {
			t.Errorf("custom confidence levels: code = %d, want %d", code, ExitAI)
		}
		if !strings.Contains(stdout.String(), "Score: 20.0, Confidence: medium") {
			t.Errorf("expected aggregate confidence 'Score: 20.0, Confidence: medium', got:\n%s", stdout.String())
		}
	}

	// Reviewer example: low=10,medium=25,high=50 returns a medium finding (score 20 <= 25).
	// Aggregate confidence must be reported as medium, not default low.
	{
		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"text",
			"--input=" + file,
			"--confidence-levels=low=10,medium=25,high=50",
		}, &stdout, &stderr)
		if code != ExitAI {
			t.Errorf("custom confidence levels (low=10,medium=25,high=50): code = %d, want %d", code, ExitAI)
		}
		if !strings.Contains(stdout.String(), "Score: 20.0, Confidence: medium") {
			t.Errorf("expected aggregate confidence 'Score: 20.0, Confidence: medium', got:\n%s", stdout.String())
		}
	}

	// JSON format should report aggregate confidence as medium when using custom thresholds.
	{
		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"text",
			"--input=" + file,
			"--format=json",
			"--confidence-levels=low=10,medium=25,high=50",
		}, &stdout, &stderr)
		if code != ExitAI {
			t.Errorf("custom confidence levels JSON: code = %d, want %d", code, ExitAI)
		}
		var result struct {
			Findings   []detection.Finding  `json:"findings"`
			Score      float64              `json:"score"`
			Confidence detection.Confidence `json:"confidence"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal json: %v", err)
		}
		if result.Confidence != detection.ConfidenceMedium {
			t.Errorf("custom confidence levels JSON aggregate confidence: got %v, want %v", result.Confidence, detection.ConfidenceMedium)
		}
		if len(result.Findings) != 1 {
			t.Fatalf("custom confidence levels JSON findings: got %d, want 1", len(result.Findings))
		}
		if result.Findings[0].Confidence != detection.ConfidenceMedium {
			t.Errorf("custom confidence levels JSON finding confidence: got %v, want %v", result.Findings[0].Confidence, detection.ConfidenceMedium)
		}
	}

	// Invalid confidence-levels format should return ExitError.
	{
		var stdout, stderr bytes.Buffer
		code := Run([]string{
			"text",
			"--input=" + file,
			"--confidence-levels=invalid",
		}, &stdout, &stderr)
		if code != ExitError {
			t.Errorf("invalid confidence-levels: code = %d, want %d", code, ExitError)
		}
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
	tests := []struct {
		name                  string
		report                scan.Report
		minConf               detection.Confidence
		wantAICommits         int
		wantFindings          []int
		wantPerDetectorScores []map[string]float64
		wantScores            []float64
		wantConfidence        detection.Confidence
		wantTool              string // optional
	}{
		{
			name: "keep low confidence and above findings",
			report: scan.Report{
				Commits: []scan.CommitResult{
					{
						Hash: "abc123",
						Findings: []detection.Finding{
							{
								Detector:   "toolmention",
								Tool:       "Claude",
								Confidence: detection.ConfidenceMedium,
								Score:      20,
							},
							{
								Detector:   "trailer",
								Tool:       "Kimi K3",
								Confidence: detection.ConfidenceHigh,
								Score:      85,
							},
							{
								Detector:   "trailer",
								Tool:       "Claude Code",
								Confidence: detection.ConfidenceHigh,
								Score:      100,
							},
						},
					},
					{
						Hash: "abc456",
						Findings: []detection.Finding{
							{
								Detector:   "toolmention",
								Tool:       "Claude",
								Confidence: detection.ConfidenceMedium,
								Score:      20,
							},
							{
								Detector:   "trailer",
								Tool:       "Claude Code",
								Confidence: detection.ConfidenceHigh,
								Score:      105,
							},
						},
					},
				},
			},
			minConf:       detection.ConfidenceMedium,
			wantAICommits: 2,
			wantFindings:  []int{3, 2},
			wantTool:      "Claude",
			wantPerDetectorScores: []map[string]float64{
				{"toolmention": 20, "trailer": 100},
				{"toolmention": 20, "trailer": 105},
			},
			wantScores: []float64{120, 125},
		},
		{
			name: "all findings filtered out",
			report: scan.Report{
				Commits: []scan.CommitResult{
					{
						Hash: "abc123",
						Findings: []detection.Finding{
							{
								Detector:   "toolmention",
								Confidence: detection.ConfidenceLow,
								Score:      20,
							},
						},
					},
				},
			},
			minConf:       detection.ConfidenceHigh,
			wantAICommits: 0,
			wantFindings:  []int{0},
			wantScores:    []float64{0},
		},
		{
			name: "empty findings",
			report: scan.Report{
				Commits: []scan.CommitResult{
					{
						Hash:     "abc123",
						Findings: nil,
					},
				},
			},
			minConf:       detection.ConfidenceMedium,
			wantAICommits: 0,
			wantFindings:  []int{0},
			wantScores:    []float64{0},
		},
		{
			name:          "no commits",
			report:        scan.Report{},
			minConf:       detection.ConfidenceHigh,
			wantAICommits: 0,
			wantFindings:  []int{},
			wantScores:    []float64{},
		},
		{
			name: "low confidence threshold returns original report",
			report: scan.Report{
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
								Score:      80,
							},
						},
						Score:      100,
						Confidence: detection.ConfidenceHigh,
					},
				},
			},
			minConf:      detection.ConfidenceLow,
			wantFindings: []int{2},
			wantScores:   []float64{100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterReport(tt.report, tt.minConf, detection.GetDefaultConfidenceLevels())

			if tt.wantFindings != nil {
				lenCommit := len(filtered.Commits)
				lenWant := len(tt.wantFindings)
				if lenCommit != lenWant {
					t.Fatalf("invalid number of items to check, commit len=%d want len=%d", lenCommit, lenWant)
				}
				for i := range filtered.Commits {
					if len(filtered.Commits[i].Findings) != tt.wantFindings[i] {
						t.Fatalf("commit findings len=%d want findings=%d", lenCommit, lenWant)
					}
				}
			}

			if tt.wantScores != nil {
				lenCommit := len(filtered.Commits)
				lenWant := len(tt.wantScores)
				if lenCommit != lenWant {
					t.Fatalf("invalid number of items to check, commit len=%d want len=%d", lenCommit, lenWant)
				}
				for i := range filtered.Commits {
					if filtered.Commits[i].Score != tt.wantScores[i] {
						t.Fatalf("commit score=%f want score=%f", filtered.Commits[i].Score, tt.wantScores[i])
					}
				}
			}

			if filtered.Summary.AICommits != tt.wantAICommits {
				t.Errorf("AICommits=%d want=%d", filtered.Summary.AICommits, tt.wantAICommits)
			}

			gotFindings := 0
			for i, commit := range filtered.Commits {
				gotFindings += len(commit.Findings)

				// compare per-detector scores for each commit
				if tt.wantPerDetectorScores == nil {
					continue
				}
				wantPerDetectorScores := tt.wantPerDetectorScores[i]
				lenCommitScores := len(commit.PerDetectorScores)
				lenWantScores := len(wantPerDetectorScores)
				if lenCommitScores != lenWantScores {
					t.Errorf("expected %d detectors in map but found=%d", lenWantScores, lenCommitScores)
				}
				for detector := range commit.PerDetectorScores {
					if wantPerDetectorScores[detector] != commit.PerDetectorScores[detector] {
						t.Errorf(
							"expected %f, found %f for detector %q",
							wantPerDetectorScores[detector],
							commit.PerDetectorScores[detector],
							detector,
						)
					}
				}
			}

			if tt.wantTool != "" {
				got := filtered.Commits[0].Findings[0].Tool
				if got != tt.wantTool {
					t.Errorf("tool=%q want=%q", got, tt.wantTool)
				}
			}
		})
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

func TestRunDocsEmptyFormatFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"docs", "--format="}, &stdout, &stderr)
	if code != ExitError {
		t.Errorf("exit code=%d want error", code)
	}
}

func TestParseKeyValueFloatList(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]float64
		wantErr bool
	}{
		{
			name:    "Valid input single pair",
			input:   "low=20",
			want:    map[string]float64{"low": 20},
			wantErr: false,
		},
		{
			name:    "Valid input multiple pairs with floats",
			input:   "trailer=0.8,toolmention=0.2",
			want:    map[string]float64{"trailer": 0.8, "toolmention": 0.2},
			wantErr: false,
		},
		{
			name:    "Valid input with spacing padding",
			input:   " low = 20 , medium = 60.5 ",
			want:    map[string]float64{"low": 20, "medium": 60.5},
			wantErr: false,
		},
		{
			name:    "Empty string yields empty map",
			input:   "   ",
			want:    map[string]float64{},
			wantErr: false,
		},
		{
			name:    "Error on malformed key value syntax",
			input:   "low:20",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Error on empty keys",
			input:   "=20,medium=60",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Error on empty values",
			input:   "low=,medium=60",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Error on invalid float strings",
			input:   "low=twenty",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Error on invalid numerical states (NaN)",
			input:   "low=NaN",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseKeyValueFloatList(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKeyValueFloatList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseKeyValueFloatList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunScanConfidenceLevels(t *testing.T) {
	dir := initTestRepo(t)

	tests := []struct {
		name        string
		args        []string
		wantCode    int
		wantErrText string
	}{
		{
			name:        "invalid min confidence",
			args:        []string{"scan", "--min-confidence=invalid", dir},
			wantCode:    ExitError,
			wantErrText: "invalid confidence",
		},
		{
			name:     "invalid confidence levels format",
			args:     []string{"scan", "--confidence-levels=low", dir},
			wantCode: ExitError,
		},
		{
			name:     "reject NaN confidence levels",
			args:     []string{"scan", "--confidence-levels=high=NaN", dir},
			wantCode: ExitError,
		},
		{
			name:     "both valid flags",
			args:     []string{"scan", "--confidence-levels=low=15,medium=55,high=95", dir},
			wantCode: ExitAI,
		},
		{
			name:     "confidence levels all valid",
			args:     []string{"scan", "--confidence-levels", "low=10,medium=50,high=90", dir},
			wantCode: ExitAI,
		},
		{
			name:     "confidence levels low and high set",
			args:     []string{"scan", "--confidence-levels", "low=10,high=90", dir},
			wantCode: ExitAI,
		},
		{
			name:     "confidence levels large intervals",
			args:     []string{"scan", "--confidence-levels", "low=1,medium=10,high=1000", dir},
			wantCode: ExitAI,
		},
		{
			name:     "confidence levels medium and high set",
			args:     []string{"scan", "--confidence-levels", "medium=50,high=100", dir},
			wantCode: ExitAI,
		},
		{
			name:     "confidence empty levels",
			args:     []string{"scan", "--confidence-levels", "", dir},
			wantCode: ExitAI,
		},
		{
			name:        "confidence missing equals",
			args:        []string{"scan", "--confidence-levels", "low", dir},
			wantCode:    ExitError,
			wantErrText: "invalid entry",
		},
		{
			name:        "confidence empty key",
			args:        []string{"scan", "--confidence-levels", "=10", dir},
			wantCode:    ExitError,
			wantErrText: "empty key in entry",
		},
		{
			name:        "confidence empty value",
			args:        []string{"scan", "--confidence-levels", "low=", dir},
			wantCode:    ExitError,
			wantErrText: "empty value in entry",
		},
		{
			name:        "confidence invalid number",
			args:        []string{"scan", "--confidence-levels", "low=abc", dir},
			wantCode:    ExitError,
			wantErrText: "invalid numeric value",
		},
		{
			name:        "confidence NaN",
			args:        []string{"scan", "--confidence-levels", "high=NaN", dir},
			wantCode:    ExitError,
			wantErrText: "invalid numeric value",
		},
		{
			name:        "confidence Inf",
			args:        []string{"scan", "--confidence-levels", "high=Inf", dir},
			wantCode:    ExitError,
			wantErrText: "invalid numeric value",
		},
		{
			name:        "confidence -Inf",
			args:        []string{"scan", "--confidence-levels", "high=-Inf", dir},
			wantCode:    ExitError,
			wantErrText: "invalid numeric value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)

			if tt.name == "both valid flags" {
				if code != ExitAI && code != ExitNoAI {
					t.Fatalf("unexpected exit code %d (stderr=%s)", code, stderr.String())
				}
				return
			}

			if code != tt.wantCode {
				t.Fatalf("exit code=%d want=%d", code, tt.wantCode)
			}

			if tt.wantErrText != "" &&
				!strings.Contains(stderr.String(), tt.wantErrText) {
				t.Fatalf("stderr=%q does not contain %q", stderr.String(), tt.wantErrText)
			}
		})
	}
}

func TestRunScanConfidenceLevelsJson(t *testing.T) {
	dir := initTestRepo(t)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "confidence levels",
			args: []string{
				"scan",
				"--format=json",
				"--confidence-levels=low=10,medium=50,high=90",
				dir,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != ExitAI && code != ExitNoAI {
				t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
			}

			var report scan.Report
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatalf("unmarshalling error, invalid json: %v", err)
			}
		})
	}
}

func TestRunTextCommandWithCheckboxDetection(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		usedLabel         string
		notUsedLabel      string
		wantTool          string
		wantScore         float64
		wantConfidence    detection.Confidence
		wantExitCode      uint
		wantFindingsCount int
	}{
		{
			name:              "AI used checkbox with tool with default checkbox labels",
			input:             "[x] This contribution was assisted or created by Generative AI tools.\n\nClaude was used for drafting.",
			usedLabel:         "",
			notUsedLabel:      "",
			wantTool:          "Claude",
			wantScore:         95,
			wantConfidence:    detection.ConfidenceHigh,
			wantExitCode:      ExitAI,
			wantFindingsCount: 1,
		},
		{
			name:              "AI not used checkbox with tool with default checkbox labels",
			input:             "[x] This contribution was NOT assisted or created by Generative AI tools.\n\nClaude was used for drafting.",
			usedLabel:         "",
			notUsedLabel:      "",
			wantTool:          "Claude",
			wantScore:         20,
			wantConfidence:    detection.ConfidenceLow,
			wantExitCode:      ExitAI,
			wantFindingsCount: 1,
		},
		{
			name:              "AI not used checkbox with tool with default checkbox labels",
			input:             "[x] This contribution was NOT assisted or created by Generative AI tools.",
			usedLabel:         "",
			notUsedLabel:      "",
			wantTool:          "",
			wantScore:         0,
			wantConfidence:    detection.ConfidenceNone,
			wantExitCode:      ExitNoAI,
			wantFindingsCount: 0,
		},
		{
			name:              "custom AI used checkbox with tool",
			input:             "[x] This PR was created with AI assistance\n\nClaude was used for drafting.",
			usedLabel:         "This PR was created with AI assistance",
			notUsedLabel:      "This PR was created without AI assistance",
			wantTool:          "Claude",
			wantScore:         95,
			wantConfidence:    detection.ConfidenceHigh,
			wantExitCode:      ExitAI,
			wantFindingsCount: 1,
		},
		{
			name:              "custom AI used checkbox without tool",
			input:             "[x] This PR was created with AI assistance",
			usedLabel:         "This PR was created with AI assistance",
			notUsedLabel:      "This PR was created without AI assistance",
			wantTool:          "",
			wantScore:         75,
			wantConfidence:    detection.ConfidenceHigh,
			wantExitCode:      ExitAI,
			wantFindingsCount: 1,
		},
		{
			name:              "custom AI not used checkbox with tool",
			input:             "[x] This PR was created without AI assistance\n\nClaude was mentioned in the discussion.",
			usedLabel:         "This PR was created with AI assistance",
			notUsedLabel:      "This PR was created without AI assistance",
			wantTool:          "Claude",
			wantScore:         20,
			wantConfidence:    detection.ConfidenceLow,
			wantExitCode:      ExitAI,
			wantFindingsCount: 1,
		},
		{
			name:              "custom AI not used checkbox without tool",
			input:             "[x] This PR was created without AI assistance\n\nNo tool mentioned in the discussion.",
			usedLabel:         "This PR was created with AI assistance",
			notUsedLabel:      "This PR was created without AI assistance",
			wantTool:          "",
			wantScore:         0,
			wantConfidence:    detection.ConfidenceLow,
			wantExitCode:      ExitNoAI,
			wantFindingsCount: 0,
		},
		{
			name:              "custom AI not used checkbox without tool",
			input:             "[x] This PR was created without AI assistance\n\nNo tool mentioned in the discussion.",
			usedLabel:         "This PR was created with AI assistance",
			notUsedLabel:      "This PR was created without AI assistance",
			wantTool:          "",
			wantScore:         0,
			wantConfidence:    detection.ConfidenceLow,
			wantExitCode:      ExitNoAI,
			wantFindingsCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			file := filepath.Join(tmp, "input.txt")
			if err := os.WriteFile(file, []byte(tt.input), 0644); err != nil {
				t.Fatalf("write input: %v", err)
			}

			var stdout, stderr bytes.Buffer
			code := Run([]string{
				"text",
				"--format=json",
				"--enable-checkbox-detection",
				"--cb-disclosed-ai=" + tt.usedLabel,
				"--cb-disclosed-noai=" + tt.notUsedLabel,
				"--input=" + file,
			}, &stdout, &stderr)
			if code != int(tt.wantExitCode) {
				t.Fatalf(
					"exit code=%d want=%d (stderr=%s, stdout=%s)",
					code, tt.wantExitCode, stderr.String(), stdout.String(),
				)
			}

			var result struct {
				Findings   []detection.Finding  `json:"findings"`
				Score      float64              `json:"score"`
				Confidence detection.Confidence `json:"confidence"`
			}

			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal findings: %v (output=%s)", err, stdout.String())
			}

			findings := result.Findings
			findingsCount := len(findings)
			if findingsCount != tt.wantFindingsCount {
				t.Fatalf(
					"findings count=%d want=%d (findings=%v)",
					findingsCount, tt.wantFindingsCount, findings,
				)
			}

			if findingsCount > 0 {
				f := findings[0]

				if f.Tool != tt.wantTool {
					t.Errorf("tool=%q want=%q", f.Tool, tt.wantTool)
				}

				if f.Score != tt.wantScore {
					t.Errorf("score=%v want=%v", f.Score, tt.wantScore)
				}

				if f.Confidence != tt.wantConfidence {
					t.Errorf("confidence=%s want=%s", f.Confidence.String(), tt.wantConfidence.String())
				}

				if f.Detector != "toolmention" {
					t.Errorf("detector=%q want=%q", f.Detector, "toolmention")
				}
			}
		})
	}
}

func TestRunTextCommandStatement(t *testing.T) {
	file := filepath.Join(t.TempDir(), "pr.txt")
	body := "Fix synchronous read recursion.\n\nDrafted with OpenAI Codex."
	if err := os.WriteFile(file, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"text", "--input=" + file, "--format=json", "--min-confidence=high"}, &stdout, &stderr)
	if code != ExitAI {
		t.Fatalf("code = %d, want %d (stderr=%s)", code, ExitAI, stderr.String())
	}
	var result struct {
		Confidence string              `json:"confidence"`
		Findings   []detection.Finding `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if result.Confidence != "high" {
		t.Errorf("confidence = %q, want high", result.Confidence)
	}
	detectors := map[string]bool{}
	for _, f := range result.Findings {
		detectors[f.Detector] = true
	}
	if !detectors["statement"] {
		t.Errorf("findings = %+v, want a statement finding", result.Findings)
	}

	stdout.Reset()
	Run([]string{"text", "--input=" + file, "--format=json", "--enable-checkbox-detection"}, &stdout, &stderr)
	if strings.Contains(stdout.String(), `"statement"`) {
		t.Errorf("checkbox detection on: got a statement finding, want the checkbox to decide")
	}
}

func TestRunTextCommandStatementCustomTool(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "pr.txt")
	if err := os.WriteFile(file, []byte("Drafted with Fooagent."), 0644); err != nil {
		t.Fatal(err)
	}
	tools := filepath.Join(dir, "tools.json")
	if err := os.WriteFile(tools, []byte(`{"version": 1, "custom_tools": ["Fooagent"]}`), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"text", "--input=" + file, "--format=json", "--custom-tools=" + tools, "--min-confidence=high"}, &stdout, &stderr)
	if code != ExitAI {
		t.Fatalf("code = %d, want %d (stdout=%s stderr=%s)", code, ExitAI, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"statement"`) {
		t.Errorf("output = %s, want a statement finding for the custom tool", stdout.String())
	}
}

func TestRunScanSkipsStatement(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("file.txt"); err != nil {
		t.Fatalf("add: %v", err)
	}
	sig := &object.Signature{Name: "Test", Email: "human@example.com", When: time.Now()}
	if _, err := wt.Commit("Fix parsing of tool calls generated by Mistral", &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var stdout, stderr bytes.Buffer
	Run([]string{"scan", "--format=json", dir}, &stdout, &stderr)
	if !strings.Contains(stdout.String(), "toolmention") {
		t.Fatalf("output = %s, want a toolmention finding", stdout.String())
	}
	if strings.Contains(stdout.String(), `"statement"`) {
		t.Errorf("output = %s, want no statement finding on a commit message", stdout.String())
	}
}
