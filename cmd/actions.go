package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/chaoss/disclosure/workflow"
	"github.com/spf13/cobra"
)

func actionsCommand(stdout, stderr io.Writer, exitCode *int) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "actions [repo-path]",
		Short: "Detect configured AI labels from GitHub Actions workflows",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			repoPath := "."
			if len(args) > 0 {
				repoPath = args[0]
			}

			config, err := workflow.DetectLabels(repoPath)
			if err != nil {
				fmt.Fprintf(stderr, "error reading workflows: %v\n", err)
				*exitCode = ExitError
				return err
			}

			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(config); err != nil {
				fmt.Fprintf(stderr, "error formatting json: %v\n", err)
				*exitCode = ExitError
				return err
			}

			return nil
		},
	}
	return cmd
}
