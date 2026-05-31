package workflow

import (
	"fmt"
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
	Uses string                 `yaml:"uses"`
	With map[string]interface{} `yaml:"with"`
}

// ActionConfig represents the extracted configuration for a single use of the action.
type ActionConfig struct {
	Label         string `json:"label"`
	MinConfidence string `json:"min_confidence"`
	ScanPRBody    string `json:"scan_pr_body"`
}

// Config represents the extracted AI configurations from workflow files.
type Config struct {
	Configs []ActionConfig `json:"configs"`
}

func getString(m map[string]interface{}, key, def string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return def
}

// DetectConfigs scans the .github/workflows directory for the chaoss/disclosure action
// and returns all configured instances. Default values are populated for missing inputs.
func DetectConfigs(repoPath string) (*Config, error) {
	workflowsDir := filepath.Join(repoPath, ".github", "workflows")
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Configs: []ActionConfig{}}, nil
		}
		return nil, err
	}

	seenConfigs := make(map[ActionConfig]bool)

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
					ac := ActionConfig{
						Label:         getString(step.With, "label", "ai-detected"),
						MinConfidence: getString(step.With, "min-confidence", "low"),
						ScanPRBody:    getString(step.With, "scan-pr-body", "true"),
					}
					seenConfigs[ac] = true
				}
			}
		}
	}

	var configs []ActionConfig
	for c := range seenConfigs {
		configs = append(configs, c)
	}

	if configs == nil {
		configs = []ActionConfig{}
	}

	return &Config{Configs: configs}, nil
}
