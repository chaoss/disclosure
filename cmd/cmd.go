package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/chaoss/ai-detection-action/detection"
	"github.com/chaoss/ai-detection-action/detection/coauthor"
	"github.com/chaoss/ai-detection-action/detection/committer"
	"github.com/chaoss/ai-detection-action/detection/gitnotes"
	"github.com/chaoss/ai-detection-action/detection/message"
	"github.com/chaoss/ai-detection-action/detection/toolmention"
	"github.com/chaoss/ai-detection-action/output"
	"github.com/chaoss/ai-detection-action/scan"
	"github.com/spf13/cobra"
)

var Version = "dev"

// Exit codes
const (
	ExitNoAI  = 0
	ExitAI    = 1
	ExitError = 2
)

func allDetectors() []detection.Detector {
	return []detection.Detector{
		&committer.Detector{},
		&coauthor.Detector{},
		&gitnotes.Detector{},
		&message.Detector{},
		&toolmention.Detector{},
	}
}

// Run is the main entry point for the CLI. Returns an exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	rootCmd := &cobra.Command{
		Use:           "ai-detection",
		Short:         "Detect AI-generated contributions",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	exitCode := ExitNoAI

	rootCmd.AddCommand(scanCommand(stdout, stderr, &exitCode))
	rootCmd.AddCommand(textCommand(stdout, stderr, &exitCode))
	rootCmd.AddCommand(versionCommand(stdout, &exitCode))

	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	return exitCode
}

func scanCommand(stdout, stderr io.Writer, exitCode *int) *cobra.Command {
	var rangeFlag string
	var formatFlag string
	var minConfFlag string

	cmd := &cobra.Command{
		Use:   "scan [repo-path]",
		Short: "Scan commits for AI signals",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			repoPath := "."
			if len(args) > 0 {
				repoPath = args[0]
			}

			minConf, err := output.ConfidenceFromString(minConfFlag)
			if err != nil {
				fmt.Fprintln(stderr, err)
				*exitCode = ExitError
				return err
			}

			detectors := allDetectors()
			report, err := scan.ScanCommitRange(repoPath, rangeFlag, detectors)
			if err != nil {
				fmt.Fprintf(stderr, "error: %v\n", err)
				*exitCode = ExitError
				return err
			}

			report = filterReport(report, minConf)

			switch formatFlag {
			case "json":
				if err := output.FormatJSON(stdout, report); err != nil {
					fmt.Fprintf(stderr, "error: %v\n", err)
					*exitCode = ExitError
					return err
				}
			case "text":
				if err := output.FormatText(stdout, report); err != nil {
					fmt.Fprintf(stderr, "error: %v\n", err)
					*exitCode = ExitError
					return err
				}
			default:
				err := fmt.Errorf("unknown format: %s", formatFlag)
				fmt.Fprintln(stderr, err)
				*exitCode = ExitError
				return err
			}

			if report.Summary.AICommits > 0 {
				*exitCode = ExitAI
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&rangeFlag, "range", "", "commit range in BASE..HEAD format")
	cmd.Flags().StringVar(&formatFlag, "format", "text", "output format: json or text")
	cmd.Flags().StringVar(&minConfFlag, "min-confidence", "low", "minimum confidence level: low, medium, high (or 1, 2, 3)")

	return cmd
}

func textCommand(stdout, stderr io.Writer, exitCode *int) *cobra.Command {
	var formatFlag string
	var inputFlag string

	cmd := &cobra.Command{
		Use:   "text",
		Short: "Scan text input for AI signals",
		RunE: func(_ *cobra.Command, args []string) error {
			var textBytes []byte
			var err error

			if inputFlag == "-" {
				textBytes, err = io.ReadAll(os.Stdin)
			} else {
				textBytes, err = os.ReadFile(inputFlag)
			}
			if err != nil {
				fmt.Fprintf(stderr, "error reading input: %v\n", err)
				*exitCode = ExitError
				return err
			}

			detectors := allDetectors()
			findings := scan.ScanText(string(textBytes), detectors)

			switch formatFlag {
			case "json":
				if err := output.FormatJSONFindings(stdout, findings); err != nil {
					fmt.Fprintf(stderr, "error: %v\n", err)
					*exitCode = ExitError
					return err
				}
			case "text":
				if err := output.FormatTextFindings(stdout, findings); err != nil {
					fmt.Fprintf(stderr, "error: %v\n", err)
					*exitCode = ExitError
					return err
				}
			default:
				err := fmt.Errorf("unknown format: %s", formatFlag)
				fmt.Fprintln(stderr, err)
				*exitCode = ExitError
				return err
			}

			if len(findings) > 0 {
				*exitCode = ExitAI
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&formatFlag, "format", "text", "output format: json or text")
	cmd.Flags().StringVar(&inputFlag, "input", "-", "input file path, or - for stdin")

	return cmd
}

func versionCommand(stdout io.Writer, exitCode *int) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintf(stdout, "ai-detection %s\n", Version)
			*exitCode = ExitNoAI
		},
	}
}

func filterReport(report scan.Report, minConf detection.Confidence) scan.Report {
	if minConf <= detection.ConfidenceLow {
		return report
	}

	filtered := scan.Report{
		Commits: make([]scan.CommitResult, 0, len(report.Commits)),
		Summary: scan.Summary{
			TotalCommits: report.Summary.TotalCommits,
			ToolCounts:   map[string]int{},
			ByConfidence: map[string]int{},
		},
	}

	for _, cr := range report.Commits {
		var kept []detection.Finding
		for _, f := range cr.Findings {
			if f.Confidence >= minConf {
				kept = append(kept, f)
			}
		}
		result := scan.CommitResult{Hash: cr.Hash, Findings: kept}
		filtered.Commits = append(filtered.Commits, result)

		if len(kept) > 0 {
			filtered.Summary.AICommits++
		}
		for _, f := range kept {
			filtered.Summary.ToolCounts[f.Tool]++
			filtered.Summary.ByConfidence[f.Confidence.String()]++
		}
	}

	return filtered
}
