package parser_test

import (
	"os"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/parser"
)

func TestParseTenK(t *testing.T) {
	data, err := os.ReadFile("../../testdata/tenk_sample.htm")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	sections, err := parser.ParseTenK(data)
	if err != nil {
		t.Fatalf("ParseTenK: %v", err)
	}

	if len(sections) == 0 {
		t.Fatal("expected at least one section")
	}

	keys := map[string]bool{}
	for _, s := range sections {
		keys[s.ID] = true
	}
	for _, want := range []string{"item_1", "item_1a", "item_7"} {
		if !keys[want] {
			t.Errorf("missing section %s", want)
		}
	}

	for _, s := range sections {
		if s.ID == "item_1a" && len(s.Text) == 0 {
			t.Error("item_1a has empty text")
		}
	}
}
