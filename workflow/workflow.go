package workflow

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Jobs map[string]Job `yaml:"jobs"`
}

type Job struct {
	Steps []Step `yaml:"steps"`
}

type Step struct {
	Uses string            `yaml:"uses"`
	With map[string]string `yaml:"with"`
}

// Config represents the extracted AI labels from workflow files.
type Config struct {
	Labels []string `json:"labels"`
}

// DetectLabels scans the .github/workflows directory for the chaoss/disclosure action
// and returns all configured labels. If the action is found but no label is specified,
// it returns the default "ai-detected".
func DetectLabels(repoPath string) (*Config, error) {
	workflowsDir := filepath.Join(repoPath, ".github", "workflows")
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Labels: []string{}}, nil
		}
		return nil, err
	}

	seenLabels := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		path := filepath.Join(workflowsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var wf Workflow
		if err := yaml.Unmarshal(data, &wf); err != nil {
			continue
		}

		for _, job := range wf.Jobs {
			for _, step := range job.Steps {
				uses := strings.TrimSpace(step.Uses)
				if strings.HasPrefix(uses, "chaoss/disclosure@") || uses == "chaoss/disclosure" {
					label := step.With["label"]
					if label == "" {
						label = "ai-detected"
					}
					seenLabels[label] = true
				}
			}
		}
	}

	var labels []string
	for l := range seenLabels {
		labels = append(labels, l)
	}

	if labels == nil {
		labels = []string{} // return empty slice rather than null in json
	}

	return &Config{Labels: labels}, nil
}
