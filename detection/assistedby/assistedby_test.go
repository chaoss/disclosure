package assistedby

import (
	"testing"

	"github.com/chaoss/ai-detection-action/detection"
)

func TestDetect(t *testing.T) {
	d := &Detector{}
	tests := []struct {
		name      string
		message   string
		wantTools []string
	}{
		{
			name:      "Claude trailer with Opus model",
			message:   "fix: update handler\n\nAssisted-By: Claude Opus 4 <noreply@anthropic.com>",
			wantTools: []string{"Claude Opus 4"},
		},
		{
			name:      "Claude trailer with Sonnet model",
			message:   "fix: update handler\n\nAssisted-By: Claude Sonnet 4 <noreply@anthropic.com>",
			wantTools: []string{"Claude Sonnet 4"},
		},
		{
			name:      "Cursor trailer",
			message:   "refactor: extract method\n\nAssisted-By: Cursor <cursoragent@cursor.com>",
			wantTools: []string{"Cursor"},
		},
		{
			name:      "Aider trailer with model name",
			message:   "feat: add endpoint\n\nAssisted-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools: []string{"Aider"},
		},
		{
			name:      "Aider trailer with different model",
			message:   "feat: add endpoint\n\nAssisted-By: aider (claude-4.7-opus) <noreply@aider.chat>",
			wantTools: []string{"Aider"},
		},
		{
			name:      "multiple trailers with Claude and human",
			message:   "fix: bug\n\nAssisted-By: Claude Opus 4 <noreply@anthropic.com>",
			wantTools: []string{"Claude Opus 4"},
		},
		{
			name:      "multiple AI trailers",
			message:   "fix: bug\n\nAssisted-By: Claude Opus 4 <noreply@anthropic.com>\nAssisted-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools: []string{"Claude Opus 4", "Aider"},
		},
		{
			name:      "case variation",
			message:   "fix: something\n\nassisted-by: Claude <noreply@anthropic.com>",
			wantTools: []string{"Claude"},
		},
		{
			name:      "ASSISTED-BY uppercase",
			message:   "fix: something\n\nASSISTED-BY: Claude <noreply@anthropic.com>",
			wantTools: []string{"Claude"},
		},
		{
			name:      "Assisted-By trailer in commit message",
			message:   "this is a commit message with\nAssisted-By: Claude Code",
			wantTools: []string{"Claude Code"},
		},
		{
			name:      "Another Assisted-By trailer in commit message 1",
			message:   "this is a commit message with\nAssisted-By: Gemini",
			wantTools: []string{"Gemini"},
		},
		{
			name:      "Another Assisted-By trailer in commit message 2",
			message:   "this is a commit message with\nAssisted-By: Kimi K2.6",
			wantTools: []string{"Kimi K2.6"},
		},
		{
			name:      "Multiple Assisted-By trailer in commit message",
			message:   "this is a commit message with\nAssisted-By: Claude Code\nAssisted-By: Gemini",
			wantTools: []string{"Claude Code", "Gemini"},
		},
		{
			name: "Multiple Assisted-By trailers (with purpose brackets) in commit message",
			message: `
this is a commit message

Co-Authored-By: Cursor <cursoragent@cursor.com>

Assisted-by: Claude 4.7 Opus
	(logic optimization and design fixes)
Assisted-by: Kimi K2.6 (unit tests, integration tests)
Assisted-by: ChatGPT (documentation review)
Assisted-by: Gemini (documentation)
`,
			wantTools: []string{"Claude 4.7 Opus", "Kimi K2.6", "ChatGPT", "Gemini"},
		},
		{
			name:      "Assisted-By trailer in commit message in lower case",
			message:   "this is a commit message with\nassisted-by: Claude Code",
			wantTools: []string{"Claude Code"},
		},
		{
			name:      "Two different attributions (assistedby and coauthor) both with email address",
			message:   "Fix bug\n\nAssisted-By: Claude Sonnet 4 <noreply@anthropic.com>\nCo-Authored-By: Copilot <copilot@github.com>",
			wantTools: []string{"Claude Sonnet 4"},
		},
		{
			name:      "Two different attributions (assistedby and coauthor) one with model name, other with email address",
			message:   "Add validation logic\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\nAssisted-by: GitHub Copilot",
			wantTools: []string{"GitHub Copilot"},
		},
		{
			name:      "Claude Opus model attribution trailer",
			message:   "Fix bug\n\nAssisted-by: Claude Opus 4 <noreply@anthropic.com>",
			wantTools: []string{"Claude Opus 4"},
		},
		{
			name:      "no trailers",
			message:   "just a normal commit message",
			wantTools: nil,
		},
		{
			name:      "empty message",
			message:   "",
			wantTools: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := d.Detect(detection.Input{CommitMessage: tt.message})
			gotTools := make([]string, len(findings))
			for i, f := range findings {
				gotTools[i] = f.Tool
				if f.Confidence != detection.ConfidenceHigh {
					t.Errorf("confidence = %d, want %d", f.Confidence, detection.ConfidenceHigh)
				}
				if f.Detector != "assistedby" {
					t.Errorf("detector = %q, want %q", f.Detector, "assistedby")
				}
			}

			if len(gotTools) == 0 {
				gotTools = nil
			}

			if len(gotTools) != len(tt.wantTools) {
				t.Errorf("tools = %v, want %v", gotTools, tt.wantTools)
				return
			}
			for i := range gotTools {
				if gotTools[i] != tt.wantTools[i] {
					t.Errorf("tools = %v, want %v", gotTools, tt.wantTools)
					return
				}
			}
		})
	}
}
