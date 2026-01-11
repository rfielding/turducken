package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{},
	}
}

type ChatInput struct {
	UserText      string
	Docs          map[string]string // id -> content
	PrologSummary string
	ToolRunner    func(context.Context, ToolCall) (ToolResult, error)
}

type ChatOutput struct {
	Text string
}

type ToolCall struct {
	CallID    string          `json:"call_id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResult struct {
	Name   string `json:"name"`
	Output any    `json:"output"`
}

type responsesCreateResponse struct {
	Output     []map[string]any `json:"output"`
	OutputText string           `json:"output_text"`
}

func (c *Client) Chat(ctx context.Context, in ChatInput) (ChatOutput, error) {
	// Model IDs are configured by OpenAI; adjust as needed.
	model := envDefault("OPENAI_MODEL", "gpt-5.2")

	tools := ToolSchemas()

	// Running input list (Responses API “Items”).
	inputList := []any{
		map[string]any{"role": "user", "content": in.UserText},
	}

	instructions := systemPrompt(in.PrologSummary, in.Docs)

	for step := 0; step < 8; step++ {
		reqBody := map[string]any{
			"model":        model,
			"instructions": instructions,
			"tools":        tools,
			"input":        inputList,
		}

		raw, err := c.do(ctx, "https://api.openai.com/v1/responses", reqBody)
		if err != nil {
			return ChatOutput{}, err
		}

		var resp responsesCreateResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			return ChatOutput{}, err
		}

		// Append model output items back into context, per OpenAI guide.
		for _, it := range resp.Output {
			inputList = append(inputList, it)
		}

		calls := extractToolCalls(resp.Output)
		if len(calls) == 0 {
			if resp.OutputText != "" {
				return ChatOutput{Text: resp.OutputText}, nil
			}
			return ChatOutput{Text: extractText(resp.Output)}, nil
		}

		for _, call := range calls {
			out, err := in.ToolRunner(ctx, call)
			if err != nil {
				out = ToolResult{Name: call.Name, Output: map[string]any{"ok": false, "error": err.Error()}}
			}

			outJSON, _ := json.Marshal(out.Output)
			inputList = append(inputList, map[string]any{
				"type":    "function_call_output",
				"call_id": call.CallID,
				"output":  string(outJSON),
			})
		}
	}

	return ChatOutput{}, fmt.Errorf("too many tool steps")
}

func (c *Client) do(ctx context.Context, url string, body any) ([]byte, error) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(res.Body)
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("openai error %d: %s", res.StatusCode, buf.String())
	}
	return buf.Bytes(), nil
}

func systemPrompt(prologSummary string, docs map[string]string) string {
	return fmt.Sprintf(
		`You are a requirements assistant.

Your job:
- Evolve a Prolog-based model (facts + rules) that acts as the sandboxed source of truth.
- Maintain human-facing artifacts as documents (Markdown / Mermaid).
- Prefer usability over theoretical perfection: help the user scale models without turning them into toys.

Prolog summary:
%s

Current documents:
%s

Call tools when you need to update docs or assert/query Prolog.`,
		prologSummary,
		summarizeDocs(docs),
	)
}

func summarizeDocs(docs map[string]string) string {
	if len(docs) == 0 {
		return "(none)\n"
	}
	out := ""
	for id := range docs {
		out += "- " + id + "\n"
	}
	return out
}

func extractToolCalls(items []map[string]any) []ToolCall {
	var calls []ToolCall
	for _, it := range items {
		t, _ := it["type"].(string)
		if t != "function_call" {
			continue
		}
		name, _ := it["name"].(string)
		callID, _ := it["call_id"].(string)
		argsStr, _ := it["arguments"].(string) // JSON string
		calls = append(calls, ToolCall{
			CallID:    callID,
			Name:      name,
			Arguments: json.RawMessage([]byte(argsStr)),
		})
	}
	return calls
}

func extractText(items []map[string]any) string {
	out := ""
	for _, it := range items {
		t, _ := it["type"].(string)
		if t != "message" {
			continue
		}
		content, ok := it["content"].([]any)
		if !ok {
			continue
		}
		for _, c := range content {
			m, ok := c.(map[string]any)
			if !ok {
				continue
			}
			ct, _ := m["type"].(string)
			if ct == "output_text" {
				if txt, ok := m["text"].(string); ok {
					out += txt
				}
			}
		}
	}
	return out
}

func envDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
