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
		"fix-codex-typo",       // "codex" appears, but not as a branch prefix
		"rooster/fix-bug",      // starts with "roo", but not "roo/"
		"windsurfer/refactor",  // starts with "windsurf", but not "windsurf/"
		"openhands-discussion", // missing trailing slash "openhands/"
		"swe-agent-docs",       // missing trailing slash "swe-agent/"
		"feature/roo-code",     // prefix appears in body, not at start
		"",
	}

	for _, branch := range cases {
		findings := d.Detect(detection.Input{BranchName: branch})
		if len(findings) != 0 {
			t.Errorf("Detect(%q): got %d findings, want 0", branch, len(findings))
		}
	}
}

func TestDetectMergedBranchFromCommitMessage(t *testing.T) {
	d := &Detector{}
	codex := detection.KnownAgentBranchPrefixes["codex/"]
	cases := map[string]string{
		"Merge pull request #188 from agrafix/codex/fix-cpu-denial-of-service-vulnerability\n\nBound captured extension route backtracking": codex,
		"Merge pull request #41 from OskarEichler/codex/security-normalize-hostname-bounds":                                                 codex,
		"Merge branch 'claude/add-tests'":                                  detection.KnownAgentBranchPrefixes["claude/"],
		"Merge remote-tracking branch 'origin/codex/fix-bug'":              codex,
		"Merge remote-tracking branch 'copilot/main'":                      "",
		"Merge remote-tracking branch 'origin/main' into copilot/fix-docs": "",
		"Merge pull request #12 from someone/feature/codex-support":        "",
		"Fix bug\n\nMerge pull request #1 from a/codex/x":                  "",
	}
	for message, want := range cases {
		findings := d.Detect(detection.Input{BranchName: "main", CommitMessage: message})
		got := ""
		if len(findings) == 1 {
			got = findings[0].Tool
		}
		if got != want {
			t.Errorf("Detect(%q): tool = %q, want %q", message, got, want)
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
