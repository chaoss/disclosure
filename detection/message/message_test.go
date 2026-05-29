package message

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
			name:           "no patterns",
			message:        "normal commit message with no AI signatures",
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

				if f.Detector != "message" {
					t.Errorf("detector = %q, want %q", f.Detector, "message")
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
