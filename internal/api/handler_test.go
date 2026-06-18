package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devanmcgeer/value-lens/internal/api"
)

func TestQueryEndpointValidation(t *testing.T) {
	h := api.NewHandler(nil)
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/query", "application/json",
		strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHealthEndpoint(t *testing.T) {
	h := api.NewHandler(nil)
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
