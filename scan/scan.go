package scan

import (
	"github.com/chaoss/disclosure/detection"
	"github.com/chaoss/disclosure/gitops"
)

// CommitResult holds findings for a single commit.
type CommitResult struct {
	Hash              string               `json:"hash"`
	Findings          []detection.Finding  `json:"findings"`
	PerDetectorScores map[string]float64   `json:"per_detector_scores"`
	Score             float64              `json:"score"`
	Confidence        detection.Confidence `json:"confidence"`
}

// Summary aggregates stats across all commits scanned.
type Summary struct {
	TotalCommits      int                `json:"total_commits"`
	AICommits         int                `json:"ai_commits"`
	ToolCounts        map[string]int     `json:"tool_counts"`
	ByConfidence      map[string]int     `json:"by_confidence"`
	PerDetectorScores map[string]float64 `json:"per_detector_scores"`
}

// Report holds the full scan results.
type Report struct {
	Commits []CommitResult `json:"commits"`
	Summary Summary        `json:"summary"`
}

// ProgressPhase identifies the current scan phase.
type ProgressPhase string

const (
	PhaseLoading  ProgressPhase = "loading"
	PhaseScanning ProgressPhase = "scanning"
)

// ProgressFunc is called to report progress during scan operations.
type ProgressFunc func(phase ProgressPhase, done, total int)

// ScanCommitRange scans all commits in the given range using the provided detectors.
func ScanCommitRange(repoPath, commitRange string, detectors []detection.Detector) (Report, error) {
	return ScanCommitRangeWithProgress(repoPath, commitRange, detectors, nil)
}

// ScanCommitRangeWithProgress scans all commits in the given range using the provided detectors,
// invoking progress during history loading and commit scanning if non-nil.
func ScanCommitRangeWithProgress(repoPath, commitRange string, detectors []detection.Detector, progress ProgressFunc) (Report, error) {
	var loadProgress gitops.ProgressFunc
	if progress != nil {
		loadProgress = func(loaded int) {
			progress(PhaseLoading, loaded, 0)
		}
	}

	commits, err := gitops.ListCommitsWithProgress(repoPath, commitRange, loadProgress)
	if err != nil {
		return Report{}, err
	}

	// Best-effort: an empty branch name (e.g. detached HEAD, common in CI
	// checkouts) simply means the branchname detector finds nothing.
	branchName, _ := gitops.GetCurrentBranch(repoPath)

	total := len(commits)
	var results []CommitResult
	for i, c := range commits {
		result := scanOneCommit(c, branchName, detectors)
		results = append(results, result)
		if progress != nil {
			progress(PhaseScanning, i+1, total)
		}
	}

	return buildReport(results), nil
}

// ScanCommit scans a single commit by hash.
func ScanCommit(repoPath, hash string, detectors []detection.Detector) (CommitResult, error) {
	c, err := gitops.GetCommit(repoPath, hash)
	if err != nil {
		return CommitResult{}, err
	}

	branchName, _ := gitops.GetCurrentBranch(repoPath)
	return scanOneCommit(c, branchName, detectors), nil
}

// ScanText runs detectors against arbitrary text (PR body, comments, etc).
func ScanText(text string, detectors []detection.Detector) []detection.Finding {
	input := detection.Input{Text: text}
	var findings []detection.Finding
	for _, d := range detectors {
		findings = append(findings, d.Detect(input)...)
	}
	return findings
}

func scanOneCommit(c gitops.Commit, branchName string, detectors []detection.Detector) CommitResult {
	input := detection.Input{
		CommitHash:    c.Hash,
		AuthorEmail:   c.AuthorEmail,
		CommitEmail:   c.CommitterEmail,
		CommitMessage: c.Message,
		Notes:         c.Notes,
		BranchName:    branchName,
	}

	var findings []detection.Finding
	for _, d := range detectors {
		findings = append(findings, d.Detect(input)...)
	}

	if len(detectors) == 0 {
		return CommitResult{
			Hash:              c.Hash,
			Findings:          findings,
			PerDetectorScores: nil,
			Score:             0.0,
			Confidence:        detection.ConfidenceNone,
		}
	}

	confidenceLevels := detectors[0].GetConfidenceLevels()
	score, perDetectorScores := detection.ConsolidateScoreByFindings(findings)
	confidence := detection.ScoreToConfidence(confidenceLevels, score)
	return CommitResult{
		Hash:              c.Hash,
		Findings:          findings,
		PerDetectorScores: perDetectorScores,
		Score:             score,
		Confidence:        confidence,
	}
}

func buildReport(results []CommitResult) Report {
	summary := Summary{
		TotalCommits: len(results),
		ToolCounts:   map[string]int{},
		ByConfidence: map[string]int{},
	}

	var allFindings []detection.Finding
	for _, r := range results {
		if len(r.Findings) > 0 {
			summary.AICommits++
		}
		for _, f := range r.Findings {
			summary.ToolCounts[f.Tool]++
			summary.ByConfidence[f.Confidence.String()]++
			allFindings = append(allFindings, f)
		}
	}

	_, perDetectorScores := detection.ConsolidateScoreByFindings(allFindings)
	summary.PerDetectorScores = perDetectorScores

	return Report{
		Commits: results,
		Summary: summary,
	}
}
