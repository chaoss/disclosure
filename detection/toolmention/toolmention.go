package toolmention

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/chaoss/disclosure/detection"
)

type toolPattern struct {
	name    string
	pattern *regexp.Regexp
}

var toolPatterns []toolPattern

func init() {
	const separator = `[\s_-]+`
	names := append([]string(nil), detection.SupportedToolsInMentions...)
	for _, name := range names {
		parts := strings.FieldsFunc(name, func(r rune) bool {
			return r == ' ' || r == '-'
		})
		for i := range parts {
			parts[i] = regexp.QuoteMeta(parts[i])
		}
		pattern := `(?i)\b` + strings.Join(parts, separator)
		last, _ := utf8.DecodeLastRuneInString(name)
		if unicode.IsLetter(last) || unicode.IsDigit(last) || last == '_' {
			pattern += `\b`
		} else {
			pattern += `(?:$|[^A-Za-z0-9_])`
		}
		toolPatterns = append(toolPatterns, toolPattern{
			name:    name,
			pattern: regexp.MustCompile(pattern),
		})
	}
}

type Detector struct{}

func (d *Detector) Name() string { return "toolmention" }

type toolMatch struct {
	start int
	end   int
	name  string
}

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	text, err := input.GetTextWithCommitMessage()
	if err != nil {
		return []detection.Finding{}
	}

	matches := make([]toolMatch, 0, len(toolPatterns))
	for _, tp := range toolPatterns {
		for _, loc := range tp.pattern.FindAllStringIndex(text, -1) {
			matches = append(matches, toolMatch{
				start: loc[0],
				end:   loc[1],
				name:  tp.name,
			})
		}
	}

	// Give priority to longer matches when there's an overlap.
	// e.g. Claude Code would be preferred over Claude
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].start != matches[j].start {
			return matches[i].start < matches[j].start
		}
		return matches[i].end-matches[i].start > matches[j].end-matches[j].start
	})

	toolMatches := make([]toolMatch, 0, len(matches))
	seen := make(map[string]struct{}, len(toolPatterns))
	lastEnd := -1
	for _, match := range matches {
		if match.start < lastEnd {
			continue
		}
		if _, ok := seen[match.name]; ok {
			continue
		}
		toolMatches = append(toolMatches, match)
		seen[match.name] = struct{}{}
		lastEnd = match.end
	}

	score := detection.ToolMentionBaseScore
	confidence, err := detection.ScoreToConfidence(score)
	if err != nil {
		log.Fatal(err)
	}

	findings := make([]detection.Finding, 0, len(toolMatches))
	for _, match := range toolMatches {
		findings = append(findings, detection.Finding{
			Detector:   d.Name(),
			Tool:       match.name,
			Confidence: detection.ConfidenceLow,
			Detail:     "text mentions " + match.name,
		})
	}
	return findings
}
