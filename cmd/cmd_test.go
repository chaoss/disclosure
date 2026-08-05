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
		name          string
		report        scan.Report
		minConf       detection.Confidence
		wantScore     float64
		wantAICommits int
		wantFindings  int
		wantTool      string // optional
	}{
		{
			name: "keep only high confidence findings",
			report: scan.Report{
				Commits: []scan.CommitResult{
					{
						Hash: "abc123",
						Findings: []detection.Finding{
							{
								Detector:   "toolmention",
								Tool:       "Claude",
								Confidence: detection.ConfidenceLow,
								Score:      20,
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
								Confidence: detection.ConfidenceLow,
								Score:      20,
							},
							{
								Detector:   "trailer",
								Tool:       "Claude Code",
								Confidence: detection.ConfidenceHigh,
								Score:      100,
							},
						},
					},
				},
				Summary: scan.Summary{
					TotalCommits:      2,
					AICommits:         2,
					ToolCounts:        map[string]int{"Claude": 2, "Claude Code": 2},
					ByConfidence:      map[string]int{"low": 2, "high": 2},
					PerDetectorScores: map[string]float64{"toolmention": 20, "trailer": 100},
					OverallScore:      120,
				},
			},
			minConf:       detection.ConfidenceHigh,
			wantScore:     100,
			wantAICommits: 2,
			wantFindings:  2, // only high confidence ones from both commits
			wantTool:      "Claude Code",
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
				Summary: scan.Summary{
					TotalCommits: 1,
				},
			},
			minConf:       detection.ConfidenceHigh,
			wantScore:     0,
			wantAICommits: 0,
			wantFindings:  0,
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
			wantScore:     0,
			wantAICommits: 0,
			wantFindings:  0,
		},
		{
			name:          "no commits",
			report:        scan.Report{},
			minConf:       detection.ConfidenceHigh,
			wantScore:     0,
			wantAICommits: 0,
			wantFindings:  0,
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
								Score:      100,
							},
						},
					},
				},
			},
			minConf:      detection.ConfidenceLow,
			wantFindings: 2,
			wantScore:    120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterReport(tt.report, tt.minConf)

			if tt.minConf == detection.ConfidenceLow {
				if got := len(filtered.Commits[0].Findings); got != tt.wantFindings {
					t.Fatalf("findings=%d want=%d", got, tt.wantFindings)
				}
				return
			}

			if filtered.Summary.OverallScore != tt.wantScore {
				t.Errorf("overall score=%v want=%v",
					filtered.Summary.OverallScore, tt.wantScore)
			}

			if filtered.Summary.AICommits != tt.wantAICommits {
				t.Errorf("AICommits=%d want=%d",
					filtered.Summary.AICommits, tt.wantAICommits)
			}

			gotFindings := 0
			for _, c := range filtered.Commits {
				gotFindings += len(c.Findings)
			}
			if gotFindings != tt.wantFindings {
				t.Errorf("findings=%d want=%d", gotFindings, tt.wantFindings)
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

func TestRunScanFlags(t *testing.T) {
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
			name:     "invalid confidence scores format",
			args:     []string{"scan", "--confidence-scores=low", dir},
			wantCode: ExitError,
		},
		{
			name:     "reject NaN confidence score",
			args:     []string{"scan", "--confidence-scores=high=NaN", dir},
			wantCode: ExitError,
		},
		{
			name:     "both valid flags",
			args:     []string{"scan", "--confidence-scores=low=15,medium=55,high=95", dir},
			wantCode: ExitAI, // or ExitNoAI
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

func TestRunScanScoreFlags(t *testing.T) {
	dir := initTestRepo(t)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "confidence scores",
			args: []string{
				"scan",
				"--format=json",
				"--confidence-scores=low=10,medium=50,high=90",
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
				t.Fatalf("unmarshal: %v", err)
			}

			var findings []detection.Finding
			for _, c := range report.Commits {
				findings = append(findings, c.Findings...)
			}

			expected, _ := detection.ConsolidateScoreByFindings(findings)

			if report.Summary.OverallScore != expected {
				t.Fatalf("overall=%v want=%v",
					report.Summary.OverallScore,
					expected)
			}
		})
	}
}

func TestScanCommandInvalidFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "confidence missing equals",
			args: []string{"--confidence-scores", "low"},
		},
		{
			name: "confidence empty key",
			args: []string{"--confidence-scores", "=10"},
		},
		{
			name: "confidence empty value",
			args: []string{"--confidence-scores", "low="},
		},
		{
			name: "confidence invalid number",
			args: []string{"--confidence-scores", "low=abc"},
		},
		{
			name: "confidence NaN",
			args: []string{"--confidence-scores", "high=NaN"},
		},
		{
			name: "confidence Inf",
			args: []string{"--confidence-scores", "high=Inf"},
		},
		{
			name: "confidence -Inf",
			args: []string{"--confidence-scores", "high=-Inf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := ExitNoAI

			cmd := scanCommand(&stdout, &stderr, &exitCode)
			cmd.SetArgs(tt.args)

			_ = cmd.Execute()

			if exitCode != ExitError {
				t.Fatalf("expected ExitError, got %d (stderr=%q)", exitCode, stderr.String())
			}
		})
	}
}
