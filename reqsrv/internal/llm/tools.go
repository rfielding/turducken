package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"reqsrv/internal/prolog"
	"reqsrv/internal/store"
)

func ToolSchemas() []any {
	// Tool schema per OpenAI function calling guide.
	return []any{
		map[string]any{
			"type":        "function",
			"name":        "doc_put",
			"description": "Create or update a document in the right pane.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":      map[string]any{"type": "string"},
					"title":   map[string]any{"type": "string"},
					"mime":    map[string]any{"type": "string"},
					"content": map[string]any{"type": "string"},
				},
				"required": []string{"id", "title", "content"},
			},
		},
		map[string]any{
			"type":        "function",
			"name":        "prolog_assertz",
			"description": "Assert a Prolog clause (fact or rule) into the kernel. Example: edge(s0,s1). or holds(s0, done).",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"clause": map[string]any{"type": "string"},
				},
				"required": []string{"clause"},
			},
		},
		map[string]any{
			"type":        "function",
			"name":        "ctl_check",
			"description": "Run a CTL query like ef(done, s0). ag(p, s0). Returns boolean.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
				},
				"required": []string{"query"},
			},
		},
		map[string]any{
			"type":        "function",
			"name":        "seed_demo_model",
			"description": "Seed a tiny demo Kripke structure (edge/2 and holds/2) plus starter docs.",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

func RunTool(ctx context.Context, call ToolCall, docs *store.DocStore, k *prolog.Kernel) (ToolResult, error) {
	switch call.Name {
	case "doc_put":
		var in struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Mime    string `json:"mime"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(call.Arguments, &in); err != nil {
			return ToolResult{}, err
		}
		if in.Mime == "" {
			in.Mime = "text/markdown"
		}
		docs.Put(in.ID, in.Title, in.Mime, in.Content)
		return ToolResult{Name: call.Name, Output: map[string]any{"ok": true}}, nil

	case "prolog_assertz":
		var in struct {
			Clause string `json:"clause"`
		}
		if err := json.Unmarshal(call.Arguments, &in); err != nil {
			return ToolResult{}, err
		}
		if err := k.Assertz(ctx, in.Clause); err != nil {
			return ToolResult{Name: call.Name, Output: map[string]any{"ok": false, "error": err.Error()}}, nil
		}
		return ToolResult{Name: call.Name, Output: map[string]any{"ok": true}}, nil

	case "ctl_check":
		var in struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(call.Arguments, &in); err != nil {
			return ToolResult{}, err
		}
		res, err := k.QueryBool(ctx, in.Query)
		if err != nil {
			return ToolResult{Name: call.Name, Output: map[string]any{"ok": false, "error": err.Error()}}, nil
		}
		return ToolResult{Name: call.Name, Output: map[string]any{"ok": true, "result": res}}, nil

	case "seed_demo_model":
		_ = k.Assertz(ctx, "edge(s0,s1)")
		_ = k.Assertz(ctx, "edge(s1,s2)")
		_ = k.Assertz(ctx, "holds(s2,done)")
		docs.Put("requirements.md", "Requirements", "text/markdown", demoRequirementsMD())
		docs.Put("program.mmd", "Program Diagram (Mermaid)", "text/plain", demoProgramMMD())
		docs.Put("interaction.mmd", "Interaction Diagram (Mermaid)", "text/plain", demoInteractionMMD())
		return ToolResult{Name: call.Name, Output: map[string]any{"ok": true}}, nil

	default:
		return ToolResult{}, fmt.Errorf("unknown tool: %s", call.Name)
	}
}

func demoRequirementsMD() string {
	return `# Requirements (demo)

This is a tiny seeded model so you can prove the plumbing works end-to-end.

## Model facts
- edge(s0,s1)
- edge(s1,s2)
- holds(s2,done)

## Example queries
- ef(done, s0).  % should be true
- ag(done, s0).  % should be false

Next step: replace this with CSP-ish process definitions and a global edge/2 derived from them.
`
}

func demoProgramMMD() string {
	return `stateDiagram-v2
  [*] --> s0
  s0 --> s1
  s1 --> s2
  s2 --> [*]
`
}

func demoInteractionMMD() string {
	return `sequenceDiagram
  participant User
  participant Model
  User->>Model: describe requirements
  Model-->>User: propose Prolog + docs updates
`
}
