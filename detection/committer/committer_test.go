package committer

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestDetectAllKnownEmails(t *testing.T) {
	d := &Detector{}
	for email, expectedName := range detection.KnownAgentCommitters {
		input := detection.Input{CommitEmail: email}
		findings := d.Detect(input)
		if len(findings) != 1 {
			t.Errorf("Detect(%q): got %d findings, want 1", email, len(findings))
			continue
		}
		if findings[0].Tool != expectedName {
			t.Errorf("Detect(%q): tool = %q, want %q", email, findings[0].Tool, expectedName)
		}
		if findings[0].Confidence != detection.ConfidenceHigh {
			t.Errorf("Detect(%q): confidence = %d, want %d", email, findings[0].Confidence, detection.ConfidenceHigh)
		}
		if findings[0].Detector != "committer" {
			t.Errorf("Detect(%q): detector = %q, want %q", email, findings[0].Detector, "committer")
		}
	}
}

func TestDetectMixedCase(t *testing.T) {
	d := &Detector{}
	cases := []struct {
		input    string
		wantTool string
	}{
		{"198982749+Copilot@users.noreply.github.com", "GitHub Copilot (agent)"},
		{"209825114+CLAUDE[BOT]@USERS.NOREPLY.GITHUB.COM", "Claude"},
		{"136622811+CodeRabbitAI[bot]@users.noreply.github.com", "CodeRabbit"},
	}

	for _, tc := range cases {
		findings := d.Detect(detection.Input{CommitEmail: tc.input})
		if len(findings) != 1 {
			t.Errorf("Detect(%q): got %d findings, want 1", tc.input, len(findings))
			continue
		}
		if findings[0].Tool != tc.wantTool {
			t.Errorf("Detect(%q): tool = %q, want %q", tc.input, findings[0].Tool, tc.wantTool)
		}
	}
}

func TestDetectWhitespace(t *testing.T) {
	d := &Detector{}
	cases := []string{
		"  209825114+claude[bot]@users.noreply.github.com",
		"209825114+claude[bot]@users.noreply.github.com  ",
		"  209825114+claude[bot]@users.noreply.github.com  ",
	}

	for _, email := range cases {
		findings := d.Detect(detection.Input{CommitEmail: email})
		if len(findings) != 1 {
			t.Errorf("Detect(%q): got %d findings, want 1", email, len(findings))
			continue
		}
		if findings[0].Tool != "Claude" {
			t.Errorf("Detect(%q): tool = %q, want %q", email, findings[0].Tool, "Claude")
		}
	}
}

func TestDetectNotFound(t *testing.T) {
	d := &Detector{}
	cases := []string{
		"user@example.com",
		"",
		"  ",
		"claude@anthropic.com",
		"not-a-bot@users.noreply.github.com",
	}

	for _, email := range cases {
		findings := d.Detect(detection.Input{CommitEmail: email})
		if len(findings) != 0 {
			t.Errorf("Detect(%q): got %d findings, want 0", email, len(findings))
		}
	}
}

func TestDetectNumericPrefix(t *testing.T) {
	d := &Detector{}
	// Simulate a renamed bot: same numeric ID, different username
	cases := []struct {
		input    string
		wantTool string
	}{
		// Claude bot with a different username
		{"209825114+renamed-claude-bot@users.noreply.github.com", "Claude"},
		// Copilot with a different username
		{"198982749+copilot-v2@users.noreply.github.com", "GitHub Copilot (agent)"},
		// CodeRabbit with a different username
		{"136622811+coderabbit-new@users.noreply.github.com", "CodeRabbit"},
	}

	for _, tc := range cases {
		findings := d.Detect(detection.Input{CommitEmail: tc.input})
		if len(findings) != 1 {
			t.Errorf("Detect(%q): got %d findings, want 1", tc.input, len(findings))
			continue
		}
		if findings[0].Tool != tc.wantTool {
			t.Errorf("Detect(%q): tool = %q, want %q", tc.input, findings[0].Tool, tc.wantTool)
		}
	}
}

func TestDetectNumericPrefixNoFalsePositive(t *testing.T) {
	d := &Detector{}
	// An email with a numeric prefix that doesn't match any known bot
	cases := []string{
		"999999999+someone@users.noreply.github.com",
		"12345+user@users.noreply.github.com",
	}

	for _, email := range cases {
		findings := d.Detect(detection.Input{CommitEmail: email})
		if len(findings) != 0 {
			t.Errorf("Detect(%q): got %d findings, want 0", email, len(findings))
		}
	}
}

func TestDetectAuthorAndCommitter(t *testing.T) {
	const (
		claudeEmail  = "209825114+claude[bot]@users.noreply.github.com"
		copilotEmail = "198982749+copilot@users.noreply.github.com"
		humanEmail   = "human@example.com"
	)

	d := &Detector{}
	tests := []struct {
		name        string
		input       detection.Input
		wantTools   []string
		wantDetails []string
	}{
		{
			name: "author only",
			input: detection.Input{
				AuthorEmail: copilotEmail,
				CommitEmail: humanEmail,
			},
			wantTools: []string{detection.KnownAgentCommitters[copilotEmail]},
			wantDetails: []string{
				"author email " + copilotEmail + " matches known AI bot",
			},
		},
		{
			name: "committer only",
			input: detection.Input{
				AuthorEmail: humanEmail,
				CommitEmail: claudeEmail,
			},
			wantTools: []string{detection.KnownAgentCommitters[claudeEmail]},
			wantDetails: []string{
				"committer email " + claudeEmail + " matches known AI bot",
			},
		},
		{
			name: "different agent identities",
			input: detection.Input{
				AuthorEmail: copilotEmail,
				CommitEmail: claudeEmail,
			},
			wantTools: []string{
				detection.KnownAgentCommitters[copilotEmail],
				detection.KnownAgentCommitters[claudeEmail],
			},
			wantDetails: []string{
				"author email " + copilotEmail + " matches known AI bot",
				"committer email " + claudeEmail + " matches known AI bot",
			},
		},
		{
			name: "same identity in both fields",
			input: detection.Input{
				AuthorEmail: copilotEmail,
				CommitEmail: copilotEmail,
			},
			wantTools: []string{detection.KnownAgentCommitters[copilotEmail]},
			wantDetails: []string{
				"author and committer email " + copilotEmail + " matches known AI bot",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := d.Detect(tt.input)
			if len(findings) != len(tt.wantTools) {
				t.Fatalf("got %d findings, want %d", len(findings), len(tt.wantTools))
			}
			for i, finding := range findings {
				if finding.Tool != tt.wantTools[i] {
					t.Errorf("finding %d tool = %q, want %q", i, finding.Tool, tt.wantTools[i])
				}
				if finding.Detail != tt.wantDetails[i] {
					t.Errorf("finding %d detail = %q, want %q", i, finding.Detail, tt.wantDetails[i])
				}
			}
		})
	}
}
