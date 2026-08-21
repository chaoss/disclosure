# CHAOSS AI disclosure detection

A standalone CLI tool and GitHub Action that detects disclosed AI-generated contributions in git repositories. It works entirely from git-level data (commit emails, messages, trailers) using [go-git](https://github.com/go-git/go-git), with no platform API dependencies in the core. A separate text-scanning mode lets wrappers pipe in PR descriptions, issue comments, or any other text.

The goal is to help open source maintainers understand when AI tools are involved in contributions, and to give community health projects like [CollectOSS](https://github.com/chaoss/collectoss/) and [GrimoireLab](https://github.com/chaoss/grimoirelab/) a way to track AI usage across repositories.

## What it detects

The built-in detectors run against each commit, each producing findings at a confidence level:

- `Co-Authored-By` and `Assisted-By` trailers with known AI tool emails (Claude Code, Cursor, Aider).
- Known commit trailers in formats unique to specific tools (such as Aider, EntireIO, Replit Agent/Assistant etc.) or footers (like `Generated with Claude Code`) that can contain values indicative of AI use.
- Known AI bot committer emails (Claude, Copilot, Cursor, Codex, Gemini Code Assist, Amazon Q, Devin, Cline, Continue.dev, Cody, JetBrains AI, CodeRabbit). Also matches on the numeric prefix of GitHub noreply emails, so bot username renames don't break detection.
- `git-ai` authorship logs stored in git notes under `refs/notes/ai`, including the attributed tool and model when available.
- Branch names following conventions used by AI coding CLIs/agents (e.g. `codex/`, `claude/`, `cursor/`, `copilot/`, `devin/`, `cline/`, `aider/`, `gemini/`).
- AI session ID trailers (such as Replit-Commit-Session-Id) combined with other known commit trailers, indicating that the commit was generated as part of an AI conversation or workflow.
- Mentions of tool names like Claude, Copilot, Cursor, Aider, ChatGPT, Windsurf, Devin, etc. This detector also runs against commit messages, and is the primary detector for the text-scanning mode (PR bodies, comments).
- Disclosure of AI use from checkboxes in pull request description or comments, in text-scanning mode. Checkbox labels configurable by user.

## CLI usage

```
disclosure scan \
	[--range=BASE..HEAD] [--format=json|text] \
	[--min-confidence=low|medium|high] \
	[--confidence-levels="low=30,medium=70,high=100"]
	[repo-path]
disclosure text \
	[--format=json|text] [--input=FILE|-] \
	[--checkbox-label-ai-used="AI was used"] \
	[--checkbox-label-ai-not-used="AI was not used"]
disclosure version
```

Exit codes: `0` = no AI detected, `1` = AI detected, `2` = error.

### Scan commits

```sh
# Scan all commits in the current repo
disclosure scan

# Scan a specific range, JSON output
disclosure scan --range=abc123..def456 --format=json

# Only report high-confidence findings
disclosure scan --min-confidence=high /path/to/repo
```

### Scan text

Reads from stdin by default, or from a file with `--input`:

```sh
echo "I used Claude to write this PR" | disclosure text --format=json

disclosure text --input=pr-body.txt
```

### Numeric scoring

Please see [SCORING.md](SCORING.md) for more information on disclosure's
scoring methodology.

### Use as a CI gate

The exit code makes it usable in shell pipelines and CI scripts:

```sh
if disclosure scan --range=$BASE..$HEAD --min-confidence=medium; then
  echo "No AI detected"
else
  echo "AI involvement detected"
fi
```

## GitHub Action

Add to your workflow to automatically label PRs with detected AI involvement:

```yaml
- uses: chaoss/disclosure/action@main
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
go get github.com/chaoss/disclosure
```

Scan a repo's commits with the built-in detectors:

```go
package main

import (
	"fmt"

	"github.com/chaoss/disclosure/detection"
	"github.com/chaoss/disclosure/detection/branchname"
	"github.com/chaoss/disclosure/detection/committer"
	"github.com/chaoss/disclosure/detection/trailer"
	"github.com/chaoss/disclosure/detection/toolmention"
	"github.com/chaoss/disclosure/scan"
)

func main() {
	detectors := []detection.Detector{
		&committer.Detector{},
		&toolmention.Detector{},
		&trailer.Detector{},
		&branchname.Detector{},
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
go build -o disclosure .
```

Requires Go 1.25+.

## Running tests

```sh
go test ./...
```

## Project layout

```
detection/              Core types: Detector interface, Finding, Confidence, Input
detection/branchname/   AI CLI/agent branch naming conventions (codex/, claude/, etc.)
detection/committer/    Known AI bot committer emails
detection/gitnotes/     git-ai authorship logs from refs/notes/ai
detection/toolmention/  AI tool name mentions in text
detection/trailer/      Look for known trailers in commit message
gitops/                 go-git wrapper for reading commits
scan/                   Orchestration: run detectors over commits or text
output/                 JSON and human-readable text formatters
cmd/                    CLI subcommands
action/                 GitHub Action (composite action + labeling)
```

## Other AI disclosure/attribution tools

- [AItrributor](https://github.com/block/aittributor) - Prepare-commit-msg hook that adds AI agent attribution to git commits.
- [Usagescale](https://usagescale.org/) - An open standard for declaring how a work was made, whose knowledge it carries, and who stands behind it.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Code is licensed under [GNU General Public License v3.0](LICENSE).
