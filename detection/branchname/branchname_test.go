package branchname

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestDetectKnownPrefixes(t *testing.T) {
	d := &Detector{}
	for prefix, expectedTool := range detection.KnownAgentBranchPrefixes {
		branch := prefix + "fix-bug"
		findings := d.Detect(detection.Input{BranchName: branch})
		if len(findings) != 1 {
			t.Errorf("Detect(%q): got %d findings, want 1", branch, len(findings))
			continue
		}
		if findings[0].Tool != expectedTool {
			t.Errorf("Detect(%q): tool = %q, want %q", branch, findings[0].Tool, expectedTool)
		}
		if findings[0].Confidence != detection.ConfidenceMedium {
			t.Errorf("Detect(%q): confidence = %d, want %d", branch, findings[0].Confidence, detection.ConfidenceMedium)
		}
		if findings[0].Detector != "branchname" {
			t.Errorf("Detect(%q): detector = %q, want %q", branch, findings[0].Detector, "branchname")
		}
	}
}

func TestDetectCaseInsensitive(t *testing.T) {
	d := &Detector{}
	findings := d.Detect(detection.Input{BranchName: "Codex/Fix-Bug"})
	if len(findings) != 1 {
		t.Fatalf("Detect: got %d findings, want 1", len(findings))
	}
	if findings[0].Tool != "OpenAI Codex" {
		t.Errorf("Detect: tool = %q, want %q", findings[0].Tool, "OpenAI Codex")
	}
}

func TestDetectNoMatch(t *testing.T) {
	d := &Detector{}
	cases := []string{
		"main",
		"feature/add-login",
		"fix-codex-typo", // "codex" appears, but not as a branch prefix
		"",
	}

	for _, branch := range cases {
		findings := d.Detect(detection.Input{BranchName: branch})
		if len(findings) != 0 {
			t.Errorf("Detect(%q): got %d findings, want 0", branch, len(findings))
		}
	}
}

func TestDetectPrefixMustBeFollowedBySeparator(t *testing.T) {
	// "codexterity" starts with "codex" but not "codex/" and should not match.
	d := &Detector{}
	findings := d.Detect(detection.Input{BranchName: "codexterity-branch"})
	if len(findings) != 0 {
		t.Errorf("Detect: got %d findings, want 0", len(findings))
	}
}
