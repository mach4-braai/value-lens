package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/devanmcgeer/value-lens/internal/rag"
)

type Handler struct {
	engine *rag.Engine
}

func NewHandler(engine *rag.Engine) *Handler {
	return &Handler{engine: engine}
}

type QueryRequest struct {
	Question string `json:"question"`
	TopK     int    `json:"top_k,omitempty"`
}

type QueryResponse struct {
	Answer  string       `json:"answer"`
	Sources []SourceInfo `json:"sources"`
}

type SourceInfo struct {
	Ticker    string  `json:"ticker"`
	FiledDate string  `json:"filed_date"`
	Section   string  `json:"section"`
	Excerpt   string  `json:"excerpt"`
	Distance  float64 `json:"distance"`
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	r.Post("/api/query", h.handleQuery)
	return r
}

func (h *Handler) handleQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.Question == "" {
		http.Error(w, `{"error":"question is required"}`, http.StatusBadRequest)
		return
	}
	if req.TopK == 0 {
		req.TopK = 5
	}

	result, err := h.engine.Query(r.Context(), req.Question, req.TopK)
	if err != nil {
		http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
		return
	}

	resp := QueryResponse{Answer: result.Answer}
	for _, s := range result.Sources {
		resp.Sources = append(resp.Sources, SourceInfo{
			Ticker:    s.Ticker,
			FiledDate: s.FiledDate,
			Section:   s.Section,
			Excerpt:   truncate(s.Content, 200),
			Distance:  s.Distance,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
