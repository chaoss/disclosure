package detection

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// Confidence represents how confident we are that a finding indicates AI involvement.
type Confidence int

const (
	ConfidenceNone   Confidence = 0 // Nil equivalent for confidence
	ConfidenceLow    Confidence = 1 // Tool name mentioned in text
	ConfidenceMedium Confidence = 2 // Commit message pattern match
	ConfidenceHigh   Confidence = 3 // Bot email, co-author trailer, git AI ref
)

func (c Confidence) String() string {
	switch c {
	case ConfidenceNone:
		return "none"
	case ConfidenceLow:
		return "low"
	case ConfidenceMedium:
		return "medium"
	case ConfidenceHigh:
		return "high"
	default:
		return "unknown"
	}
}

func (c Confidence) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func (c *Confidence) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "none":
		*c = ConfidenceNone
	case "low":
		*c = ConfidenceLow
	case "medium":
		*c = ConfidenceMedium
	case "high":
		*c = ConfidenceHigh
	default:
		return fmt.Errorf("invalid confidence %q", s)
	}

	return nil
}

func (c *Confidence) Increment() {
	*c = min(*c+1, ConfidenceHigh)
}

func GetDefaultConfidenceLevels() map[Confidence]float64 {
	return map[Confidence]float64{
		ConfidenceLow:    30.0,
		ConfidenceMedium: 70.0,
		ConfidenceHigh:   100.0,
	}
}

func ScoreToConfidence(confidenceLevels map[Confidence]float64, score float64) Confidence {
	if score >= confidenceLevels[ConfidenceHigh] {
		return ConfidenceHigh
	}
	if score <= confidenceLevels[ConfidenceLow] {
		return ConfidenceLow
	}
	levels := []Confidence{ConfidenceLow, ConfidenceMedium, ConfidenceHigh}
	for _, level := range levels {
		if math.Round(score) <= confidenceLevels[level] {
			return level
		}
	}
	return ConfidenceNone
}

// SetConfidenceLevelsFromStrings allows to update confidenceLevels using a custom map
func SetConfidenceLevelsFromStrings(
	confidenceLevels map[Confidence]float64,
	userMapping map[string]float64,
) (map[Confidence]float64, error) {
	for k, v := range userMapping {
		k = strings.ToLower(strings.TrimSpace(k))
		switch k {
		case "low":
			confidenceLevels[ConfidenceLow] = v
		case "medium":
			confidenceLevels[ConfidenceMedium] = v
		case "high":
			confidenceLevels[ConfidenceHigh] = v
		default:
			return nil, fmt.Errorf("unsupported confidence key: %s", k)
		}
	}
	// set defaults if unspecified in user mapping
	for c, def := range GetDefaultConfidenceLevels() {
		if _, ok := confidenceLevels[c]; !ok {
			confidenceLevels[c] = def
		}
	}

	if confidenceLevels[ConfidenceLow] >= confidenceLevels[ConfidenceMedium] ||
		confidenceLevels[ConfidenceMedium] >= confidenceLevels[ConfidenceHigh] {
		return nil, fmt.Errorf("low, medium, high must be in ascending order")
	}
	return confidenceLevels, nil
}

// ConfidenceFromString parses a confidence string or numeric value.
func ConfidenceFromString(s string) (Confidence, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "low":
		return ConfidenceLow, nil
	case "2", "medium":
		return ConfidenceMedium, nil
	case "3", "high":
		return ConfidenceHigh, nil
	default:
		return 0, fmt.Errorf("invalid confidence %q: use low/1, medium/2, or high/3", s)
	}
}

// Finding represents a single detection of AI involvement.
type Finding struct {
	Detector   string     `json:"detector"`
	Tool       string     `json:"tool"`
	Model      string     `json:"model,omitempty"`
	Confidence Confidence `json:"confidence"`
	Score      float64    `json:"score,omitempty"`
	Detail     string     `json:"detail"`
}

func (f Finding) DisplayTool() string {
	tool := strings.TrimSpace(f.Tool)
	model := strings.TrimSpace(f.Model)

	switch {
	case tool == "":
		return model
	case model == "":
		return tool
	default:
		return fmt.Sprintf("%s [%s]", tool, model)
	}
}

// Detector is the interface that all detection strategies implement.
type Detector interface {
	Name() string
	Detect(input Input) []Finding
	GetConfidenceLevels() map[Confidence]float64
}

// Input provides data for detectors to examine. Each detector reads the fields
// it cares about and ignores the rest.
type Input struct {
	CommitHash    string
	AuthorEmail   string
	CommitEmail   string // CommitEmail is the committer email.
	CommitMessage string
	Notes         string // Content from refs/notes/ai, if any
	Text          string // For text-only scans (PR body, comments)
	RepoPath      string
	BranchName    string // Name of the branch the commit/PR is on, if known.
}

func (input *Input) GetBranchName() (string, error) {
	return getCommitField(input.BranchName, fieldBranchName)
}

func (input *Input) GetAuthorEmail() (string, error) {
	return getCommitField(input.AuthorEmail, fieldAuthorEmail)
}

func (input *Input) GetCommitEmail() (string, error) {
	return getCommitField(input.CommitEmail, fieldCommitEmail)
}

func (input *Input) GetCommitMessage() (string, error) {
	return getCommitField(input.CommitMessage, fieldCommitMessage)
}

func (input *Input) GetTextWithCommitMessage() (string, error) {
	text := strings.TrimSpace(input.Text)
	commitMessage, _ := input.GetCommitMessage()

	var result []string
	if text != "" {
		result = append(result, text)
	}
	if commitMessage != "" {
		result = append(result, commitMessage)
	}
	if len(result) == 0 {
		return "", fmt.Errorf("no text or commit message, blank contents")
	}

	return strings.Join(result, "\n"), nil
}

func (input *Input) GetNotes() (GitnoteParseResult, error) {
	return parseGitnotes(input.Notes)
}

// ConsolidateScoreByFindings computes per-detector scores and a total score from findings
func ConsolidateScoreByFindings(findings []Finding) (float64, map[string]float64) {
	// For each detector, we take max of all findings for that detector.
	perDetectorScores := map[string]float64{}
	for _, f := range findings {
		detectorName := strings.TrimSpace(f.Detector)
		if cur, ok := perDetectorScores[detectorName]; !ok || f.Score > cur {
			perDetectorScores[detectorName] = f.Score
		}
	}

	// No detectors found
	if len(perDetectorScores) == 0 {
		return 0.0, perDetectorScores
	}

	// Compute total score for findings
	return CalculateTotalScore(perDetectorScores), perDetectorScores
}

// CalculateTotalScore computes total score from a per-detector scores map
func CalculateTotalScore(perDetectorScores map[string]float64) float64 {
	var totalScore float64
	for _, score := range perDetectorScores {
		totalScore += score
	}
	return totalScore
}
