package branchname

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestDetectKnownPrefixes(t *testing.T) {
	d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
	for prefix, expectedTool := range detection.KnownAgentBranchPrefixes {
		branch := prefix + "fix-bug"
		findings := d.Detect(detection.Input{BranchName: branch})
		if len(findings) != 1 {
			t.Errorf("Detect(%q): got %d findings, want 1", branch, len(findings))
			continue
		}
		finding := findings[0]
		if finding.Tool != expectedTool {
			t.Errorf("Detect(%q): tool = %q, want %q", branch, finding.Tool, expectedTool)
		}
		if finding.Score != detection.BranchNameBaseScore {
			t.Errorf("Detect(%q): score = %f, want %f", branch, finding.Score, detection.BranchNameBaseScore)
		}
		if finding.Confidence != detection.ConfidenceHigh {
			t.Errorf(
				"Detect(%q): confidence = %s, want %s",
				branch, finding.Confidence.String(), detection.ConfidenceHigh.String(),
			)
		}
		if finding.Detector != "branchname" {
			t.Errorf("Detect(%q): detector = %q, want %q", branch, finding.Detector, "branchname")
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
