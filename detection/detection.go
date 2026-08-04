package detection

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Confidence represents how confident we are that a finding indicates AI involvement.
type Confidence int

const (
	ConfidenceNone              = 0 // Nil equivalent for confidence
	ConfidenceLow    Confidence = 1 // Tool name mentioned in text
	ConfidenceMedium Confidence = 2 // Commit message pattern match
	ConfidenceHigh   Confidence = 3 // Bot email, co-author trailer, git AI ref
)

func (c Confidence) String() string {
	switch c {
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

func (c *Confidence) Increment() {
	*c = min(*c+1, ConfidenceHigh)
}

// Default mapping from Confidence -> numeric score (0..100).
var defaultConfidenceScores = map[Confidence]float64{
	ConfidenceLow:    30.0,
	ConfidenceMedium: 70.0,
	ConfidenceHigh:   100.0,
}

// confidenceScores holds the active mapping, can be overridence in tests or via cli.
var confidenceScores = map[Confidence]float64{
	ConfidenceLow:    defaultConfidenceScores[ConfidenceLow],
	ConfidenceMedium: defaultConfidenceScores[ConfidenceMedium],
	ConfidenceHigh:   defaultConfidenceScores[ConfidenceHigh],
}

func ScoreToConfidence(score float64) (Confidence, error) {
	if score < 0 || score > 100 {
		return ConfidenceNone, fmt.Errorf("invalid score, should be between 0 and 100")
	}
	levels := []Confidence{ConfidenceLow, ConfidenceMedium, ConfidenceHigh}
	for _, level := range levels {
		if math.Round(score) <= confidenceScores[level] {
			return level, nil
		}
	}
	return ConfidenceNone, fmt.Errorf("confidence intervals unable to categorize score")
}

// SetConfidenceScoresFromStrings allows to update confidenceScores using a custom map
func SetConfidenceScoresFromStrings(userMapping map[string]float64) error {
	tmp := map[Confidence]float64{}
	for k, v := range userMapping {
		k = strings.ToLower(strings.TrimSpace(k))
		switch k {
		case "low":
			tmp[ConfidenceLow] = v
		case "medium":
			tmp[ConfidenceMedium] = v
		case "high":
			tmp[ConfidenceHigh] = v
		default:
			return fmt.Errorf("unsupported confidence key: %s", k)
		}
	}
	// set defaults if unspecified in user mapping
	for c, def := range defaultConfidenceScores {
		if _, ok := tmp[c]; !ok {
			tmp[c] = def
		}
	}
	confidenceScores = tmp
	return nil
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

// ConsolidateFindingScore computes per-detector scores and a consolidated overall
// score (0..100) using a weighted average across detectors.
// Things to note:
//   - If weights is nil, detectors are equally weighted. We normalize provided weights so they sum to 1.
//   - Detectors with missing weight entries are treated as zero weight.
//   - Normalization will fallback to equal weights if total weight is zero.
func ConsolidateFindingScore(findings []Finding, weights map[string]float64) (float64, map[string]float64) {
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

	// Prepare normalized weights
	norm := map[string]float64{}
	// equal weights for all detectors if weights unspecified
	if weights == nil {
		weight := 1.0 / float64(len(perDetectorScores))
		for detectorName := range perDetectorScores {
			norm[detectorName] = weight
		}
	} else {
		var sum float64
		for detectorName := range perDetectorScores {
			currWeight := max(0, weights[detectorName])
			norm[detectorName] = currWeight
			sum += currWeight
		}
		// if user-supplied sum is zero, fallback to equal weights
		if sum == 0 {
			weight := 1.0 / float64(len(perDetectorScores))
			for detectorName := range perDetectorScores {
				norm[detectorName] = weight
			}
		} else {
			for detectorName := range norm {
				norm[detectorName] = norm[detectorName] / sum
			}
		}
	}

	// Compute overall weighted average (deterministic order)
	detectorNames := make([]string, 0, len(perDetectorScores))
	for detectorName := range perDetectorScores {
		detectorNames = append(detectorNames, detectorName)
	}
	sort.Strings(detectorNames)
	var overall float64
	for _, detectorName := range detectorNames {
		overall += perDetectorScores[detectorName] * norm[detectorName]
	}
	overall = max(0, min(overall, 100))
	overall, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", overall), 64)
	return overall, perDetectorScores
}
