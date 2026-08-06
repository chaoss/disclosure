package toolmention

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name                string
		input               detection.Input
		includeModelCatalog bool
		wantTools           []string
	}{
		{
			name:      "Claude mention in text",
			input:     detection.Input{Text: "I used Claude to write this PR"},
			wantTools: []string{"Claude"},
		},
		{
			name:      "Claude Code mention in text",
			input:     detection.Input{Text: "Generated with Claude Code"},
			wantTools: []string{"Claude Code"},
		},
		{
			name:      "GitHub Copilot mention",
			input:     detection.Input{Text: "GitHub Copilot helped with this"},
			wantTools: []string{"GitHub Copilot"},
		},
		{
			name:      "Copilot mention",
			input:     detection.Input{Text: "Copilot was used to generate docs"},
			wantTools: []string{"Copilot"},
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
			wantTools: []string{"Claude Code"},
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
			name:      "empty input with spaces",
			input:     detection.Input{Text: "   ", CommitMessage: "\n   \n"},
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
			name:      "GPT-4 mention uses canonical display name once",
			input:     detection.Input{Text: "I asked gpt-4 for help"},
			wantTools: []string{"GPT-4"},
		},
		{
			name:      "generated model mention is off by default",
			input:     detection.Input{Text: "I used Aion-RP 1.0 for a comparison"},
			wantTools: nil,
		},
		{
			name:                "generated model mention when catalogue included",
			input:               detection.Input{Text: "I used Aion-RP 1.0 for a comparison"},
			includeModelCatalog: true,
			wantTools:           []string{"Aion-RP 1.0"},
		},
		{
			name:                "longest generated model match wins",
			input:               detection.Input{Text: "I used Claude Opus 4.6 and GPT-4 Turbo"},
			includeModelCatalog: true,
			wantTools:           []string{"Claude Opus 4.6", "GPT-4 Turbo"},
		},
		{
			name:      "generic router words do not match",
			input:     detection.Input{Text: "Use the free router setting"},
			wantTools: nil,
		},
		{
			name:                "keyboard shortcut prose does not match ambiguous generated model",
			input:               detection.Input{Text: "Press Command A to select all text"},
			includeModelCatalog: true,
			wantTools:           nil,
		},
		{
			name:      "electronics prose does not match ambiguous model",
			input:     detection.Input{Text: "The R1 resistor value changed during testing"},
			wantTools: nil,
		},
		{
			name:      "sonar prose does not match ambiguous model",
			input:     detection.Input{Text: "The sonar data was noisy in shallow water"},
			wantTools: nil,
		},
		{
			name:      "spotlight prose does not match ambiguous model",
			input:     detection.Input{Text: "The UI spotlight highlighted the primary button"},
			wantTools: nil,
		},
		{
			name:                "body builder prose does not match generated model",
			input:               detection.Input{Text: "A body builder updated the training plan"},
			includeModelCatalog: true,
			wantTools:           nil,
		},
		{
			name:                "weaver prose does not match generated model",
			input:               detection.Input{Text: "The weaver repaired the fabric"},
			includeModelCatalog: true,
			wantTools:           nil,
		},
		{
			name:                "bodybuilder prose does not match generated model",
			input:               detection.Input{Text: "The bodybuilder updated the training plan"},
			includeModelCatalog: true,
			wantTools:           nil,
		},
		{
			name:                "command a prose does not match generated model",
			input:               detection.Input{Text: "Press Command A to select all text"},
			includeModelCatalog: true,
			wantTools:           nil,
		},
		{
			name:                "common short generated model names do not match prose",
			input:               detection.Input{Text: "The o1 visa, o3 zone, and Saba reference were unrelated"},
			includeModelCatalog: true,
			wantTools:           nil,
		},
		{
			name:                "uncensored prose does not match generated model",
			input:               detection.Input{Text: "The report included uncensored logs"},
			includeModelCatalog: true,
			wantTools:           nil,
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
		{
			name:      "Qwen coder variant match",
			input:     detection.Input{Text: "Running Qwen as a local autocomplete provider"},
			wantTools: []string{"Qwen"},
		},
		{
			name:      "Kimi K3 tool match",
			input:     detection.Input{Text: "Passed the massive log files into Kimi K3 for error analysis"},
			wantTools: []string{"Kimi"},
		},
		{
			name:      "DeepSeek, Qwen and Llama open weights",
			input:     detection.Input{Text: "Tested using DeepSeek, Qwen, Llama locally"},
			wantTools: []string{"DeepSeek", "Qwen", "Llama"},
		},
		{
			name:      "Mistral and Codestral variations",
			input:     detection.Input{Text: "Switched our completions from Mistral to Codestral"},
			wantTools: []string{"Mistral", "Codestral"},
		},
		{
			name:      "Z.ai ecosystem tools",
			input:     detection.Input{Text: "Used GLM-4 inside ZCode via Z.ai orchestrator"},
			wantTools: []string{"GLM-4", "ZCode", "Z.ai"},
		},
		{
			name:      "Vibe coding platforms bolt and lovable",
			input:     detection.Input{Text: "Scaffolded with Bolt.new and then customized via Lovable.dev"},
			wantTools: []string{"Bolt.new", "Lovable.dev"},
		},
		{
			name:      "Codeium and rebranded Qodo",
			input:     detection.Input{Text: "Codeium handles completions while Qodo and CodiumAI run test suites"},
			wantTools: []string{"Codeium", "Qodo", "CodiumAI"},
		},
		{
			name:      "Replit Agent full stack",
			input:     detection.Input{Text: "Replit Agent deployed the workspace inside Replit"},
			wantTools: []string{"Replit Agent", "Replit"},
		},
		{
			name:      "Mastra and CodeGPT automation",
			input:     detection.Input{Text: "Automated PR reviews handled by Mastra with CodeGPT engine"},
			wantTools: []string{"Mastra", "CodeGPT"},
		},
		{
			name:      "Vercel v0 explicit tool match",
			input:     detection.Input{Text: "Frontend generated entirely using Vercel v0 templates"},
			wantTools: []string{"Vercel v0"},
		},
		{
			name:      "Tabnine and specialized assistants",
			input:     detection.Input{Text: "Compared Tabnine vs Sourcery vs Augment Code"},
			wantTools: []string{"Tabnine", "Sourcery", "Augment Code"},
		},
		{
			name:      "OpenClaw core ecosystem detection",
			input:     detection.Input{Text: "Configured our local orchestration suite via OpenClaw"},
			wantTools: []string{"OpenClaw"},
		},
		{
			name:      "OpenClaw alternative frameworks",
			input:     detection.Input{Text: "Running the daemon through nanoclaw and picoclaw wrappers"},
			wantTools: []string{"NanoClaw", "PicoClaw"},
		},
		{
			name:      "OpenClaude generated code in PR",
			input:     detection.Input{Text: "Code in this PR has been generated with OpenClaude"},
			wantTools: []string{"OpenClaude"},
		},
		{
			name:      "Yi AI",
			input:     detection.Input{Text: "Test cases generated with Yi AI"},
			wantTools: []string{"Yi AI"},
		},
		{
			name:      "Specific model from 01.ai",
			input:     detection.Input{Text: "Test cases generated with Yi-Large by 01.ai"},
			wantTools: []string{"Yi-Large", "01.ai"},
		},
		{
			name:      "No Continue AI false positives",
			input:     detection.Input{Text: "So I will continue to look into this issue"},
			wantTools: nil,
		},
		{
			name:      "Continue AI disclosure",
			input:     detection.Input{Text: "Documentation generated using Continue.dev"},
			wantTools: []string{"Continue.dev"},
		},
		{
			name:      "Command R disclosure",
			input:     detection.Input{Text: "Using Command-R for bug fixes"},
			wantTools: []string{"Command-R"},
		},
		{
			name:      "Command R+ disclosure",
			input:     detection.Input{Text: "Using Command-R+ for documentation"},
			wantTools: []string{"Command-R+"},
		},
		{
			name:      "Command R and R+ used and disclosed together",
			input:     detection.Input{Text: "Using Command R for bug fixes and Command R+ for documentation"},
			wantTools: []string{"Command-R", "Command-R+"},
		},
		{
			name:      "Command-R+ in backticks",
			input:     detection.Input{Text: "Running `Command-R+` via the Cohere API"},
			wantTools: []string{"Command-R+"},
		},
		{
			name:      "tool name in markdown backticks",
			input:     detection.Input{Text: "I used `Claude` to draft the summary"},
			wantTools: []string{"Claude"},
		},
		{
			name:      "slash-separated tool names both detected",
			input:     detection.Input{Text: "Compared Claude/ChatGPT for this task"},
			wantTools: []string{"Claude", "ChatGPT"},
		},
		{
			name:      "Cohere ordinary word does not match as tool",
			input:     detection.Input{Text: "The changes cohere with existing conventions"},
			wantTools: nil,
		},
		{
			name:      "Gemini ordinary reference does not match as tool",
			input:     detection.Input{Text: "The Gemini spacecraft mission was discussed in class"},
			wantTools: nil,
		},
		{
			name:      "Llama ordinary animal reference will also match (false positive)",
			input:     detection.Input{Text: "The llama walked across the field"},
			wantTools: []string{"Llama"},
		},
		{
			name:      "Gemini inside a longer word does not match",
			input:     detection.Input{Text: "The geminized configuration was rejected"},
			wantTools: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Detector{IncludeModelCatalog: tt.includeModelCatalog}
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

func BenchmarkDetectNoMatchDefault(b *testing.B) {
	d := &Detector{}
	input := detection.Input{Text: "This ordinary engineering note has no AI disclosure signal."}

	for b.Loop() {
		_ = d.Detect(input)
	}
}

func BenchmarkDetectNoMatchWithModelCatalog(b *testing.B) {
	d := &Detector{IncludeModelCatalog: true}
	input := detection.Input{Text: "This ordinary engineering note has no AI disclosure signal."}

	for b.Loop() {
		_ = d.Detect(input)
	}
}
