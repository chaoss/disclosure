package branchname

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestDetectCodexBranch(t *testing.T) {
	d := &Detector{}
	findings := d.Detect(detection.Input{BranchName: "codex/fix-collectoss-metrics"})

	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(findings))
	}

	f := findings[0]
	if f.Tool != "OpenAI Codex" {
		t.Errorf("tool = %q, want OpenAI Codex", f.Tool)
	}
	if f.Confidence != detection.ConfidenceMedium {
		t.Errorf("confidence = %v, want medium", f.Confidence)
	}
	if f.Detector != "branchname" {
		t.Errorf("detector = %q, want branchname", f.Detector)
	}
}

func TestDetectCodexBranchCaseInsensitive(t *testing.T) {
	d := &Detector{}
	findings := d.Detect(detection.Input{BranchName: "Codex/add-feature"})

	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(findings))
	}
}

func TestDetectNoMatch(t *testing.T) {
	d := &Detector{}

	for _, branch := range []string{"", "main", "feature/fix-bug", "codec/typo"} {
		findings := d.Detect(detection.Input{BranchName: branch})
		if len(findings) != 0 {
			t.Errorf("branch %q: expected no findings, got %v", branch, findings)
		}
	}
}

func TestDetectIgnoresOtherFields(t *testing.T) {
	d := &Detector{}
	findings := d.Detect(detection.Input{
		BranchName:    "main",
		CommitMessage: "Generated with Claude Code",
		Text:          "I used Claude",
	})

	if len(findings) != 0 {
		t.Errorf("expected branch detector to ignore non-branch fields, got %v", findings)
	}
}
