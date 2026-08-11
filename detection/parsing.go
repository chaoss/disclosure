package detection

import (
	"encoding/json"
	"fmt"
	"strings"
)

type GitnoteParseResult struct {
	Metadata             metadata `json:"metadata"`
	AttributionFileCount int      `json:"attribution_file_count"`
}

// metadata represents the JSON metadata section of a git-ai authorship log.
type metadata struct {
	SchemaVersion string                  `json:"schema_version"`
	Prompts       map[string]promptRecord `json:"prompts"`
}

type promptRecord struct {
	AgentID agentID `json:"agent_id"`
}

type agentID struct {
	Tool  string `json:"tool"`
	Model string `json:"model"`
}

var (
	fieldAuthorEmail   = "author_email"
	fieldCommitEmail   = "commit_email"
	fieldCommitMessage = "commit_message"
	fieldBranchName    = "branch_name"
)

func getCommitField(value string, field string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return "", fmt.Errorf("%s is empty", field)
	}

	switch field {
	case fieldAuthorEmail, fieldCommitEmail:
		return strings.ToLower(value), nil

	case fieldCommitMessage:
		return value, nil

	case fieldBranchName:
		return value, nil

	default:
		return "", fmt.Errorf("unsupported field: %s", field)
	}
}

func parseGitnotes(notes string) (GitnoteParseResult, error) {
	if notes == "" {
		return GitnoteParseResult{}, fmt.Errorf("no notes found in input")
	}

	parts := strings.SplitN(notes, "\n---\n", 2)
	if len(parts) != 2 {
		return GitnoteParseResult{}, fmt.Errorf("missing parts in notes, unable to parse")
	}

	attestation := parts[0]
	jsonSection := parts[1]

	var meta metadata
	if err := json.Unmarshal([]byte(jsonSection), &meta); err != nil {
		return GitnoteParseResult{}, fmt.Errorf("unable to unmarshal notes json")
	}

	if !strings.HasPrefix(meta.SchemaVersion, GitNotesAuthorshipPrefix) {
		return GitnoteParseResult{}, fmt.Errorf("authorship prefix missing in schema version")
	}

	// Count attributed files from the attestation section
	fileCount := 0
	for line := range strings.SplitSeq(attestation, "\n") {
		if line == "" {
			continue
		}
		// File paths start at column 0, attestation entries are indented
		if !strings.HasPrefix(line, " ") {
			fileCount++
		}
	}

	return GitnoteParseResult{
		Metadata:             meta,
		AttributionFileCount: fileCount,
	}, nil
}
