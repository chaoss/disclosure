package message

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/chaoss/ai-detection-action/detection"
)

var commitMessagePatterns = []struct {
	check func(string) (detection.Confidence, bool)
	name  string
}{
	{
		check: func(msg string) (detection.Confidence, bool) {
			return detection.ConfidenceMedium, strings.HasPrefix(strings.ToLower(msg), "aider:")
		},
		name: "Aider",
	},
	{
		check: func(msg string) (detection.Confidence, bool) {
			return detection.ConfidenceMedium, strings.Contains(msg, "Generated with Claude Code")
		},
		name: "Claude Code",
	},
	{
		check: func(msg string) (detection.Confidence, bool) {
			trailers := []string{
				"Entire-Metadata",
				"Entire-Metadata-Task",
				"Entire-Strategy",
				"Entire-Session",
				"Entire-Condensation",
				"Entire-Source-Ref",
				"Entire-Checkpoint",
				"Entire-Agent",
			}
			for _, trailer := range trailers {
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
			trailerRegex := regexp.MustCompile(`(?m)^Replit-Commit-Author:\s*(Agent|Assistant)(?:\r?\nReplit-Commit-Session-Id:\s*([a-fA-F0-9-]+))?(?:\r?\n|$)`)

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

func (d *Detector) Name() string { return "message" }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	if input.CommitMessage == "" {
		return nil
	}

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
