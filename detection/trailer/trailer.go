package trailer

import (
	"fmt"
	"log"
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

	text = strings.Join(strings.Fields(text), " ")
	text = strings.TrimSpace(strings.TrimRight(text, ".,;:!?"))
	if len(text) >= 2 && strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]") {
		text = strings.TrimSpace(text[1 : len(text)-1])
		text = strings.TrimSpace(strings.TrimRight(text, ".,;:!?"))
	}
	if text == "" {
		return "", fmt.Errorf("no contents found in text")
	}

	return cases.Title(language.English, cases.NoLower).String(text), nil
}

var commitMessagePatterns = []struct {
	check func(string) (float64, bool)
	name  string
}{
	{
		check: func(msg string) (float64, bool) {
			if strings.HasPrefix(strings.ToLower(msg), detection.AiderCommitPrefix) {
				return detection.TrailerMatchBaseScore, true
			}
			return detection.TrailerNotMatchedScore, false
		},
		name: "Aider",
	},
	{
		check: func(msg string) (float64, bool) {
			if strings.Contains(msg, detection.ClaudeAttributionText) {
				return detection.TrailerMatchBaseScore, true
			}
			return detection.TrailerNotMatchedScore, false
		},
		name: "Claude Code",
	},
	{
		check: func(msg string) (float64, bool) {
			matchedTrailerCount := 0
			for _, trailer := range detection.EntireIOTrailers {
				if strings.Contains(msg, fmt.Sprintf("\n%s:", trailer)) {
					matchedTrailerCount += 1
				}
			}
			if matchedTrailerCount > 0 {
				score := detection.TrailerMatchBaseScore + (float64(matchedTrailerCount-1))*detection.AdditionalTrailerBonusPoints
				return score, true
			}
			return detection.TrailerNotMatchedScore, false
		},
		name: "EntireIO",
	},
	{
		check: func(msg string) (float64, bool) {
			matchResult := detection.ReplitAttributionPattern.FindStringSubmatch(msg)
			if len(matchResult) == 0 {
				// replit not detected
				return detection.TrailerNotMatchedScore, false
			}

			var score float64 = 0
			switch matchResult[1] {
			case "Agent", "Assistant":
				score += detection.TrailerMatchBaseScore
			default:
				// unknown replit product, we cannot confirm ai use
				return detection.TrailerNotMatchedScore, false
			}

			// bonus points if commit session id also present
			if matchResult[2] != "" {
				score += detection.SessionIDBonusPoints
			}

			return score, true
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
		score := detection.CoauthoredByTrailerBaseScore

		if name, ok := detection.KnownCoAuthorEmails[email]; ok {
			model := extractCoauthorModel(name, namePart)
			if model != "" {
				score += detection.CoauthorModelBonusPoints
			}
			key := toolModelPair{tool: name, model: model}
			if seen[key] {
				continue
			}
			score += detection.CoauthorKnownEmailBonusPoints
			confidence, err := detection.ScoreToConfidence(score)
			if err != nil {
				log.Fatal(err)
			}
			findings = append(findings, detection.Finding{
				Detector:   d.Name(),
				Tool:       name,
				Model:      model,
				Score:      score,
				Confidence: confidence,
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
		if err != nil {
			continue
		}
		matchedToolKey := cases.Fold().String(matchedTool)
		if seen[matchedToolKey] {
			continue
		}

		score := detection.AssistedByTrailerBaseScore
		confidence, err := detection.ScoreToConfidence(score)
		if err != nil {
			log.Fatal(err)
		}

		findings = append(findings, detection.Finding{
			Detector:   d.Name(),
			Tool:       matchedTool,
			Score:      score,
			Confidence: confidence,
			Detail:     fmt.Sprintf("Assisted-By trailer with tool %s", matchedTool),
		})
		seen[matchedToolKey] = true
	}
	return findings
}

func (d *Detector) detectMessagePatterns(commitMessage string) []detection.Finding {
	var findings []detection.Finding
	for _, p := range commitMessagePatterns {
		if score, isDetected := p.check(commitMessage); isDetected {
			confidence, err := detection.ScoreToConfidence(score)
			if err != nil {
				log.Fatal(err)
			}
			findings = append(findings, detection.Finding{
				Detector:   d.Name(),
				Tool:       p.name,
				Score:      score,
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
