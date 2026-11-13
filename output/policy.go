package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/chaoss/disclosure/policy"
)

// FormatPolicyJSON writes a policy-file inventory as indented JSON.
func FormatPolicyJSON(w io.Writer, report policy.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// FormatPolicyText writes a human-readable policy-file inventory.
func FormatPolicyText(w io.Writer, report policy.Report) error {
	if _, err := fmt.Fprintf(w, "Scanned repository %s at commit %s\n", report.Repository, report.ScannedCommit); err != nil {
		return err
	}
	if len(report.Files) == 0 {
		_, err := fmt.Fprintln(w, "No policy-related files found.")
		return err
	}

	if _, err := fmt.Fprintln(w, "Policy-related files:"); err != nil {
		return err
	}
	for _, file := range report.Files {
		if _, err := fmt.Fprintf(w, "  %s (%s)\n", file.Path, file.Type); err != nil {
			return err
		}
	}
	return nil
}
