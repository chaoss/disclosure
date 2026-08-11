package toolmention

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

type Detector struct {
	ConfidenceLevels       map[detection.Confidence]float64
	CheckboxAIUsedRegex    *regexp.Regexp
	CheckboxAINotUsedRegex *regexp.Regexp
	initOnce               sync.Once
}

func (d *Detector) initCheckboxLabels() {
	d.initOnce.Do(func() {
		d.SetCheckboxAILabels(
			detection.CheckboxAIUsedLabel,
			detection.CheckboxAINotUsedLabel,
		)
	})
}

func (d *Detector) Name() string { return "toolmention" }

func (d *Detector) GetConfidenceLevels() map[detection.Confidence]float64 { return d.ConfidenceLevels }

type toolMatch struct {
	start int
	end   int
	name  string
}

func newCheckboxRegex(label string) *regexp.Regexp {
	return regexp.MustCompile(
		`(?im)^[ \t]*\[[ \t]*x[ \t]*\][ \t]*` +
			regexp.QuoteMeta(label) +
			`[ \t]*$`,
	)
}
func (d *Detector) setCheckboxAIUsed(label string) {
	if d.CheckboxAIUsedRegex != nil {
		return
	}
	if label == "" {
		label = detection.CheckboxAIUsedLabel
	}
	d.CheckboxAIUsedRegex = newCheckboxRegex(label)
}

func (d *Detector) setCheckboxAINotUsed(label string) {
	if d.CheckboxAINotUsedRegex != nil {
		return
	}
	if label == "" {
		label = detection.CheckboxAINotUsedLabel
	}
	d.CheckboxAINotUsedRegex = newCheckboxRegex(label)
}

func (d *Detector) appendFinding(findings *[]detection.Finding, toolName string, score float64, detail string) {
	*findings = append(*findings, detection.Finding{
		Detector:   d.Name(),
		Tool:       toolName,
		Score:      score,
		Confidence: detection.ScoreToConfidence(d.ConfidenceLevels, score),
		Detail:     detail,
	})
}

func (d *Detector) SetConfidenceLevels(confidenceLevels map[detection.Confidence]float64) {
	d.ConfidenceLevels = confidenceLevels
}

// SetCheckboxAILabels configures the checkbox labels.
// Call this before the first Detect call to use custom labels,
// otherwise Detect initializes the default labels.
func (d *Detector) SetCheckboxAILabels(aiUsedLabel, aiNotUsedLabel string) {
	d.setCheckboxAIUsed(aiUsedLabel)
	d.setCheckboxAINotUsed(aiNotUsedLabel)
}

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	d.initCheckboxLabels()

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

	aiUsed := d.CheckboxAIUsedRegex.MatchString(text)
	aiNotUsed := d.CheckboxAINotUsedRegex.MatchString(text)
	isAIUseConfirmed := aiUsed && !aiNotUsed

	detail := ""
	if isAIUseConfirmed {
		detail = "checkbox confirms AI was used and "
	}

	findings := make([]detection.Finding, 0, len(toolMatches)+1)

	// tool mentions and checkbox
	if len(toolMatches) > 0 {
		score := detection.ToolMentionBaseScore
		if isAIUseConfirmed {
			score += detection.CheckboxAIUsedBaseScore
		}
		for _, match := range toolMatches {
			d.appendFinding(&findings, match.name, score, detail+"text mentions "+match.name)
		}
		return findings
	}

	// checkbox only
	if isAIUseConfirmed {
		score := detection.CheckboxAIUsedBaseScore
		d.appendFinding(&findings, "", score, detail+"text does not mention any specific tool")
	}
	return findings
}
