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

func TestScanCommandConfidenceLevelsValid(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "confidence levels all valid",
			args: []string{"--confidence-levels", "low=10,medium=50,high=90"},
		},
		{
			name: "confidence levels low and high set",
			args: []string{"--confidence-levels", "low=10,high=90"},
		},
		{
			name: "confidence levels large intervals",
			args: []string{"--confidence-levels", "low=1,medium=10,high=1000"},
		},
		{
			name: "confidence levels medium and high set",
			args: []string{"--confidence-levels", "medium=50,high=100"},
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

func TestScanCommandConfidenceLevelsInvalid(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "confidence empty levels",
			args: []string{"--confidence-levels", ""},
		},
		{
			name: "confidence missing equals",
			args: []string{"--confidence-levels", "low"},
		},
		{
			name: "confidence empty key",
			args: []string{"--confidence-levels", "=10"},
		},
		{
			name: "confidence empty value",
			args: []string{"--confidence-levels", "low="},
		},
		{
			name: "confidence invalid number",
			args: []string{"--confidence-levels", "low=abc"},
		},
		{
			name: "confidence NaN",
			args: []string{"--confidence-levels", "high=NaN"},
		},
		{
			name: "confidence Inf",
			args: []string{"--confidence-levels", "high=Inf"},
		},
		{
			name: "confidence -Inf",
			args: []string{"--confidence-levels", "high=-Inf"},
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

func TestRunTextCommandCustomCheckboxLabels(t *testing.T) {
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
				"--checkbox-label-ai-used=" + tt.usedLabel,
				"--checkbox-label-ai-not-used=" + tt.notUsedLabel,
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
				t.Fatalf(
					"failed to unmarshal output: %v (output=%s)",
					err,
					stdout.String(),
				)
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
