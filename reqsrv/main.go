package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"reqsrv/internal/llm"
	"reqsrv/internal/prolog"
	"reqsrv/internal/store"
)

//go:embed web/*
var webFS embed.FS

type Server struct {
	Docs   *store.DocStore
	Kernel *prolog.Kernel
	LLM    *llm.Client
}

func main() {
	addr := env("ADDR", ":8080")
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}

	s := &Server{
		Docs:   store.NewDocStore(),
		Kernel: prolog.NewKernel(),
		LLM:    llm.NewClient(apiKey),
	}

	mux := http.NewServeMux()

	// UI
	mux.Handle("/", http.FileServer(http.FS(sub)))

	// API
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/chat", s.handleChat)
	mux.HandleFunc("/api/docs", s.handleSetDoc)
	mux.HandleFunc("/api/docs/", s.handleGetDoc)

	srv := &http.Server{
		Addr:              addr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"docs": s.Docs.List(),
	}
	writeJSON(w, resp)
}

type chatReq struct {
	Message string `json:"message"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if req.Message == "" {
		http.Error(w, "message required", 400)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	out, err := s.LLM.Chat(ctx, llm.ChatInput{
		UserText:      req.Message,
		Docs:          s.Docs.Snapshot(),
		PrologSummary: s.Kernel.Summary(ctx),
		ToolRunner: func(ctx context.Context, call llm.ToolCall) (llm.ToolResult, error) {
			return llm.RunTool(ctx, call, s.Docs, s.Kernel)
		},
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "timeout", 504)
			return
		}
		http.Error(w, err.Error(), 500)
		return
	}

	writeJSON(w, map[string]any{
		"assistant": out.Text,
		"docs":      s.Docs.List(),
	})
}

type setDocReq struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Mime    string `json:"mime"`
}

func (s *Server) handleSetDoc(w http.ResponseWriter, r *http.Request) {
	var req setDocReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if req.ID == "" {
		http.Error(w, "id required", 400)
		return
	}
	if req.Mime == "" {
		req.Mime = "text/markdown"
	}
	s.Docs.Put(req.ID, req.Title, req.Mime, req.Content)
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleGetDoc(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/docs/"):]
	doc, ok := s.Docs.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", doc.Mime)
	_, _ = w.Write([]byte(doc.Content))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
