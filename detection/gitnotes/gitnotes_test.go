package gitnotes

import (
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func assertFindingMetadata(t *testing.T, finding detection.Finding, wantScore float64) {
	t.Helper()

	if finding.Score != wantScore {
		t.Errorf("score = %f, want %f", finding.Score, wantScore)
	}

	expectedConfidence := detection.ScoreToConfidence(
		detection.GetDefaultConfidenceLevels(), wantScore,
	)

	if finding.Confidence != expectedConfidence {
		t.Errorf("confidence = %d, want %d", finding.Confidence, expectedConfidence)
	}

	if finding.Detector != "gitnotes" {
		t.Errorf("detector = %q, want %q", finding.Detector, "gitnotes")
	}
}

func TestDetect(t *testing.T) {
	d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}

	validNote := `src/main.rs
  abcd1234abcd1234 1-10,15-20
src/lib.rs
  abcd1234abcd1234 1-50
---
{
  "schema_version": "authorship/3.0.0",
  "base_commit_sha": "7734793b756b3921c88db5375a8c156e9532447b",
  "prompts": {
    "abcd1234abcd1234": {
      "agent_id": {
        "tool": "cursor",
        "id": "6ef2299e-a67f-432b-aa80-3d2fb4d28999",
        "model": "claude-4.5-opus"
      },
      "total_additions": 25,
      "total_deletions": 5,
      "accepted_lines": 20,
      "overriden_lines": 0
    }
  }
}`

	multiToolNote := `src/main.rs
  abcd1234abcd1234 1-10
  efgh5678efgh5678 25,30-35
---
{
  "schema_version": "authorship/3.0.0",
  "base_commit_sha": "abc123",
  "prompts": {
    "abcd1234abcd1234": {
      "agent_id": {
        "tool": "cursor",
        "model": "claude-4.5-opus"
      },
      "total_additions": 10,
      "total_deletions": 0,
      "accepted_lines": 10,
      "overriden_lines": 0
    },
    "efgh5678efgh5678": {
      "agent_id": {
        "tool": "claude-code",
        "model": "claude-3-sonnet"
      },
      "total_additions": 6,
      "total_deletions": 0,
      "accepted_lines": 6,
      "overriden_lines": 0
    }
  }
}`

	tests := []struct {
		name       string
		notes      string
		wantTools  []string
		wantModels []string
		wantScore  float64
	}{
		{
			name:       "valid git-ai note with single tool",
			notes:      validNote,
			wantTools:  []string{"cursor"},
			wantModels: []string{"claude-4.5-opus"},
			wantScore:  detection.GitNotesMatchBaseScore,
		},
		{
			name:       "multiple tools in note",
			notes:      multiToolNote,
			wantTools:  []string{"cursor", "claude-code"},
			wantModels: []string{"claude-4.5-opus", "claude-3-sonnet"},
			wantScore:  detection.GitNotesMatchBaseScore,
		},
		{
			name:      "empty notes",
			notes:     "",
			wantTools: nil,
		},
		{
			name:      "no separator",
			notes:     "just some random text in notes",
			wantTools: nil,
		},
		{
			name:      "invalid JSON in metadata",
			notes:     "src/main.rs\n  abc 1-10\n---\nnot json",
			wantTools: nil,
		},
		{
			name:      "wrong schema version",
			notes:     "src/main.rs\n  abc 1-10\n---\n{\"schema_version\": \"wrong/1.0\", \"prompts\": {}}",
			wantTools: nil,
		},
		{
			name:      "no tool in agent_id",
			notes:     "src/main.rs\n  abc 1-10\n---\n{\"schema_version\": \"authorship/3.0.0\", \"prompts\": {\"abc\": {\"agent_id\": {\"tool\": \"\"}}}}",
			wantTools: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := d.Detect(detection.Input{Notes: tt.notes})
			gotTools := make([]string, len(findings))
			gotModels := make([]string, len(findings))
			for i, f := range findings {
				gotTools[i] = f.Tool
				gotModels[i] = f.Model

				assertFindingMetadata(t, f, tt.wantScore)
			}

			if len(gotTools) == 0 {
				gotTools = nil
				gotModels = nil
			}

			if len(gotTools) != len(tt.wantTools) {
				t.Errorf("tools = %v, want %v", gotTools, tt.wantTools)
				return
			}

			// Check all expected tools are present (order may vary due to map iteration)
			wantSet := map[string]bool{}
			for _, w := range tt.wantTools {
				wantSet[w] = true
			}
			for _, g := range gotTools {
				if !wantSet[g] {
					t.Errorf("unexpected tool %q, want one of %v", g, tt.wantTools)
				}
			}

			if tt.wantModels != nil {
				if len(gotModels) != len(tt.wantModels) {
					t.Errorf("models = %v, want %v", gotModels, tt.wantModels)
					return
				}
				for i := range gotModels {
					if gotModels[i] != tt.wantModels[i] {
						t.Errorf("models = %v, want %v", gotModels, tt.wantModels)
						return
					}
				}
			}
		})
	}
}

func TestDetectPreservesDistinctToolModelPairs(t *testing.T) {
	d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
	note := `src/main.rs
  first 1-10
  second 11-20
  third 21-30
---
{
  "schema_version": "authorship/3.0.0",
  "base_commit_sha": "abc",
  "prompts": {
    "a-first": {
      "agent_id": {
        "tool": "cursor",
        "model": "claude-4.5-opus"
      }
    },
    "b-second": {
      "agent_id": {
        "tool": "cursor",
        "model": "gpt-4o"
      }
    },
    "c-third": {
      "agent_id": {
        "tool": "cursor",
        "model": "claude-4.5-opus"
      }
    }
  }
}`

	findings := d.Detect(detection.Input{Notes: note})
	wantTools := []string{"cursor", "cursor"}
	wantModels := []string{"claude-4.5-opus", "gpt-4o"}
	wantScore := detection.GitNotesMatchBaseScore

	if len(findings) != len(wantTools) {
		t.Fatalf("expected %d findings, got %d: %#v", len(wantTools), len(findings), findings)
	}
	for i, finding := range findings {
		if finding.Tool != wantTools[i] {
			t.Errorf("tool[%d] = %q, want %q", i, finding.Tool, wantTools[i])
		}
		if finding.Model != wantModels[i] {
			t.Errorf("model[%d] = %q, want %q", i, finding.Model, wantModels[i])
		}
		assertFindingMetadata(t, finding, wantScore)
	}
}

func TestDetectDetailIncludesModel(t *testing.T) {
	d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
	note := `src/main.rs
  abcd1234abcd1234 1-10
---
{
  "schema_version": "authorship/3.0.0",
  "base_commit_sha": "abc",
  "prompts": {
    "abcd1234abcd1234": {
      "agent_id": {
        "tool": "cursor",
        "model": "claude-4.5-opus"
      },
      "total_additions": 10,
      "total_deletions": 0,
      "accepted_lines": 10,
      "overriden_lines": 0
    }
  }
}`

	findings := d.Detect(detection.Input{Notes: note})
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	assertFindingMetadata(t, findings[0], detection.GitNotesMatchBaseScore)

	if findings[0].Detail == "" {
		t.Error("expected non-empty detail")
	}

	if !contains(findings[0].Detail, "claude-4.5-opus") {
		t.Errorf("detail should mention model, got: %s", findings[0].Detail)
	}

	if !contains(findings[0].Detail, "1 file(s)") {
		t.Errorf("detail should mention file count, got: %s", findings[0].Detail)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
