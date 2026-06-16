package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/chaoss/disclosure/detection"
	"github.com/chaoss/disclosure/detection/coauthor"
	"github.com/chaoss/disclosure/detection/committer"
	"github.com/chaoss/disclosure/detection/gitnotes"
	"github.com/chaoss/disclosure/detection/message"
	"github.com/chaoss/disclosure/detection/toolmention"
	"github.com/chaoss/disclosure/output"
	"github.com/chaoss/disclosure/scan"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
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
		Use:           "disclosure",
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
	rootCmd.AddCommand(generateDocs(&exitCode))

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
			fmt.Fprintf(stdout, "disclosure %s\n", Version)
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

func generateDocs(exitCode *int) *cobra.Command {
	var outputDir string
	var formatFlag string

	defaultOutputDir := filepath.FromSlash("./docs/cli")
	supportedFormats := []string{"markdown", "manpages", "rest"}
	exampleCustomDir := filepath.FromSlash("./documentation")
	cmd := &cobra.Command{
		Use:   "docs",
		Short: fmt.Sprintf("Build docs in %s formats", strings.Join(supportedFormats, ", ")),
		Example: fmt.Sprintf(`  # simply build markdown docs at default output dir (%s)
  ai-detection-action docs

  # build rest docs at default output dir
  ai-detection-action docs --format rest

  # build manpages docs at a specific 'documentation' dir
  ai-detection-action docs --format manpages --out %s`, defaultOutputDir, exampleCustomDir),
		Args: cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			prepareError := func(err error) error {
				log.Println(err)
				*exitCode = ExitError
				return err
			}

			var docDir string
			var err error

			root := cmd.Root()
			// as per cobra docs: disable autogen tag for
			// stable, reproducible files (no timestamp footer)
			root.DisableAutoGenTag = true

			// create required dir for docs inside output dir
			if slices.Contains(supportedFormats, formatFlag) {
				docDir = filepath.Clean(filepath.Join(outputDir, formatFlag))
				err = os.MkdirAll(docDir, 0o755)
			} else {
				err = fmt.Errorf("unknown format: %s\n", formatFlag)
			}
			if err != nil {
				return prepareError(err)
			}

			// gen docs as per specified flag
			switch formatFlag {
			case "markdown":
				log.Println("Building docs in Markdown format.")
				err = doc.GenMarkdownTree(root, docDir)
			case "manpages":
				log.Println("Building docs in Manpages format.")
				hdr := &doc.GenManHeader{Title: strings.ToUpper(root.Name()), Section: "1"}
				err = doc.GenManTree(root, hdr, docDir)
			case "rest":
				log.Println("Building docs in ReST (reStructuredText) format.")
				err = doc.GenReSTTree(root, docDir)
			}

			if err != nil {
				return prepareError(err)
			}
			log.Printf("Docs built successfully at %s\n", docDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&outputDir, "out", defaultOutputDir, "output directory")
	cmd.Flags().StringVar(&formatFlag, "format", "markdown", strings.Join(supportedFormats, "|"))

	return cmd
}
