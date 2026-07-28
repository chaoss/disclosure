package gitnotes

import (
	"fmt"

	"github.com/chaoss/disclosure/detection"
)

type Detector struct{}

func (d *Detector) Name() string { return "gitnotes" }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	parseResult, err := input.GetNotes()
	if err != nil {
		return []detection.Finding{}
	}

	seen := map[string]bool{}
	var findings []detection.Finding
	for _, prompt := range parseResult.Metadata.Prompts {
		tool := prompt.AgentID.Tool
		if tool == "" || seen[tool] {
			continue
		}
		seen[tool] = true

		detail := fmt.Sprintf("git-ai authorship log (refs/notes/ai) attributes code to %s", tool)
		if prompt.AgentID.Model != "" {
			detail += fmt.Sprintf(" (model: %s)", prompt.AgentID.Model)
		}
		if parseResult.AttributionFileCount > 0 {
			detail += fmt.Sprintf(", %d file(s) attributed", parseResult.AttributionFileCount)
		}

		findings = append(findings, detection.Finding{
			Detector:   d.Name(),
			Tool:       tool,
			Confidence: detection.ConfidenceHigh,
			Detail:     detail,
		})
	}

	return findings
}
