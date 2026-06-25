package assistedby

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/chaoss/disclosure/detection"
)

var assistedByPattern = regexp.MustCompile(`(?im)^assisted-by\s*:\s*([^\r\n]+?)\s*$`)
var toolLineReplacePattern = regexp.MustCompile(`\s*<[^>]+>`)

func getMatchedTools(toolLine string) []string {
	toolLine = strings.TrimSpace(toolLine)
	if toolLine == "" {
		return nil
	}
	toolLine = toolLineReplacePattern.ReplaceAllString(toolLine, "")
	parts := strings.Split(toolLine, "\n")
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

type Detector struct{}

func (d *Detector) Name() string { return "assistedby" }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	if input.CommitMessage == "" {
		return nil
	}

	matches := assistedByPattern.FindAllStringSubmatch(
		input.CommitMessage,
		-1,
	)

	if len(matches) == 0 {
		return nil
	}

	var findings []detection.Finding

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		for _, matchedTool := range getMatchedTools(match[1]) {
			findings = append(findings, detection.Finding{
				Detector:   d.Name(),
				Tool:       matchedTool,
				Confidence: detection.ConfidenceHigh,
				Detail: fmt.Sprintf(
					"Assisted-By trailer with tool %s",
					matchedTool,
				),
			})
		}
	}

	return findings
}
