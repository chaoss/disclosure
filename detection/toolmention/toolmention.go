package toolmention

//go:generate go run generate.go

import (
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/chaoss/disclosure/detection"
)

type toolPattern struct {
	name    string
	pattern *regexp.Regexp
}

type toolMatch struct {
	name  string
	start int
	end   int
}

var baseToolPatterns = compileToolPatterns(detection.SupportedToolMentions(false))

var generatedToolPatterns struct {
	once     sync.Once
	patterns []toolPattern
}

func compileToolPatterns(names []string) []toolPattern {
	sort.SliceStable(names, func(i, j int) bool {
		if len(names[i]) != len(names[j]) {
			return len(names[i]) > len(names[j])
		}
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})

	patterns := make([]toolPattern, 0, len(names))
	for _, name := range names {
		patterns = append(patterns, toolPattern{
			name:    name,
			pattern: regexp.MustCompile(toolMentionPattern(name)),
		})
	}
	return patterns
}

func toolMentionPattern(name string) string {
	const separator = `[\s_-]+`

	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == ' ' || r == '-'
	})
	for i := range parts {
		parts[i] = regexp.QuoteMeta(parts[i])
	}

	pattern := `(?i)\b` + strings.Join(parts, separator)
	last, _ := utf8.DecodeLastRuneInString(name)
	if unicode.IsLetter(last) || unicode.IsDigit(last) || last == '_' {
		return pattern + `\b`
	}
	return pattern + `(?:$|[^A-Za-z0-9_])`
}

func patterns(includeModelCatalog bool) []toolPattern {
	if !includeModelCatalog {
		return baseToolPatterns
	}

	generatedToolPatterns.once.Do(func() {
		generatedToolPatterns.patterns = compileToolPatterns(detection.SupportedToolMentions(true))
	})
	return generatedToolPatterns.patterns
}

func overlaps(candidate toolMatch, accepted []toolMatch) bool {
	for _, match := range accepted {
		if candidate.start < match.end && match.start < candidate.end {
			return true
		}
	}
	return false
}

type Detector struct {
	IncludeModelCatalog bool
}

func (d *Detector) Name() string { return "toolmention" }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	text, err := input.GetTextWithCommitMessage()
	if err != nil {
		return []detection.Finding{}
	}

	var candidates []toolMatch
	for _, tp := range patterns(d.IncludeModelCatalog) {
		for _, loc := range tp.pattern.FindAllStringIndex(text, -1) {
			candidates = append(candidates, toolMatch{name: tp.name, start: loc[0], end: loc[1]})
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].end-candidates[i].start != candidates[j].end-candidates[j].start {
			return candidates[i].end-candidates[i].start > candidates[j].end-candidates[j].start
		}
		if candidates[i].start != candidates[j].start {
			return candidates[i].start < candidates[j].start
		}
		return strings.ToLower(candidates[i].name) < strings.ToLower(candidates[j].name)
	})

	var accepted []toolMatch
	seen := map[string]bool{}
	for _, candidate := range candidates {
		seenKey := strings.ToLower(candidate.name)
		if seen[seenKey] || overlaps(candidate, accepted) {
			continue
		}
		accepted = append(accepted, candidate)
		seen[seenKey] = true
	}

	sort.SliceStable(accepted, func(i, j int) bool {
		return accepted[i].start < accepted[j].start
	})

	findings := make([]detection.Finding, 0, len(accepted))
	for _, match := range accepted {
		findings = append(findings, detection.Finding{
			Detector:   d.Name(),
			Tool:       match.name,
			Confidence: detection.ConfidenceLow,
			Detail:     "text mentions " + match.name,
		})
	}

	return findings
}
