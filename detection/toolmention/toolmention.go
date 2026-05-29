package toolmention

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/chaoss/disclosure/detection"
)

// toolPatterns maps AI tool names to compiled word-boundary regexes.
var toolPatterns []struct {
	name    string
	pattern *regexp.Regexp
}

func init() {
	tools := []string{
		"Claude Code",
		"Claude",
		"GitHub Copilot",
		"Copilot",
		"Cursor",
		"Aider",
		"OpenAI Codex",
		"Codex",
		"Gemini Code Assist",
		"Amazon Q Developer",
		"Amazon Q",
		"Devin",
		"Cline",
		"Continue.dev",
		"Sourcegraph Cody",
		"Cody",
		"JetBrains AI",
		"CodeRabbit",
		"ChatGPT",
		"GPT-4",
		"Windsurf",
	}

	for _, name := range tools {
		escaped := regexp.QuoteMeta(name)
		pattern := regexp.MustCompile(`(?i)\b` + escaped + `\b`)
		toolPatterns = append(toolPatterns, struct {
			name    string
			pattern *regexp.Regexp
		}{name: name, pattern: pattern})
	}
}

type Detector struct{}

func (d *Detector) Name() string { return "toolmention" }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	text := input.Text
	if input.CommitMessage != "" {
		if text != "" {
			text = text + "\n" + input.CommitMessage
		} else {
			text = input.CommitMessage
		}
	}

	if strings.TrimSpace(text) == "" {
		return nil
	}

	var findings []detection.Finding
	seen := map[string]bool{}

	for _, tp := range toolPatterns {
		if seen[tp.name] {
			continue
		}
		if tp.pattern.MatchString(text) {
			findings = append(findings, detection.Finding{
				Detector:   d.Name(),
				Tool:       tp.name,
				Confidence: detection.ConfidenceLow,
				Detail:     fmt.Sprintf("text mentions %s", tp.name),
			})
			seen[tp.name] = true
		}
	}

	return findings
}
