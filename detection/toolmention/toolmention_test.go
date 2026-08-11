package toolmention

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestDetect(t *testing.T) {
	d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}

	tests := []struct {
		name           string
		input          detection.Input
		wantTools      []string
		wantScore      float64
		wantConfidence detection.Confidence
	}{
		{
			name:           "Claude mention in text",
			input:          detection.Input{Text: "I used Claude to write this PR"},
			wantTools:      []string{"Claude"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Claude Code mention in text",
			input:          detection.Input{Text: "Generated with Claude Code"},
			wantTools:      []string{"Claude Code"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "GitHub Copilot mention",
			input:          detection.Input{Text: "GitHub Copilot helped with this"},
			wantTools:      []string{"GitHub Copilot"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Copilot mention",
			input:          detection.Input{Text: "Copilot was used to generate docs"},
			wantTools:      []string{"Copilot"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "multiple tools mentioned",
			input:          detection.Input{Text: "I used Cursor and Aider for this PR"},
			wantTools:      []string{"Cursor", "Aider"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "case insensitive",
			input:          detection.Input{Text: "I used CLAUDE to write this"},
			wantTools:      []string{"Claude"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "commit message scanned too",
			input:          detection.Input{CommitMessage: "feat: add feature\n\nGenerated with Claude Code"},
			wantTools:      []string{"Claude Code"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "text and commit message combined",
			input:          detection.Input{Text: "Used Cursor", CommitMessage: "aider: fix bug"},
			wantTools:      []string{"Cursor", "Aider"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "no mentions",
			input:          detection.Input{Text: "This is a normal PR description"},
			wantTools:      nil,
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "empty input with spaces",
			input:          detection.Input{Text: "   ", CommitMessage: "\n   \n"},
			wantTools:      nil,
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "empty input",
			input:          detection.Input{},
			wantTools:      nil,
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "word boundary prevents partial match",
			input:          detection.Input{Text: "The cursory review found nothing"},
			wantTools:      nil,
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "ChatGPT mention",
			input:          detection.Input{Text: "I asked ChatGPT for help"},
			wantTools:      []string{"ChatGPT"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "t3.chat mention",
			input:          detection.Input{Text: "I used t3.chat to compare model outputs"},
			wantTools:      []string{"t3.chat"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "t3.chat mention is case insensitive",
			input:          detection.Input{Text: "Generated with T3.CHAT"},
			wantTools:      []string{"t3.chat"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "t3.chat word boundary prevents partial match",
			input:          detection.Input{Text: "This mentions t3.chatty, not the tool"},
			wantTools:      nil,
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Windsurf mention",
			input:          detection.Input{Text: "Written with Windsurf IDE"},
			wantTools:      []string{"Windsurf"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Devin mention",
			input:          detection.Input{Text: "Devin created this PR"},
			wantTools:      []string{"Devin"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "duplicate tool mentions only produce one finding",
			input:          detection.Input{Text: "Claude helped here. Claude helped there."},
			wantTools:      []string{"Claude"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Qwen coder variant match",
			input:          detection.Input{Text: "Running Qwen as a local autocomplete provider"},
			wantTools:      []string{"Qwen"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Kimi K3 tool match",
			input:          detection.Input{Text: "Passed the massive log files into Kimi K3 for error analysis"},
			wantTools:      []string{"Kimi"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "DeepSeek, Qwen and Llama open weights",
			input:          detection.Input{Text: "Tested using DeepSeek, Qwen, Llama locally"},
			wantTools:      []string{"DeepSeek", "Qwen", "Llama"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Mistral and Codestral variations",
			input:          detection.Input{Text: "Switched our completions from Mistral to Codestral"},
			wantTools:      []string{"Mistral", "Codestral"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Z.ai ecosystem tools",
			input:          detection.Input{Text: "Used GLM-4 inside ZCode via Z.ai orchestrator"},
			wantTools:      []string{"GLM-4", "ZCode", "Z.ai"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Vibe coding platforms bolt and lovable",
			input:          detection.Input{Text: "Scaffolded with Bolt.new and then customized via Lovable.dev"},
			wantTools:      []string{"Bolt.new", "Lovable.dev"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Codeium and rebranded Qodo",
			input:          detection.Input{Text: "Codeium handles completions while Qodo and CodiumAI run test suites"},
			wantTools:      []string{"Codeium", "Qodo", "CodiumAI"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Replit Agent full stack",
			input:          detection.Input{Text: "Replit Agent deployed the workspace inside Replit"},
			wantTools:      []string{"Replit Agent", "Replit"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Mastra and CodeGPT automation",
			input:          detection.Input{Text: "Automated PR reviews handled by Mastra with CodeGPT engine"},
			wantTools:      []string{"Mastra", "CodeGPT"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Vercel v0 explicit tool match",
			input:          detection.Input{Text: "Frontend generated entirely using Vercel v0 templates"},
			wantTools:      []string{"Vercel v0"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Tabnine and specialized assistants",
			input:          detection.Input{Text: "Compared Tabnine vs Sourcery vs Augment Code"},
			wantTools:      []string{"Tabnine", "Sourcery", "Augment Code"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "OpenClaw core ecosystem detection",
			input:          detection.Input{Text: "Configured our local orchestration suite via OpenClaw"},
			wantTools:      []string{"OpenClaw"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "OpenClaw alternative frameworks",
			input:          detection.Input{Text: "Running the daemon through nanoclaw and picoclaw wrappers"},
			wantTools:      []string{"NanoClaw", "PicoClaw"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "OpenClaude generated code in PR",
			input:          detection.Input{Text: "Code in this PR has been generated with OpenClaude"},
			wantTools:      []string{"OpenClaude"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Yi AI",
			input:          detection.Input{Text: "Test cases generated with Yi AI"},
			wantTools:      []string{"Yi AI"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Specific model from 01.ai",
			input:          detection.Input{Text: "Test cases generated with Yi-Large by 01.ai"},
			wantTools:      []string{"Yi-Large", "01.ai"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "No Continue AI false positives",
			input:          detection.Input{Text: "So I will continue to look into this issue"},
			wantTools:      nil,
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Continue AI disclosure",
			input:          detection.Input{Text: "Documentation generated using Continue.dev"},
			wantTools:      []string{"Continue.dev"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Command R disclosure",
			input:          detection.Input{Text: "Using Command-R for bug fixes"},
			wantTools:      []string{"Command-R"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Command R+ disclosure",
			input:          detection.Input{Text: "Using Command-R+ for documentation"},
			wantTools:      []string{"Command-R+"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Command R and R+ used and disclosed together",
			input:          detection.Input{Text: "Using Command R for bug fixes and Command R+ for documentation"},
			wantTools:      []string{"Command-R", "Command-R+"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Command-R+ in backticks",
			input:          detection.Input{Text: "Running `Command-R+` via the Cohere API"},
			wantTools:      []string{"Command-R+"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "tool name in markdown backticks",
			input:          detection.Input{Text: "I used `Claude` to draft the summary"},
			wantTools:      []string{"Claude"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "slash-separated tool names both detected",
			input:          detection.Input{Text: "Compared Claude/ChatGPT for this task"},
			wantTools:      []string{"Claude", "ChatGPT"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Cohere ordinary word does not match as tool",
			input:          detection.Input{Text: "The changes cohere with existing conventions"},
			wantTools:      nil,
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Gemini ordinary reference does not match as tool",
			input:          detection.Input{Text: "The Gemini spacecraft mission was discussed in class"},
			wantTools:      nil,
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Llama ordinary animal reference will also match (false positive)",
			input:          detection.Input{Text: "The llama walked across the field"},
			wantTools:      []string{"Llama"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name:           "Gemini inside a longer word does not match",
			input:          detection.Input{Text: "The geminized configuration was rejected"},
			wantTools:      nil,
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},

		// Checkbox-based AI disclosure tests.
		{
			name: "AI used checkbox with tool mention",
			input: detection.Input{
				Text: "[x] This contribution was assisted or created by Generative AI tools.\n\nWhich AI tool was used? Claude",
			},
			wantTools:      []string{"Claude"},
			wantScore:      95,
			wantConfidence: detection.ConfidenceHigh,
		},
		{
			name: "AI used checkbox with multiple tool mentions",
			input: detection.Input{
				Text: "[x] This contribution was assisted or created by Generative AI tools.\n\nTools used: Claude and Cursor",
			},
			wantTools:      []string{"Claude", "Cursor"},
			wantScore:      95,
			wantConfidence: detection.ConfidenceHigh,
		},
		{
			name: "AI used checkbox without tool mention",
			input: detection.Input{
				Text: "[x] This contribution was assisted or created by Generative AI tools.",
			},
			wantTools:      []string{""},
			wantScore:      75,
			wantConfidence: detection.ConfidenceHigh,
		},
		{
			name: "AI used checkbox with detailed disclosure",
			input: detection.Input{
				Text: `Generative AI disclosure

Please select one option:

[ ] This contribution was NOT assisted or created by Generative AI tools.
[x] This contribution was assisted or created by Generative AI tools.

If AI tools were used, please provide details below:

- What tools were used? Claude
- How were these tools used? Draft.
- Did you review these outputs before submitting this PR? Yes.`,
			},
			wantTools:      []string{"Claude"},
			wantScore:      95,
			wantConfidence: detection.ConfidenceHigh,
		},
		{
			name: "AI used checkbox with lowercase x and whitespace",
			input: detection.Input{
				Text: "[ X ]   This contribution was assisted or created by Generative AI tools.   ",
			},
			wantTools:      []string{""},
			wantScore:      75,
			wantConfidence: detection.ConfidenceHigh,
		},
		{
			name: "AI used checkbox unchecked",
			input: detection.Input{
				Text: "[ ] This contribution was assisted or created by Generative AI tools.\n\nClaude was not used.",
			},
			wantTools:      []string{"Claude"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name: "AI not used checkbox checked",
			input: detection.Input{
				Text: "[x] This contribution was NOT assisted or created by Generative AI tools.\n\nClaude was mentioned in documentation.",
			},
			wantTools:      []string{"Claude"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name: "both AI checkboxes checked, ambiguous so ignored",
			input: detection.Input{
				Text: "[x] This contribution was assisted or created by Generative AI tools.\n[x] This contribution was NOT assisted or created by Generative AI tools.\n\nClaude",
			},
			wantTools:      []string{"Claude"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
		{
			name: "AI used checkbox in commit message",
			input: detection.Input{
				Text:          "Claude was used for this change.",
				CommitMessage: "[x] This contribution was assisted or created by Generative AI tools.",
			},
			wantTools:      []string{"Claude"},
			wantScore:      95,
			wantConfidence: detection.ConfidenceHigh,
		},
		{
			name: "both AI checkboxes unchecked without tool mention",
			input: detection.Input{
				Text: "[] This contribution was assisted or created by Generative AI tools.\n[] This contribution was NOT assisted or created by Generative AI tools.\n\nSome other text",
			},
			wantTools:      nil,
			wantScore:      0,
			wantConfidence: detection.ConfidenceLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := d.Detect(tt.input)

			gotTools := make([]string, len(findings))
			for i, f := range findings {
				gotTools[i] = f.Tool

				if f.Confidence != tt.wantConfidence {
					t.Errorf(
						"confidence = %d, want %d", f.Confidence, tt.wantConfidence,
					)
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

func TestDetectCustomCheckboxLabels(t *testing.T) {
	d := &Detector{}
	d.SetConfidenceLevels(detection.GetDefaultConfidenceLevels())
	d.SetCheckboxAILabels(
		"This PR was created with AI assistance",
		"This PR was created without AI assistance",
	)

	tests := []struct {
		name           string
		input          detection.Input
		wantTools      []string
		wantScore      float64
		wantConfidence detection.Confidence
	}{
		{
			name: "custom AI used checkbox with tool",
			input: detection.Input{
				Text: "[x] This PR was created with AI assistance\n\nClaude was used for drafting.",
			},
			wantTools:      []string{"Claude"},
			wantScore:      95,
			wantConfidence: detection.ConfidenceHigh,
		},
		{
			name: "custom AI used checkbox without tool",
			input: detection.Input{
				Text: "[x] This PR was created with AI assistance",
			},
			wantTools:      []string{""},
			wantScore:      75,
			wantConfidence: detection.ConfidenceHigh,
		},
		{
			name: "custom AI not used checkbox",
			input: detection.Input{
				Text: "[x] This PR was created without AI assistance\n\nClaude was mentioned in the discussion.",
			},
			wantTools:      []string{"Claude"},
			wantScore:      20,
			wantConfidence: detection.ConfidenceLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := d.Detect(tt.input)

			if len(findings) != len(tt.wantTools) {
				t.Fatalf("findings count = %d, want %d. findings=%v", len(findings), len(tt.wantTools), findings)
			}

			for i, f := range findings {
				if f.Tool != tt.wantTools[i] {
					t.Errorf("tool[%d] = %q, want %q", i, f.Tool, tt.wantTools[i])
				}

				if f.Score != tt.wantScore {
					t.Errorf("score[%d] = %v, want %v", i, f.Score, tt.wantScore)
				}

				if f.Confidence != tt.wantConfidence {
					t.Errorf("confidence[%d] = %s, want %s", i, f.Confidence.String(), tt.wantConfidence.String())
				}

				if f.Detector != "toolmention" {
					t.Errorf("detector[%d] = %q, want %q", i, f.Detector, "toolmention")
				}
			}
		})
	}
}
