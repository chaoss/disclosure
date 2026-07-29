package toolmention

import (
	"fmt"
	"regexp"

	"github.com/chaoss/disclosure/detection"
)

// toolPatterns maps AI tool names to compiled word-boundary regexes.
var toolPatterns []struct {
	name    string
	pattern *regexp.Regexp
}

func init() {
	for _, name := range detection.SupportedToolsInMentions {
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
	text, err := input.GetTextWithCommitMessage()
	if err != nil {
		return []detection.Finding{}
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
