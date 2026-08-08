package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/chaoss/disclosure/detection"
	"github.com/chaoss/disclosure/scan"
)

// FormatJSON writes the report as JSON to w.
func FormatJSON(w io.Writer, report scan.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// FormatText writes a human-readable summary to w.
func FormatText(w io.Writer, report scan.Report) error {
	fmt.Fprintf(w, "Scanned %d commits, %d with AI signals\n\n", report.Summary.TotalCommits, report.Summary.AICommits)

	if report.Summary.AICommits == 0 {
		fmt.Fprintln(w, "No AI involvement detected.")
		return nil
	}

	// Tool summary
	tools := sortedKeys(report.Summary.ToolCounts)
	fmt.Fprintln(w, "Tools detected:")
	for _, tool := range tools {
		fmt.Fprintf(w, "  %s: %d\n", tool, report.Summary.ToolCounts[tool])
	}
	fmt.Fprintln(w)

	// Per-commit detail
	for _, cr := range report.Commits {
		if len(cr.Findings) == 0 {
			continue
		}
		hash := cr.Hash
		if len(hash) > 12 {
			hash = hash[:12]
		}
		fmt.Fprintf(w, "Commit %s (score: %.1f, confidence: %s)\n", hash, cr.Score, cr.Confidence.String())
		for _, f := range cr.Findings {
			fmt.Fprintf(
				w, "  [score: %.1f, confidence: %s] %s (%s): %s\n",
				f.Score, f.Confidence, f.DisplayTool(), f.Detector, f.Detail,
			)
		}
	}

	return nil
}

// FormatTextFindings writes findings (from a text scan) in human-readable form.
func FormatTextFindings(w io.Writer, findings []detection.Finding) error {
	if len(findings) == 0 {
		fmt.Fprintln(w, "No AI involvement detected.")
		return nil
	}

	fmt.Fprintf(w, "Found %d AI signal(s):\n", len(findings))

	// compute consolidated score and confidence for these findings
	score, _ := detection.ConsolidateScoreByFindings(findings)
	confidence := detection.ScoreToConfidence(detection.GetDefaultConfidenceLevels(), score)
	fmt.Fprintf(w, "Score: %.1f, Confidence: %s\n", score, confidence.String())

	for _, f := range findings {
		fmt.Fprintf(
			w, "  [score: %.1f, confidence: %s] %s (%s): %s\n",
			f.Score, f.Confidence.String(), f.DisplayTool(), f.Detector, f.Detail,
		)
	}
	return nil
}

// FormatJSONFindings writes findings as JSON to w.
func FormatJSONFindings(w io.Writer, findings []detection.Finding) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	score, _ := detection.ConsolidateScoreByFindings(findings)
	confidence := detection.ScoreToConfidence(detection.GetDefaultConfidenceLevels(), score)
	return enc.Encode(struct {
		Findings   []detection.Finding  `json:"findings"`
		Score      float64              `json:"score"`
		Confidence detection.Confidence `json:"confidence"`
	}{
		Findings:   findings,
		Score:      score,
		Confidence: confidence,
	})
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
