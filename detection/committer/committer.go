package committer

import (
	"fmt"
	"strings"

	"github.com/chaoss/disclosure/detection"
)

// numericPrefixIndex maps the numeric prefix from GitHub noreply emails to tool names.
// This handles issue #4: when a bot's username changes, the numeric ID stays the same.
var numericPrefixIndex map[string]string

func init() {
	numericPrefixIndex = make(map[string]string, len(detection.KnownAgentCommitters))
	for email, name := range detection.KnownAgentCommitters {
		if idx := strings.Index(email, "+"); idx > 0 {
			numericPrefixIndex[email[:idx]] = name
		}
	}
}

type Detector struct {
	ConfidenceLevels map[detection.Confidence]float64
}

func (d *Detector) Name() string { return "committer" }

func (d *Detector) GetConfidenceLevels() map[detection.Confidence]float64 { return d.ConfidenceLevels }

func (d *Detector) detectEmail(email, identityField string) []detection.Finding {
	// Direct match against known emails
	if name, ok := detection.KnownAgentCommitters[email]; ok {
		score := detection.CommitterMatchBaseScore + detection.CommitterKnownEmailBonusPoints
		confidence := detection.ScoreToConfidence(d.ConfidenceLevels, score)
		return []detection.Finding{{
			Detector:   d.Name(),
			Tool:       name,
			Score:      score,
			Confidence: confidence,
			Detail:     fmt.Sprintf("%s email %s matches known AI bot", identityField, email),
		}}
	}

	// Numeric prefix match for GitHub noreply emails (#4).
	// Format: <numeric-id>+<username>@users.noreply.github.com
	if strings.HasSuffix(email, detection.GithubNoReplyEmailSuffix) {
		score := detection.CommitterMatchBaseScore + detection.CommitterEmailSuffixBonusPoints
		confidence := detection.ScoreToConfidence(d.ConfidenceLevels, score)
		if idx := strings.Index(email, "+"); idx > 0 {
			prefix := email[:idx]
			if name, ok := numericPrefixIndex[prefix]; ok {
				return []detection.Finding{{
					Detector:   d.Name(),
					Tool:       name,
					Score:      score,
					Confidence: confidence,
					Detail:     fmt.Sprintf("%s email %s matches known AI bot", identityField, email),
				}}
			}
		}
	}

	return nil
}

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	authorEmail, _ := input.GetAuthorEmail()
	committerEmail, _ := input.GetCommitEmail()

	if authorEmail != "" && authorEmail == committerEmail {
		return d.detectEmail(authorEmail, "author and committer")
	}

	findings := d.detectEmail(authorEmail, "author")
	return append(findings, d.detectEmail(committerEmail, "committer")...)
}
