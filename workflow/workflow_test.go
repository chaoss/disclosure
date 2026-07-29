package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectConfigs(t *testing.T) {
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}

	yamlContent := `
jobs:
  test:
    steps:
      - uses: actions/checkout@v4
      - uses: chaoss/disclosure@main
        with:
          label: custom-ai-label
          labeling-enabled: "true"
          min-confidence: medium
          scan-pr-body: "false"
`
	if err := os.WriteFile(filepath.Join(workflowsDir, "test.yml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := DetectConfigs(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(config.Configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(config.Configs))
	}

	ac := config.Configs[0]
	if ac.Label != "custom-ai-label" {
		t.Errorf("expected label custom-ai-label, got %v", ac.Label)
	}
	if ac.LabelingEnabled != "true" {
		t.Errorf("expected labeling-enabled true, got %v", ac.LabelingEnabled)
	}
	if ac.MinConfidence != "medium" {
		t.Errorf("expected min-confidence medium, got %v", ac.MinConfidence)
	}
	if ac.ScanPRBody != "false" {
		t.Errorf("expected scan-pr-body false, got %v", ac.ScanPRBody)
	}
}

func TestDetectConfigsDefault(t *testing.T) {
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}

	yamlContent := `
jobs:
  test:
    steps:
      - uses: chaoss/disclosure@v1
`
	if err := os.WriteFile(filepath.Join(workflowsDir, "test.yml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := DetectConfigs(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(config.Configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(config.Configs))
	}

	ac := config.Configs[0]
	if ac.Label != "ai-detected" {
		t.Errorf("expected label ai-detected, got %v", ac.Label)
	}
	if ac.LabelingEnabled != "false" {
		t.Errorf("expected labeling-enabled false, got %v", ac.LabelingEnabled)
	}
	if ac.MinConfidence != "low" {
		t.Errorf("expected min-confidence low, got %v", ac.MinConfidence)
	}
	if ac.ScanPRBody != "true" {
		t.Errorf("expected scan-pr-body true, got %v", ac.ScanPRBody)
	}
}

func TestDetectConfigsNoWorkflows(t *testing.T) {
	tempDir := t.TempDir()
	config, err := DetectConfigs(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Configs) != 0 {
		t.Fatalf("expected empty configs, got %v", config.Configs)
	}
}

func TestDetectConfigsMultipleSortedAndDeduplicated(t *testing.T) {
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}

	firstWorkflow := `
jobs:
  test:
    steps:
      - uses: chaoss/disclosure@main
        with:
          label: z-ai
          labeling-enabled: "true"
          min-confidence: high
          scan-pr-body: "false"
`
	secondWorkflow := `
jobs:
  test:
    steps:
      - uses: chaoss/disclosure@main
        with:
          label: a-ai
          labeling-enabled: "false"
          min-confidence: low
          scan-pr-body: "true"
      - uses: chaoss/disclosure@main
        with:
          label: z-ai
          labeling-enabled: "true"
          min-confidence: high
          scan-pr-body: "false"
`
	if err := os.WriteFile(filepath.Join(workflowsDir, "z.yml"), []byte(firstWorkflow), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowsDir, "a.yml"), []byte(secondWorkflow), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := DetectConfigs(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	want := []ActionConfig{
		{
			Label:           "a-ai",
			LabelingEnabled: "false",
			MinConfidence:   "low",
			ScanPRBody:      "true",
		},
		{
			Label:           "z-ai",
			LabelingEnabled: "true",
			MinConfidence:   "high",
			ScanPRBody:      "false",
		},
	}
	if len(config.Configs) != len(want) {
		t.Fatalf("expected %d configs, got %d: %#v", len(want), len(config.Configs), config.Configs)
	}
	for i := range want {
		if config.Configs[i] != want[i] {
			t.Errorf("config[%d] = %#v, want %#v", i, config.Configs[i], want[i])
		}
	}
}
