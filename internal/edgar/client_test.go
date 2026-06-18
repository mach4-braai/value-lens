package edgar_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/edgar"
)

func TestFetchSubmissions(t *testing.T) {
	body, err := os.ReadFile("../../testdata/submissions_sample.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	c := edgar.NewClient(srv.URL, "Test test@example.com")
	subs, err := c.FetchSubmissions(context.Background(), "0000320193")
	if err != nil {
		t.Fatalf("FetchSubmissions: %v", err)
	}
	filings := subs.TenKFilings()
	if len(filings) == 0 {
		t.Fatal("expected at least one 10-K filing")
	}
}
