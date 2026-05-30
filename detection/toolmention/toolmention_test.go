package toolmention

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestDetect(t *testing.T) {
	d := &Detector{}

	tests := []struct {
		name      string
		input     detection.Input
		wantTools []string
	}{
		{
			name:      "Claude mention in text",
			input:     detection.Input{Text: "I used Claude to write this PR"},
			wantTools: []string{"Claude"},
		},
		{
			name:      "Claude Code mention in text",
			input:     detection.Input{Text: "Generated with Claude Code"},
			wantTools: []string{"Claude Code", "Claude"},
		},
		{
			name:      "Copilot mention",
			input:     detection.Input{Text: "GitHub Copilot helped with this"},
			wantTools: []string{"GitHub Copilot", "Copilot"},
		},
		{
			name:      "multiple tools mentioned",
			input:     detection.Input{Text: "I used Cursor and Aider for this PR"},
			wantTools: []string{"Cursor", "Aider"},
		},
		{
			name:      "case insensitive",
			input:     detection.Input{Text: "I used CLAUDE to write this"},
			wantTools: []string{"Claude"},
		},
		{
			name:      "commit message scanned too",
			input:     detection.Input{CommitMessage: "feat: add feature\n\nGenerated with Claude Code"},
			wantTools: []string{"Claude Code", "Claude"},
		},
		{
			name:      "text and commit message combined",
			input:     detection.Input{Text: "Used Cursor", CommitMessage: "aider: fix bug"},
			wantTools: []string{"Cursor", "Aider"},
		},
		{
			name:      "no mentions",
			input:     detection.Input{Text: "This is a normal PR description"},
			wantTools: nil,
		},
		{
			name:      "empty input",
			input:     detection.Input{},
			wantTools: nil,
		},
		{
			name:      "word boundary prevents partial match",
			input:     detection.Input{Text: "The cursory review found nothing"},
			wantTools: nil,
		},
		{
			name:      "ChatGPT mention",
			input:     detection.Input{Text: "I asked ChatGPT for help"},
			wantTools: []string{"ChatGPT"},
		},
		{
			name:      "t3.chat mention",
			input:     detection.Input{Text: "I used t3.chat to compare model outputs"},
			wantTools: []string{"t3.chat"},
		},
		{
			name:      "t3.chat mention is case insensitive",
			input:     detection.Input{Text: "Generated with T3.CHAT"},
			wantTools: []string{"t3.chat"},
		},
		{
			name:      "t3.chat word boundary prevents partial match",
			input:     detection.Input{Text: "This mentions t3.chatty, not the tool"},
			wantTools: nil,
		},
		{
			name:      "Windsurf mention",
			input:     detection.Input{Text: "Written with Windsurf IDE"},
			wantTools: []string{"Windsurf"},
		},
		{
			name:      "Devin mention",
			input:     detection.Input{Text: "Devin created this PR"},
			wantTools: []string{"Devin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := d.Detect(tt.input)
			gotTools := make([]string, len(findings))
			for i, f := range findings {
				gotTools[i] = f.Tool
				if f.Confidence != detection.ConfidenceLow {
					t.Errorf("confidence = %d, want %d", f.Confidence, detection.ConfidenceLow)
				}
				if f.Detector != "toolmention" {
					t.Errorf("detector = %q, want %q", f.Detector, "toolmention")
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
