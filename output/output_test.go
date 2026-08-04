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
					},
				},
				Score: 100.0,
			},
			{
				Hash:     "def789ghi012",
				Findings: nil,
				Score:    0.0,
			},
		},
		Summary: scan.Summary{
			TotalCommits: 2,
			AICommits:    1,
			ToolCounts:   map[string]int{"Claude Code": 1},
			ByConfidence: map[string]int{"high": 1},
			OverallScore: 100.0,
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
	if !strings.Contains(out, "Overall score") {
		t.Errorf("expected overall score in output, got:\n%s", out)
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
	if decoded.Summary.OverallScore != 0 {
		t.Fatalf("overall score = %v, want 0", decoded.Summary.OverallScore)
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
			OverallScore: 0,
			ToolCounts:   map[string]int{},
		},
	}
	if err := FormatText(&buf, report); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "Overall score") {
		t.Error("did not expect overall score for zero")
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
	if !strings.Contains(buf.String(), "Overall score:") {
		t.Fatal("missing score")
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
			Detector: "toolmention",
			Tool:     "Claude",
			Score:    100,
		},
	}
	if err := FormatJSONFindings(&buf, findings); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Findings []detection.Finding `json:"findings"`
		Score    float64             `json:"overall_score"`
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
}

func TestSortedKeys(t *testing.T) {
	got := sortedKeys(map[string]int{"c": 1, "h": 1, "a": 1, "o": 1, "s": 1})
	want := []string{"a", "c", "h", "o", "s"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
