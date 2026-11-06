// Package branchname detects AI tool involvement from branch naming conventions,
// e.g. branches created by CLI agents such as "codex/fix-bug" or "claude/add-tests".
package branchname

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/chaoss/disclosure/detection"
)

// Merge commits keep the source branch name in their subject line, which is
// the only record of it once the branch is deleted.
var (
	githubMergeSubject = regexp.MustCompile(`^Merge pull request #\d+ from [^/\s]+/(\S+)`)
	gitMergeSubject    = regexp.MustCompile(`^Merge (?:remote-tracking )?branch '([^']+)'`)
)

type Detector struct {
	ConfidenceLevels map[detection.Confidence]float64
}

func (d *Detector) Name() string { return "branchname" }

func (d *Detector) GetConfidenceLevels() map[detection.Confidence]float64 { return d.ConfidenceLevels }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	if branch, err := input.GetBranchName(); err == nil {
		if f := d.match(branch, "branch name"); f != nil {
			return f
		}
	}
	if branch := mergedBranch(input.CommitMessage); branch != "" {
		return d.match(branch, "merged branch")
	}
	return nil
}

func (d *Detector) match(branch, source string) []detection.Finding {
	score := detection.BranchNameBaseScore
	confidence := detection.ScoreToConfidence(d.ConfidenceLevels, score)
	lower := strings.ToLower(branch)
	for prefix, tool := range detection.KnownAgentBranchPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return []detection.Finding{{
				Detector:   d.Name(),
				Tool:       tool,
				Score:      score,
				Confidence: confidence,
				Detail:     fmt.Sprintf("%s %q matches %s convention", source, branch, tool),
			}}
		}
	}
	return nil
}

func mergedBranch(message string) string {
	subject, _, _ := strings.Cut(strings.TrimSpace(message), "\n")
	for _, re := range []*regexp.Regexp{githubMergeSubject, gitMergeSubject} {
		if m := re.FindStringSubmatch(subject); m != nil {
			return m[1]
		}
	}
	return ""
}
