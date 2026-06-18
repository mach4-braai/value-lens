package chunker

import "strings"

type Options struct {
	MaxTokens int // approximate max tokens per chunk
	Overlap   int // number of overlapping tokens between chunks
}

type Result struct {
	Text       string
	Index      int
	TokenCount int
}

// Chunk splits text into overlapping chunks. Uses whitespace tokenization
// as a rough approximation — accurate enough for embedding purposes.
func Chunk(text string, opts Options) []Result {
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 512
	}
	words := strings.Fields(text)
	if len(words) <= opts.MaxTokens {
		return []Result{{Text: text, Index: 0, TokenCount: len(words)}}
	}

	var chunks []Result
	step := opts.MaxTokens - opts.Overlap
	if step <= 0 {
		step = opts.MaxTokens / 2
	}

	idx := 0
	for start := 0; start < len(words); start += step {
		end := start + opts.MaxTokens
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[start:end], " ")
		chunks = append(chunks, Result{
			Text:       chunk,
			Index:      idx,
			TokenCount: end - start,
		})
		idx++
		if end == len(words) {
			break
		}
	}
	return chunks
}
