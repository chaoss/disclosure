package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	Label           string `json:"label"`
	LabelingEnabled string `json:"labeling_enabled"`
	MinConfidence   string `json:"min_confidence"`
	ScanPRBody      string `json:"scan_pr_body"`
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
	configs := []ActionConfig{}

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
						Label:           getString(step.With, "label", "ai-detected"),
						LabelingEnabled: getString(step.With, "labeling-enabled", "false"),
						MinConfidence:   getString(step.With, "min-confidence", "low"),
						ScanPRBody:      getString(step.With, "scan-pr-body", "true"),
					}
					if seenConfigs[ac] {
						continue
					}
					seenConfigs[ac] = true
					configs = append(configs, ac)
				}
			}
		}
	}

	sort.Slice(configs, func(i, j int) bool {
		if configs[i].Label != configs[j].Label {
			return configs[i].Label < configs[j].Label
		}
		if configs[i].LabelingEnabled != configs[j].LabelingEnabled {
			return configs[i].LabelingEnabled < configs[j].LabelingEnabled
		}
		if configs[i].MinConfidence != configs[j].MinConfidence {
			return configs[i].MinConfidence < configs[j].MinConfidence
		}
		return configs[i].ScanPRBody < configs[j].ScanPRBody
	})

	return &Config{Configs: configs}, nil
}
