package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/chaoss/disclosure/detection"
	"github.com/chaoss/disclosure/scan"
)

func sampleReport() scan.Report {
	return scan.Report{
		Commits: []scan.CommitResult{
			{
				Hash: "abc123def456",
				Findings: []detection.Finding{
					{
						Detector:   "trailer",
						Tool:       "Claude Code",
						Model:      "Opus 4",
						Confidence: detection.ConfidenceHigh,
						Detail:     "Co-Authored-By trailer with email noreply@anthropic.com",
						Score:      100.0,
					},
				},
				Score:      100.0,
				Confidence: detection.ConfidenceHigh,
			},
			{
				Hash:       "def789ghi012",
				Findings:   nil,
				Score:      20.0,
				Confidence: detection.ConfidenceLow,
			},
		},
		Summary: scan.Summary{
			TotalCommits: 2,
			AICommits:    1,
			ToolCounts:   map[string]int{"Claude Code": 1},
			ByConfidence: map[string]int{"high": 1},
		},
	}
}

func TestFormatJSON(t *testing.T) {
	var buf bytes.Buffer
	report := sampleReport()

	if err := FormatJSON(&buf, report); err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	var decoded scan.Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Summary.TotalCommits != 2 {
		t.Errorf("total_commits = %d, want 2", decoded.Summary.TotalCommits)
	}
	if decoded.Summary.AICommits != 1 {
		t.Errorf("ai_commits = %d, want 1", decoded.Summary.AICommits)
	}
}

func TestFormatText(t *testing.T) {
	var buf bytes.Buffer
	report := sampleReport()

	if err := FormatText(&buf, report); err != nil {
		t.Fatalf("FormatText: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "2 commits") {
		t.Errorf("expected commit count in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1 with AI signals") {
		t.Errorf("expected AI commit count in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Claude Code") {
		t.Errorf("expected tool name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Claude Code [Opus 4]") {
		t.Errorf("expected tool model in output, got:\n%s", out)
	}
	if !strings.Contains(out, "abc123def456") {
		t.Errorf("expected commit hash in output, got:\n%s", out)
	}
}

func TestFormatTextNoFindings(t *testing.T) {
	var buf bytes.Buffer
	report := scan.Report{
		Commits: []scan.CommitResult{
			{Hash: "abc123def456", Findings: nil},
		},
		Summary: scan.Summary{
			TotalCommits: 1,
			ToolCounts:   map[string]int{},
			ByConfidence: map[string]int{},
		},
	}

	if err := FormatText(&buf, report); err != nil {
		t.Fatalf("FormatText: %v", err)
	}

	if !strings.Contains(buf.String(), "No AI involvement detected") {
		t.Errorf("expected no-detection message, got:\n%s", buf.String())
	}
}

func TestFormatJSONFindings(t *testing.T) {
	var buf bytes.Buffer
	findings := []detection.Finding{
		{Detector: "toolmention", Tool: "Claude", Confidence: detection.ConfidenceLow, Detail: "text mentions Claude"},
	}

	if err := FormatJSONFindings(&buf, findings); err != nil {
		t.Fatalf("FormatJSONFindings: %v", err)
	}

	if !strings.Contains(buf.String(), `"tool": "Claude"`) {
		t.Errorf("expected Claude in JSON output, got:\n%s", buf.String())
	}
}

func TestFormatTextFindings(t *testing.T) {
	var buf bytes.Buffer
	findings := []detection.Finding{
		{Detector: "toolmention", Tool: "Claude", Confidence: detection.ConfidenceLow, Detail: "text mentions Claude"},
		{Detector: "gitnotes", Model: "gpt-4o", Confidence: detection.ConfidenceHigh, Detail: "git notes declares model"},
	}

	if err := FormatTextFindings(&buf, findings); err != nil {
		t.Fatalf("FormatTextFindings: %v", err)
	}

	if !strings.Contains(buf.String(), "Claude") {
		t.Errorf("expected Claude in text output, got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "gpt-4o") {
		t.Errorf("expected model-only finding in text output, got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "2 AI signal") {
		t.Errorf("expected signal count in output, got:\n%s", buf.String())
	}
}

func TestFormatTextFindingsEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := FormatTextFindings(&buf, nil); err != nil {
		t.Fatalf("FormatTextFindings: %v", err)
	}
	if !strings.Contains(buf.String(), "No AI involvement detected") {
		t.Errorf("expected no-detection message, got:\n%s", buf.String())
	}
}

func TestFormatJSONEmptyReport(t *testing.T) {
	var buf bytes.Buffer
	report := scan.Report{}
	if err := FormatJSON(&buf, report); err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}
	var decoded scan.Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("write failed")
}

func TestFormatJSONWriterError(t *testing.T) {
	err := FormatJSON(failingWriter{}, sampleReport())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatTextZeroScore(t *testing.T) {
	var buf bytes.Buffer
	report := scan.Report{
		Summary: scan.Summary{
			TotalCommits: 1,
			AICommits:    1,
			ToolCounts:   map[string]int{},
		},
	}
	if err := FormatText(&buf, report); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "Score") {
		t.Error("did not expect score for zero")
	}
}

func TestFormatTextShortHash(t *testing.T) {
	var buf bytes.Buffer
	report := scan.Report{
		Commits: []scan.CommitResult{
			{
				Hash: "abc",
				Findings: []detection.Finding{
					{Detector: "test"},
				},
			},
		},
		Summary: scan.Summary{
			AICommits: 1,
		},
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	if err := FormatText(&buf, report); err != nil {
		t.Fatal(err)
	}
}

func TestFormatTextFindingsIncludesScore(t *testing.T) {
	var buf bytes.Buffer
	findings := []detection.Finding{
		{
			Detector: "test",
			Score:    100,
		},
	}
	if err := FormatTextFindings(&buf, findings); err != nil {
		t.Fatal(err)
	}
	formatTextStr := buf.String()
	if !strings.Contains(formatTextStr, "Score:") {
		t.Fatal("missing score")
	}
	if !strings.Contains(formatTextStr, "Confidence:") {
		t.Fatal("missing confidence")
	}
}

func TestFormatTextFindingsEmptySlice(t *testing.T) {
	var buf bytes.Buffer
	if err := FormatTextFindings(&buf, []detection.Finding{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No AI involvement detected") {
		t.Fatal("expected no detection message")
	}
}

func TestFormatJSONFindingsStructure(t *testing.T) {
	var buf bytes.Buffer
	findings := []detection.Finding{
		{
			Detector:   "toolmention",
			Tool:       "Claude",
			Score:      100,
			Confidence: detection.ConfidenceHigh,
		},
	}
	if err := FormatJSONFindings(&buf, findings); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Findings   []detection.Finding  `json:"findings"`
		Score      float64              `json:"score"`
		Confidence detection.Confidence `json:"confidence"`
	}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Findings) != 1 {
		t.Fatalf("findings=%d want 1", len(decoded.Findings))
	}
	if decoded.Score != 100 {
		t.Fatalf("score=%v want 100", decoded.Score)
	}
	if decoded.Confidence != detection.ConfidenceHigh {
		t.Fatalf("confidence=%v want high", decoded.Confidence)
	}
}

func TestSortedKeys(t *testing.T) {
	got := sortedKeys(map[string]int{"c": 1, "h": 1, "a": 1, "o": 1, "s": 1})
	want := []string{"a", "c", "h", "o", "s"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestFormatTextExactOutput(t *testing.T) {
	var buf bytes.Buffer

	report := sampleReport()

	if err := FormatText(&buf, report); err != nil {
		t.Fatalf("FormatText: %v", err)
	}

	want := `Scanned 2 commits, 1 with AI signals

Tools detected:
  Claude Code: 1

Commit abc123def456 (score: 100.0, confidence: high)
  [score: 100.0, confidence: high] Claude Code [Opus 4] (trailer): Co-Authored-By trailer with email noreply@anthropic.com
`

	if got := buf.String(); got != want {
		t.Errorf("FormatText output mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestFormatTextExactOutputMultipleTools(t *testing.T) {
	var buf bytes.Buffer

	report := scan.Report{
		Commits: []scan.CommitResult{
			{
				Hash: "1234567890abcdef",
				Findings: []detection.Finding{
					{
						Detector:   "toolmention",
						Tool:       "Copilot",
						Confidence: detection.ConfidenceLow,
						Score:      20,
						Detail:     "copilot finding",
					},
					{
						Detector:   "toolmention",
						Tool:       "Kimi",
						Confidence: detection.ConfidenceLow,
						Score:      20,
						Detail:     "kimi finding",
					},
					{
						Detector:   "trailer",
						Tool:       "Claude",
						Confidence: detection.ConfidenceHigh,
						Score:      75,
						Detail:     "claude finding",
					},
				},
				Score:      95,
				Confidence: detection.ConfidenceHigh,
			},
		},
		Summary: scan.Summary{
			TotalCommits: 1,
			AICommits:    1,
			ToolCounts: map[string]int{
				"Claude":  1,
				"Copilot": 1,
				"Kimi":    1,
			},
		},
	}

	if err := FormatText(&buf, report); err != nil {
		t.Fatalf("FormatText: %v", err)
	}

	want := `Scanned 1 commits, 1 with AI signals

Tools detected:
  Claude: 1
  Copilot: 1
  Kimi: 1

Commit 1234567890ab (score: 95.0, confidence: high)
  [score: 20.0, confidence: low] Copilot (toolmention): copilot finding
  [score: 20.0, confidence: low] Kimi (toolmention): kimi finding
  [score: 75.0, confidence: high] Claude (trailer): claude finding
`

	if got := buf.String(); got != want {
		t.Errorf("FormatText output mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestFormatTextNoAIExactOutput(t *testing.T) {
	var buf bytes.Buffer

	report := scan.Report{
		Summary: scan.Summary{
			TotalCommits: 5,
			AICommits:    0,
			ToolCounts:   map[string]int{},
		},
	}

	if err := FormatText(&buf, report); err != nil {
		t.Fatalf("FormatText: %v", err)
	}

	want := "Scanned 5 commits, 0 with AI signals\n\nNo AI involvement detected.\n"

	if got := buf.String(); got != want {
		t.Errorf("FormatText output mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestFormatTextSkipsCommitsWithoutFindings(t *testing.T) {
	var buf bytes.Buffer

	report := scan.Report{
		Commits: []scan.CommitResult{
			{
				Hash: "with-findings",
				Findings: []detection.Finding{
					{
						Detector:   "test",
						Tool:       "Claude",
						Confidence: detection.ConfidenceHigh,
						Score:      100,
						Detail:     "detected",
					},
				},
				Score:      100,
				Confidence: detection.ConfidenceHigh,
			},
			{
				Hash:       "without-findings",
				Findings:   nil,
				Score:      0,
				Confidence: detection.ConfidenceNone,
			},
		},
		Summary: scan.Summary{
			TotalCommits: 2,
			AICommits:    1,
			ToolCounts: map[string]int{
				"Claude": 1,
			},
		},
	}

	if err := FormatText(&buf, report); err != nil {
		t.Fatalf("FormatText: %v", err)
	}

	want := `Scanned 2 commits, 1 with AI signals

Tools detected:
  Claude: 1

Commit with-finding (score: 100.0, confidence: high)
  [score: 100.0, confidence: high] Claude (test): detected
`

	if got := buf.String(); got != want {
		t.Errorf("FormatText output mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestFormatTextFindingsExactOutput(t *testing.T) {
	var buf bytes.Buffer

	findings := []detection.Finding{
		{
			Detector:   "toolmention",
			Tool:       "Claude",
			Confidence: detection.ConfidenceLow,
			Score:      25,
			Detail:     "text mentions Claude",
		},
		{
			Detector:   "gitnotes",
			Model:      "gpt-4o",
			Confidence: detection.ConfidenceHigh,
			Score:      100,
			Detail:     "git notes declares model",
		},
	}

	if err := FormatTextFindings(&buf, findings); err != nil {
		t.Fatalf("FormatTextFindings: %v", err)
	}

	want := `Found 2 AI signal(s):
Score: 125.0, Confidence: high
  [score: 25.0, confidence: low] Claude (toolmention): text mentions Claude
  [score: 100.0, confidence: high] gpt-4o (gitnotes): git notes declares model
`

	if got := buf.String(); got != want {
		t.Errorf("FormatTextFindings output mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestFormatTextFindingsEmptyExactOutput(t *testing.T) {
	var buf bytes.Buffer

	if err := FormatTextFindings(&buf, nil); err != nil {
		t.Fatalf("FormatTextFindings: %v", err)
	}

	want := "No AI involvement detected.\n"

	if got := buf.String(); got != want {
		t.Errorf("FormatTextFindings output mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestFormatTextFindingsEmptySliceExactOutput(t *testing.T) {
	var buf bytes.Buffer

	if err := FormatTextFindings(&buf, []detection.Finding{}); err != nil {
		t.Fatalf("FormatTextFindings: %v", err)
	}

	want := "No AI involvement detected.\n"

	if got := buf.String(); got != want {
		t.Errorf("FormatTextFindings output mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestFormatJSONExactIndentation(t *testing.T) {
	var buf bytes.Buffer

	report := scan.Report{}

	if err := FormatJSON(&buf, report); err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	got := buf.String()

	if !strings.HasSuffix(got, "\n") {
		t.Errorf("FormatJSON output does not end with newline: %q", got)
	}

	if !strings.Contains(got, "\n  ") {
		t.Errorf("FormatJSON output is not indented with two spaces:\n%s", got)
	}
}

func TestFormatJSONFindingsExactFormatting(t *testing.T) {
	var buf bytes.Buffer

	findings := []detection.Finding{
		{
			Detector:   "toolmention",
			Tool:       "Claude",
			Confidence: detection.ConfidenceLow,
			Score:      100,
			Detail:     "text mentions Claude",
		},
	}

	if err := FormatJSONFindings(&buf, findings); err != nil {
		t.Fatalf("FormatJSONFindings: %v", err)
	}

	got := buf.String()

	if !strings.HasSuffix(got, "\n") {
		t.Errorf("expected trailing newline, got %q", got)
	}

	if !strings.Contains(got, "\n  ") {
		t.Errorf("expected two-space indentation:\n%s", got)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := decoded["findings"]; !ok {
		t.Errorf("missing findings field:\n%s", got)
	}

	if _, ok := decoded["score"]; !ok {
		t.Errorf("missing score field:\n%s", got)
	}

	if _, ok := decoded["confidence"]; !ok {
		t.Errorf("missing confidence field:\n%s", got)
	}
}
