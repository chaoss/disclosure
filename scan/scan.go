package scan

import (
	"github.com/chaoss/ai-detection-action/detection"
	"github.com/chaoss/ai-detection-action/gitops"
)

// CommitResult holds findings for a single commit.
type CommitResult struct {
	Hash     string              `json:"hash"`
	Findings []detection.Finding `json:"findings"`
}

// Summary aggregates stats across all commits scanned.
type Summary struct {
	TotalCommits int            `json:"total_commits"`
	AICommits    int            `json:"ai_commits"`
	ToolCounts   map[string]int `json:"tool_counts"`
	ByConfidence map[string]int `json:"by_confidence"`
}

// Report holds the full scan results.
type Report struct {
	Commits []CommitResult `json:"commits"`
	Summary Summary        `json:"summary"`
}

// ScanCommitRange scans all commits in the given range using the provided detectors.
func ScanCommitRange(repoPath, commitRange string, detectors []detection.Detector) (Report, error) {
	commits, err := gitops.ListCommits(repoPath, commitRange)
	if err != nil {
		return Report{}, err
	}

	var results []CommitResult
	for _, c := range commits {
		result := scanOneCommit(c, detectors)
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

	return scanOneCommit(c, detectors), nil
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

func scanOneCommit(c gitops.Commit, detectors []detection.Detector) CommitResult {
	input := detection.Input{
		CommitHash:    c.Hash,
		CommitEmail:   c.CommitterEmail,
		CommitMessage: c.Message,
		Notes:         c.Notes,
	}

	var findings []detection.Finding
	for _, d := range detectors {
		findings = append(findings, d.Detect(input)...)
	}

	return CommitResult{
		Hash:     c.Hash,
		Findings: findings,
	}
}

func buildReport(results []CommitResult) Report {
	summary := Summary{
		TotalCommits: len(results),
		ToolCounts:   map[string]int{},
		ByConfidence: map[string]int{},
	}

	for _, r := range results {
		if len(r.Findings) > 0 {
			summary.AICommits++
		}
		for _, f := range r.Findings {
			summary.ToolCounts[f.Tool]++
			summary.ByConfidence[f.Confidence.String()]++
		}
	}

	return Report{
		Commits: results,
		Summary: summary,
	}
}
