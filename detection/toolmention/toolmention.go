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
	replaceChars := `[\s_-]`
	for _, name := range detection.SupportedToolsInMentions {
		escaped := regexp.QuoteMeta(name)
		escaped = strings.ReplaceAll(escaped, "-", replaceChars)
		escaped = strings.ReplaceAll(escaped, " ", replaceChars)
		trailingBoundary := `(?:\z|\s|[\.,!?;:)\]"'])`
		if match, _ := regexp.MatchString(`\W$`, name); match {
			trailingBoundary = `(?:\b|\z|\s|[\.,!?;:)\]"'])`
		}
		pattern := regexp.MustCompile(`(?i)\b` + escaped + trailingBoundary)
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
