package output

import (
	"bytes"
	"encoding/json"
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
						Detector:   "coauthor",
						Tool:       "Claude Code",
						Confidence: detection.ConfidenceHigh,
						Detail:     "Co-Authored-By trailer with email noreply@anthropic.com",
					},
				},
			},
			{
				Hash:     "def789ghi012",
				Findings: nil,
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
	}

	if err := FormatTextFindings(&buf, findings); err != nil {
		t.Fatalf("FormatTextFindings: %v", err)
	}

	if !strings.Contains(buf.String(), "Claude") {
		t.Errorf("expected Claude in text output, got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "1 AI signal") {
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

func TestConfidenceFromString(t *testing.T) {
	tests := []struct {
		input string
		want  detection.Confidence
		err   bool
	}{
		{"low", detection.ConfidenceLow, false},
		{"1", detection.ConfidenceLow, false},
		{"medium", detection.ConfidenceMedium, false},
		{"2", detection.ConfidenceMedium, false},
		{"high", detection.ConfidenceHigh, false},
		{"3", detection.ConfidenceHigh, false},
		{"HIGH", detection.ConfidenceHigh, false},
		{"  low  ", detection.ConfidenceLow, false},
		{"invalid", 0, true},
		{"4", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		got, err := ConfidenceFromString(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("ConfidenceFromString(%q): err = %v, wantErr = %v", tt.input, err, tt.err)
			continue
		}
		if got != tt.want {
			t.Errorf("ConfidenceFromString(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
