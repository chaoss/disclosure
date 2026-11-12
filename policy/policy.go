// Package policy finds known policy-related files without interpreting their contents.
package policy

import (
	"fmt"
	"path"
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
	baseName := path.Base(normalizedPath)

	switch {
	case baseName == "agents.md" || baseName == "claude.md" || baseName == "gemini.md":
		return "agent_instructions", true
	case isStandardCommunityPath(normalizedPath, "code_of_conduct.md"):
		return "conduct", true
	case isStandardCommunityPath(normalizedPath, "contributing.md"):
		return "contributing", true
	case isStandardCommunityPath(normalizedPath, "security.md"):
		return "security", true
	}

	switch {
	case normalizedPath == ".github/copilot-instructions.md",
		isUnder(normalizedPath, ".github/instructions") && strings.HasSuffix(normalizedPath, ".instructions.md"),
		isUnder(normalizedPath, ".cursor/rules") && strings.HasSuffix(normalizedPath, ".mdc"),
		normalizedPath == ".clinerules",
		isUnder(normalizedPath, ".clinerules"),
		isUnder(normalizedPath, ".windsurf/rules") && strings.HasSuffix(normalizedPath, ".md"),
		isUnder(normalizedPath, ".continue/rules") && strings.HasSuffix(normalizedPath, ".md"):
		return "agent_instructions", true
	case isTemplatePath(normalizedPath, "pull_request_template"):
		return "pull_request_template", true
	case isTemplatePath(normalizedPath, "issue_template"):
		return "issue_template", true
	default:
		return "", false
	}
}

func isStandardCommunityPath(filePath, name string) bool {
	return filePath == name ||
		filePath == ".github/"+name ||
		filePath == "docs/"+name
}

func isTemplatePath(filePath, name string) bool {
	if isStandardCommunityPath(filePath, name+".md") {
		return true
	}

	for _, directory := range []string{name, ".github/" + name, "docs/" + name} {
		prefix := directory + "/"
		if !strings.HasPrefix(filePath, prefix) {
			continue
		}
		fileName := strings.TrimPrefix(filePath, prefix)
		if fileName != "" && !strings.Contains(fileName, "/") {
			return true
		}
	}
	return false
}

func isUnder(filePath, directory string) bool {
	return strings.HasPrefix(filePath, directory+"/")
}
