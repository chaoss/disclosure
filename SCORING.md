# Disclosure numeric scoring (20th May 2026)

Related issue: https://github.com/chaoss/ai-detection-action/issues/12

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
- low confidence for score 0 to 30
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
```

In above commit, scoring will be as follows:
1. Trailer - Yes, one Co-Author finding, matches known trailer Claude Code (40) with known email (35) = 75.0
2. Committer - No, committer email address doesn't match known AI bot email addresses = 0.0
3. Branch - No, branch does not have known tools = 0.0
4. Gitnotes - No gitnotes found = 0.0
5. toolmention - Yes, one finding, tool Claude matched = 20.0

**Total score: 75 + 0 + 0 + 0 + 20 = 95 pts**

95 pts lies in 71 to 100 range, so it falls in confidence level here is **high**.
