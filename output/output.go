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

	// overall numeric score
	if report.Summary.OverallScore > 0 {
		fmt.Fprintf(w, "Overall score: %.1f / 100\n\n", report.Summary.OverallScore)
	}

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
		if cr.Score > 0 {
			fmt.Fprintf(w, "Commit %s (score: %.1f)\n", hash, cr.Score)
		} else {
			fmt.Fprintf(w, "Commit %s\n", hash)
		}
		for _, f := range cr.Findings {
			fmt.Fprintf(w, "  [%s] %s (%s): %s\n", f.Confidence, f.DisplayTool(), f.Detector, f.Detail)
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
	// compute consolidated score for these findings
	overall, _ := detection.ConsolidateFindingScore(findings, nil)
	fmt.Fprintf(w, "Overall score: %.1f / 100\n", overall)

	for _, f := range findings {
		fmt.Fprintf(w, "  [%s] %s (%s): %s\n", f.Confidence, f.DisplayTool(), f.Detector, f.Detail)
	}
	return nil
}

// FormatJSONFindings writes findings as JSON to w.
func FormatJSONFindings(w io.Writer, findings []detection.Finding) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		Findings     []detection.Finding `json:"findings"`
		OverallScore float64             `json:"overall_score"`
	}{
		Findings:     findings,
		OverallScore: func() float64 { s, _ := detection.ConsolidateFindingScore(findings, nil); return s }(),
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
