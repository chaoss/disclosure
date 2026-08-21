package cmd

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/chaoss/disclosure/detection"
	"github.com/chaoss/disclosure/detection/branchname"
	"github.com/chaoss/disclosure/detection/committer"
	"github.com/chaoss/disclosure/detection/gitnotes"
	"github.com/chaoss/disclosure/detection/toolmention"
	"github.com/chaoss/disclosure/detection/trailer"
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

// detectorConfig struct to hold config properties for any of the detectors.
type detectorConfig struct {
	checkboxAIUsedLabel     string
	checkboxAINotUsedLabel  string
	enableCheckboxDetection bool
}

func allDetectors(confidenceLevels map[detection.Confidence]float64, config detectorConfig) []detection.Detector {
	toolmentionDetector := &toolmention.Detector{}
	toolmentionDetector.SetConfidenceLevels(confidenceLevels)
	toolmentionDetector.SetCheckboxConfig(
		config.enableCheckboxDetection, config.checkboxAIUsedLabel, config.checkboxAINotUsedLabel,
	)
	return []detection.Detector{
		&committer.Detector{ConfidenceLevels: confidenceLevels},
		&gitnotes.Detector{ConfidenceLevels: confidenceLevels},
		&trailer.Detector{ConfidenceLevels: confidenceLevels},
		toolmentionDetector,
		&branchname.Detector{ConfidenceLevels: confidenceLevels},
	}
}

// parseKeyValueFloatList parses strings like "a=1,b=2.5" into a map[string]float64.
func parseKeyValueFloatList(s string) (map[string]float64, error) {
	out := map[string]float64{}
	if strings.TrimSpace(s) == "" {
		return out, nil
	}
	parts := strings.SplitSeq(s, ",")
	for p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid entry: %q", p)
		}
		key := strings.TrimSpace(kv[0])
		if key == "" {
			return nil, fmt.Errorf("empty key in entry: %q", p)
		}
		valStr := strings.TrimSpace(kv[1])
		if valStr == "" {
			return nil, fmt.Errorf("empty value in entry: %q", p)
		}
		v, err := strconv.ParseFloat(valStr, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("invalid numeric value %q in entry %q", valStr, p)
		}
		out[key] = v
	}
	return out, nil
}

// Run is the main entry point for the CLI. Returns an exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	rootCmd := &cobra.Command{
		Use:   "disclosure",
		Short: "Detect AI-generated contributions",
		Long: `Disclosure is a standalone CLI tool that detects AI-generated contributions
in git repositories. It works entirely from git-level data (commit emails,
messages, trailers) with no platform API dependencies.

The tool detects when AI tools are disclosed in contributions — not whether
AI was actually used. It checks for known AI bot emails, Co-Authored-By
trailers, git-ai notes, and tool name mentions in commit messages and text.

Exit codes:
  0  No AI detected
  1  AI detected
  2  Error`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	exitCode := ExitNoAI

	rootCmd.AddCommand(scanCommand(stdout, stderr, &exitCode))
	rootCmd.AddCommand(textCommand(stdout, stderr, &exitCode))
	rootCmd.AddCommand(findConfigCommand(stdout, stderr, &exitCode))
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
	var confidenceLevelsFlag string

	cmd := &cobra.Command{
		Use:   "scan [repo-path]",
		Short: "Scan commits for AI signals",
		Long: `Scan commits in a git repository for signals of AI tool usage.

Checks each commit for:
  - Known AI bot committer emails (Claude, Copilot, Cursor, etc.)
  - Co-Authored-By trailers with AI tool emails
  - git-ai authorship logs in git notes
  - AI session ID trailers
  - Commit message patterns (aider:, Generated with Claude Code, etc.)
  - Tool name mentions in commit messages
  - Branch naming conventions used by AI CLIs (codex/, claude/, cursor/, etc.)

Examples:
  disclosure scan
  disclosure scan --range=abc123..def456 --format=json
  disclosure scan --min-confidence=high /path/to/repo
  disclosure scan --range=$BASE..HEAD --min-confidence=medium`,
		Example: `  # Scan current directory
  disclosure scan

  # Scan a specific commit range with JSON output
  disclosure scan --range=abc123..def456 --format=json

  # Only show high-confidence findings
  disclosure scan --min-confidence=high /path/to/repo

  # Scan and use in CI pipeline
  if disclosure scan --range=$BASE..HEAD --min-confidence=medium; then
    echo "No AI detected"
  else
    echo "AI involvement detected"
    exit 1
  fi

  # Set custom confidence levels
  disclosure scan --confidence-levels=low=20,medium=50, high=100 --format=json
  `,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			repoPath := "."
			if len(args) > 0 {
				repoPath = args[0]
			}

			minConf, err := detection.ConfidenceFromString(minConfFlag)
			if err != nil {
				fmt.Fprintln(stderr, err)
				*exitCode = ExitError
				return err
			}

			// parse confidence-levels override if provided
			confidenceLevels := detection.GetDefaultConfidenceLevels()
			if strings.TrimSpace(confidenceLevelsFlag) != "" {
				flagMap, err := parseKeyValueFloatList(confidenceLevelsFlag)
				if err != nil {
					fmt.Fprintln(stderr, err)
					*exitCode = ExitError
					return err
				}
				if confidenceLevels, err = detection.SetConfidenceLevelsFromStrings(
					confidenceLevels,
					flagMap,
				); err != nil {
					fmt.Fprintln(stderr, err)
					*exitCode = ExitError
					return err
				}
			}

			detectors := allDetectors(confidenceLevels, detectorConfig{})
			report, err := scan.ScanCommitRange(repoPath, rangeFlag, detectors)
			if err != nil {
				fmt.Fprintf(stderr, "error: %v\n", err)
				*exitCode = ExitError
				return err
			}

			report = filterReport(report, minConf, confidenceLevels)

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
	cmd.Flags().StringVar(&confidenceLevelsFlag, "confidence-levels", "", "override confidence->score mapping, e.g. 'low=20,medium=60,high=100'")

	return cmd
}

func textCommand(stdout, stderr io.Writer, exitCode *int) *cobra.Command {
	var formatFlag string
	var inputFlag string
	var checkboxAIUsedLabel string
	var checkboxAINotUsedLabel string
	var enableCheckboxDetection bool

	cmd := &cobra.Command{
		Use:   "text",
		Short: "Scan text input for AI signals",
		Long: `Scan arbitrary text for AI tool name mentions and signals.

Useful for scanning PR descriptions, issue comments, or any text input
for mentions of AI tools like Claude, Copilot, Cursor, ChatGPT, etc.

The text scanner uses word-boundary matching to find tool names and
is the primary detector for non-commit text analysis.

Examples:
  echo "I used Claude to write this" | disclosure text --format=json
  disclosure text --input=pr-body.txt
  cat comment.txt | disclosure text --min-confidence=medium
  disclosure text --input=review.txt --format=json | jq '.findings'`,
		Example: `  # Scan text from stdin
  echo "I used Claude to write this" | disclosure text --format=json

  # Scan a file
  disclosure text --input=pr-body.txt

  # Scan with medium confidence threshold
  cat comment.txt | disclosure text --min-confidence=medium

  # Use in a pipeline
  disclosure text --input=review.txt --format=json | jq '.findings'

  # Use checkbox labels
  disclosure text \
	--enable-checkbox-detection
	--cb-disclosed-ai="AI was used in this PR" \
	--cb-disclosed-noai="AI was not used in this PR" \
	--input=pr-body.txt
  `,
		RunE: func(_ *cobra.Command, args []string) error {
			var textBytes []byte
			var err error

			if inputFlag == "-" {
				textBytes, err = io.ReadAll(os.Stdin)
			} else {
				textBytes, err = os.ReadFile(filepath.Clean(inputFlag))
			}
			if err != nil {
				fmt.Fprintf(stderr, "error reading input: %v\n", err)
				*exitCode = ExitError
				return err
			}

			detectors := allDetectors(detection.GetDefaultConfidenceLevels(), detectorConfig{
				checkboxAIUsedLabel:     checkboxAIUsedLabel,
				checkboxAINotUsedLabel:  checkboxAINotUsedLabel,
				enableCheckboxDetection: enableCheckboxDetection,
			})
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

	cmd.Flags().BoolVar(
		&enableCheckboxDetection,
		"enable-checkbox-detection",
		false,
		"flag to enable detecting ai use checkboxes in specified text",
	)
	cmd.Flags().StringVar(
		&checkboxAIUsedLabel,
		"cb-disclosed-ai",
		detection.DefaultCheckboxAIUsedLabel,
		"string label for checkbox for AI use declaration",
	)
	cmd.Flags().StringVar(
		&checkboxAINotUsedLabel,
		"cb-disclosed-noai",
		detection.DefaultCheckboxAINotUsedLabel,
		"string label for checkbox for no AI use declaration",
	)
	cmd.Flags().StringVar(&formatFlag, "format", "text", "output format: json or text")
	cmd.Flags().StringVar(&inputFlag, "input", "-", "input file path, or - for stdin")

	return cmd
}

func versionCommand(stdout io.Writer, exitCode *int) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Long: `Print the disclosure version information.

Examples:
  disclosure version
  disclosure version --format=json`,
		Example: `  disclosure version
  disclosure version --format=json`,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintf(stdout, "disclosure %s\n", Version)
			*exitCode = ExitNoAI
		},
	}
}

func filterReport(
	report scan.Report,
	minConf detection.Confidence,
	confidenceLevels map[detection.Confidence]float64,
) scan.Report {
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

	for _, commit := range report.Commits {
		var commitFindings []detection.Finding
		for _, f := range commit.Findings {
			if f.Confidence >= minConf {
				commitFindings = append(commitFindings, f)
			}
		}

		// Recompute per-commit score from the kept findings
		commitScore, perDetectorScores := detection.ConsolidateScoreByFindings(commitFindings)
		confidence := detection.ScoreToConfidence(confidenceLevels, commitScore)
		result := scan.CommitResult{
			Hash:              commit.Hash,
			Findings:          commitFindings,
			Score:             commitScore,
			Confidence:        confidence,
			PerDetectorScores: perDetectorScores,
		}
		filtered.Commits = append(filtered.Commits, result)
		if len(commitFindings) > 0 {
			filtered.Summary.AICommits++
		}
		for _, commitFinding := range commitFindings {
			filtered.Summary.ToolCounts[commitFinding.Tool]++
			filtered.Summary.ByConfidence[commitFinding.Confidence.String()]++
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
  disclosure docs

  # build rest docs at default output dir
  disclosure docs --format rest

  # build manpages docs at a specific 'documentation' dir
  disclosure docs --format manpages --out %s`, defaultOutputDir, exampleCustomDir),
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
				err = os.MkdirAll(docDir, 0o750)
			} else {
				err = fmt.Errorf("unknown format: %s", formatFlag)
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
