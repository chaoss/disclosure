package coauthor

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestDetect(t *testing.T) {
	d := &Detector{}
	tests := []struct {
		name       string
		message    string
		wantTools  []string
		wantModels []string
	}{
		{
			name:       "Claude trailer with Opus model",
			message:    "fix: update handler\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>",
			wantTools:  []string{"Claude Code"},
			wantModels: []string{"Claude Opus 4"},
		},
		{
			name:       "Claude trailer with Sonnet model",
			message:    "fix: update handler\n\nCo-Authored-By: Claude Sonnet 4 <noreply@anthropic.com>",
			wantTools:  []string{"Claude Code"},
			wantModels: []string{"Claude Sonnet 4"},
		},
		{
			name:       "Cursor trailer",
			message:    "refactor: extract method\n\nCo-Authored-By: Cursor <cursoragent@cursor.com>",
			wantTools:  []string{"Cursor"},
			wantModels: []string{"Cursor"},
		},
		{
			name:       "Aider trailer with model name",
			message:    "feat: add endpoint\n\nCo-Authored-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools:  []string{"Aider"},
			wantModels: []string{"gpt-4o"},
		},
		{
			name:       "Aider trailer with different model",
			message:    "feat: add endpoint\n\nCo-Authored-By: aider (claude-3.5-sonnet) <noreply@aider.chat>",
			wantTools:  []string{"Aider"},
			wantModels: []string{"claude-3.5-sonnet"},
		},
		{
			name:       "multiple trailers with Claude and human",
			message:    "fix: bug\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>\nCo-Authored-By: Alice <alice@example.com>",
			wantTools:  []string{"Claude Code"},
			wantModels: []string{"Claude Opus 4"},
		},
		{
			name:       "multiple AI trailers",
			message:    "fix: bug\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>\nCo-Authored-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools:  []string{"Claude Code", "Aider"},
			wantModels: []string{"Claude Opus 4", "gpt-4o"},
		},
		{
			name:       "case variation",
			message:    "fix: thing\n\nco-authored-by: Claude <noreply@anthropic.com>",
			wantTools:  []string{"Claude Code"},
			wantModels: []string{""},
		},
		{
			name:       "CO-AUTHORED-BY uppercase",
			message:    "fix: thing\n\nCO-AUTHORED-BY: Claude <noreply@anthropic.com>",
			wantTools:  []string{"Claude Code"},
			wantModels: []string{""},
		},
		{
			name:       "no trailers",
			message:    "just a normal commit message",
			wantTools:  nil,
			wantModels: nil,
		},
		{
			name:       "human co-author only",
			message:    "pair programming\n\nCo-Authored-By: Bob <bob@company.com>",
			wantTools:  nil,
			wantModels: nil,
		},
		{
			name:       "empty message",
			message:    "",
			wantTools:  nil,
			wantModels: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := d.Detect(detection.Input{CommitMessage: tt.message})
			gotTools := make([]string, len(findings))
			gotModels := make([]string, len(findings))
			for i, f := range findings {
				gotTools[i] = f.Tool
				gotModels[i] = f.Model
				if f.Confidence != detection.ConfidenceHigh {
					t.Errorf("confidence = %d, want %d", f.Confidence, detection.ConfidenceHigh)
				}
				if f.Detector != "coauthor" {
					t.Errorf("detector = %q, want %q", f.Detector, "coauthor")
				}
			}

			if len(gotTools) == 0 {
				gotTools = nil
				gotModels = nil
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
				if gotModels[i] != tt.wantModels[i] {
					t.Errorf("models = %q, want %q", gotModels, tt.wantModels)
					return
				}
			}
		})
	}
}
