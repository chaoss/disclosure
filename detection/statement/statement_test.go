package statement

import (
	"strings"
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
		if findings[0].Score != detection.StatementBaseScore {
			t.Errorf("Detect(%q): score = %f, want %f", text, findings[0].Score, detection.StatementBaseScore)
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
		"Not generated with Claude or any other AI tool.",
		"No AI was used; this was not written using Copilot.",
		"This wasn't drafted with Codex.",
		"No Copilot was used here",
		"Neither Copilot nor ChatGPT was used",
		"AI use: Claude was not used",
		"AI disclosure: none. No Copilot or ChatGPT.",
		"Assisted-by: Claude",
		"Generated-by: Cursor",
		"Co-authored-by: Claude <noreply@anthropic.com>",
		"<!-- AI disclosure: say which tool, e.g. Claude or Copilot -->",
		"<!--\nAI disclosure: e.g.\nDrafted with Codex\n-->\nFixes a typo.",
		"",
	}
	for _, text := range cases {
		if findings := d.Detect(detection.Input{Text: text}); len(findings) != 0 {
			t.Errorf("Detect(%q): got %d findings, want 0", text, len(findings))
		}
	}
}

func TestDetectIgnoresCommitMessage(t *testing.T) {
	d := &Detector{}
	findings := d.Detect(detection.Input{CommitMessage: "feat: add feature\n\nGenerated with Claude Code"})
	if len(findings) != 0 {
		t.Fatalf("Detect: got %+v, want no findings", findings)
	}
}

func TestDetectStatementNextToNegatedSentence(t *testing.T) {
	d := &Detector{}
	findings := d.Detect(detection.Input{Text: "This was not trivial. Drafted with Codex."})
	if len(findings) != 1 || findings[0].Tool != "Codex" {
		t.Fatalf("Detect: got %+v, want one Codex finding", findings)
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

func BenchmarkDetect(b *testing.B) {
	d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
	plain := strings.Repeat("Update the parser and add regression coverage for empty input.\n", 200)
	tests := []struct {
		name string
		text string
	}{
		{name: "short match", text: "Drafted with OpenAI Codex."},
		{name: "12k chars no match", text: plain},
		{name: "12k chars match at end", text: plain + "Drafted with OpenAI Codex."},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			input := detection.Input{Text: tt.text}
			b.ReportAllocs()
			for range b.N {
				d.Detect(input)
			}
		})
	}
}
