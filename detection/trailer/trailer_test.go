package trailer

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestDetect(t *testing.T) {
	d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
	tests := []struct {
		name           string
		message        string
		wantTools      []string
		wantModels     []string
		wantScore      []float64
		wantConfidence []detection.Confidence
	}{
		// Co-Authored-By tests start here
		{
			name:           "coauthor: Claude trailer with Opus model",
			message:        "fix: update handler\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Code"},
			wantModels:     []string{"Opus 4"},
			wantScore:      []float64{85},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: Claude trailer with Sonnet model",
			message:        "fix: update handler\n\nCo-Authored-By: Claude Sonnet 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Code"},
			wantModels:     []string{"Sonnet 4"},
			wantScore:      []float64{85},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: Cursor trailer",
			message:        "refactor: extract method\n\nCo-Authored-By: Cursor <cursoragent@cursor.com>",
			wantTools:      []string{"Cursor"},
			wantModels:     []string{""},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: Aider trailer with model name",
			message:        "feat: add endpoint\n\nCo-Authored-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools:      []string{"Aider"},
			wantModels:     []string{"gpt-4o"},
			wantScore:      []float64{85},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: Aider trailer with different model",
			message:        "feat: add endpoint\n\nCo-Authored-By: aider (claude-3.5-sonnet) <noreply@aider.chat>",
			wantTools:      []string{"Aider"},
			wantModels:     []string{"claude-3.5-sonnet"},
			wantScore:      []float64{85},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: Continue.dev trailer",
			message:        "feat: add feature\n\nCo-Authored-By: Continue <noreply@continue.dev>",
			wantTools:      []string{"Continue.dev"},
			wantModels:     []string{""},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: Windsurf trailer",
			message:        "feat: add feature\n\nCo-Authored-By: Windsurf <noreply@windsurf.com>",
			wantTools:      []string{"Windsurf"},
			wantModels:     []string{""},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: Cursor trailer with parenthesized model",
			message:        "refactor: extract method\n\nCo-Authored-By: Cursor (composer 2.5) <cursoragent@cursor.com>",
			wantTools:      []string{"Cursor"},
			wantModels:     []string{"composer 2.5"},
			wantScore:      []float64{85},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: Copilot trailer with parenthesized model",
			message:        "feat: add endpoint\n\nCo-Authored-By: Copilot (gpt-4.1) <copilot@github.com>",
			wantTools:      []string{"Copilot"},
			wantModels:     []string{"gpt-4.1"},
			wantScore:      []float64{85},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: Claude trailer with parenthesized model",
			message:        "fix: update handler\n\nCo-Authored-By: Claude Code (Opus 4.1) <noreply@anthropic.com>",
			wantTools:      []string{"Claude Code"},
			wantModels:     []string{"Opus 4.1"},
			wantScore:      []float64{85},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: multiple trailers with Claude and human",
			message:        "fix: bug\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>\nCo-Authored-By: Alice <alice@example.com>",
			wantTools:      []string{"Claude Code"},
			wantModels:     []string{"Opus 4"},
			wantScore:      []float64{85},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: multiple AI trailers",
			message:        "fix: bug\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>\nCo-Authored-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools:      []string{"Claude Code", "Aider"},
			wantModels:     []string{"Opus 4", "gpt-4o"},
			wantScore:      []float64{85, 85},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: same tool with distinct models",
			message:        "fix: bug\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>\nCo-Authored-By: Claude Sonnet 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Code", "Claude Code"},
			wantModels:     []string{"Opus 4", "Sonnet 4"},
			wantScore:      []float64{85, 85},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: same tool and model deduplicated",
			message:        "fix: bug\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Code"},
			wantModels:     []string{"Opus 4"},
			wantScore:      []float64{85},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: case variation",
			message:        "fix: thing\n\nco-authored-by: Claude <noreply@anthropic.com>",
			wantTools:      []string{"Claude Code"},
			wantModels:     []string{""},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: CO-AUTHORED-BY uppercase",
			message:        "fix: thing\n\nCO-AUTHORED-BY: Claude <noreply@anthropic.com>",
			wantTools:      []string{"Claude Code"},
			wantModels:     []string{""},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "coauthor: human co-author only",
			message:        "pair programming\n\nCo-Authored-By: Bob <bob@company.com>",
			wantTools:      nil,
			wantScore:      nil,
			wantConfidence: nil,
		},
		// Co-Authored-By tests end here

		// Assisted-By tests start here
		{
			name:           "assistedby: Claude trailer with Opus model",
			message:        "fix: update handler\n\nAssisted-By: Claude Opus 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Opus 4"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Claude trailer with Sonnet model",
			message:        "fix: update handler\n\nAssisted-By: Claude Sonnet 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Sonnet 4"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Cursor trailer",
			message:        "refactor: extract method\n\nAssisted-By: Cursor <cursoragent@cursor.com>",
			wantTools:      []string{"Cursor"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Aider trailer with model name",
			message:        "feat: add endpoint\n\nAssisted-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools:      []string{"Aider"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Aider trailer with different model",
			message:        "feat: add endpoint\n\nAssisted-By: aider (claude-4.7-opus) <noreply@aider.chat>",
			wantTools:      []string{"Aider"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: multiple trailers with Claude and human",
			message:        "fix: bug\n\nAssisted-By: Claude Opus 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Opus 4"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: multiple AI trailers",
			message:        "fix: bug\n\nAssisted-By: Claude Opus 4 <noreply@anthropic.com>\nAssisted-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools:      []string{"Claude Opus 4", "Aider"},
			wantScore:      []float64{75, 75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: case variation",
			message:        "fix: something\n\nassisted-by: Claude <noreply@anthropic.com>",
			wantTools:      []string{"Claude"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: ASSISTED-BY uppercase",
			message:        "fix: something\n\nASSISTED-BY: Claude <noreply@anthropic.com>",
			wantTools:      []string{"Claude"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Assisted-By trailer in commit message",
			message:        "this is a commit message with\nAssisted-By: Claude Code",
			wantTools:      []string{"Claude Code"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Another Assisted-By trailer in commit message 1",
			message:        "this is a commit message with\nAssisted-By: Gemini",
			wantTools:      []string{"Gemini"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Another Assisted-By trailer in commit message 2",
			message:        "this is a commit message with\nAssisted-By: Kimi K2.6",
			wantTools:      []string{"Kimi K2.6"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Multiple Assisted-By trailer in commit message",
			message:        "this is a commit message with\nAssisted-By: Claude Code\nAssisted-By: Gemini",
			wantTools:      []string{"Claude Code", "Gemini"},
			wantScore:      []float64{75, 75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name: "assistedby: Multiple Assisted-By trailers (with purpose brackets) in commit message",
			message: `
this is a commit message

Co-Authored-By: Cursor <cursoragent@cursor.com>
Assisted-by: Claude 4.7 Opus
	(logic optimization and design fixes)
Assisted-By: Claude Sonnet 4 <noreply@anthropic.com>
Assisted-by: Kimi K2.6 (unit tests, integration tests)
Assisted-by: ChatGPT (documentation review)
Assisted-by: Gemini
Co-Authored-By: Copilot <copilot@github.com>
Signed-off-by: some human <test@example.com>
`,
			wantTools: []string{"Cursor", "Copilot", "Claude 4.7 Opus", "Claude Sonnet 4", "Kimi K2.6", "ChatGPT", "Gemini"},
			wantScore: []float64{75, 75, 75, 75, 75, 75, 75},
			wantConfidence: []detection.Confidence{
				detection.ConfidenceHigh, // Cursor
				detection.ConfidenceHigh, // Copilot
				detection.ConfidenceHigh, // Claude 4.7 Opus
				detection.ConfidenceHigh, // Claude Sonnet 4
				detection.ConfidenceHigh, // Kimi K2.6
				detection.ConfidenceHigh, // ChatGPT
				detection.ConfidenceHigh, // Gemini
			},
		},
		{
			name:           "assistedby: Same tool has 2 Assisted-By trailers in commit message",
			message:        "this is a commit message with\nAssisted-By: Claude Code\nAssisted-By: Claude Code",
			wantTools:      []string{"Claude Code"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Equivalent tools are deduplicated across multiple trailers",
			message:        "this is a commit message with\nAssisted-By: Claude Code\nAssisted-By: CLAUDE CODE\nAssisted-By: [Claude   Code].\nAssisted-By: GitHub Copilot",
			wantTools:      []string{"Claude Code", "GitHub Copilot"},
			wantScore:      []float64{75, 75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Tool with enclosing square brackets",
			message:        "this is a commit message with\nAssisted-By: [Claude Code]",
			wantTools:      []string{"Claude Code"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Tool with extra whitespace and terminal punctuation",
			message:        "this is a commit message with\nAssisted-By: Claude   Code.",
			wantTools:      []string{"Claude Code"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Internal model punctuation is preserved",
			message:        "this is a commit message with\nAssisted-By: GPT-5.5",
			wantTools:      []string{"GPT-5.5"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Internal tool punctuation is preserved",
			message:        "this is a commit message with\nAssisted-By: Continue.dev",
			wantTools:      []string{"Continue.dev"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Assisted-By trailer in commit message in lower case",
			message:        "this is a commit message with\nassisted-by: Claude Code",
			wantTools:      []string{"Claude Code"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Two different attributions (assistedby and coauthor) both with email address",
			message:        "Fix bug\n\nAssisted-By: Claude Sonnet 4 <noreply@anthropic.com>\nCo-Authored-By: Copilot <copilot@github.com>",
			wantTools:      []string{"Copilot", "Claude Sonnet 4"},
			wantScore:      []float64{75, 75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Two different attributions (assistedby and coauthor) one with model name, other with email address",
			message:        "Add validation logic\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\nAssisted-by: GitHub Copilot",
			wantTools:      []string{"Claude Code", "GitHub Copilot"},
			wantModels:     []string{"Sonnet 4.6", ""},
			wantScore:      []float64{85, 75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name:           "assistedby: Claude Opus model attribution trailer",
			message:        "Fix bug\n\nAssisted-by: Claude Opus 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Opus 4"},
			wantScore:      []float64{75},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		//Assisted-by tests end here

		// Other trailer tests start here
		{
			name:           "aider prefix",
			message:        "aider: fix the login bug",
			wantTools:      []string{"Aider"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "aider prefix uppercase",
			message:        "Aider: refactor auth module",
			wantTools:      []string{"Aider"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Claude Code footer",
			message:        "Add user validation\n\nGenerated with Claude Code",
			wantTools:      []string{"Claude Code"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Claude Code footer with link",
			message:        "Add validation\n\nGenerated with Claude Code\nhttps://claude.ai",
			wantTools:      []string{"Claude Code"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "EntireIO trailer present in commit",
			message:        "this is some commit message\n\nEntire-Checkpoint: ab123cdefg12",
			wantTools:      []string{"EntireIO"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Another EntireIO trailer present in commit",
			message:        "this is some commit message\n\nEntire-Metadata: ab123cdefg12",
			wantTools:      []string{"EntireIO"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Another EntireIO trailer present in commit with CRLF line endings",
			message:        "this is some commit message\r\n\r\nEntire-Metadata: ab123cdefg12",
			wantTools:      []string{"EntireIO"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "EntireIO trailer not used, only mentioned in a commit",
			message:        "this is a commit message with\nEntire-Metadata mentioned",
			wantTools:      nil,
			wantScore:      nil,
			wantConfidence: nil,
		},
		{
			name:           "Replit Agent trailer present in a commit",
			message:        "this is a commit message with\nReplit-Commit-Author: Agent",
			wantTools:      []string{"Replit"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Replit Agent trailer present in a commit with session id",
			message:        "this is a commit message with\nReplit-Commit-Author: Agent\nReplit-Commit-Session-Id: 1234a1ab-12ab-1234-abcd-0123456a1234",
			wantTools:      []string{"Replit"},
			wantScore:      []float64{80},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Replit Assistant trailer present in a commit",
			message:        "this is a commit message with\nReplit-Commit-Author: Assistant",
			wantTools:      []string{"Replit"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Replit Assistant trailer present in a commit with session id",
			message:        "this is a commit message with\nReplit-Commit-Author: Assistant\nReplit-Commit-Session-Id: 1234a1ab-12ab-1234-abcd-0123456a1234",
			wantTools:      []string{"Replit"},
			wantScore:      []float64{80},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Replit Agent trailer present in commit with CRLF line endings",
			message:        "this is some commit message\r\n\r\nReplit-Commit-Author: Agent",
			wantTools:      []string{"Replit"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Replit Assistant trailer present in commit with CRLF line endings",
			message:        "this is some commit message\r\n\r\nReplit-Commit-Author: Assistant",
			wantTools:      []string{"Replit"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Replit Agent trailer present in commit with another trailer with CRLF line endings",
			message:        "this is some commit message\r\n\r\nReplit-Commit-Author: Agent\r\nSomeOther: Trailer",
			wantTools:      []string{"Replit"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Replit Assistant trailer present in commit with another trailer with CRLF line endings",
			message:        "this is some commit message\r\n\r\nReplit-Commit-Author: Assistant\r\nSomeOther: Trailer",
			wantTools:      []string{"Replit"},
			wantScore:      []float64{35},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Some other Replit product trailer (not agent or asst) present in a commit",
			message:        "this is a commit message with\nReplit-Commit-Author: SomeOtherReplitProduct",
			wantTools:      nil,
			wantScore:      nil,
			wantConfidence: nil,
		},
		{
			name:           "Replit trailer not used, only mentioned in a commit",
			message:        "this is a commit message with\nReplit-Commit-Author: Assistant mentioned",
			wantTools:      nil,
			wantScore:      nil,
			wantConfidence: nil,
		},
		{
			name:           "aider in middle of message not prefix",
			message:        "fix the aider: integration test",
			wantTools:      nil,
			wantScore:      nil,
			wantConfidence: nil,
		},
		{
			name:           "aider as substring of a word",
			message:        "raider: fix the tests",
			wantTools:      nil,
			wantScore:      nil,
			wantConfidence: nil,
		},
		{
			name:           "no trailers",
			message:        "just a normal commit message with no AI signatures",
			wantTools:      nil,
			wantScore:      nil,
			wantConfidence: nil,
		},
		{
			name:           "empty message",
			message:        "",
			wantTools:      nil,
			wantScore:      nil,
			wantConfidence: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := d.Detect(detection.Input{CommitMessage: tt.message})
			gotTools := make([]string, len(findings))
			gotModels := make([]string, len(findings))
			gotScore := make([]float64, len(findings))
			gotConfidence := make([]detection.Confidence, len(findings))
			for i, f := range findings {
				gotTools[i] = f.Tool
				gotModels[i] = f.Model
				gotScore[i] = f.Score
				gotConfidence[i] = f.Confidence

				if f.Detector != "trailer" {
					t.Errorf("detector = %q, want %q", f.Detector, "trailer")
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
			}

			if tt.wantModels != nil {
				if len(gotModels) != len(tt.wantModels) {
					t.Errorf("models = %v, want %v", gotModels, tt.wantModels)
					return
				}
				for i := range gotModels {
					if gotModels[i] != tt.wantModels[i] {
						t.Errorf("models = %v, want %v", gotModels, tt.wantModels)
						return
					}
				}
			}

			if tt.wantScore != nil {
				if len(gotScore) == 0 {
					gotScore = nil
				}
				if len(gotScore) != len(tt.wantScore) {
					t.Errorf("score = %v, want %v", gotScore, tt.wantScore)
					return
				}
				for i := range gotScore {
					if gotScore[i] != tt.wantScore[i] {
						t.Errorf("score = %v, want %v", gotScore, tt.wantScore)
						return
					}
				}

			}

			if len(gotConfidence) == 0 {
				gotConfidence = nil
			}
			if len(gotConfidence) != len(tt.wantConfidence) {
				t.Errorf("confidence = %v, want %v", gotConfidence, tt.wantConfidence)
				return
			}
			for i := range gotConfidence {
				if gotConfidence[i] != tt.wantConfidence[i] {
					t.Errorf("confidence = %v, want %v", gotConfidence, tt.wantConfidence)
					return
				}
			}
		})
	}
}

func BenchmarkDetect(b *testing.B) {
	d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "no match",
			message: "fix: handle an ordinary commit without attribution",
		},
		{
			name:    "Replit match",
			message: "feat: update handler\n\nReplit-Commit-Author: Agent\nReplit-Commit-Session-Id: 1234a1ab-12ab-1234-abcd-0123456a1234",
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			input := detection.Input{CommitMessage: tt.message}
			b.ReportAllocs()
			for range b.N {
				d.Detect(input)
			}
		})
	}
}
