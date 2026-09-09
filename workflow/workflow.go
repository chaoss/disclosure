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
	SourceFile      string   `json:"source_file"`
	ActionRef       string   `json:"action_ref"`
	Label           string   `json:"label"`
	LabelingEnabled string   `json:"labeling_enabled"`
	MinConfidence   string   `json:"min_confidence"`
	ScanPRBody      string   `json:"scan_pr_body"`
	ExplicitInputs  []string `json:"explicit_inputs"`
	DefaultedInputs []string `json:"defaulted_inputs"`
}

// Config represents the extracted AI configurations from workflow files.
type Config struct {
	Configs    []ActionConfig `json:"configs"`
	Warnings   []Warning      `json:"warnings,omitempty"`
	Incomplete bool           `json:"incomplete"`
}

// Warning represents a recoverable workflow parsing or read problem.
type Warning struct {
	SourceFile string `json:"source_file"`
	Error      string `json:"error"`
}

const disclosureActionUse = "chaoss/disclosure/action"

var actionInputDefaults = map[string]string{
	"label":            "ai-detected",
	"labeling-enabled": "false",
	"min-confidence":   "low",
	"scan-pr-body":     "true",
}

func actionRef(uses string) (string, bool) {
	uses = strings.TrimSpace(uses)
	if uses == disclosureActionUse {
		return "", true
	}
	if strings.HasPrefix(uses, disclosureActionUse+"@") {
		return strings.TrimPrefix(uses, disclosureActionUse+"@"), true
	}
	return "", false
}

func getInput(m map[string]interface{}, key string) (string, bool) {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v), true
	}
	return actionInputDefaults[key], false
}

func actionConfig(sourceFile, ref string, with map[string]interface{}) ActionConfig {
	explicitInputs := []string{}
	defaultedInputs := []string{}
	values := map[string]string{}

	inputKeys := []string{"label", "labeling-enabled", "min-confidence", "scan-pr-body"}
	for _, key := range inputKeys {
		value, explicit := getInput(with, key)
		values[key] = value
		if explicit {
			explicitInputs = append(explicitInputs, key)
		} else {
			defaultedInputs = append(defaultedInputs, key)
		}
	}

	return ActionConfig{
		SourceFile:      sourceFile,
		ActionRef:       ref,
		Label:           values["label"],
		LabelingEnabled: values["labeling-enabled"],
		MinConfidence:   values["min-confidence"],
		ScanPRBody:      values["scan-pr-body"],
		ExplicitInputs:  explicitInputs,
		DefaultedInputs: defaultedInputs,
	}
}

func configKey(ac ActionConfig) string {
	parts := []string{
		ac.SourceFile,
		ac.ActionRef,
		ac.Label,
		ac.LabelingEnabled,
		ac.MinConfidence,
		ac.ScanPRBody,
		strings.Join(ac.ExplicitInputs, ","),
		strings.Join(ac.DefaultedInputs, ","),
	}
	return strings.Join(parts, "\x00")
}

func sortConfigs(configs []ActionConfig) {
	sort.Slice(configs, func(i, j int) bool {
		return configKey(configs[i]) < configKey(configs[j])
	})
}

func detectConfigsInWorkflow(sourceFile string, data []byte) ([]ActionConfig, error) {
	var wf Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, err
	}

	var configs []ActionConfig
	for _, job := range wf.Jobs {
		for _, step := range job.Steps {
			ref, ok := actionRef(step.Uses)
			if !ok {
				continue
			}
			configs = append(configs, actionConfig(sourceFile, ref, step.With))
		}
	}
	return configs, nil
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

	seenConfigs := make(map[string]bool)
	configs := []ActionConfig{}
	warnings := []Warning{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		sourceFile := filepath.ToSlash(filepath.Join(".github", "workflows", entry.Name()))
		path := filepath.Join(workflowsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			warnings = append(warnings, Warning{SourceFile: sourceFile, Error: err.Error()})
			continue
		}

		fileConfigs, err := detectConfigsInWorkflow(sourceFile, data)
		if err != nil {
			warnings = append(warnings, Warning{SourceFile: sourceFile, Error: err.Error()})
			continue
		}
		for _, ac := range fileConfigs {
			key := configKey(ac)
			if seenConfigs[key] {
				continue
			}
			seenConfigs[key] = true
			configs = append(configs, ac)
		}
	}

	sortConfigs(configs)

	return &Config{Configs: configs, Warnings: warnings, Incomplete: len(warnings) > 0}, nil
}
