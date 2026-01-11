# reqsrv

Single-binary Go webserver with an embedded Prolog sandbox (`github.com/ichiban/prolog`) and a 2-pane UI:

- Left: chat
- Right: documents produced by tools (Markdown / Mermaid / charts later)

## Run

```bash
export OPENAI_API_KEY=...
go mod tidy
go run .
```

Open http://localhost:8080

Try: `please call seed_demo_model` to populate demo docs and a tiny graph.

## Why this shape?

- Prolog is the sandbox: facts + rules are authoritative.
- The LLM edits the sandbox and the documents by tool calls.
- CTL checks are Prolog predicates (EF/AG already stubbed).

## References

This project’s tool-calling loop follows the OpenAI “Responses API” function calling flow (function_call + function_call_output items). 
