package rag_test

import (
	"testing"

	"github.com/devanmcgeer/value-lens/internal/rag"
)

func TestBuildPrompt(t *testing.T) {
	chunks := []rag.RetrievedChunk{
		{Content: "Apple faces significant competition.", Section: "item_1a", Ticker: "AAPL", FiledDate: "2024-11-01"},
		{Content: "Revenue was $394.3 billion.", Section: "item_7", Ticker: "AAPL", FiledDate: "2024-11-01"},
	}
	prompt := rag.BuildPrompt("What are Apple's main risks?", chunks)

	if len(prompt) == 0 {
		t.Fatal("expected non-empty prompt")
	}
	if !containsAll(prompt, "Apple's main risks", "significant competition", "394.3 billion") {
		t.Errorf("prompt missing expected content: %s", prompt[:200])
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
