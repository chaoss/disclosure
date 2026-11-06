package disclosurephrase

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestDetectDisclosureStatements(t *testing.T) {
	d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
	cases := map[string]string{
		"Fixes the stack overflow.\n\nDrafted with OpenAI Codex.":                                  "OpenAI Codex",
		"AI disclosure: OpenAI Codex assisted with investigation, implementation, and validation.": "OpenAI Codex",
		"Generated with Claude Code":                  "Claude Code",
		"This PR was written using Cursor":            "Cursor",
		"Created with the help of GitHub Copilot":     "GitHub Copilot",
		"Prepared with assistance from OpenAI Codex.": "OpenAI Codex",
		"Copilot was used to generate docs":           "Copilot",
		"Aider wrote the first version of this":       "Aider",
		"AI-assistance: ChatGPT":                      "ChatGPT",
	}
	for text, want := range cases {
		findings := d.Detect(detection.Input{Text: text})
		if len(findings) != 1 {
			t.Errorf("Detect(%q): got %d findings, want 1", text, len(findings))
			continue
		}
		if findings[0].Tool != want {
			t.Errorf("Detect(%q): tool = %q, want %q", text, findings[0].Tool, want)
		}
		if findings[0].Score != detection.DisclosurePhraseBaseScore {
			t.Errorf("Detect(%q): score = %f, want %f", text, findings[0].Score, detection.DisclosurePhraseBaseScore)
		}
		if findings[0].Confidence != detection.ConfidenceHigh {
			t.Errorf("Detect(%q): confidence = %s, want high", text, findings[0].Confidence.String())
		}
	}
}

func TestDetectPlainMentionsAreIgnored(t *testing.T) {
	d := &Detector{}
	cases := []string{
		"Add support for the Claude API",
		"Fix Copilot integration bug",
		"Generated docs for the Copilot plugin",
		"Tested with Cursor and VS Code",
		"Bump the Gemini SDK",
		"Drafted with care.\n\nWorks with Codex sessions.",
		"",
	}
	for _, text := range cases {
		if findings := d.Detect(detection.Input{Text: text}); len(findings) != 0 {
			t.Errorf("Detect(%q): got %d findings, want 0", text, len(findings))
		}
	}
}

func TestDetectCommitMessageFooter(t *testing.T) {
	d := &Detector{}
	findings := d.Detect(detection.Input{CommitMessage: "feat: add feature\n\nGenerated with Claude Code"})
	if len(findings) != 1 || findings[0].Tool != "Claude Code" {
		t.Fatalf("Detect: got %+v, want one Claude Code finding", findings)
	}
}

func TestDetectReportsEachToolOnce(t *testing.T) {
	d := &Detector{}
	findings := d.Detect(detection.Input{Text: "Drafted with Codex.\nTests generated with Codex and Cursor."})
	tools := map[string]int{}
	for _, f := range findings {
		tools[f.Tool]++
	}
	if tools["Codex"] != 1 || tools["Cursor"] != 1 || len(tools) != 2 {
		t.Errorf("Detect: tools = %v, want Codex and Cursor once each", tools)
	}
}
