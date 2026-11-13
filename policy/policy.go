// Package policy finds known policy-related files without interpreting their contents.
package policy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chaoss/disclosure/gitops"
)

// File describes a policy-related file found in the committed repository tree.
type File struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// Report describes a neutral inventory from one repository commit.
type Report struct {
	Repository    string `json:"repository"`
	ScannedCommit string `json:"scanned_commit"`
	Files         []File `json:"files"`
}

// Scan inventories known policy-related file paths at the repository's current HEAD.
// It does not read file contents or inspect repository history.
func Scan(repoPath string) (Report, error) {
	snapshot, err := gitops.GetHeadTree(repoPath)
	if err != nil {
		return Report{}, fmt.Errorf("scanning policy files: %w", err)
	}

	report := Report{
		Repository:    repoPath,
		ScannedCommit: snapshot.Commit,
		Files:         make([]File, 0),
	}
	for _, filePath := range snapshot.Paths {
		fileType, ok := typeForPath(filePath)
		if !ok {
			continue
		}
		report.Files = append(report.Files, File{Path: filePath, Type: fileType})
	}

	// Keep this explicit even though the git tree paths are sorted, so callers can
	// depend on the inventory order if the source of paths changes later.
	sort.Slice(report.Files, func(i, j int) bool {
		return report.Files[i].Path < report.Files[j].Path
	})

	return report, nil
}

func typeForPath(filePath string) (string, bool) {
	// Git paths are case-sensitive, but common policy filenames are often written
	// with different capitalization. Normalize only for matching; report the path
	// exactly as it appears in the repository.
	normalizedPath := strings.ToLower(filePath)
	for _, pattern := range KnownFilePatterns {
		if pattern.Pattern.MatchString(normalizedPath) {
			return pattern.Type, true
		}
	}
	return "", false
}
