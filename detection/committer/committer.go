package committer

import (
	"fmt"
	"strings"

	"github.com/chaoss/disclosure/detection"
)

// knownAgentCommitters maps GitHub noreply emails to AI tool names.
var knownAgentCommitters = map[string]string{
	"209825114+claude[bot]@users.noreply.github.com":               "Claude",
	"215619710+anthropic-claude[bot]@users.noreply.github.com":     "Claude (Anthropic)",
	"208546643+claude-code-action[bot]@users.noreply.github.com":   "Claude Code Action",
	"198982749+copilot@users.noreply.github.com":                   "GitHub Copilot (agent)",
	"167198135+copilot[bot]@users.noreply.github.com":              "GitHub Copilot (chat)",
	"206951365+cursor[bot]@users.noreply.github.com":               "Cursor",
	"215057067+openai-codex[bot]@users.noreply.github.com":         "OpenAI Codex",
	"199175422+chatgpt-codex-connector[bot]@users.noreply.github.com": "Codex via ChatGPT",
	"176961590+gemini-code-assist[bot]@users.noreply.github.com":   "Gemini Code Assist",
	"208079219+amazon-q-developer[bot]@users.noreply.github.com":   "Amazon Q Developer",
	"158243242+devin-ai-integration[bot]@users.noreply.github.com": "Devin",
	"205137888+cline[bot]@users.noreply.github.com":                "Cline",
	"230936708+continue[bot]@users.noreply.github.com":             "Continue.dev",
	"201248094+sourcegraph-cody[bot]@users.noreply.github.com":     "Sourcegraph Cody",
	"220155983+jetbrains-ai[bot]@users.noreply.github.com":         "JetBrains AI",
	"136622811+coderabbitai[bot]@users.noreply.github.com":         "CodeRabbit",
}

// numericPrefixIndex maps the numeric prefix from GitHub noreply emails to tool names.
// This handles issue #4: when a bot's username changes, the numeric ID stays the same.
var numericPrefixIndex map[string]string

func init() {
	numericPrefixIndex = make(map[string]string, len(knownAgentCommitters))
	for email, name := range knownAgentCommitters {
		if idx := strings.Index(email, "+"); idx > 0 {
			numericPrefixIndex[email[:idx]] = name
		}
	}
}

type Detector struct{}

func (d *Detector) Name() string { return "committer" }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	email := strings.ToLower(strings.TrimSpace(input.CommitEmail))
	if email == "" {
		return nil
	}

	// Direct match against known emails
	if name, ok := knownAgentCommitters[email]; ok {
		return []detection.Finding{{
			Detector:   d.Name(),
			Tool:       name,
			Confidence: detection.ConfidenceHigh,
			Detail:     fmt.Sprintf("committer email %s matches known AI bot", email),
		}}
	}

	// Numeric prefix match for GitHub noreply emails (#4).
	// Format: <numeric-id>+<username>@users.noreply.github.com
	if strings.HasSuffix(email, "@users.noreply.github.com") {
		if idx := strings.Index(email, "+"); idx > 0 {
			prefix := email[:idx]
			if name, ok := numericPrefixIndex[prefix]; ok {
				return []detection.Finding{{
					Detector:   d.Name(),
					Tool:       name,
					Confidence: detection.ConfidenceHigh,
					Detail:     fmt.Sprintf("committer email %s matches known AI bot", email),
				}}
			}
		}
	}

	return nil
}
