package trailer

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/chaoss/disclosure/detection"
)

var toolLineReplacePattern = regexp.MustCompile(`\s*<[^>]+>`)

func extractToolsFromText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	text = toolLineReplacePattern.ReplaceAllString(text, "")
	parts := strings.Split(text, "\n")
	var tools []string
	for _, p := range parts {
		p = strings.TrimSpace(strings.Split(strings.TrimSpace(p), "(")[0])
		if p == "" {
			continue
		}
		words := strings.Fields(p)
		for i, w := range words {
			if len(w) > 0 {
				words[i] = strings.ToUpper(w[:1]) + w[1:]
			}
		}
		tools = append(tools, strings.Join(words, " "))
	}
	return tools
}

var commitMessagePatterns = []struct {
	check func(string) (detection.Confidence, bool)
	name  string
}{
	{
		check: func(msg string) (detection.Confidence, bool) {
			return detection.ConfidenceMedium, strings.HasPrefix(strings.ToLower(msg), detection.AiderCommitPrefix)
		},
		name: "Aider",
	},
	{
		check: func(msg string) (detection.Confidence, bool) {
			return detection.ConfidenceMedium, strings.Contains(msg, detection.ClaudeAttributionText)
		},
		name: "Claude Code",
	},
	{
		check: func(msg string) (detection.Confidence, bool) {
			for _, trailer := range detection.EntireIOTrailers {
				if strings.Contains(msg, fmt.Sprintf("\n%s:", trailer)) {
					return detection.ConfidenceMedium, true
				}
			}
			return detection.ConfidenceMedium, false
		},
		name: "EntireIO",
	},
	{
		check: func(msg string) (detection.Confidence, bool) {
			trailerRegex := regexp.MustCompile(detection.ReplitAttributionRegex)

			matchResult := trailerRegex.FindStringSubmatch(msg)
			if len(matchResult) == 0 {
				// replit not detected
				return detection.ConfidenceMedium, false
			}

			var confidence detection.Confidence
			switch matchResult[1] {
			case "Agent":
				confidence = detection.ConfidenceMedium
			case "Assistant":
				confidence = detection.ConfidenceLow
			default:
				// unknown replit product, we cannot confirm ai use
				return detection.ConfidenceLow, false
			}

			// if commit session id also present, increase confidence
			if matchResult[2] != "" {
				confidence.Increment()
			}

			return confidence, true
		},
		name: "Replit",
	},
}

type Detector struct{}

func (d *Detector) Name() string { return "trailer" }

func (d *Detector) detectTrailerCoauthoredBy(input detection.Input) []detection.Finding {
	var findings []detection.Finding

	matches := detection.CoAuthorPattern.FindAllStringSubmatch(input.CommitMessage, -1)
	if len(matches) == 0 {
		return findings
	}

	seen := map[string]bool{}
	for _, match := range matches {
		email := strings.ToLower(strings.TrimSpace(match[1]))

		if name, ok := detection.KnownCoAuthorEmails[email]; ok && !seen[name] {
			findings = append(findings, detection.Finding{
				Detector:   d.Name(),
				Tool:       name,
				Confidence: detection.ConfidenceHigh,
				Detail:     fmt.Sprintf("Co-Authored-By trailer with email %s", email),
			})
			seen[name] = true
		}
	}

	return findings
}

func (d *Detector) detectTrailerAssistedBy(input detection.Input) []detection.Finding {
	var findings []detection.Finding

	matches := detection.AssistedByPattern.FindAllStringSubmatch(
		input.CommitMessage,
		-1,
	)
	if len(matches) == 0 {
		return findings
	}

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		for _, matchedTool := range extractToolsFromText(match[1]) {
			findings = append(findings, detection.Finding{
				Detector:   d.Name(),
				Tool:       matchedTool,
				Confidence: detection.ConfidenceHigh,
				Detail:     fmt.Sprintf("Assisted-By trailer with tool %s", matchedTool),
			})
		}
	}
	return findings
}

func (d *Detector) detectCustomTrailers(input detection.Input) []detection.Finding {
	var findings []detection.Finding
	for _, p := range commitMessagePatterns {
		if confidence, isDetected := p.check(input.CommitMessage); isDetected {
			findings = append(findings, detection.Finding{
				Detector:   d.Name(),
				Tool:       p.name,
				Confidence: confidence,
				Detail:     fmt.Sprintf("commit message matches %s pattern", p.name),
			})
		}
	}
	return findings
}

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	if input.CommitMessage == "" {
		return nil
	}

	var findings []detection.Finding

	findings = slices.Concat(
		// add findings for co-authored-by
		d.detectTrailerCoauthoredBy(input),

		// add findings for assisted-by
		d.detectTrailerAssistedBy(input),

		// add findings for other custom trailers
		d.detectCustomTrailers(input),
	)

	return findings
}
