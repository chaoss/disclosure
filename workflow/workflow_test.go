package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func writeWorkflow(t *testing.T, repoPath, name, content string) {
	t.Helper()

	workflowsDir := filepath.Join(repoPath, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowsDir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertConfig(t *testing.T, got, want ActionConfig) {
	t.Helper()

	if got.SourceFile != want.SourceFile {
		t.Errorf("source_file = %q, want %q", got.SourceFile, want.SourceFile)
	}
	if got.ActionRef != want.ActionRef {
		t.Errorf("action_ref = %q, want %q", got.ActionRef, want.ActionRef)
	}
	if got.Label != want.Label {
		t.Errorf("label = %q, want %q", got.Label, want.Label)
	}
	if got.LabelingEnabled != want.LabelingEnabled {
		t.Errorf("labeling_enabled = %q, want %q", got.LabelingEnabled, want.LabelingEnabled)
	}
	if got.MinConfidence != want.MinConfidence {
		t.Errorf("min_confidence = %q, want %q", got.MinConfidence, want.MinConfidence)
	}
	if got.ScanPRBody != want.ScanPRBody {
		t.Errorf("scan_pr_body = %q, want %q", got.ScanPRBody, want.ScanPRBody)
	}
	if !sameStrings(got.ExplicitInputs, want.ExplicitInputs) {
		t.Errorf("explicit_inputs = %v, want %v", got.ExplicitInputs, want.ExplicitInputs)
	}
	if !sameStrings(got.DefaultedInputs, want.DefaultedInputs) {
		t.Errorf("defaulted_inputs = %v, want %v", got.DefaultedInputs, want.DefaultedInputs)
	}
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func TestDetectConfigs(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflow(t, tempDir, "test.yml", `
jobs:
  test:
    steps:
      - uses: actions/checkout@v4
      - uses: chaoss/disclosure/action@main
        with:
          label: custom-ai-label
          labeling-enabled: "true"
          min-confidence: medium
          scan-pr-body: "false"
`)

	config, err := DetectConfigs(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(config.Configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(config.Configs))
	}
	if config.Incomplete {
		t.Fatal("expected complete result")
	}
	assertConfig(t, config.Configs[0], ActionConfig{
		SourceFile:      ".github/workflows/test.yml",
		ActionRef:       "main",
		Label:           "custom-ai-label",
		LabelingEnabled: "true",
		MinConfidence:   "medium",
		ScanPRBody:      "false",
		ExplicitInputs:  []string{"label", "labeling-enabled", "min-confidence", "scan-pr-body"},
		DefaultedInputs: nil,
	})
}

func TestDetectConfigsDefault(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflow(t, tempDir, "test.yml", `
jobs:
  test:
    steps:
      - uses: chaoss/disclosure/action@v1
`)

	config, err := DetectConfigs(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(config.Configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(config.Configs))
	}
	assertConfig(t, config.Configs[0], ActionConfig{
		SourceFile:      ".github/workflows/test.yml",
		ActionRef:       "v1",
		Label:           "ai-detected",
		LabelingEnabled: "false",
		MinConfidence:   "low",
		ScanPRBody:      "true",
		ExplicitInputs:  nil,
		DefaultedInputs: []string{"label", "labeling-enabled", "min-confidence", "scan-pr-body"},
	})
}

func TestDetectConfigsReadmeSyntax(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflow(t, tempDir, "ai_detection.yml", `
name: AI Disclosure
on:
  pull_request:
jobs:
  disclose:
    runs-on: ubuntu-latest
    steps:
      - uses: chaoss/disclosure/action@main
`)

	config, err := DetectConfigs(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(config.Configs) != 1 {
		t.Fatalf("expected 1 config, got %d: %#v", len(config.Configs), config.Configs)
	}
	assertConfig(t, config.Configs[0], ActionConfig{
		SourceFile:      ".github/workflows/ai_detection.yml",
		ActionRef:       "main",
		Label:           "ai-detected",
		LabelingEnabled: "false",
		MinConfidence:   "low",
		ScanPRBody:      "true",
		ExplicitInputs:  nil,
		DefaultedInputs: []string{"label", "labeling-enabled", "min-confidence", "scan-pr-body"},
	})
}

func TestDetectConfigsIgnoresRepositoryRootReference(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflow(t, tempDir, "test.yml", `
jobs:
  test:
    steps:
      - uses: chaoss/disclosure@main
`)

	config, err := DetectConfigs(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Configs) != 0 {
		t.Fatalf("expected empty configs, got %#v", config.Configs)
	}
}

func TestDetectConfigsNoWorkflows(t *testing.T) {
	config, err := DetectConfigs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Configs) != 0 {
		t.Fatalf("expected empty configs, got %v", config.Configs)
	}
	if config.Incomplete {
		t.Fatal("expected complete result")
	}
}

func TestDetectConfigsMultipleSortedAndDeduplicated(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflow(t, tempDir, "z.yml", `
jobs:
  test:
    steps:
      - uses: chaoss/disclosure/action@main
        with:
          label: z-ai
          labeling-enabled: "true"
          min-confidence: high
          scan-pr-body: "false"
`)
	writeWorkflow(t, tempDir, "a.yml", `
jobs:
  test:
    steps:
      - uses: chaoss/disclosure/action@main
        with:
          label: a-ai
          labeling-enabled: "false"
          min-confidence: low
          scan-pr-body: "true"
      - uses: chaoss/disclosure/action@main
        with:
          label: a-ai
          labeling-enabled: "false"
          min-confidence: low
          scan-pr-body: "true"
`)

	config, err := DetectConfigs(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	want := []ActionConfig{
		{
			SourceFile:      ".github/workflows/a.yml",
			ActionRef:       "main",
			Label:           "a-ai",
			LabelingEnabled: "false",
			MinConfidence:   "low",
			ScanPRBody:      "true",
			ExplicitInputs:  []string{"label", "labeling-enabled", "min-confidence", "scan-pr-body"},
			DefaultedInputs: nil,
		},
		{
			SourceFile:      ".github/workflows/z.yml",
			ActionRef:       "main",
			Label:           "z-ai",
			LabelingEnabled: "true",
			MinConfidence:   "high",
			ScanPRBody:      "false",
			ExplicitInputs:  []string{"label", "labeling-enabled", "min-confidence", "scan-pr-body"},
			DefaultedInputs: nil,
		},
	}
	if len(config.Configs) != len(want) {
		t.Fatalf("expected %d configs, got %d: %#v", len(want), len(config.Configs), config.Configs)
	}
	for i := range want {
		assertConfig(t, config.Configs[i], want[i])
	}
}

func TestDetectConfigsWarningsForInvalidWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflow(t, tempDir, "valid.yml", `
jobs:
  test:
    steps:
      - uses: chaoss/disclosure/action@main
`)
	writeWorkflow(t, tempDir, "invalid.yml", `
jobs:
  test:
    steps:
      - uses: [
`)

	config, err := DetectConfigs(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(config.Configs))
	}
	if !config.Incomplete {
		t.Fatal("expected incomplete result")
	}
	if len(config.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %#v", config.Warnings)
	}
	if config.Warnings[0].SourceFile != ".github/workflows/invalid.yml" {
		t.Errorf("warning source_file = %q", config.Warnings[0].SourceFile)
	}
	if config.Warnings[0].Error == "" {
		t.Error("expected warning error")
	}
}
