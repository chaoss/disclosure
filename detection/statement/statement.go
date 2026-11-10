// Package statement detects explicit statements of AI use, such as
// "Drafted with OpenAI Codex" or "AI disclosure: Claude assisted with tests",
// where a tool name appears inside wording that says the tool was used.
package statement

import (
	"regexp"
	"strings"

	"github.com/chaoss/disclosure/detection"
	"github.com/chaoss/disclosure/detection/toolmention"
)

var htmlComment = regexp.MustCompile(detection.HtmlCommentPattern)

type Detector struct {
	ConfidenceLevels map[detection.Confidence]float64
	// Mentions finds tool names in a line. Defaults to the built-in names.
	Mentions func(text string) []toolmention.Mention
}

func (d *Detector) Name() string { return "statement" }

func (d *Detector) GetConfidenceLevels() map[detection.Confidence]float64 { return d.ConfidenceLevels }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	mentions := d.Mentions
	if mentions == nil {
		mentions = toolmention.Mentions
	}
	// Template hints inside HTML comments are not statements by the author.
	text := htmlComment.ReplaceAllString(input.Text, "")

	var findings []detection.Finding
	seen := map[string]bool{}
	for line := range strings.SplitSeq(text, "\n") {
		for _, m := range mentions(line) {
			if seen[m.Tool] || !isDisclosed(line, m) {
				continue
			}
			seen[m.Tool] = true
			score := detection.StatementBaseScore
			findings = append(findings, detection.Finding{
				Detector:   d.Name(),
				Tool:       m.Tool,
				Score:      score,
				Confidence: detection.ScoreToConfidence(d.ConfidenceLevels, score),
				Detail:     "text states " + m.Tool + " was used",
			})
		}
	}
	return findings
}

func isDisclosed(line string, m toolmention.Mention) bool {
	before, after := line[:m.Start], line[m.End:]
	if loc := detection.StatementLeadingPattern.FindStringIndex(before); loc != nil &&
		!detection.StatementNegatedBeforePattern.MatchString(before[:loc[0]]) {
		return true
	}
	if detection.StatementTrailingPattern.MatchString(after) &&
		!detection.StatementNegatedBeforePattern.MatchString(before) {
		return true
	}
	if loc := detection.StatementLabelPattern.FindStringIndex(line); loc != nil &&
		!detection.StatementNegationPattern.MatchString(line[loc[1]:]) {
		return true
	}
	return false
}
