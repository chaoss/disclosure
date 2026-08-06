package modelgen

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestBuildSnapshotFromFixture(t *testing.T) {
	fixture := []byte(`{
		"data": [
			{"id": "anthropic/claude-opus-4.6", "name": "Anthropic: Claude Opus 4.6"},
			{"id": "openai/gpt-4-turbo", "name": "OpenAI: GPT-4 Turbo (Preview)"},
			{"id": "cohere/command-r", "name": "Cohere: Command R"},
			{"id": "perplexity/sonar", "name": "Perplexity: Sonar"},
			{"id": "misc/body-builder", "name": "Body Builder"},
			{"id": "misc/bodybuilder", "name": "BodyBuilder"},
			{"id": "misc/uncensored", "name": "Uncensored"},
			{"id": "cohere/command-a", "name": "Cohere: Command A"},
			{"id": "ai21/saba", "name": "Saba"},
			{"id": "openai/o1", "name": "OpenAI: o1"},
			{"id": "openai/o3", "name": "OpenAI: o3"},
			{"id": "openai/GPT-4-TURBO", "name": "OpenAI: GPT-4 Turbo"}
		]
	}`)

	snapshot, err := BuildSnapshot(fixture)
	if err != nil {
		t.Fatal(err)
	}

	wantModels := []string{
		"Claude Opus 4.6",
		"GPT-4 Turbo",
		"claude-opus-4.6",
		"gpt-4-turbo",
	}
	if snapshot.SourceURL != SourceURL {
		t.Errorf("SourceURL = %q, want %q", snapshot.SourceURL, SourceURL)
	}
	if snapshot.SourceItemCount != 12 {
		t.Errorf("SourceItemCount = %d, want 12", snapshot.SourceItemCount)
	}
	if snapshot.ModelCount != len(wantModels) {
		t.Errorf("ModelCount = %d, want %d", snapshot.ModelCount, len(wantModels))
	}
	if !slices.Equal(snapshot.Models, wantModels) {
		t.Errorf("Models = %#v, want %#v", snapshot.Models, wantModels)
	}
	if snapshot.ModelHash == "" {
		t.Error("expected non-empty model hash")
	}
}

func TestBuildSnapshotRejectsEmptyData(t *testing.T) {
	if _, err := BuildSnapshot([]byte(`{"data":[]}`)); err == nil {
		t.Fatal("expected empty data error")
	}
}

func TestRenderModelsGoIncludesMetadata(t *testing.T) {
	snapshot := Snapshot{
		SourceURL:       SourceURL,
		SourceItemCount: 2,
		ModelCount:      2,
		ModelHash:       "abc123",
		Models:          []string{"Claude Opus 4.6", "GPT-4 Turbo"},
	}

	var buf bytes.Buffer
	if err := RenderModelsGo(&buf, snapshot); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	for _, want := range []string{
		"// Source: https://openrouter.ai/api/v1/models",
		"// Source item count: 2",
		"// Normalized model count: 2",
		"// Normalized model SHA256: abc123",
		"var GeneratedModelMentions = []string{",
		`"Claude Opus 4.6"`,
		`"GPT-4 Turbo"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output missing %q:\n%s", want, out)
		}
	}
}
