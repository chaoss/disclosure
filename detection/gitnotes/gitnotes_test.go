package gitnotes

import (
	"testing"

	"github.com/chaoss/ai-detection-action/detection"
)

func TestDetect(t *testing.T) {
	d := &Detector{}

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
		name      string
		notes     string
		wantTools []string
	}{
		{
			name:      "valid git-ai note with single tool",
			notes:     validNote,
			wantTools: []string{"cursor"},
		},
		{
			name:      "multiple tools in note",
			notes:     multiToolNote,
			wantTools: []string{"cursor", "claude-code"},
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
			for i, f := range findings {
				gotTools[i] = f.Tool
				if f.Confidence != detection.ConfidenceHigh {
					t.Errorf("confidence = %d, want %d", f.Confidence, detection.ConfidenceHigh)
				}
				if f.Detector != "gitnotes" {
					t.Errorf("detector = %q, want %q", f.Detector, "gitnotes")
				}
			}

			if len(gotTools) == 0 {
				gotTools = nil
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
		})
	}
}

func TestDetectDetailIncludesModel(t *testing.T) {
	d := &Detector{}
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
