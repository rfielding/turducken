package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
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
	mux      *http.ServeMux

	// Metrics
	mu         sync.RWMutex
	counters   map[string]int64
	timeSeries []TimePoint

	// Cached simulation result - computed once when spec loads
	cachedSimulation *SimulationResult
}

type TimePoint struct {
	Time    time.Time `json:"time"`
	Counter string    `json:"counter"`
	Value   int64     `json:"value"`
}

func (s *Server) incCounter(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name]++

	// Record time series point
	s.timeSeries = append(s.timeSeries, TimePoint{
		Time:    time.Now(),
		Counter: name,
		Value:   s.counters[name],
	})

	// Keep only last 1000 points
	if len(s.timeSeries) > 1000 {
		s.timeSeries = s.timeSeries[len(s.timeSeries)-1000:]
	}
}

func (s *Server) getCounters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]int64)
	for k, v := range s.counters {
		result[k] = v
	}
	return result
}

func (s *Server) getTimeSeries() []TimePoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]TimePoint, len(s.timeSeries))
	copy(result, s.timeSeries)
	return result
}

// New creates a new server instance
func New(specFile string) (*Server, error) {
	engine, err := prolog.New()
	if err != nil {
		return nil, fmt.Errorf("creating prolog engine: %w", err)
	}

	s := &Server{
		engine:     engine,
		llm:        llm.New(),
		specFile:   specFile,
		counters:   make(map[string]int64),
		timeSeries: []TimePoint{},
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
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/openapi", s.handleOpenAPI)
	mux.HandleFunc("/api/simulate", s.handleSimulate) // Add this line

	// Static files (embedded)
	mux.HandleFunc("/", s.handleStatic)

	s.mux = mux
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
				"source":     req.Source,
				"canAutoFix": true,
			})
			return
		}

		// Run simulation immediately and cache the result
		s.runAndCacheSimulation(1000)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})

		s.incCounter("spec_loads")

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

	s.incCounter("queries")
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

	s.incCounter("visualizations")
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

// ActorStateMachine represents a single actor's state machine
type ActorStateMachine struct {
	Actor       string              `json:"actor"`
	States      []string            `json:"states"`
	Transitions []map[string]string `json:"transitions"`
	Initial     string              `json:"initial"`
}

// TODO: Implement when engine supports variable binding extraction
func (s *Server) extractActorStateMachines(ctx context.Context) ([]ActorStateMachine, error) {
	// Requires engine.GetActorStateMachines() to be implemented
	return nil, nil
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

	s.incCounter("chat_requests")
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

	s.incCounter("ctl_checks")
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

// handleProperties returns named properties from the spec with check results
func (s *Server) handleProperties(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	properties, err := s.engine.GetProperties(ctx)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Check each property and include results
	type PropertyResult struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Formula     string `json:"formula"`
		Satisfied   *bool  `json:"satisfied,omitempty"`
		Error       string `json:"error,omitempty"`
	}

	results := make([]PropertyResult, len(properties))
	for i, prop := range properties {
		results[i] = PropertyResult{
			Name:        prop.Name,
			Description: prop.Description,
			Formula:     prop.Formula,
		}

		// Try to check the property
		query := fmt.Sprintf("check_ctl(%s).", prop.Formula)
		satisfied, err := s.engine.QueryOne(ctx, query)
		if err != nil {
			results[i].Error = err.Error()
		} else {
			results[i].Satisfied = &satisfied
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"properties": results,
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

// handleMetrics returns server metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"counters":   s.getCounters(),
		"timeSeries": s.getTimeSeries(),
	})
}

// handleOpenAPI returns OpenAPI 3.0 spec generated from Prolog API definitions
func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Query API info from spec
	info := make(map[string]string)
	infoResults, _ := s.engine.Query(ctx, "api_info(Key, Value).")
	for _, row := range infoResults {
		key := row["Key"]
		val := row["Value"]
		if key != "" && val != "" {
			info[key] = val
		}
	}

	// Query endpoints
	type Endpoint struct {
		Method      string
		Path        string
		Description string
		OperationID string
	}
	var endpoints []Endpoint
	epResults, _ := s.engine.Query(ctx, "api_endpoint(Method, Path, Desc, OpID).")
	for _, row := range epResults {
		endpoints = append(endpoints, Endpoint{
			Method:      row["Method"],
			Path:        row["Path"],
			Description: row["Desc"],
			OperationID: row["OpID"],
		})
	}

	// Query request fields
	type Field struct {
		Name        string
		Type        string
		Required    bool
		Description string
	}
	requestFields := make(map[string][]Field)
	reqResults, _ := s.engine.Query(ctx, "api_request(OpID, Name, Type, Req, Desc).")
	for _, row := range reqResults {
		opID := row["OpID"]
		reqBool := row["Req"] == "true"
		requestFields[opID] = append(requestFields[opID], Field{
			Name:        row["Name"],
			Type:        row["Type"],
			Required:    reqBool,
			Description: row["Desc"],
		})
	}

	// Query response fields
	responseFields := make(map[string][]Field)
	respResults, _ := s.engine.Query(ctx, "api_response(OpID, Name, Type, Desc).")
	for _, row := range respResults {
		opID := row["OpID"]
		responseFields[opID] = append(responseFields[opID], Field{
			Name:        row["Name"],
			Type:        row["Type"],
			Description: row["Desc"],
		})
	}

	// Build OpenAPI spec
	paths := make(map[string]interface{})
	for _, ep := range endpoints {
		method := strings.ToLower(ep.Method)
		pathObj := make(map[string]interface{})

		opObj := map[string]interface{}{
			"operationId": ep.OperationID,
			"summary":     ep.Description,
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Success",
				},
			},
		}

		// Add request body for POST
		if method == "post" {
			if fields, ok := requestFields[ep.OperationID]; ok && len(fields) > 0 {
				props := make(map[string]interface{})
				var required []string
				for _, f := range fields {
					props[f.Name] = map[string]interface{}{
						"type":        f.Type,
						"description": f.Description,
					}
					if f.Required {
						required = append(required, f.Name)
					}
				}
				opObj["requestBody"] = map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"type":       "object",
								"properties": props,
								"required":   required,
							},
						},
					},
				}
			}
		}

		pathObj[method] = opObj
		paths[ep.Path] = pathObj
	}

	openapi := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       info["title"],
			"version":     info["version"],
			"description": info["description"],
		},
		"servers": []map[string]string{
			{"url": "/api"},
		},
		"paths": paths,
	}

	json.NewEncoder(w).Encode(openapi)
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

type SimulationResult struct {
	ByType   map[string]int64  `json:"byType"`
	BySrc    map[string]int64  `json:"bySrc"`
	ByDst    map[string]int64  `json:"byDst"`
	Timeline []SimulationEvent `json:"timeline"`
	Total    int64             `json:"total"`
	Steps    int               `json:"steps"`
}

type SimulationEvent struct {
	Step  int    `json:"step"`
	Label string `json:"label"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// runAndCacheSimulation runs the simulation and stores the result
func (s *Server) runAndCacheSimulation(steps int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := SimulationResult{
		ByType:   make(map[string]int64),
		BySrc:    make(map[string]int64),
		ByDst:    make(map[string]int64),
		Timeline: make([]SimulationEvent, 0),
	}

	ctx := context.Background()
	sm, err := s.engine.GetStateMachine(ctx)
	if err != nil || len(sm.Initial) == 0 || len(sm.Transitions) == 0 {
		s.cachedSimulation = &result
		return
	}

	// Build transition map for quick lookup
	transitionMap := make(map[string][]prolog.Transition)
	for _, t := range sm.Transitions {
		transitionMap[t.From] = append(transitionMap[t.From], t)
	}

	// Extract actor from state name (e.g., "proposer_idle" -> "proposer")
	extractActor := func(state string) string {
		if idx := strings.Index(state, "_"); idx > 0 {
			return state[:idx]
		}
		return state
	}

	// Run simulation - walk the state machine
	currentStates := make(map[string]string)
	for _, init := range sm.Initial {
		actor := extractActor(init)
		currentStates[actor] = init
	}

	for step := 0; step < steps; step++ {
		dice := rand.Float64()
		s.setDiceValue(ctx, dice)

		// Collect all possible transitions from current states
		var possibleTransitions []prolog.Transition
		for _, state := range currentStates {
			for _, t := range transitionMap[state] {
				if s.transitionAllowed(ctx, state, t) {
					possibleTransitions = append(possibleTransitions, t)
				}
			}
		}

		if len(possibleTransitions) == 0 {
			s.clearDiceValue(ctx)
			break
		}

		// Pick a random transition
		t := possibleTransitions[rand.Intn(len(possibleTransitions))]

		// Update state
		actor := extractActor(t.From)
		currentStates[actor] = t.To

		// Record metrics
		result.ByType[t.Label]++
		result.BySrc[extractActor(t.From)]++
		result.ByDst[extractActor(t.To)]++
		result.Total++

		result.Timeline = append(result.Timeline, SimulationEvent{
			Step:  step,
			Label: t.Label,
			From:  t.From,
			To:    t.To,
		})
		s.clearDiceValue(ctx)
	}

	result.Steps = steps

	s.cachedSimulation = &result
}

func (s *Server) transitionAllowed(ctx context.Context, state string, t prolog.Transition) bool {
	if !s.stateGuardSatisfied(ctx, state) {
		return false
	}
	if !s.transitionGuardSatisfied(ctx, t) {
		return false
	}
	return true
}

func (s *Server) stateGuardSatisfied(ctx context.Context, state string) bool {
	stateAtom := prologAtom(state)
	hasGuard, err := s.engine.QueryOne(ctx, fmt.Sprintf("state_guard(%s, _).", stateAtom))
	if err != nil {
		log.Printf("state_guard lookup error: %v", err)
		return true
	}
	if !hasGuard {
		return true
	}
	ok, err := s.engine.QueryOne(ctx, fmt.Sprintf("state_guard(%s, Guard), call(Guard).", stateAtom))
	if err != nil {
		log.Printf("state_guard eval error: %v", err)
		return false
	}
	return ok
}

func (s *Server) transitionGuardSatisfied(ctx context.Context, t prolog.Transition) bool {
	fromAtom := prologAtom(t.From)
	labelAtom := prologAtom(t.Label)
	toAtom := prologAtom(t.To)
	hasGuard, err := s.engine.QueryOne(ctx, fmt.Sprintf("transition_guard(%s, %s, %s, _).", fromAtom, labelAtom, toAtom))
	if err != nil {
		log.Printf("transition_guard lookup error: %v", err)
		return true
	}
	if !hasGuard {
		return true
	}
	ok, err := s.engine.QueryOne(ctx, fmt.Sprintf("transition_guard(%s, %s, %s, Guard), call(Guard).", fromAtom, labelAtom, toAtom))
	if err != nil {
		log.Printf("transition_guard eval error: %v", err)
		return false
	}
	return ok
}

func (s *Server) setDiceValue(ctx context.Context, value float64) {
	_, _ = s.engine.QueryOne(ctx, "retractall(dice0_value(_)).")
	_, _ = s.engine.QueryOne(ctx, fmt.Sprintf("assertz(dice0_value(%s)).", formatFloat(value)))
}

func (s *Server) clearDiceValue(ctx context.Context) {
	_, _ = s.engine.QueryOne(ctx, "retractall(dice0_value(_)).")
}

func prologAtom(value string) string {
	if value == "" {
		return "''"
	}
	for i, r := range value {
		if (r >= 'a' && r <= 'z') || (i > 0 && ((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_')) {
			continue
		}
		if i == 0 {
			break
		}
		goto needsQuote
	}
	if value[0] >= 'a' && value[0] <= 'z' {
		return value
	}
needsQuote:
	escaped := strings.ReplaceAll(value, "'", "''")
	return "'" + escaped + "'"
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// handleSimulate returns the cached simulation result
func (s *Server) handleSimulate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.mu.RLock()
	result := s.cachedSimulation
	s.mu.RUnlock()

	if result == nil {
		// No simulation cached, return empty result
		json.NewEncoder(w).Encode(SimulationResult{
			ByType:   make(map[string]int64),
			BySrc:    make(map[string]int64),
			ByDst:    make(map[string]int64),
			Timeline: make([]SimulationEvent, 0),
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}
