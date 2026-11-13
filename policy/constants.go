package policy

import "regexp"

// FilePattern associates a known policy-related path pattern with a file type.
type FilePattern struct {
	Pattern *regexp.Regexp
	Type    string
}

// KnownFilePatterns lists policy-related file patterns in matching order.
// Patterns match lowercased Git paths; reports preserve the original spelling.
// The first matching entry determines the file type. Add new patterns here and
// cover them with matching and non-matching examples in policy_test.go.
var KnownFilePatterns = []FilePattern{
	{Pattern: regexp.MustCompile(`(^|/)agents\.md$`), Type: "agent_instructions"},
	{Pattern: regexp.MustCompile(`(^|/)claude\.md$`), Type: "agent_instructions"},
	{Pattern: regexp.MustCompile(`(^|/)gemini\.md$`), Type: "agent_instructions"},
	{Pattern: regexp.MustCompile(`^(\.github/|docs/)?code_of_conduct\.md$`), Type: "conduct"},
	{Pattern: regexp.MustCompile(`^(\.github/|docs/)?contributing\.md$`), Type: "contributing"},
	{Pattern: regexp.MustCompile(`^(\.github/|docs/)?security\.md$`), Type: "security"},
	{Pattern: regexp.MustCompile(`^\.github/copilot-instructions\.md$`), Type: "agent_instructions"},
	{Pattern: regexp.MustCompile(`^\.github/instructions/([^/]+/)*[^/]*\.instructions\.md$`), Type: "agent_instructions"},
	{Pattern: regexp.MustCompile(`^\.cursor/rules/([^/]+/)*[^/]*\.mdc$`), Type: "agent_instructions"},
	{Pattern: regexp.MustCompile(`^\.clinerules(/[^/]+)*$`), Type: "agent_instructions"},
	{Pattern: regexp.MustCompile(`^\.windsurf/rules/([^/]+/)*[^/]*\.md$`), Type: "agent_instructions"},
	{Pattern: regexp.MustCompile(`^\.continue/rules/([^/]+/)*[^/]*\.md$`), Type: "agent_instructions"},
	{Pattern: regexp.MustCompile(`^(\.github/|docs/)?pull_request_template(\.md|/[^/]+)$`), Type: "pull_request_template"},
	{Pattern: regexp.MustCompile(`^(\.github/|docs/)?issue_template(\.md|/[^/]+)$`), Type: "issue_template"},
}
