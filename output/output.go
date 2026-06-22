package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

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
		fmt.Fprintf(w, "Commit %s\n", cr.Hash[:12])
		for _, f := range cr.Findings {
			fmt.Fprintf(w, "  [%s] %s (%s): %s\n", f.Confidence, formatTool(f), f.Detector, f.Detail)
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
	for _, f := range findings {
		fmt.Fprintf(w, "  [%s] %s (%s): %s\n", f.Confidence, formatTool(f), f.Detector, f.Detail)
	}
	return nil
}

// FormatJSONFindings writes findings as JSON to w.
func FormatJSONFindings(w io.Writer, findings []detection.Finding) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		Findings []detection.Finding `json:"findings"`
	}{Findings: findings})
}

func formatTool(f detection.Finding) string {
	var extras []string
	if f.Model != "" {
		extras = append(extras, f.Model)
	}
	if f.Version != "" {
		extras = append(extras, "v"+f.Version)
	}
	if len(extras) > 0 {
		return fmt.Sprintf("%s [%s]", f.Tool, strings.Join(extras, " "))
	}
	return f.Tool
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ConfidenceFromString parses a confidence string or numeric value.
func ConfidenceFromString(s string) (detection.Confidence, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "low":
		return detection.ConfidenceLow, nil
	case "2", "medium":
		return detection.ConfidenceMedium, nil
	case "3", "high":
		return detection.ConfidenceHigh, nil
	default:
		return 0, fmt.Errorf("invalid confidence %q: use low/1, medium/2, or high/3", s)
	}
}
