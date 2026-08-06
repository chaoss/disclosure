package detection

import (
	"regexp"
	"strings"
)

// KnownAgentCommitters maps GitHub noreply emails to AI tool names.
var KnownAgentCommitters = map[string]string{
	"209825114+claude[bot]@users.noreply.github.com":                  "Claude",
	"215619710+anthropic-claude[bot]@users.noreply.github.com":        "Claude (Anthropic)",
	"208546643+claude-code-action[bot]@users.noreply.github.com":      "Claude Code Action",
	"198982749+copilot@users.noreply.github.com":                      "GitHub Copilot (agent)",
	"167198135+copilot[bot]@users.noreply.github.com":                 "GitHub Copilot (chat)",
	"206951365+cursor[bot]@users.noreply.github.com":                  "Cursor",
	"215057067+openai-codex[bot]@users.noreply.github.com":            "OpenAI Codex",
	"199175422+chatgpt-codex-connector[bot]@users.noreply.github.com": "Codex via ChatGPT",
	"176961590+gemini-code-assist[bot]@users.noreply.github.com":      "Gemini Code Assist",
	"208079219+amazon-q-developer[bot]@users.noreply.github.com":      "Amazon Q Developer",
	"158243242+devin-ai-integration[bot]@users.noreply.github.com":    "Devin",
	"205137888+cline[bot]@users.noreply.github.com":                   "Cline",
	"230936708+continue[bot]@users.noreply.github.com":                "Continue.dev",
	"201248094+sourcegraph-cody[bot]@users.noreply.github.com":        "Sourcegraph Cody",
	"220155983+jetbrains-ai[bot]@users.noreply.github.com":            "JetBrains AI",
	"136622811+coderabbitai[bot]@users.noreply.github.com":            "CodeRabbit",
}

// GithubNoReplyEmailSuffix GitHub noreply emails suffix to check committer emails.
var GithubNoReplyEmailSuffix = "@users.noreply.github.com"

// SupportedToolsInMentions List of all supported tools to detect in tool mentions within commits.
var SupportedToolsInMentions = []string{
	"Claude Code",
	"Claude",
	"GitHub Copilot",
	"Copilot",
	"Cursor",
	"Aider",
	"OpenAI Codex",
	"Codex",
	"Gemini Code Assist",
	"Amazon Q Developer",
	"Amazon Q",
	"Devin",
	"Cline",
	"Continue.dev",
	"Continue CLI",
	"Continue AI",
	"Sourcegraph Cody",
	"Cody",
	"JetBrains AI",
	"CodeRabbit",
	"ChatGPT",
	"t3.chat",
	"GPT-4",
	"Windsurf",
	"Antigravity",
	"Google Gemini",
	"Kimi",
	"DeepSeek",
	"Qwen",
	"Llama",
	"Code Llama",
	"Mistral",
	"Codestral",
	"Mixtral",
	"Magistral",
	"Zed AI",
	"Zhipu AI",
	"Z.ai",
	"GLM-4",
	"ChatGLM",
	"ZCode",
	"Gemma",
	"Yi-Coder",
	"Yi-Lightning",
	"Yi-Large",
	"Yi 01.ai",
	"Yi AI",
	"01.ai",
	"Phind",
	"StarCoder",
	"Tabnine",
	"Codeium",
	"Qodo",
	"CodiumAI",
	"Replit Agent",
	"Replit",
	"Sourcery",
	"Augment Code",
	"OpenCode",
	"Vercel v0",
	"Bolt.new",
	"Lovable.dev",
	"CodeReviewBot",
	"CodeGPT",
	"Bito",
	"CodeGeeX",
	"GitKraken AI",
	"Glide AI",
	"Mastra",
	"Pieces.app",
	"PiecesOS",
	"Greptile",
	"Sweep AI",
	"Supermaven",
	"Trae",
	"OpenHands",
	"PR-Agent",
	"SWE-Agent",
	"OpenClaw",
	"ClawWork",
	"ZeroClaw",
	"NanoClaw",
	"TinyClaw",
	"MicroClaw",
	"MimiClaw",
	"IronClaw",
	"PicoClaw",
	"SupaClaw",
	"OpenClaude",
	"xAI Grok",
	"Grok AI",
	"Grok Code Fast",
	"Command-R+",
	"Command-R",
	"Roo Code",
	"SantaCoder",
	"PearAI",
	"Phi-4",
	"Nemotron",
	"Salesforce Codegen",
	"CodeWhisperer",
}

var genericModelMentionKeys = map[string]struct{}{
	"auto":             {},
	"autorouter":       {},
	"bodybuilder":      {},
	"commanda":         {},
	"commandr":         {},
	"free":             {},
	"freemodelsrouter": {},
	"o1":               {},
	"o3":               {},
	"r1":               {},
	"router":           {},
	"saba":             {},
	"sonar":            {},
	"spotlight":        {},
	"uncensored":       {},
	"weaver":           {},
}

func SupportedToolMentions(includeGenerated bool) []string {
	mentions := append([]string(nil), SupportedToolsInMentions...)
	if includeGenerated {
		mentions = appendCanonicalMentions(mentions, GeneratedModelMentions...)
	}
	return mentions
}

func appendCanonicalMentions(existing []string, candidates ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(candidates))
	for _, item := range existing {
		key := strings.ToLower(strings.TrimSpace(item))
		if key != "" {
			seen[key] = struct{}{}
		}
	}

	for _, candidate := range candidates {
		canonical := strings.TrimSpace(candidate)
		key := strings.ToLower(canonical)
		if key == "" {
			continue
		}
		if _, isGeneric := genericModelMentionKeys[modelMentionKey(canonical)]; isGeneric {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		existing = append(existing, canonical)
		seen[key] = struct{}{}
	}

	return existing
}

func modelMentionKey(model string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, model)
}

// KnownCoAuthorEmails Known emails present with Co-Authored-By trailers
var KnownCoAuthorEmails = map[string]string{
	"noreply@anthropic.com":  "Claude Code",
	"cursoragent@cursor.com": "Cursor",
	"noreply@aider.chat":     "Aider",
	"copilot@github.com":     "Copilot",
}

// CoAuthorPattern Regex to look for Co-Authored-By trailer with name and email
var CoAuthorPattern = regexp.MustCompile(`(?im)^co-authored-by:\s*([^<]*)<([^>]+)>`)

// AssistedByPattern Regex to look for Assisted-By trailer with tool name
var AssistedByPattern = regexp.MustCompile(`(?im)^assisted-by\s*:\s*([^\r\n]+?)\s*$`)

// AiderCommitPrefix The commit prefix added by Aider
var AiderCommitPrefix = "aider:"

// ClaudeAttributionText
var ClaudeAttributionText = "Generated with Claude Code"

// EntireIOTrailers List of EntireIO related trailers to check for
var EntireIOTrailers = []string{
	"Entire-Metadata",
	"Entire-Metadata-Task",
	"Entire-Strategy",
	"Entire-Session",
	"Entire-Condensation",
	"Entire-Source-Ref",
	"Entire-Checkpoint",
	"Entire-Agent",
}

// ReplitAttributionRegex Regex to detect use of Replit Agent or Assistant
var ReplitAttributionRegex = `(?m)^Replit-Commit-Author:\s*(Agent|Assistant)(?:\r?\nReplit-Commit-Session-Id:\s*([a-fA-F0-9-]+))?(?:\r?\n|$)`

// ReplitAttributionPattern compiled pattern for Replit Agent or Assistant attribution
var ReplitAttributionPattern = regexp.MustCompile(ReplitAttributionRegex)

// GitNotesAuthorshipPrefix Authorship prefix to check as a precondition in git notes
var GitNotesAuthorshipPrefix = "authorship/"

// TrailerEmailPattern Regex to match email address in commit trailers
var TrailerEmailPattern = regexp.MustCompile(`\s*<[^>]+>`)
