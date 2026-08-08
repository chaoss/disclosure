package detection

import (
	"math"
	"reflect"
	"testing"
)

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

func TestConsolidateFindings(t *testing.T) {
	tests := []struct {
		name               string
		findings           []Finding
		wantTotalScore     float64
		wantDetectorScores map[string]float64
		wantNaN            bool
	}{
		{
			name: "sum of individual detector scores",
			findings: []Finding{
				{Detector: "A", Score: 100},
				{Detector: "B", Score: 50},
				{Detector: "C", Score: 75},
			},
			wantTotalScore: 225,
			wantDetectorScores: map[string]float64{
				"A": 100,
				"B": 50,
				"C": 75,
			},
		},
		{
			name: "mix of positive and negative scores",
			findings: []Finding{
				{Detector: "A", Score: 100},
				{Detector: "B", Score: -50},
				{Detector: "C", Score: 75},
			},
			wantTotalScore: 125,
			wantDetectorScores: map[string]float64{
				"A": 100,
				"B": -50,
				"C": 75,
			},
		},
		{
			name: "max used to aggregated detector level scores",
			findings: []Finding{
				{Detector: "A", Score: 100},
				{Detector: "B", Score: 50},
				{Detector: "A", Score: 120},
				{Detector: "C", Score: 75},
				{Detector: "B", Score: -10},
			},
			wantTotalScore: 245,
			wantDetectorScores: map[string]float64{
				"A": 120,
				"B": 50,
				"C": 75,
			},
		},
		{
			name:               "nil findings",
			findings:           nil,
			wantTotalScore:     0,
			wantDetectorScores: map[string]float64{},
		},
		{
			name:               "empty findings",
			findings:           []Finding{},
			wantTotalScore:     0,
			wantDetectorScores: map[string]float64{},
		},
		{
			name: "single detector across all findings",
			findings: []Finding{
				{Detector: "A", Score: 20},
				{Detector: "A", Score: 80},
				{Detector: "A", Score: 50},
			},
			wantTotalScore: 80,
			wantDetectorScores: map[string]float64{
				"A": 80,
			},
		},
		{
			name: "detector names trimmed",
			findings: []Finding{
				{Detector: " A ", Score: 60},
				{Detector: "A", Score: 90},
			},
			wantTotalScore: 90,
			wantDetectorScores: map[string]float64{
				"A": 90,
			},
		},
		{
			name: "blank detector names collapse",
			findings: []Finding{
				{Detector: "", Score: 10},
				{Detector: "   ", Score: 55},
			},
			wantTotalScore: 55,
			wantDetectorScores: map[string]float64{
				"": 55,
			},
		},
		{
			name: "NaN score",
			findings: []Finding{
				{Detector: "A", Score: math.NaN()},
				{Detector: "B", Score: 50},
			},
			wantTotalScore: math.NaN(),
			wantNaN:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalScore, perDetectorScores := ConsolidateScoreByFindings(tt.findings)

			if tt.wantNaN {
				if !math.IsNaN(totalScore) {
					t.Fatalf("total score = %v, want NaN", totalScore)
				}
				return
			}

			if totalScore != tt.wantTotalScore {
				t.Fatalf("total score = %v, want %v", totalScore, tt.wantTotalScore)
			}

			if !reflect.DeepEqual(perDetectorScores, tt.wantDetectorScores) {
				t.Fatalf("per = %#v, want %#v", perDetectorScores, tt.wantDetectorScores)
			}
		})
	}
}

func TestScoreToConfidence(t *testing.T) {
	confidenceLevels := GetDefaultConfidenceLevels()
	tests := []struct {
		name  string
		score float64
		want  Confidence
	}{
		{
			name:  "zero score",
			score: 0,
			want:  ConfidenceLow,
		},
		{
			name:  "normal low score",
			score: 25,
			want:  ConfidenceLow,
		},
		{
			name:  "medium boundary",
			score: 50,
			want:  ConfidenceMedium,
		},
		{
			name:  "high boundary",
			score: 75,
			want:  ConfidenceHigh,
		},
		{
			name:  "maximum score",
			score: 100,
			want:  ConfidenceHigh,
		},
		{
			name:  "negative score",
			score: -1,
			want:  ConfidenceLow,
		},
		{
			name:  "above maximum score",
			score: 101,
			want:  ConfidenceHigh,
		},
		{
			name:  "positive infinity",
			score: math.Inf(1),
			want:  ConfidenceHigh,
		},
		{
			name:  "negative infinity",
			score: math.Inf(-1),
			want:  ConfidenceLow,
		},
		{
			name:  "NaN score",
			score: math.NaN(),
			want:  ConfidenceNone,
		},
		{
			name:  "rounding to low boundary",
			score: confidenceLevels[ConfidenceLow] - 0.4,
			want:  ConfidenceLow,
		},
		{
			name:  "rounding past low boundary",
			score: confidenceLevels[ConfidenceLow] + 0.5,
			want:  ConfidenceMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreToConfidence(confidenceLevels, tt.score)
			if got != tt.want {
				t.Fatalf("confidence=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetConfidenceLevelsFromStrings(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]float64
		wantErr bool
		check   func(t *testing.T, confidenceLevels map[Confidence]float64)
	}{
		{
			name: "full custom mapping",
			input: map[string]float64{
				"low":    30,
				"medium": 60,
				"high":   90,
			},
			wantErr: false,
			check: func(t *testing.T, confidenceLevels map[Confidence]float64) {
				if confidenceLevels[ConfidenceLow] != 30 {
					t.Fatalf("low score mismatch")
				}
				if confidenceLevels[ConfidenceMedium] != 60 {
					t.Fatalf("medium score mismatch")
				}
				if confidenceLevels[ConfidenceHigh] != 90 {
					t.Fatalf("high score mismatch")
				}
			},
		},
		{
			name: "case insensitive keys",
			input: map[string]float64{
				"LOW":    20,
				"Medium": 50,
				"HIGH":   80,
			},
			wantErr: false,
			check: func(t *testing.T, confidenceLevels map[Confidence]float64) {
				if confidenceLevels[ConfidenceLow] != 20 {
					t.Fatalf("low score mismatch")
				}
			},
		},
		{
			name: "keys with whitespace",
			input: map[string]float64{
				" low ": 25,
			},
			wantErr: false,
			check: func(t *testing.T, confidenceLevels map[Confidence]float64) {
				if confidenceLevels[ConfidenceLow] != 25 {
					t.Fatalf("low score mismatch")
				}
			},
		},
		{
			name:    "empty mapping uses defaults",
			input:   map[string]float64{},
			wantErr: false,
			check: func(t *testing.T, confidenceLevels map[Confidence]float64) {
				defaultConfidenceLevels := GetDefaultConfidenceLevels()
				if confidenceLevels[ConfidenceLow] != defaultConfidenceLevels[ConfidenceLow] {
					t.Fatalf("default low mismatch")
				}
			},
		},
		{
			name: "partial mapping fills defaults",
			input: map[string]float64{
				"low": 10,
			},
			wantErr: false,
			check: func(t *testing.T, confidenceLevels map[Confidence]float64) {
				defaultConfidenceLevels := GetDefaultConfidenceLevels()
				if confidenceLevels[ConfidenceLow] != 10 {
					t.Fatalf("custom low missing")
				}
				if confidenceLevels[ConfidenceHigh] != defaultConfidenceLevels[ConfidenceHigh] {
					t.Fatalf("high should use default")
				}
			},
		},
		{
			name: "all levels must be in ascending order",
			input: map[string]float64{
				"low":    40,
				"medium": 30,
				"high":   90,
			},
			wantErr: true,
		},
		{
			name: "levels must be in ascending order even with defaults",
			input: map[string]float64{
				"high": 30,
			},
			wantErr: true,
		},
		{
			name: "levels cannot be equal when defaults are used",
			input: map[string]float64{
				"low":    70,
				"medium": 70,
				"high":   100,
			},
			wantErr: true,
		},
		{
			name: "levels cannot be equal even when defaults are used",
			input: map[string]float64{
				"low": 70,
				// medium defaults to 70
				"high": 100,
			},
			wantErr: true,
		},
		{
			name: "unsupported key",
			input: map[string]float64{
				"critical": 100,
			},
			wantErr: true,
		},
		{
			name: "invalid key does not overwrite existing mapping",
			input: map[string]float64{
				"low":     10,
				"invalid": 20,
			},
			wantErr: true,
		},
		{
			name: "negative threshold accepted",
			input: map[string]float64{
				"low": -10,
			},
			wantErr: false,
			check: func(t *testing.T, confidenceLevels map[Confidence]float64) {
				if confidenceLevels[ConfidenceLow] != -10 {
					t.Fatalf("expected negative threshold to be accepted")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidenceLevels := GetDefaultConfidenceLevels()
			confidenceLevels, err := SetConfidenceLevelsFromStrings(confidenceLevels, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.check != nil {
				tt.check(t, confidenceLevels)
			}
		})
	}
}

func TestConfidenceFromString(t *testing.T) {
	tests := []struct {
		input string
		want  Confidence
		err   bool
	}{
		{"low", ConfidenceLow, false},
		{"1", ConfidenceLow, false},
		{"medium", ConfidenceMedium, false},
		{"2", ConfidenceMedium, false},
		{"high", ConfidenceHigh, false},
		{"3", ConfidenceHigh, false},
		{"HIGH", ConfidenceHigh, false},
		{"  low  ", ConfidenceLow, false},
		{"invalid", 0, true},
		{"4", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		got, err := ConfidenceFromString(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("ConfidenceFromString(%q): err = %v, wantErr = %v", tt.input, err, tt.err)
			continue
		}
		if got != tt.want {
			t.Errorf("ConfidenceFromString(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
