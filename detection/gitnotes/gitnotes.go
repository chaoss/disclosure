package gitnotes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chaoss/ai-detection-action/detection"
)

// metadata represents the JSON metadata section of a git-ai authorship log.
type metadata struct {
	SchemaVersion string                    `json:"schema_version"`
	Prompts       map[string]promptRecord   `json:"prompts"`
}

type promptRecord struct {
	AgentID agentID `json:"agent_id"`
}

type agentID struct {
	Tool  string `json:"tool"`
	Model string `json:"model"`
}

type Detector struct{}

func (d *Detector) Name() string { return "gitnotes" }

func (d *Detector) Detect(input detection.Input) []detection.Finding {
	if input.Notes == "" {
		return nil
	}

	parts := strings.SplitN(input.Notes, "\n---\n", 2)
	if len(parts) != 2 {
		return nil
	}

	attestation := parts[0]
	jsonSection := parts[1]

	var meta metadata
	if err := json.Unmarshal([]byte(jsonSection), &meta); err != nil {
		return nil
	}

	if !strings.HasPrefix(meta.SchemaVersion, "authorship/") {
		return nil
	}

	// Count attributed files from the attestation section
	fileCount := 0
	for _, line := range strings.Split(attestation, "\n") {
		if line == "" {
			continue
		}
		// File paths start at column 0, attestation entries are indented
		if !strings.HasPrefix(line, " ") {
			fileCount++
		}
	}

	seen := map[string]bool{}
	var findings []detection.Finding

	for _, prompt := range meta.Prompts {
		tool := prompt.AgentID.Tool
		if tool == "" || seen[tool] {
			continue
		}
		seen[tool] = true

		detail := fmt.Sprintf("git-ai authorship log (refs/notes/ai) attributes code to %s", tool)
		if prompt.AgentID.Model != "" {
			detail += fmt.Sprintf(" (model: %s)", prompt.AgentID.Model)
		}
		if fileCount > 0 {
			detail += fmt.Sprintf(", %d file(s) attributed", fileCount)
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
