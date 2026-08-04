package scan

import (
	"github.com/chaoss/disclosure/detection"
	"github.com/chaoss/disclosure/gitops"
)

// CommitResult holds findings for a single commit.
type CommitResult struct {
	Hash     string              `json:"hash"`
	Findings []detection.Finding `json:"findings"`
	Score    float64             `json:"score"`
}

// Summary aggregates stats across all commits scanned.
type Summary struct {
	TotalCommits      int                `json:"total_commits"`
	AICommits         int                `json:"ai_commits"`
	ToolCounts        map[string]int     `json:"tool_counts"`
	ByConfidence      map[string]int     `json:"by_confidence"`
	PerDetectorScores map[string]float64 `json:"per_detector_scores"`
	OverallScore      float64            `json:"overall_score"`
}

// Report holds the full scan results.
type Report struct {
	Commits []CommitResult `json:"commits"`
	Summary Summary        `json:"summary"`
}

// Weights can be set (e.g., from cli) to control detector weighting used when consolidating scores.
// If nil, detectors are equally weighted.
var Weights map[string]float64

// ScanCommitRange scans all commits in the given range using the provided detectors.
func ScanCommitRange(repoPath, commitRange string, detectors []detection.Detector) (Report, error) {
	commits, err := gitops.ListCommits(repoPath, commitRange)
	if err != nil {
		return Report{}, err
	}

	// Best-effort: an empty branch name (e.g. detached HEAD, common in CI
	// checkouts) simply means the branchname detector finds nothing.
	branchName, _ := gitops.GetCurrentBranch(repoPath)

	var results []CommitResult
	for _, c := range commits {
		result := scanOneCommit(c, branchName, detectors)
		results = append(results, result)
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

	score, _ := detection.ConsolidateFindingScore(findings, Weights)

	return CommitResult{
		Hash:     c.Hash,
		Findings: findings,
		Score:    score,
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

	overall, perDetectorScores := detection.ConsolidateFindingScore(allFindings, Weights)
	summary.PerDetectorScores = perDetectorScores
	summary.OverallScore = overall

	return Report{
		Commits: results,
		Summary: summary,
	}
}
