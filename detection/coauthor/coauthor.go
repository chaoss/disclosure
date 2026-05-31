package coauthor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/chaoss/disclosure/detection"
)

var knownCoAuthorEmails = map[string]string{
	"noreply@anthropic.com":  "Claude Code",
	"cursoragent@cursor.com": "Cursor",
	"noreply@aider.chat":     "Aider",
}

var coAuthorPattern = regexp.MustCompile(`(?im)^co-authored-by:\s*([^<]*)<([^>]+)>`)

type Detector struct{}

func (d *Detector) Name() string { return "coauthor" }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	if input.CommitMessage == "" {
		return nil
	}

	matches := coAuthorPattern.FindAllStringSubmatch(input.CommitMessage, -1)
	var findings []detection.Finding
	seen := map[string]bool{}

	for _, match := range matches {
		namePart := strings.TrimSpace(match[1])
		email := strings.ToLower(strings.TrimSpace(match[2]))
		if name, ok := knownCoAuthorEmails[email]; ok && !seen[name] {
			model := ""
			if name == "Aider" && strings.Contains(namePart, "(") {
				start := strings.Index(namePart, "(")
				end := strings.Index(namePart, ")")
				if end > start {
					model = namePart[start+1 : end]
				}
			} else if name == "Claude Code" {
				model = namePart
				if model == "Claude" {
					model = ""
				}
			}

			findings = append(findings, detection.Finding{
				Detector:   d.Name(),
				Tool:       name,
				Model:      model,
				Confidence: detection.ConfidenceHigh,
				Detail:     fmt.Sprintf("Co-Authored-By trailer with email %s", email),
			})
			seen[name] = true
		}
	}

	return findings
}
