package branchname

import (
	"fmt"
	"strings"

	"github.com/chaoss/disclosure/detection"
)

// branchPrefixPatterns maps branch name prefixes to AI tools. Matching is
// case-insensitive on the branch name.
var branchPrefixPatterns = []struct {
	prefix     string
	tool       string
	confidence detection.Confidence
}{
	{prefix: "codex/", tool: "OpenAI Codex", confidence: detection.ConfidenceMedium},
}

type Detector struct{}

func (d *Detector) Name() string { return "branchname" }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	branch := strings.TrimSpace(input.BranchName)
	if branch == "" {
		return nil
	}

	lowerBranch := strings.ToLower(branch)
	var findings []detection.Finding

	for _, p := range branchPrefixPatterns {
		if strings.HasPrefix(lowerBranch, strings.ToLower(p.prefix)) {
			findings = append(findings, detection.Finding{
				Detector:   d.Name(),
				Tool:       p.tool,
				Confidence: p.confidence,
				Detail:     fmt.Sprintf("branch name %q matches %s prefix", branch, p.prefix),
			})
		}
	}

	return findings
}
