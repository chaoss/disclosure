package detection

import "testing"

func TestFindingDisplayTool(t *testing.T) {
	tests := []struct {
		name    string
		finding Finding
		want    string
	}{
		{
			name: "tool only",
			finding: Finding{
				Tool: "Cursor",
			},
			want: "Cursor",
		},
		{
			name: "tool and model",
			finding: Finding{
				Tool:  "Claude Code",
				Model: "Opus 4",
			},
			want: "Claude Code [Opus 4]",
		},
		{
			name: "model only",
			finding: Finding{
				Model: "gpt-4o",
			},
			want: "gpt-4o",
		},
		{
			name: "empty finding",
			finding: Finding{
				Tool:  " ",
				Model: " ",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.finding.DisplayTool(); got != tt.want {
				t.Errorf("DisplayTool() = %q, want %q", got, tt.want)
			}
		})
	}
}
