package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/chaoss/disclosure/policy"
)

func TestFormatPolicyJSON(t *testing.T) {
	report := policy.Report{
		Repository:    ".",
		ScannedCommit: "abc123",
		Files: []policy.File{
			{Path: "AGENTS.md", Type: "agent_instructions"},
		},
	}

	var out bytes.Buffer
	if err := FormatPolicyJSON(&out, report); err != nil {
		t.Fatalf("FormatPolicyJSON: %v", err)
	}

	var got policy.Report
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}
	if got.Repository != report.Repository || got.ScannedCommit != report.ScannedCommit || len(got.Files) != 1 || got.Files[0] != report.Files[0] {
		t.Errorf("JSON report = %#v, want %#v", got, report)
	}
}

func TestFormatPolicyText(t *testing.T) {
	var out bytes.Buffer
	if err := FormatPolicyText(&out, policy.Report{
		Repository:    ".",
		ScannedCommit: "abc123",
		Files:         []policy.File{{Path: "AGENTS.md", Type: "agent_instructions"}},
	}); err != nil {
		t.Fatalf("FormatPolicyText: %v", err)
	}
	for _, want := range []string{"Scanned repository . at commit abc123", "AGENTS.md (agent_instructions)"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("text output %q does not contain %q", out.String(), want)
		}
	}
}

func TestFormatPolicyTextEmpty(t *testing.T) {
	var out bytes.Buffer
	if err := FormatPolicyText(&out, policy.Report{Repository: ".", ScannedCommit: "abc123"}); err != nil {
		t.Fatalf("FormatPolicyText: %v", err)
	}
	if !strings.Contains(out.String(), "No policy-related files found.") {
		t.Errorf("text output = %q, want no-files message", out.String())
	}
}
