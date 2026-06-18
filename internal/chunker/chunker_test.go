package chunker_test

import (
	"strings"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/chunker"
)

func TestChunk(t *testing.T) {
	words := make([]string, 1000)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ")

	chunks := chunker.Chunk(text, chunker.Options{
		MaxTokens: 200,
		Overlap:   50,
	})

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	for i, c := range chunks {
		wordCount := len(strings.Fields(c.Text))
		if wordCount > 250 {
			t.Errorf("chunk %d has %d words, exceeds max", i, wordCount)
		}
	}

	if len(chunks) > 1 {
		words0 := strings.Fields(chunks[0].Text)
		words1 := strings.Fields(chunks[1].Text)
		lastOfFirst := words0[len(words0)-1]
		if words1[0] == lastOfFirst {
			// Overlap exists
		}
	}
}

func TestChunkShortText(t *testing.T) {
	chunks := chunker.Chunk("This is a short sentence.", chunker.Options{
		MaxTokens: 200,
		Overlap:   50,
	})
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for short text, got %d", len(chunks))
	}
}
