package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/chaoss/disclosure/detection"
)

func writeCustomToolsTestFile(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunTextCustomTools(t *testing.T) {
	toolsPath := writeCustomToolsTestFile(t, "tools.json", `{"version":1,"custom_tools":[" Example Assistant ","example assistant","CURSOR","GPT-4 Turbo"]}`)
	inputPath := writeCustomToolsTestFile(t, "text.txt", "Example Assistant helped alongside Cursor and GPT-4 Turbo.")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"text", "--custom-tools=" + toolsPath, "--input=" + inputPath, "--format=json"}, &stdout, &stderr)
	if code != ExitAI || stderr.Len() != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, &stderr)
	}
	var report struct {
		Findings   []detection.Finding  `json:"findings"`
		Score      float64              `json:"score"`
		Confidence detection.Confidence `json:"confidence"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode output: %v (%s)", err, &stdout)
	}
	if report.Score != detection.ToolMentionBaseScore || report.Confidence != detection.ConfidenceLow {
		t.Errorf("unexpected aggregate score: %+v", report)
	}
	var names []string
	for _, f := range report.Findings {
		names = append(names, f.Tool)
		if f.Detector != "toolmention" || f.Score != detection.ToolMentionBaseScore || f.Confidence != detection.ConfidenceLow {
			t.Errorf("unexpected finding: %+v", f)
		}
	}
	if want := []string{"Example Assistant", "Cursor", "GPT-4 Turbo"}; !slices.Equal(names, want) {
		t.Errorf("tools = %v, want %v", names, want)
	}
}

func TestRunTextCustomToolsOptIn(t *testing.T) {
	toolsPath := writeCustomToolsTestFile(t, "tools.json", `{"version":1,"custom_tools":["Example Assistant"]}`)
	inputPath := writeCustomToolsTestFile(t, "text.txt", "Example Assistant helped.")
	for _, enabled := range []bool{false, true, false} {
		var stdout, stderr bytes.Buffer
		args := []string{"text", "--input=" + inputPath}
		want := ExitNoAI
		if enabled {
			args = append(args, "--custom-tools="+toolsPath)
			want = ExitAI
		}
		if code := Run(args, &stdout, &stderr); code != want || stderr.Len() != 0 {
			t.Fatalf("enabled=%t: exit = %d, want %d; stderr = %s", enabled, code, want, &stderr)
		}
		if enabled && !strings.Contains(stdout.String(), "Example Assistant") {
			t.Errorf("missing custom name in output: %s", &stdout)
		}
	}
}

func TestRunTextCustomToolsValidation(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{"empty file", "", "parse custom tools file"},
		{"malformed JSON", "{", "parse custom tools file"},
		{"top-level array", "[]", "parse custom tools file"},
		{"null document", "null", "version must be 1"},
		{"missing version", `{"custom_tools":[]}`, "version must be 1"},
		{"unsupported version", `{"version":2,"custom_tools":[]}`, "version must be 1"},
		{"wrong version type", `{"version":"1","custom_tools":[]}`, "parse custom tools file"},
		{"missing list", `{"version":1}`, "custom_tools must be an array"},
		{"null list", `{"version":1,"custom_tools":null}`, "custom_tools must be an array"},
		{"wrong list type", `{"version":1,"custom_tools":"Example Assistant"}`, "parse custom tools file"},
		{"wrong entry type", `{"version":1,"custom_tools":[42]}`, "parse custom tools file"},
		{"structured entry", `{"version":1,"custom_tools":[{"name":"Example Assistant"}]}`, "parse custom tools file"},
		{"null entry", `{"version":1,"custom_tools":[null]}`, "custom_tools[0] must be a non-empty name"},
		{"empty entry", `{"version":1,"custom_tools":[""]}`, "custom_tools[0] must be a non-empty name"},
		{"blank entry", `{"version":1,"custom_tools":[" \t\n"]}`, "custom_tools[0] must be a non-empty name"},
		{"unknown field", `{"version":1,"custom_tools":[],"models":[]}`, "unknown field"},
		{"trailing document", `{"version":1,"custom_tools":[]} {}`, "single JSON object"},
		{"trailing garbage", `{"version":1,"custom_tools":[]} oops`, "single JSON object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeCustomToolsTestFile(t, "tools.json", tt.json)
			var stdout, stderr bytes.Buffer
			// Configuration errors must be reported before reading input.
			code := Run([]string{"text", "--custom-tools=" + path, "--input=nonexistent-input.txt"}, &stdout, &stderr)
			if code != ExitError || !strings.Contains(stderr.String(), tt.want) || stdout.Len() != 0 {
				t.Fatalf("exit = %d, stdout = %s, stderr = %s; want error containing %q", code, &stdout, &stderr, tt.want)
			}
		})
	}
}

func TestRunTextCustomToolsUnreadableFile(t *testing.T) {
	for _, path := range []string{"", t.TempDir(), filepath.Join(t.TempDir(), "missing.json")} {
		var stdout, stderr bytes.Buffer
		code := Run([]string{"text", "--custom-tools=" + path, "--input=nonexistent-input.txt"}, &stdout, &stderr)
		if code != ExitError || !strings.Contains(stderr.String(), "read custom tools file") || stdout.Len() != 0 {
			t.Errorf("path %q: exit = %d, stdout = %s, stderr = %s", path, code, &stdout, &stderr)
		}
	}
}

func TestRunTextEmptyCustomTools(t *testing.T) {
	toolsPath := writeCustomToolsTestFile(t, "tools.json", "{\"version\":1,\"custom_tools\":[]}\n")
	for _, input := range []string{"A normal contribution", "Used Cursor"} {
		inputPath := writeCustomToolsTestFile(t, "text.txt", input)
		var defaultOut, defaultErr, customOut, customErr bytes.Buffer
		args := []string{"text", "--input=" + inputPath, "--format=json"}
		defaultCode := Run(args, &defaultOut, &defaultErr)
		customCode := Run(append(args, "--custom-tools="+toolsPath), &customOut, &customErr)
		if customCode != defaultCode || customOut.String() != defaultOut.String() || customErr.String() != defaultErr.String() {
			t.Errorf("empty list changed detection for %q: default (%d, %s, %s), custom (%d, %s, %s)", input, defaultCode, &defaultOut, &defaultErr, customCode, &customOut, &customErr)
		}
	}
}

func TestRunTextCustomToolsStdinAndCheckboxes(t *testing.T) {
	toolsPath := writeCustomToolsTestFile(t, "tools.json", `{"version":1,"custom_tools":["Example Assistant"]}`)
	inputPath := writeCustomToolsTestFile(t, "text.txt", "[x] Used AI\nExample Assistant")
	input, err := os.Open(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	oldStdin := os.Stdin
	os.Stdin = input
	t.Cleanup(func() {
		os.Stdin = oldStdin
		input.Close()
	})
	var stdout, stderr bytes.Buffer
	code := Run([]string{"text", "--custom-tools=" + toolsPath, "--enable-checkbox-detection", "--cb-disclosed-ai=Used AI", "--format=json"}, &stdout, &stderr)
	if code != ExitAI || stderr.Len() != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, &stderr)
	}
	var report struct {
		Findings   []detection.Finding  `json:"findings"`
		Score      float64              `json:"score"`
		Confidence detection.Confidence `json:"confidence"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	wantScore := detection.ToolMentionBaseScore + detection.CheckboxAIUsedBaseScore
	if len(report.Findings) != 1 || report.Findings[0].Tool != "Example Assistant" || report.Findings[0].Score != wantScore || report.Score != wantScore || report.Confidence != detection.ConfidenceHigh {
		t.Fatalf("unexpected report: %+v", report)
	}
}
