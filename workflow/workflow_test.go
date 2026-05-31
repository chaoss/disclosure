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
