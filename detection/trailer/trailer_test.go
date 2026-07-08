package trailer

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestDetect(t *testing.T) {
	d := &Detector{}
	tests := []struct {
		name           string
		message        string
		wantTools      []string
		wantConfidence []detection.Confidence
	}{
		// Co-Authored-By tests start here
		{
			name:           "Claude trailer with Opus model",
			message:        "fix: update handler\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Code"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Claude trailer with Sonnet model",
			message:        "fix: update handler\n\nCo-Authored-By: Claude Sonnet 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Code"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Cursor trailer",
			message:        "refactor: extract method\n\nCo-Authored-By: Cursor <cursoragent@cursor.com>",
			wantTools:      []string{"Cursor"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Aider trailer with model name",
			message:        "feat: add endpoint\n\nCo-Authored-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools:      []string{"Aider"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Aider trailer with different model",
			message:        "feat: add endpoint\n\nCo-Authored-By: aider (claude-3.5-sonnet) <noreply@aider.chat>",
			wantTools:      []string{"Aider"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "multiple trailers with Claude and human",
			message:        "fix: bug\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>\nCo-Authored-By: Alice <alice@example.com>",
			wantTools:      []string{"Claude Code"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "multiple AI trailers",
			message:        "fix: bug\n\nCo-Authored-By: Claude Opus 4 <noreply@anthropic.com>\nCo-Authored-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools:      []string{"Claude Code", "Aider"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name:           "case variation",
			message:        "fix: thing\n\nco-authored-by: Claude <noreply@anthropic.com>",
			wantTools:      []string{"Claude Code"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "CO-AUTHORED-BY uppercase",
			message:        "fix: thing\n\nCO-AUTHORED-BY: Claude <noreply@anthropic.com>",
			wantTools:      []string{"Claude Code"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "human co-author only",
			message:        "pair programming\n\nCo-Authored-By: Bob <bob@company.com>",
			wantTools:      nil,
			wantConfidence: nil,
		},
		// Co-Authored-By tests end here

		// Assisted-By tests start here
		{
			name:           "Claude trailer with Opus model",
			message:        "fix: update handler\n\nAssisted-By: Claude Opus 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Opus 4"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Claude trailer with Sonnet model",
			message:        "fix: update handler\n\nAssisted-By: Claude Sonnet 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Sonnet 4"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Cursor trailer",
			message:        "refactor: extract method\n\nAssisted-By: Cursor <cursoragent@cursor.com>",
			wantTools:      []string{"Cursor"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Aider trailer with model name",
			message:        "feat: add endpoint\n\nAssisted-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools:      []string{"Aider"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Aider trailer with different model",
			message:        "feat: add endpoint\n\nAssisted-By: aider (claude-4.7-opus) <noreply@aider.chat>",
			wantTools:      []string{"Aider"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "multiple trailers with Claude and human",
			message:        "fix: bug\n\nAssisted-By: Claude Opus 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Opus 4"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "multiple AI trailers",
			message:        "fix: bug\n\nAssisted-By: Claude Opus 4 <noreply@anthropic.com>\nAssisted-By: aider (gpt-4o) <noreply@aider.chat>",
			wantTools:      []string{"Claude Opus 4", "Aider"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name:           "case variation",
			message:        "fix: something\n\nassisted-by: Claude <noreply@anthropic.com>",
			wantTools:      []string{"Claude"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "ASSISTED-BY uppercase",
			message:        "fix: something\n\nASSISTED-BY: Claude <noreply@anthropic.com>",
			wantTools:      []string{"Claude"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Assisted-By trailer in commit message",
			message:        "this is a commit message with\nAssisted-By: Claude Code",
			wantTools:      []string{"Claude Code"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Another Assisted-By trailer in commit message 1",
			message:        "this is a commit message with\nAssisted-By: Gemini",
			wantTools:      []string{"Gemini"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Another Assisted-By trailer in commit message 2",
			message:        "this is a commit message with\nAssisted-By: Kimi K2.6",
			wantTools:      []string{"Kimi K2.6"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Multiple Assisted-By trailer in commit message",
			message:        "this is a commit message with\nAssisted-By: Claude Code\nAssisted-By: Gemini",
			wantTools:      []string{"Claude Code", "Gemini"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name: "Multiple Assisted-By trailers (with purpose brackets) in commit message",
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
			name:           "Assisted-By trailer in commit message in lower case",
			message:        "this is a commit message with\nassisted-by: Claude Code",
			wantTools:      []string{"Claude Code"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Two different attributions (assistedby and coauthor) both with email address",
			message:        "Fix bug\n\nAssisted-By: Claude Sonnet 4 <noreply@anthropic.com>\nCo-Authored-By: Copilot <copilot@github.com>",
			wantTools:      []string{"Copilot", "Claude Sonnet 4"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name:           "Two different attributions (assistedby and coauthor) one with model name, other with email address",
			message:        "Add validation logic\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\nAssisted-by: GitHub Copilot",
			wantTools:      []string{"Claude Code", "GitHub Copilot"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh, detection.ConfidenceHigh},
		},
		{
			name:           "Claude Opus model attribution trailer",
			message:        "Fix bug\n\nAssisted-by: Claude Opus 4 <noreply@anthropic.com>",
			wantTools:      []string{"Claude Opus 4"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		//Assisted-by tests end here

		// Other trailer tests start here
		{
			name:           "aider prefix",
			message:        "aider: fix the login bug",
			wantTools:      []string{"Aider"},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "aider prefix uppercase",
			message:        "Aider: refactor auth module",
			wantTools:      []string{"Aider"},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Claude Code footer",
			message:        "Add user validation\n\nGenerated with Claude Code",
			wantTools:      []string{"Claude Code"},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Claude Code footer with link",
			message:        "Add validation\n\nGenerated with Claude Code\nhttps://claude.ai",
			wantTools:      []string{"Claude Code"},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "EntireIO trailer present in commit",
			message:        "this is some commit message\n\nEntire-Checkpoint: ab123cdefg12",
			wantTools:      []string{"EntireIO"},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Another EntireIO trailer present in commit",
			message:        "this is some commit message\n\nEntire-Metadata: ab123cdefg12",
			wantTools:      []string{"EntireIO"},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Another EntireIO trailer present in commit with CRLF line endings",
			message:        "this is some commit message\r\n\r\nEntire-Metadata: ab123cdefg12",
			wantTools:      []string{"EntireIO"},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "EntireIO trailer not used, only mentioned in a commit",
			message:        "this is a commit message with\nEntire-Metadata mentioned",
			wantTools:      nil,
			wantConfidence: nil,
		},
		{
			name:           "Replit Agent trailer present in a commit",
			message:        "this is a commit message with\nReplit-Commit-Author: Agent",
			wantTools:      []string{"Replit"},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Replit Agent trailer present in a commit with session id",
			message:        "this is a commit message with\nReplit-Commit-Author: Agent\nReplit-Commit-Session-Id: 1234a1ab-12ab-1234-abcd-0123456a1234",
			wantTools:      []string{"Replit"},
			wantConfidence: []detection.Confidence{detection.ConfidenceHigh},
		},
		{
			name:           "Replit Assistant trailer present in a commit",
			message:        "this is a commit message with\nReplit-Commit-Author: Assistant",
			wantTools:      []string{"Replit"},
			wantConfidence: []detection.Confidence{detection.ConfidenceLow},
		},
		{
			name:           "Replit Assistant trailer present in a commit with session id",
			message:        "this is a commit message with\nReplit-Commit-Author: Assistant\nReplit-Commit-Session-Id: 1234a1ab-12ab-1234-abcd-0123456a1234",
			wantTools:      []string{"Replit"},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Replit Agent trailer present in commit with CRLF line endings",
			message:        "this is some commit message\r\n\r\nReplit-Commit-Author: Agent",
			wantTools:      []string{"Replit"},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Replit Assistant trailer present in commit with CRLF line endings",
			message:        "this is some commit message\r\n\r\nReplit-Commit-Author: Assistant",
			wantTools:      []string{"Replit"},
			wantConfidence: []detection.Confidence{detection.ConfidenceLow},
		},
		{
			name:           "Replit Agent trailer present in commit with another trailer with CRLF line endings",
			message:        "this is some commit message\r\n\r\nReplit-Commit-Author: Agent\r\nSomeOther: Trailer",
			wantTools:      []string{"Replit"},
			wantConfidence: []detection.Confidence{detection.ConfidenceMedium},
		},
		{
			name:           "Replit Assistant trailer present in commit with another trailer with CRLF line endings",
			message:        "this is some commit message\r\n\r\nReplit-Commit-Author: Assistant\r\nSomeOther: Trailer",
			wantTools:      []string{"Replit"},
			wantConfidence: []detection.Confidence{detection.ConfidenceLow},
		},
		{
			name:           "Some other Replit product trailer (not agent or asst) present in a commit",
			message:        "this is a commit message with\nReplit-Commit-Author: SomeOtherReplitProduct",
			wantTools:      nil,
			wantConfidence: nil,
		},
		{
			name:           "Replit trailer not used, only mentioned in a commit",
			message:        "this is a commit message with\nReplit-Commit-Author: Assistant mentioned",
			wantTools:      nil,
			wantConfidence: nil,
		},
		{
			name:           "aider in middle of message not prefix",
			message:        "fix the aider: integration test",
			wantTools:      nil,
			wantConfidence: nil,
		},
		{
			name:           "aider as substring of a word",
			message:        "raider: fix the tests",
			wantTools:      nil,
			wantConfidence: nil,
		},
		{
			name:           "no trailers",
			message:        "just a normal commit message with no AI signatures",
			wantTools:      nil,
			wantConfidence: nil,
		},
		{
			name:           "empty message",
			message:        "",
			wantTools:      nil,
			wantConfidence: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := d.Detect(detection.Input{CommitMessage: tt.message})
			gotTools := make([]string, len(findings))
			gotConfidence := make([]detection.Confidence, len(findings))
			for i, f := range findings {
				gotTools[i] = f.Tool
				gotConfidence[i] = f.Confidence

				if f.Detector != "trailer" {
					t.Errorf("detector = %q, want %q", f.Detector, "trailer")
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
