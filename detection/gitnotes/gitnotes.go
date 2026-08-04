package gitnotes

import (
	"fmt"
	"sort"

	"github.com/chaoss/disclosure/detection"
)

type Detector struct{}

func (d *Detector) Name() string { return "gitnotes" }

type toolModelPair struct {
	tool  string
	model string
}

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	parseResult, err := input.GetNotes()
	if err != nil {
		return []detection.Finding{}
	}

	seen := map[toolModelPair]bool{}
	var findings []detection.Finding
	promptIDs := make([]string, 0, len(parseResult.Metadata.Prompts))
	for promptID := range parseResult.Metadata.Prompts {
		promptIDs = append(promptIDs, promptID)
	}
	sort.Strings(promptIDs)
	score := detection.GitNotesMatchBaseScore
	confidence, err := detection.ScoreToConfidence(score)
	if err != nil {
		confidence = detection.ConfidenceNone
	}

	for _, promptID := range promptIDs {
		prompt := parseResult.Metadata.Prompts[promptID]
		tool := prompt.AgentID.Tool
		model := prompt.AgentID.Model
		if tool == "" {
			continue
		}
		key := toolModelPair{tool: tool, model: model}
		if seen[key] {
			continue
		}
		seen[key] = true

		detail := fmt.Sprintf("git-ai authorship log (refs/notes/ai) attributes code to %s", tool)
		if model != "" {
			detail += fmt.Sprintf(" (model: %s)", model)
		}
		if parseResult.AttributionFileCount > 0 {
			detail += fmt.Sprintf(", %d file(s) attributed", parseResult.AttributionFileCount)
		}

		findings = append(findings, detection.Finding{
			Detector:   d.Name(),
			Tool:       tool,
			Model:      model,
			Score:      score,
			Confidence: confidence,
			Detail:     detail,
		})
	}

	return findings
}
