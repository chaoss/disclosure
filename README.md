# CHAOSS AI Detection Tool

A standalone CLI tool and GitHub Action that detects AI-generated contributions in git repositories. It works entirely from git-level data (commit emails, messages, trailers) using [go-git](https://github.com/go-git/go-git), with no platform API dependencies in the core. A separate text-scanning mode lets wrappers pipe in PR descriptions, issue comments, or any other text.

The goal is to help open source maintainers understand when AI tools are involved in contributions, and to give community health projects like [Augur](https://github.com/chaoss/augur/) and [GrimoireLab](https://github.com/chaoss/grimoirelab/) a way to track AI usage across repositories.

## What it detects

Four detectors run against each commit, each producing findings at a confidence level:

**High confidence** -- strong signals that an AI tool authored or co-authored the commit:
- Known AI bot committer emails (Claude, Copilot, Cursor, Codex, Gemini Code Assist, Amazon Q, Devin, Cline, Continue.dev, Cody, JetBrains AI, CodeRabbit). Also matches on the numeric prefix of GitHub noreply emails, so bot username renames don't break detection.
- `Co-Authored-By` trailers with known AI tool emails (Claude Code, Cursor, Aider).
- AI session ID trailers (such as Replit-Commit-Session-Id) combined with other known commit trailers, indicating that the commit was generated as part of an AI conversation or workflow.

**Medium confidence** -- patterns in the commit message itself:
- `aider:` prefix (Aider's default commit format).
- `Generated with Claude Code` footer.
- Known commit trailers in formats unique to specific tools (such as EntireIO, Replit Agent/Assistant) that can contain values indicative of AI use.


**Low confidence** -- mentions of AI tool names in text:
- Word-boundary matches for tool names like Claude, Copilot, Cursor, Aider, ChatGPT, Windsurf, Devin, etc. This detector also runs against commit messages, and is the primary detector for the text-scanning mode (PR bodies, comments).

## CLI usage

```
ai-detection scan [--range=BASE..HEAD] [--format=json|text] [--min-confidence=low|medium|high] [repo-path]
ai-detection text [--format=json|text] [--input=FILE|-]
ai-detection version
```

Exit codes: `0` = no AI detected, `1` = AI detected, `2` = error.

### Scan commits

```sh
# Scan all commits in the current repo
ai-detection scan

# Scan a specific range, JSON output
ai-detection scan --range=abc123..def456 --format=json

# Only report high-confidence findings
ai-detection scan --min-confidence=high /path/to/repo
```

### Scan text

Reads from stdin by default, or from a file with `--input`:

```sh
echo "I used Claude to write this PR" | ai-detection text --format=json

ai-detection text --input=pr-body.txt
```

### Use as a CI gate

The exit code makes it usable in shell pipelines and CI scripts:

```sh
if ai-detection scan --range=$BASE..$HEAD --min-confidence=medium; then
  echo "No AI detected"
else
  echo "AI involvement detected"
fi
```

## GitHub Action

Add to your workflow to automatically label PRs with detected AI involvement:

```yaml
- uses: chaoss/ai-detection-action/action@main
  with:
    label: 'ai-detected'        # label to apply (default: ai-detected)
    min-confidence: 'low'        # low, medium, or high (default: low)
    scan-pr-body: 'true'         # scan PR description for tool mentions (default: true)
```

The action builds the CLI from source, scans the PR's commits and optionally its body, then applies the configured label if anything is found. It exposes two outputs:

- `ai-detected` -- `true` or `false`
- `report` -- JSON object with the full findings from both the commit scan and text scan

The labeling logic lives entirely in the action layer. The CLI reports findings; the action decides what to do with them.

## Go module

The detection packages can be imported directly into other Go projects:

```sh
go get github.com/chaoss/ai-detection-action
```

Scan a repo's commits with the built-in detectors:

```go
package main

import (
	"fmt"

	"github.com/chaoss/ai-detection-action/detection"
	"github.com/chaoss/ai-detection-action/detection/coauthor"
	"github.com/chaoss/ai-detection-action/detection/committer"
	"github.com/chaoss/ai-detection-action/detection/message"
	"github.com/chaoss/ai-detection-action/detection/toolmention"
	"github.com/chaoss/ai-detection-action/scan"
)

func main() {
	detectors := []detection.Detector{
		&committer.Detector{},
		&coauthor.Detector{},
		&message.Detector{},
		&toolmention.Detector{},
	}

	report, err := scan.ScanCommitRange("/path/to/repo", "base..head", detectors)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d commits, %d with AI signals\n", report.Summary.TotalCommits, report.Summary.AICommits)
	for _, cr := range report.Commits {
		for _, f := range cr.Findings {
			fmt.Printf("  [%s] %s: %s\n", f.Confidence, f.Tool, f.Detail)
		}
	}
}
```

Scan arbitrary text without a git repo:

```go
findings := scan.ScanText("I used Claude to write this", detectors)
```

You can also write your own detector by implementing the `detection.Detector` interface:

```go
type Detector interface {
	Name() string
	Detect(input detection.Input) []detection.Finding
}
```

Pass it alongside the built-in detectors and the scan functions will run it the same way.

## Building from source

```sh
go build -o ai-detection .
```

Requires Go 1.24+.

## Running tests

```sh
go test ./...
```

## Project layout

```
detection/              Core types: Detector interface, Finding, Confidence, Input
detection/committer/    Known AI bot committer emails
detection/coauthor/     Co-Authored-By trailer parsing
detection/message/      Commit message pattern matching
detection/toolmention/  AI tool name mentions in text
gitops/                 go-git wrapper for reading commits
scan/                   Orchestration: run detectors over commits or text
output/                 JSON and human-readable text formatters
cmd/                    CLI subcommands
action/                 GitHub Action (composite action + labeling)
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
