package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Provider specifies which LLM to use
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
)

// Client handles LLM API interactions
type Client struct {
	provider      Provider
	anthropicKey  string
	openaiKey     string
	anthropicURL  string
	openaiURL     string
	claudeModel   string
	gptModel      string
	httpClient    *http.Client
}

// New creates a new LLM client
func New() *Client {
	c := &Client{
		anthropicKey:  os.Getenv("ANTHROPIC_API_KEY"),
		openaiKey:     os.Getenv("OPENAI_API_KEY"),
		anthropicURL:  "https://api.anthropic.com/v1",
		openaiURL:     "https://api.openai.com/v1",
		claudeModel:   "claude-sonnet-4-20250514",
		gptModel:      "gpt-4o",
		httpClient:    http.DefaultClient,
	}
	
	// Default to OpenAI if available, else Anthropic
	if c.openaiKey != "" {
		c.provider = ProviderOpenAI
	} else if c.anthropicKey != "" {
		c.provider = ProviderAnthropic
	}
	
	return c
}

// SetProvider sets the LLM provider
func (c *Client) SetProvider(p Provider) {
	c.provider = p
}

// GetProvider returns the current provider
func (c *Client) GetProvider() Provider {
	return c.provider
}

// SystemPrompt is the system prompt for specification generation
const SystemPrompt = `You are a formal methods assistant that helps write Prolog specifications for system modeling.

You generate Prolog code that can be used for:
1. State machine modeling (states, transitions, initial/accepting states)
2. CTL model checking (temporal properties)
3. CSP-style message passing (channels, send/recv)
4. Process algebra (recursive process definitions)
5. Visualization data (sequence diagrams, charts)

PROLOG SYNTAX FOR SPECIFICATIONS:

State Machines:
  state(name, [prop1, prop2]).     % State with properties
  transition(from, label, to).     % Labeled transition
  initial(state).                  % Mark initial state
  accepting(state).                % Mark accepting state
  prop(state, property).           % State satisfies property

CSP Channels:
  channel(name, capacity).         % Declare buffered channel
  send(chan, msg, s1, s2).        % Send transition
  recv(chan, msg, s1, s2).        % Receive transition

Process Definitions (recursive equations):
  proc(Name, Definition).
  % Where Definition can be:
  %   stop                         - deadlock
  %   skip                         - successful termination
  %   prefix(event, NextProc)      - do event then NextProc
  %   choice(P1, P2)               - external choice
  %   parallel(P1, P2)             - parallel composition

CTL Properties:
  % Use check_ctl(Formula) to verify
  % Formulas: atom(p), not(F), and(F1,F2), or(F1,F2)
  %          ex(F), ax(F), ef(F), af(F), eg(F), ag(F)
  %          eu(F1,F2), au(F1,F2)

Sequence Diagrams:
  lifeline(actor).                 % Declare participant
  message(seq, from, to, label).   % Message at sequence number

Charts:
  pie_slice(label, value).         % Pie chart slice
  line_point(series, x, y).        % Line chart point
  bar_value(label, value).         % Bar chart value

When the user describes a system, generate clean Prolog code enclosed in ` + "```prolog" + ` blocks.
Focus on capturing the essential behavior and properties they care about.`

// BuildPrompt constructs the full prompt for the LLM
func (c *Client) BuildPrompt(userMessage, currentSpec, additionalContext string) string {
	var prompt bytes.Buffer
	
	if currentSpec != "" {
		prompt.WriteString("Current specification:\n```prolog\n")
		prompt.WriteString(currentSpec)
		prompt.WriteString("\n```\n\n")
	}
	
	if additionalContext != "" {
		prompt.WriteString("Context:\n")
		prompt.WriteString(additionalContext)
		prompt.WriteString("\n\n")
	}
	
	prompt.WriteString("User request:\n")
	prompt.WriteString(userMessage)
	
	return prompt.String()
}

// Chat sends a message to the LLM and returns the response
func (c *Client) Chat(ctx context.Context, prompt string) (string, error) {
	switch c.provider {
	case ProviderOpenAI:
		if c.openaiKey == "" {
			return c.mockResponse(prompt), nil
		}
		return c.chatOpenAI(ctx, prompt)
	case ProviderAnthropic:
		if c.anthropicKey == "" {
			return c.mockResponse(prompt), nil
		}
		return c.chatAnthropic(ctx, prompt)
	default:
		return c.mockResponse(prompt), nil
	}
}

// chatOpenAI calls the OpenAI API
func (c *Client) chatOpenAI(ctx context.Context, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": c.gptModel,
		"messages": []map[string]string{
			{"role": "system", "content": SystemPrompt},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 4096,
	}
	
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.openaiURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.openaiKey)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(body))
	}
	
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI")
	}
	
	return result.Choices[0].Message.Content, nil
}

// chatAnthropic calls the Anthropic API
func (c *Client) chatAnthropic(ctx context.Context, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model":      c.claudeModel,
		"max_tokens": 4096,
		"system":     SystemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.anthropicURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.anthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(body))
	}
	
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response from Anthropic")
	}
	
	return result.Content[0].Text, nil
}

// mockResponse returns a helpful mock response when no API key is configured
func (c *Client) mockResponse(prompt string) string {
	return `I can help you create a Prolog specification. Here's an example based on your request:

` + "```prolog" + `
% Example state machine for a simple protocol
state(idle, [waiting]).
state(busy, [processing]).
state(done, [complete]).

initial(idle).
accepting(done).

transition(idle, start, busy).
transition(busy, finish, done).
transition(busy, error, idle).

% Properties
prop(idle, waiting).
prop(busy, processing).
prop(done, complete).

% CTL property: from idle, we can eventually reach done
% check_ctl(ef(atom(complete))).

% Sequence diagram for the protocol
lifeline(client).
lifeline(server).

message(1, client, server, request).
message(2, server, client, response).
` + "```" + `

To use this specification:
1. Click "Apply Spec" to load it into the engine
2. Use the Query tab to run Prolog queries
3. Use the Visualize tab to see state machines and diagrams
4. Use the Check tab to verify CTL properties

**Note:** Set OPENAI_API_KEY or ANTHROPIC_API_KEY environment variable to enable AI-powered specification generation.`
}

// SetGPTModel sets the OpenAI model to use
func (c *Client) SetGPTModel(model string) {
	c.gptModel = model
}

// SetClaudeModel sets the Anthropic model to use
func (c *Client) SetClaudeModel(model string) {
	c.claudeModel = model
}

// SetOpenAIKey sets the OpenAI API key
func (c *Client) SetOpenAIKey(key string) {
	c.openaiKey = key
}

// SetAnthropicKey sets the Anthropic API key
func (c *Client) SetAnthropicKey(key string) {
	c.anthropicKey = key
}

// HasAPIKey returns true if at least one API key is configured
func (c *Client) HasAPIKey() bool {
	return c.openaiKey != "" || c.anthropicKey != ""
}

// ProviderName returns a human-readable provider name
func (c *Client) ProviderName() string {
	switch c.provider {
	case ProviderOpenAI:
		return "ChatGPT (" + c.gptModel + ")"
	case ProviderAnthropic:
		return "Claude (" + c.claudeModel + ")"
	default:
		return "Mock (no API key)"
	}
}
