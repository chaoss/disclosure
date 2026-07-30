package trailer

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/chaoss/disclosure/detection"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func extractToolFromText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("empty text found")
	}

	// removing the email part e.g. Aider <noreply@aider.chat>
	text = strings.TrimSpace(detection.TrailerEmailPattern.ReplaceAllString(text, ""))

	// trimming any values in brackets e.g. Claude (Anthropic)
	text, _, _ = strings.Cut(text, "(")

	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("no contents found in text")
	}

	return cases.Title(language.English, cases.NoLower).String(text), nil
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

type toolModelPair struct {
	tool  string
	model string
}

func extractParenthesizedModel(text string) string {
	start := strings.Index(text, "(")
	if start < 0 {
		return ""
	}

	end := strings.Index(text[start:], ")")
	if end <= 0 {
		return ""
	}

	return strings.TrimSpace(text[start+1 : start+end])
}

func extractCoauthorModel(tool, namePart string) string {
	namePart = strings.TrimSpace(namePart)
	if model := extractParenthesizedModel(namePart); model != "" {
		return model
	}

	switch tool {
	case "Claude Code":
		if strings.EqualFold(namePart, "Claude") || strings.EqualFold(namePart, "Claude Code") {
			return ""
		}
		namePartLower := strings.ToLower(namePart)
		if strings.HasPrefix(namePartLower, "claude code ") {
			return strings.TrimSpace(namePart[len("Claude Code "):])
		}
		if strings.HasPrefix(namePartLower, "claude ") {
			return strings.TrimSpace(namePart[len("Claude "):])
		}
		return namePart
	}

	return ""
}

func (d *Detector) detectTrailerCoauthoredBy(commitMessage string) []detection.Finding {
	var findings []detection.Finding

	matches := detection.CoAuthorPattern.FindAllStringSubmatch(commitMessage, -1)
	if len(matches) == 0 {
		return findings
	}

	seen := map[toolModelPair]bool{}
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		namePart := strings.TrimSpace(match[1])
		email := strings.ToLower(strings.TrimSpace(match[2]))

		if name, ok := detection.KnownCoAuthorEmails[email]; ok {
			model := extractCoauthorModel(name, namePart)
			key := toolModelPair{tool: name, model: model}
			if seen[key] {
				continue
			}
			findings = append(findings, detection.Finding{
				Detector:   d.Name(),
				Tool:       name,
				Model:      model,
				Confidence: detection.ConfidenceHigh,
				Detail:     fmt.Sprintf("Co-Authored-By trailer with email %s", email),
			})
			seen[key] = true
		}
	}

	return findings
}

func (d *Detector) detectTrailerAssistedBy(commitMessage string) []detection.Finding {
	var findings []detection.Finding

	matches := detection.AssistedByPattern.FindAllStringSubmatch(commitMessage, -1)
	if len(matches) == 0 {
		return findings
	}

	seen := map[string]bool{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		matchedTool, err := extractToolFromText(match[1])
		if err != nil || seen[matchedTool] {
			continue
		}

		findings = append(findings, detection.Finding{
			Detector:   d.Name(),
			Tool:       matchedTool,
			Confidence: detection.ConfidenceHigh,
			Detail:     fmt.Sprintf("Assisted-By trailer with tool %s", matchedTool),
		})
		seen[matchedTool] = true
	}
	return findings
}

func (d *Detector) detectMessagePatterns(commitMessage string) []detection.Finding {
	var findings []detection.Finding
	for _, p := range commitMessagePatterns {
		if confidence, isDetected := p.check(commitMessage); isDetected {
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
	commitMessage, err := input.GetCommitMessage()
	if err != nil {
		return []detection.Finding{}
	}

	return slices.Concat(
		// add findings for co-authored-by
		d.detectTrailerCoauthoredBy(commitMessage),

		// add findings for assisted-by
		d.detectTrailerAssistedBy(commitMessage),

		// add findings for other custom trailers
		d.detectMessagePatterns(commitMessage),
	)
}
