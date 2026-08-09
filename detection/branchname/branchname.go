// Package branchname detects AI tool involvement from branch naming conventions,
// e.g. branches created by CLI agents such as "codex/fix-bug" or "claude/add-tests".
package branchname

import (
	"fmt"
	"strings"

	"github.com/chaoss/disclosure/detection"
)

type Detector struct{}

func (d *Detector) Name() string { return "branchname" }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	branch, err := input.GetBranchName()
	if err != nil {
		return nil
	}

	lower := strings.ToLower(branch)
	for prefix, tool := range detection.KnownAgentBranchPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return []detection.Finding{{
				Detector:   d.Name(),
				Tool:       tool,
				Confidence: detection.ConfidenceMedium,
				Detail:     fmt.Sprintf("branch name %q matches %s convention", branch, tool),
			}}
		}
	}

	return nil
}
