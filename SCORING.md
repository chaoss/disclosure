# Disclosure numeric scoring (20th May 2026)

Related issue: <https://github.com/chaoss/disclosure/issues/12>

## Simple additive scoring

We use simple additive scoring per detector to compute the final score. Steps are as follows:

1. Every detector produces one or more findings per commit.
2. For every commit, the scoring is then grouped per-detector e.g. for commit C if there are two
   findings for detector `trailer`, one with score 35 and another with score 45, then `max()` is used
   to aggregate per detector findings at each commit. So in this case, commit C will have score 45 for
   detector type `trailer`.
3. The per detector scores are then adds for each commit to get the score for a particular commit.
   e.g. if commit C gets per detector scores of 75.0 and 85.0 from detectors `trailer` and `toolmention`
   detectors respectively, then the total score for commit C will be 75 + 85 = 160.0
4. Confidence is calculated at commit as well as finding level. It's based on the default confidence
   levels unless user-specified:

- no confidence has score 0
- low confidence for score 1 to 30
- medium confidence for score 31 to 70
- high confidece for score 71 to 100

### Example (branch feature/sample-commit)

```git
Author: Jon Snow <jon.snow@example.com>
Date:   Sat May 17 11:42:08 2026 +0530

feat(auth): add JWT refresh token rotation and session invalidation

Implemented refresh token rotation for improved session security.
Users now receive a new refresh token on every refresh request,
and reused/expired tokens invalidate the session automatically.

Co-authored-by: Claude <noreply@anthropic.com>
Assisted-by: Github Copilot
```

In above commit, scoring will be as follows:

1. Trailer - Yes, one Co-Author finding matches known trailer with tool Claude Code (40) and known email (35). Another known Assisted-by finding matches known trailer using tool Github Copilot (75.0) = 75.0
2. Committer - No, committer email address doesn't match known AI bot email addresses = 0.0
3. Branch - No, branch does not have known tools = 0.0
4. Gitnotes - No gitnotes found = 0.0
5. toolmention - Yes, one finding, tool Claude matched = 20.0

**Total score**: 75 + 0 + 0 + 0 + 20 = **95 pts**

95 pts lies in 71 to 100 range, so it falls in confidence level here is **high**.

### CLI Output Example

#### Example 1: disclosure scan

When `disclosure scan` is run on its [own git repository](https://github.com/chaoss/disclosure),
at the time of this writing:

```sh
$ disclosure scan --confidence-levels=low=30,medium=70,high=100
Scanned 103 commits, 9 with AI signals

Tools detected:
  Aider: 3
  Claude: 2
  Claude Code: 1
  Cline: 1
  Codex: 1
  Copilot: 1
  Cursor: 2
  Devin: 1
  GLM-4: 1
  Kimi: 1
  Replit: 3
  t3.chat: 2

Commit 64cddd958faa (score: 20.0, confidence: low)
  [score: 20.0, confidence: low] Codex (toolmention): text mentions Codex
  [score: 20.0, confidence: low] Claude (toolmention): text mentions Claude
  [score: 20.0, confidence: low] Cursor (toolmention): text mentions Cursor
  [score: 20.0, confidence: low] Copilot (toolmention): text mentions Copilot
  [score: 20.0, confidence: low] Devin (toolmention): text mentions Devin
  [score: 20.0, confidence: low] Cline (toolmention): text mentions Cline
  [score: 20.0, confidence: low] Aider (toolmention): text mentions Aider
Commit 496ebded2c69 (score: 20.0, confidence: low)
  [score: 20.0, confidence: low] Kimi (toolmention): text mentions Kimi
  [score: 20.0, confidence: low] GLM-4 (toolmention): text mentions GLM-4
Commit f4f9781121c8 (score: 20.0, confidence: low)
  [score: 20.0, confidence: low] Replit (toolmention): text mentions Replit
Commit 1d7b738fedb3 (score: 20.0, confidence: low)
  [score: 20.0, confidence: low] t3.chat (toolmention): text mentions t3.chat
Commit 497c2fb26185 (score: 20.0, confidence: low)
  [score: 20.0, confidence: low] Replit (toolmention): text mentions Replit
Commit 2082e559760d (score: 20.0, confidence: low)
  [score: 20.0, confidence: low] Cursor (toolmention): text mentions Cursor
  [score: 20.0, confidence: low] Aider (toolmention): text mentions Aider
  [score: 20.0, confidence: low] Claude Code (toolmention): text mentions Claude Code
Commit 7c1dd7d8eed9 (score: 20.0, confidence: low)
  [score: 20.0, confidence: low] Aider (toolmention): text mentions Aider
  [score: 20.0, confidence: low] Claude (toolmention): text mentions Claude
Commit b90a1f0530a4 (score: 20.0, confidence: low)
  [score: 20.0, confidence: low] Replit (toolmention): text mentions Replit
Commit 938740b59216 (score: 20.0, confidence: low)
  [score: 20.0, confidence: low] t3.chat (toolmention): text mentions t3.chat
```

In above output, the `low` confidence findings indicate mentions of AI tools in
commit messages during development.

#### Example 2: disclosure text command

For a file `file.txt`:

```text
Claude and Chatgpt were used for the code, while Copilot was used for reviews,
and some other AI tools may have been used for documentation.
```

When run through `disclosure text`, it produces findings that confirm AI use:

```sh
$ disclosure text --input=file.txt --format=text
Found 3 AI signal(s):
Score: 20.0, Confidence: low
  [score: 20.0, confidence: low] Claude (toolmention): text mentions Claude
  [score: 20.0, confidence: low] ChatGPT (toolmention): text mentions ChatGPT
  [score: 20.0, confidence: low] Copilot (toolmention): text mentions Copilot
```

In the above output, `low` confidence indicates that the specified tools are mentioned in the text
body. Given that the tools are simply mentioned, further research is warranted to understand the
extent to which the those tools were in the given context of the specified text.

#### Example 3: disclosure text command with checkbox detection enabled

For a file `pr-body.txt` with checkboxes:

```text
Generative AI disclosure

Please select one option:

[x] This PR uses AI/LLMs
[ ] This PR does not use AI/LLMs

If AI tools were used, please provide details below:
- What tools were used? Claude
- How were these tools used? For code review
- Did you review these outputs before submitting this PR? Yes
```

When run through `disclosure text`, it produces findings that confirm AI use:

```sh
$ disclosure text --input=pr-body.txt \
  --format=text --enable-checkbox-detection \
  --cb-disclosed-ai="This PR uses AI/LLMs" --cb-disclosed-noai="This PR does not AI/LLMs"
Found 1 AI signal(s):
Score: 95.0, Confidence: high
  [score: 95.0, confidence: high] Claude (toolmention): checkbox confirms AI was used and text mentions Claude
```

In above output, a `high` confidence indicates that the user disclosed AI use by ticking the
`This PR uses AI/LLMs` checkbox, with Claude as the mentioned tool.
