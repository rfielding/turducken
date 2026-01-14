package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rfielding/turducken/pkg/llm"
	"github.com/rfielding/turducken/pkg/prolog"
)

//go:embed static/*
var staticFiles embed.FS

// Server is the main HTTP server for turducken
type Server struct {
	engine   *prolog.Engine
	llm      *llm.Client
	specFile string
}

// New creates a new server instance
func New(specFile string) (*Server, error) {
	engine, err := prolog.New()
	if err != nil {
		return nil, fmt.Errorf("creating prolog engine: %w", err)
	}

	s := &Server{
		engine:   engine,
		llm:      llm.New(),
		specFile: specFile,
	}

	// Load spec file if provided
	if specFile != "" {
		content, err := os.ReadFile(specFile)
		if err != nil {
			return nil, fmt.Errorf("reading spec file: %w", err)
		}
		if err := engine.LoadSpec(string(content)); err != nil {
			return nil, fmt.Errorf("loading spec: %w", err)
		}
	}

	return s, nil
}

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/spec", s.handleSpec)
	mux.HandleFunc("/api/query", s.handleQuery)
	mux.HandleFunc("/api/visualize", s.handleVisualize)
	mux.HandleFunc("/api/chat", s.handleChat)
	mux.HandleFunc("/api/check", s.handleCheck)
	mux.HandleFunc("/api/reset", s.handleReset)
	mux.HandleFunc("/api/provider", s.handleProvider)
	mux.HandleFunc("/api/properties", s.handleProperties)
	mux.HandleFunc("/api/docs", s.handleDocs)
	mux.HandleFunc("/api/actors", s.handleActors)

	// Static files (embedded)
	mux.HandleFunc("/", s.handleStatic)

	return http.ListenAndServe(addr, mux)
}

// handleSpec handles GET/POST for the Prolog specification
func (s *Server) handleSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(map[string]string{
			"source": s.engine.GetSource(),
		})

	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req struct {
			Source string `json:"source"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Reset and reload
		if err := s.engine.Reset(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := s.engine.LoadSpec(req.Source); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    false,
				"error":      err.Error(),
				"source":     req.Source, // Return source for retry
				"canAutoFix": true,       // Signal UI can try auto-fix
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleQuery executes a raw Prolog query
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := s.engine.RawQuery(ctx, req.Query)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

// handleVisualize generates visualization data from the current spec
func (s *Server) handleVisualize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	visType := r.URL.Query().Get("type")
	if visType == "" {
		visType = "all"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result := make(map[string]interface{})

	if visType == "statemachine" || visType == "all" {
		sm, err := s.extractStateMachine(ctx)
		if err != nil {
			log.Printf("Error extracting state machine: %v", err)
		} else {
			result["stateMachine"] = sm
		}
	}

	if visType == "sequence" || visType == "all" {
		seq, err := s.extractSequence(ctx)
		if err != nil {
			log.Printf("Error extracting sequence: %v", err)
		} else {
			result["sequence"] = seq
		}
	}

	if visType == "pie" || visType == "all" {
		pie, err := s.extractPie(ctx)
		if err != nil {
			log.Printf("Error extracting pie: %v", err)
		} else {
			result["pie"] = pie
		}
	}

	if visType == "line" || visType == "all" {
		line, err := s.extractLine(ctx)
		if err != nil {
			log.Printf("Error extracting line: %v", err)
		} else {
			result["line"] = line
		}
	}

	json.NewEncoder(w).Encode(result)
}

func (s *Server) extractStateMachine(ctx context.Context) (map[string]interface{}, error) {
	sm, err := s.engine.GetStateMachine(ctx)
	if err != nil {
		return nil, err
	}

	// Convert transitions to map format for JSON
	transitions := make([]map[string]string, len(sm.Transitions))
	for i, t := range sm.Transitions {
		transitions[i] = map[string]string{
			"from":  t.From,
			"label": t.Label,
			"to":    t.To,
		}
	}

	return map[string]interface{}{
		"states":      sm.States,
		"transitions": transitions,
		"initial":     sm.Initial,
		"accepting":   sm.Accepting,
	}, nil
}

func (s *Server) extractSequence(ctx context.Context) (map[string]interface{}, error) {
	seq, err := s.engine.GetSequenceDiagram(ctx)
	if err != nil {
		return nil, err
	}

	// Convert messages to map format for JSON
	messages := make([]map[string]interface{}, len(seq.Messages))
	for i, m := range seq.Messages {
		messages[i] = map[string]interface{}{
			"seq":   m.Seq,
			"from":  m.From,
			"to":    m.To,
			"label": m.Label,
		}
	}

	return map[string]interface{}{
		"lifelines": seq.Lifelines,
		"messages":  messages,
	}, nil
}

func (s *Server) extractPie(ctx context.Context) (map[string]interface{}, error) {
	slices, err := s.engine.GetPieChart(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to map format for JSON
	sliceData := make([]map[string]interface{}, len(slices))
	for i, slice := range slices {
		sliceData[i] = map[string]interface{}{
			"label": slice.Label,
			"value": slice.Value,
		}
	}

	return map[string]interface{}{
		"slices": sliceData,
	}, nil
}

func (s *Server) extractLine(ctx context.Context) (map[string]interface{}, error) {
	points, err := s.engine.GetLineChart(ctx)
	if err != nil {
		return nil, err
	}

	// Group points by series
	seriesMap := make(map[string][]map[string]float64)
	for _, p := range points {
		seriesMap[p.Series] = append(seriesMap[p.Series], map[string]float64{
			"x": p.X,
			"y": p.Y,
		})
	}

	// Convert to array format
	var series []map[string]interface{}
	for name, pts := range seriesMap {
		series = append(series, map[string]interface{}{
			"name":   name,
			"points": pts,
		})
	}

	return map[string]interface{}{
		"series": series,
	}, nil
}

// handleChat handles LLM chat requests
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Message string `json:"message"`
		Context string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Build prompt with current spec context
	currentSpec := s.engine.GetSource()
	prompt := s.llm.BuildPrompt(req.Message, currentSpec, req.Context)

	// Get LLM response
	response, err := s.llm.Chat(r.Context(), prompt)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Extract Prolog code from response
	prologCode := extractPrologCode(response)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"response": response,
		"prolog":   prologCode,
	})
}

// handleCheck checks a CTL property
func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Property string `json:"property"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Build CTL check query
	query := fmt.Sprintf("check_ctl(%s).", req.Property)

	satisfied, err := s.engine.QueryOne(ctx, query)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"satisfied": satisfied,
	})
}

// handleReset resets the Prolog engine
func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := s.engine.Reset(); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Reload spec file if one was provided
	if s.specFile != "" {
		content, err := os.ReadFile(s.specFile)
		if err == nil {
			s.engine.LoadSpec(string(content))
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleProvider handles GET/POST for LLM provider settings
func (s *Server) handleProvider(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"provider": string(s.llm.GetProvider()),
			"name":     s.llm.ProviderName(),
			"hasKey":   s.llm.HasAPIKey(),
		})

	case http.MethodPost:
		var req struct {
			Provider string `json:"provider"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch req.Provider {
		case "openai":
			s.llm.SetProvider(llm.ProviderOpenAI)
		case "anthropic":
			s.llm.SetProvider(llm.ProviderAnthropic)
		default:
			http.Error(w, "Unknown provider", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"provider": string(s.llm.GetProvider()),
			"name":     s.llm.ProviderName(),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProperties returns named properties from the spec
func (s *Server) handleProperties(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	properties, err := s.engine.GetProperties(ctx)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"properties": properties,
	})
}

// handleDocs returns documentation from the spec
func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	docs, err := s.engine.GetDocs(ctx)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"docs":    docs,
	})
}

// handleActors returns actor list from the spec
func (s *Server) handleActors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	actors, err := s.engine.GetActors(ctx)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"actors":  actors,
	})
}

// handleStatic serves static files
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	content, err := staticFiles.ReadFile("static" + path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set content type
	switch {
	case strings.HasSuffix(path, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(path, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	}

	w.Write(content)
}

// extractPrologCode extracts Prolog code blocks from LLM response
func extractPrologCode(response string) string {
	// Look for ```prolog ... ``` blocks
	start := strings.Index(response, "```prolog")
	if start == -1 {
		start = strings.Index(response, "```Prolog")
	}
	if start == -1 {
		return ""
	}

	start = strings.Index(response[start:], "\n") + start + 1
	end := strings.Index(response[start:], "```")
	if end == -1 {
		return ""
	}

	return strings.TrimSpace(response[start : start+end])
}
