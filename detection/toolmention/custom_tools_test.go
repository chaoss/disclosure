package toolmention

import (
	"fmt"
	"slices"
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func TestCustomTools(t *testing.T) {
	tests := []struct {
		name  string
		tools []string
		text  string
		want  []string
	}{
		{"supplement built-ins", []string{"Example Assistant", "Example Model 1"}, "Example Assistant and Example Model 1 helped alongside Cursor", []string{"Example Assistant", "Example Model 1", "Cursor"}},
		{"trim and deduplicate", []string{" Example Assistant ", "example assistant", "CURSOR"}, "EXAMPLE ASSISTANT and cursor, then Example Assistant", []string{"Example Assistant", "Cursor"}},
		{"built-in spelling wins across mentions", []string{"CURSOR"}, "Cursor and CURSOR", []string{"Cursor"}},
		{"longest mention", []string{"GPT-4 Turbo", "Claude Opus 4.6"}, "GPT-4 Turbo and Claude Opus 4.6", []string{"GPT-4 Turbo", "Claude Opus 4.6"}},
		{"separator variants", []string{"Example Model-1"}, "example_model 1", []string{"Example Model-1"}},
		{"literal regex characters", []string{"Example.AI", "Example[1]", "Example++"}, "Used Example.AI, Example[1], and Example++.", []string{"Example.AI", "Example[1]", "Example++"}},
		{"no regex interpretation", []string{"Example.AI", "Example[1]", "Example.*"}, "ExampleXAI Example1 ExampleAnything", nil},
		{"word boundaries", []string{"ExampleBot"}, "PreExampleBot ExampleBots", nil},
		{"no match", []string{"Example Assistant"}, "A normal contribution without assistance", nil},
		{"explicit common word", []string{"Weaver"}, "The weaver repaired the fabric", []string{"Weaver"}},
		{"separators are not an empty pattern", []string{"---"}, "A normal contribution", nil},
		{"empty list keeps built-ins", []string{}, "Used Cursor", []string{"Cursor"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
			if err := d.SetCustomTools(tt.tools); err != nil {
				t.Fatal(err)
			}
			findings := d.Detect(detection.Input{Text: tt.text})
			var got []string
			for _, f := range findings {
				got = append(got, f.Tool)
				if f.Detector != "toolmention" || f.Score != detection.ToolMentionBaseScore || f.Confidence != detection.ConfidenceLow {
					t.Errorf("unexpected finding: %+v", f)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("tools = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomToolsIsolation(t *testing.T) {
	first := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
	second := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
	names := []string{"Example Assistant"}
	if err := first.SetCustomTools(names); err != nil {
		t.Fatal(err)
	}
	names[0] = "Different Assistant"
	input := detection.Input{Text: "Example Assistant"}
	if got := first.Detect(input); len(got) != 1 || got[0].Tool != "Example Assistant" {
		t.Fatalf("configured detector lost its names: %+v", got)
	}
	if got := second.Detect(input); len(got) != 0 {
		t.Fatalf("custom names leaked to another detector: %+v", got)
	}
	if err := first.SetCustomTools([]string{"Another Assistant"}); err != nil {
		t.Fatal(err)
	}
	if got := first.Detect(input); len(got) != 0 {
		t.Fatalf("old custom names survived replacement: %+v", got)
	}
	if err := first.SetCustomTools(nil); err != nil {
		t.Fatal(err)
	}
	if got := first.Detect(detection.Input{Text: "Another Assistant"}); len(got) != 0 {
		t.Fatalf("custom names survived reset: %+v", got)
	}
	if got := first.Detect(detection.Input{Text: "Cursor"}); len(got) != 1 {
		t.Fatalf("reset removed built-ins: %+v", got)
	}
}

func TestCustomToolsInvalidName(t *testing.T) {
	d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
	for _, name := range []string{"", " \t\n"} {
		if err := d.SetCustomTools([]string{"Example Assistant", name}); err == nil {
			t.Errorf("expected error for name %q", name)
		}
	}
	if got := d.Detect(detection.Input{Text: "Example Assistant"}); len(got) != 0 {
		t.Fatalf("invalid configuration was partially applied: %+v", got)
	}
}

func TestCustomToolsCheckboxDetection(t *testing.T) {
	d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
	d.SetCheckboxConfig(true, "Used AI", "No AI")
	if err := d.SetCustomTools([]string{"Example Assistant"}); err != nil {
		t.Fatal(err)
	}
	findings := d.Detect(detection.Input{Text: "[x] Used AI\nExample Assistant"})
	if len(findings) != 1 || findings[0].Tool != "Example Assistant" || findings[0].Score != detection.ToolMentionBaseScore+detection.CheckboxAIUsedBaseScore {
		t.Fatalf("unexpected checkbox finding: %+v", findings)
	}
	findings = d.Detect(detection.Input{Text: "<!-- Example Assistant -->\nA normal contribution"})
	if len(findings) != 0 {
		t.Fatalf("custom name inside an HTML comment was not ignored: %+v", findings)
	}
}

func BenchmarkDetectCustomToolsNoMatch(b *testing.B) {
	for _, count := range []int{0, 600} {
		b.Run(fmt.Sprintf("names=%d", count), func(b *testing.B) {
			d := &Detector{ConfidenceLevels: detection.GetDefaultConfidenceLevels()}
			names := make([]string, count)
			for i := range names {
				names[i] = fmt.Sprintf("Example Model %d", i)
			}
			if err := d.SetCustomTools(names); err != nil {
				b.Fatal(err)
			}
			input := detection.Input{Text: "Update the parser and add regression coverage for empty input."}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				d.Detect(input)
			}
		})
	}
}
