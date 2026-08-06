//go:build ignore

package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/chaoss/disclosure/detection/toolmention/modelgen"
)

func main() {
	resp, err := http.Get("https://openrouter.ai/api/v1/models")
	if err != nil {
		log.Fatalf("failed to fetch models: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Fatalf("failed to fetch models: unexpected HTTP status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("failed to read response: %v", err)
	}
	if strings.TrimSpace(string(body)) == "" {
		log.Fatal("failed to fetch models: empty response body")
	}

	snapshot, err := modelgen.BuildSnapshot(body)
	if err != nil {
		log.Fatalf("failed to build model snapshot: %v", err)
	}

	out, err := os.Create("../models.go")
	if err != nil {
		log.Fatalf("failed to create models.go: %v", err)
	}
	defer out.Close()

	if err := modelgen.RenderModelsGo(out, snapshot); err != nil {
		log.Fatalf("failed to write models.go: %v", err)
	}
}
