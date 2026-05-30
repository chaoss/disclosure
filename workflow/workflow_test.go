package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLabels(t *testing.T) {
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
`
	if err := os.WriteFile(filepath.Join(workflowsDir, "test.yml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := DetectLabels(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(config.Labels) != 1 || config.Labels[0] != "custom-ai-label" {
		t.Fatalf("expected [custom-ai-label], got %v", config.Labels)
	}
}

func TestDetectLabelsDefault(t *testing.T) {
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

	config, err := DetectLabels(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(config.Labels) != 1 || config.Labels[0] != "ai-detected" {
		t.Fatalf("expected [ai-detected], got %v", config.Labels)
	}
}

func TestDetectLabelsNoWorkflows(t *testing.T) {
	tempDir := t.TempDir()
	config, err := DetectLabels(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Labels) != 0 {
		t.Fatalf("expected [], got %v", config.Labels)
	}
}
