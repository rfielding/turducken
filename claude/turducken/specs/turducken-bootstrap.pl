% filepath: /home/rfielding/code/turducken/claude/turducken/specs/turducken-bootstrap.pl
% ============================================================================
% TURDUCKEN BOOTSTRAP SPECIFICATION v2
% ============================================================================
% 
% PURPOSE: Complete specification to regenerate turducken in any language.
% Feed this to an LLM to one-shot the entire application.
%
% ============================================================================
% LANGUAGE-AGNOSTIC REQUIREMENTS
% ============================================================================
%
% MUST HAVE:
%   - Embedded Prolog interpreter (or implement minimal subset)
%   - HTTP server with JSON API
%   - Single HTML file served from binary/executable
%   - LLM API client (OpenAI and/or Anthropic)
%
% LANGUAGE OPTIONS:
%   Go:    github.com/ichiban/prolog (pure Go, embeds well)
%   Rust:  scryer-prolog as library, or implement mini-prolog
%   Python: pyswip (SWI-Prolog bindings) or kanren (mini-prolog)
%   TypeScript: tau-prolog (pure JS, runs in browser too)
%
% ============================================================================
% PART 1: FORMAL SPECIFICATION (Executable Prolog)
% ============================================================================

% === DOCUMENTATION ===
doc(purpose, 'Formal specification tool: natural language -> Prolog -> visualization -> CTL verification').
doc(name_origin, 'Layered: LLM wraps Prolog engine wraps state machines. Turkey/duck/chicken.').
doc(workflow, 'Chat describes system -> LLM generates Prolog -> Apply loads spec -> Visualize shows diagrams -> Check verifies properties').

% === ACTORS ===
actor(ui).
actor(http_server).
actor(prolog_engine).
actor(llm_client).

% === ACTOR INITIAL STATES ===
actor_initial(ui, ui_idle).
actor_initial(http_server, server_ready).
actor_initial(prolog_engine, engine_empty).
actor_initial(llm_client, llm_idle).

% === ACTOR STATES WITH PROPERTIES ===
actor_state(ui, ui_idle, [accepting_input]).
actor_state(ui, ui_chatting, [waiting_response]).
actor_state(ui, ui_applying, [loading_spec]).
actor_state(ui, ui_querying, [running_query]).
actor_state(http_server, server_ready, [listening]).
actor_state(http_server, server_handling, [processing]).
actor_state(prolog_engine, engine_empty, [can_load, no_spec]).
actor_state(prolog_engine, engine_loaded, [can_query, can_reset, has_spec]).
actor_state(prolog_engine, engine_error, [can_reset]).
actor_state(llm_client, llm_idle, [ready]).
actor_state(llm_client, llm_requesting, [busy]).
actor_state(llm_client, llm_received, [has_response]).
actor_state(llm_client, llm_error, [failed]).

% === ACTOR TRANSITIONS ===
actor_transition(ui, ui_idle, submit_chat, ui_chatting).
actor_transition(ui, ui_idle, apply_spec, ui_applying).
actor_transition(ui, ui_idle, run_query, ui_querying).
actor_transition(ui, ui_chatting, chat_response, ui_idle).
actor_transition(ui, ui_chatting, chat_error, ui_idle).
actor_transition(ui, ui_applying, spec_loaded, ui_idle).
actor_transition(ui, ui_applying, spec_error, ui_idle).
actor_transition(ui, ui_querying, query_result, ui_idle).
actor_transition(http_server, server_ready, receive_request, server_handling).
actor_transition(http_server, server_handling, send_response, server_ready).
actor_transition(prolog_engine, engine_empty, load_spec, engine_loaded).
actor_transition(prolog_engine, engine_empty, load_spec_error, engine_error).
actor_transition(prolog_engine, engine_loaded, reset, engine_empty).
actor_transition(prolog_engine, engine_loaded, reload_spec, engine_loaded).
actor_transition(prolog_engine, engine_loaded, query, engine_loaded).
actor_transition(prolog_engine, engine_error, reset, engine_empty).
actor_transition(llm_client, llm_idle, send_prompt, llm_requesting).
actor_transition(llm_client, llm_requesting, receive_response, llm_received).
actor_transition(llm_client, llm_requesting, timeout, llm_error).
actor_transition(llm_client, llm_requesting, api_error, llm_error).
actor_transition(llm_client, llm_received, extract_prolog, llm_idle).
actor_transition(llm_client, llm_received, no_prolog, llm_idle).
actor_transition(llm_client, llm_error, retry, llm_idle).

% === DERIVED PREDICATES (for CTL) ===
prop(State, Prop) :- actor_state(_, State, Props), member(Prop, Props).
initial(S) :- actor_initial(_, S).
transition(From, Label, To) :- actor_transition(_, From, Label, To).

% === NAMED PROPERTIES ===
property(no_deadlock, 'From any state, can eventually accept input', 'ag(ef(atom(accepting_input)))').
property(can_reset, 'From any state, can eventually reset', 'ag(ef(atom(no_spec)))').
property(llm_recovery, 'LLM errors are recoverable', 'ag(or(not(atom(failed)), ef(atom(ready))))').
property(server_liveness, 'Server always listening or processing', 'ag(or(atom(listening), atom(processing)))').

% === SEQUENCE DIAGRAM ===
lifeline(user).
lifeline(ui).
lifeline(http_server).
lifeline(llm_client).
lifeline(prolog_engine).

message(1, user, ui, type_message).
message(2, ui, http_server, post_chat).
message(3, http_server, llm_client, send_prompt).
message(4, llm_client, http_server, llm_response).
message(5, http_server, ui, chat_response).
message(6, ui, user, display_with_apply_button).
message(7, user, ui, click_apply).
message(8, ui, http_server, post_spec).
message(9, http_server, prolog_engine, load_spec).
message(10, prolog_engine, http_server, load_result).
message(11, http_server, ui, spec_status).

% ============================================================================
% PART 2: CTL MODEL CHECKING IMPLEMENTATION
% ============================================================================
%
% These predicates MUST be embedded in the Prolog engine. Copy verbatim.
%
% --- UTILITY PREDICATES ---
%
% member(X, [X|_]).
% member(X, [_|T]) :- member(X, T).
%
% append([], L, L).
% append([H|T], L, [H|R]) :- append(T, L, R).
%
% forall(Cond, Action) :- \+ (Cond, \+ Action).
%
% --- CTL SATISFACTION ---
%
% sat(S, atom(P)) :- prop(S, P).
% sat(S, not(F)) :- \+ sat(S, F).
% sat(S, and(F, G)) :- sat(S, F), sat(S, G).
% sat(S, or(F, G)) :- sat(S, F).
% sat(S, or(F, G)) :- sat(S, G).
%
% % EX: exists next state satisfying F
% sat(S, ex(F)) :- transition(S, _, S2), sat(S2, F).
%
% % AX: all next states satisfy F
% sat(S, ax(F)) :- forall(transition(S, _, S2), sat(S2, F)).
%
% % EF: exists path eventually reaching F (with cycle detection)
% sat(S, ef(F)) :- ef_check(S, F, []).
% ef_check(S, F, _) :- sat(S, F), !.
% ef_check(S, F, Visited) :-
%     \+ member(S, Visited),
%     transition(S, _, S2),
%     ef_check(S2, F, [S|Visited]).
%
% % AF: all paths eventually reach F
% sat(S, af(F)) :- af_check(S, F, []).
% af_check(S, F, _) :- sat(S, F), !.
% af_check(S, F, Visited) :-
%     \+ member(S, Visited),
%     forall(transition(S, _, S2), af_check(S2, F, [S|Visited])).
%
% % EG: exists path where F holds globally
% sat(S, eg(F)) :- eg_check(S, F, []).
% eg_check(S, F, Visited) :- member(S, Visited), !.  % cycle = success
% eg_check(S, F, Visited) :-
%     sat(S, F),
%     transition(S, _, S2),
%     eg_check(S2, F, [S|Visited]).
%
% % AG: F holds on all paths globally
% sat(S, ag(F)) :- ag_check(S, F, []).
% ag_check(S, _, Visited) :- member(S, Visited), !.
% ag_check(S, F, Visited) :-
%     sat(S, F),
%     forall(transition(S, _, S2), ag_check(S2, F, [S|Visited])).
%
% % EU: E[F U G] - exists path where F until G
% sat(S, eu(F, G)) :- eu_check(S, F, G, []).
% eu_check(S, _, G, _) :- sat(S, G), !.
% eu_check(S, F, G, Visited) :-
%     \+ member(S, Visited),
%     sat(S, F),
%     transition(S, _, S2),
%     eu_check(S2, F, G, [S|Visited]).
%
% % AU: A[F U G] - all paths F until G
% sat(S, au(F, G)) :- au_check(S, F, G, []).
% au_check(S, _, G, _) :- sat(S, G), !.
% au_check(S, F, G, Visited) :-
%     \+ member(S, Visited),
%     sat(S, F),
%     forall(transition(S, _, S2), au_check(S2, F, G, [S|Visited])).
%
% % Entry point: check from all initial states
% check_ctl(Formula) :- forall(initial(S), sat(S, Formula)).
%
% ============================================================================
% PART 3: HTTP API SPECIFICATION
% ============================================================================

% --- ENDPOINTS ---
api_endpoint('POST', '/api/spec', 'Load Prolog source code').
api_endpoint('POST', '/api/query', 'Execute Prolog query').
api_endpoint('POST', '/api/check', 'Check CTL formula').
api_endpoint('POST', '/api/chat', 'Send message to LLM').
api_endpoint('GET', '/api/visualize', 'Get visualization data').
api_endpoint('GET', '/api/metrics', 'Get request counters').
api_endpoint('POST', '/api/reset', 'Reset to original spec').
api_endpoint('GET', '/api/properties', 'Get named properties with results').

% --- REQUEST/RESPONSE SCHEMAS ---
% Format: api_field(endpoint, direction, field_name, field_type, description)

% POST /api/spec
api_field('/api/spec', request, source, string, 'Prolog source code').
api_field('/api/spec', response, success, boolean, 'Whether load succeeded').
api_field('/api/spec', response, error, 'string|null', 'Error message if failed').

% POST /api/query  
api_field('/api/query', request, query, string, 'Prolog query e.g. transition(X,Y,Z).').
api_field('/api/query', response, results, 'array of bindings', 'List of variable bindings').
api_field('/api/query', response, error, 'string|null', 'Error message if failed').

% POST /api/check
api_field('/api/check', request, formula, string, 'CTL formula e.g. ag(ef(atom(ready)))').
api_field('/api/check', response, satisfied, boolean, 'Whether formula holds').
api_field('/api/check', response, error, 'string|null', 'Error message if failed').

% POST /api/chat
api_field('/api/chat', request, message, string, 'User message').
api_field('/api/chat', request, context, 'string|null', 'Current spec for context').
api_field('/api/chat', response, response, string, 'LLM response text').
api_field('/api/chat', response, prolog, 'string|null', 'Extracted prolog block').
api_field('/api/chat', response, error, 'string|null', 'Error message if failed').

% GET /api/visualize
api_field('/api/visualize', request, type, 'query param', 'stateMachine|sequence|pie|line').
api_field('/api/visualize', response, stateMachine, object, 'transitions array, initial array').
api_field('/api/visualize', response, sequence, object, 'lifelines array, messages array').

% GET /api/metrics
api_field('/api/metrics', response, counters, object, 'Map of counter name to value').
api_field('/api/metrics', response, timeSeries, array, 'Array of time/counters snapshots').

% GET /api/properties
api_field('/api/properties', response, properties, array, 'Array of name/description/formula/satisfied').

% --- METRICS TO TRACK ---
metric(spec_loads, counter, 'Number of times spec was loaded').
metric(queries, counter, 'Number of Prolog queries executed').
metric(ctl_checks, counter, 'Number of CTL checks performed').
metric(chat_requests, counter, 'Number of LLM chat requests').
metric(errors, counter, 'Number of errors encountered').

% ============================================================================
% PART 4: LLM INTEGRATION
% ============================================================================

% --- SYSTEM PROMPT (split into parts, concatenate in code) ---
llm_prompt_part(1, 'You are a Prolog specification generator for the Turducken formal methods tool.').
llm_prompt_part(2, 'When the user describes a system, generate a Prolog specification with:').
llm_prompt_part(3, '1. actor/1 facts for each component').
llm_prompt_part(4, '2. actor_initial/2 for initial states').
llm_prompt_part(5, '3. actor_state/3 for states with property lists').
llm_prompt_part(6, '4. actor_transition/4 for state transitions').
llm_prompt_part(7, '5. property/3 for CTL properties to verify').
llm_prompt_part(8, '6. Derived predicates: prop/2, initial/1, transition/3').
llm_prompt_part(9, 'IMPORTANT CONSTRAINTS:').
llm_prompt_part(10, '- All clauses for the same predicate must be contiguous').
llm_prompt_part(11, '- Use single quotes for atoms, not double quotes').
llm_prompt_part(12, '- No discontiguous directive').
llm_prompt_part(13, '- No -> operator, use or(not(A), B) instead').
llm_prompt_part(14, 'Example property formulas:').
llm_prompt_part(15, '- ag(ef(atom(ready))) means can always eventually reach ready').
llm_prompt_part(16, '- ag(or(atom(p), atom(q))) means always p or q').
llm_prompt_part(17, '- ef(atom(done)) means can eventually reach done').
llm_prompt_part(18, 'Wrap your Prolog code in ```prolog blocks.').

% --- PROLOG EXTRACTION REGEX ---
llm_extract_pattern('```prolog([^`]*)```').

% --- API CONFIGS ---
llm_config(openai, endpoint, 'https://api.openai.com/v1/chat/completions').
llm_config(openai, model, 'gpt-4-turbo-preview').
llm_config(openai, env_key, 'OPENAI_API_KEY').

llm_config(anthropic, endpoint, 'https://api.anthropic.com/v1/messages').
llm_config(anthropic, model, 'claude-3-opus-20240229').
llm_config(anthropic, env_key, 'ANTHROPIC_API_KEY').

% ============================================================================
% PART 5: FRONTEND SPECIFICATION
% ============================================================================

% --- CSS THEME (GitHub dark) ---
css_var('--bg-primary', '#0d1117').
css_var('--bg-secondary', '#161b22').
css_var('--bg-tertiary', '#21262d').
css_var('--border-color', '#30363d').
css_var('--text-primary', '#e6edf3').
css_var('--text-secondary', '#8b949e').
css_var('--accent-blue', '#58a6ff').
css_var('--accent-green', '#3fb950').
css_var('--accent-red', '#f85149').
css_var('--accent-yellow', '#d29922').
css_var('--accent-purple', '#a371f7').

% --- LAYOUT ---
ui_layout(container, 'max-width: 1600px; margin: 0 auto; padding: 20px').
ui_layout(main_grid, 'display: grid; grid-template-columns: 1fr 1fr; gap: 20px; height: calc(100vh - 120px)').
ui_layout(panel, 'background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px').

% --- TABS ---
ui_tab(visualize, 'Visualize', 'Shows state machines per actor, sequence diagrams').
ui_tab(editor, 'Prolog Editor', 'Text area to edit spec, Apply button').
ui_tab(query, 'Query', 'Run arbitrary Prolog queries').
ui_tab(check, 'Check', 'Shows named properties with pass/fail status').

% --- MERMAID CONFIG ---
mermaid_config(theme, 'dark').
mermaid_config(securityLevel, 'loose').
mermaid_config(startOnLoad, false).

% --- ACTOR PANE LAYOUT ---
ui_actor_panes(layout, 'display: flex; flex-direction: column; gap: 16px').
ui_actor_panes(pane, 'background: var(--bg-tertiary); padding: 12px; border-radius: 6px; min-height: 120px').
ui_actor_panes(direction, 'LR').  % Left-to-right state diagrams

% --- STATE PREFIX MAPPING ---
state_prefix('ui_', ui).
state_prefix('server_', http_server).
state_prefix('engine_', prolog_engine).
state_prefix('llm_', llm_client).

% --- MESSAGE ANNOTATIONS FOR TRANSITIONS ---
msg_annotation(submit_chat, send, http_server).
msg_annotation(apply_spec, send, http_server).
msg_annotation(run_query, send, http_server).
msg_annotation(chat_response, recv, http_server).
msg_annotation(spec_loaded, recv, http_server).
msg_annotation(query_result, recv, http_server).
msg_annotation(receive_request, recv, ui).
msg_annotation(send_response, send, ui).
msg_annotation(load_spec, recv, http_server).
msg_annotation(query, recv, http_server).
msg_annotation(send_prompt, recv, http_server).
msg_annotation(receive_response, recv, external_api).

% ============================================================================
% PART 6: RUST-SPECIFIC IMPLEMENTATION HINTS
% ============================================================================
%
% If implementing in Rust:
%
% Cargo.toml dependencies:
%   axum = "0.7"           # HTTP framework
%   tokio = { version = "1", features = ["full"] }
%   serde = { version = "1", features = ["derive"] }
%   serde_json = "1"
%   reqwest = { version = "0.11", features = ["json"] }
%   rust-embed = "8"       # Embed static files
%   scryer-prolog = "0.9"  # OR implement mini-prolog below
%
% Mini-Prolog alternative (if scryer too heavy):
%   Implement just enough for CTL:
%   - Term enum: Atom, Var, Compound(name, args)
%   - Unification
%   - Backtracking via iterator/generator
%   - Built-ins: member/2, append/3, \+, forall/2
%   - Parse transition/3, initial/1, prop/2 from source
%
% Project structure:
%   src/
%     main.rs          - Entry point, CLI (clap)
%     prolog/
%       mod.rs         - Engine trait
%       engine.rs      - Prolog interpreter wrapper
%       ctl.rs         - CTL checker (can be pure Rust)
%     server/
%       mod.rs
%       handlers.rs    - Axum handlers
%       static.rs      - Embedded files (rust-embed)
%     llm/
%       mod.rs
%       client.rs      - OpenAI/Anthropic client
%
% Embedding HTML:
%   #[derive(RustEmbed)]
%   #[folder = "static/"]
%   struct Asset;
%
% ============================================================================
% PART 7: GO-SPECIFIC IMPLEMENTATION HINTS  
% ============================================================================
%
% go.mod:
%   module turducken
%   go 1.21
%   require github.com/ichiban/prolog v1.2.0
%
% CRITICAL ichiban/prolog notes:
%   - Scan() signature: sols.Scan(&struct{ Field interface{} }{})
%   - Exec() to load predicates, QueryContext() to query
%   - No :- discontiguous (will fail silently or error)
%   - Double quotes = character list, use single quotes
%
% Embedding:
%   //go:embed static/*
%   var staticFiles embed.FS
%
% ============================================================================
% PART 8: PYTHON-SPECIFIC IMPLEMENTATION HINTS
% ============================================================================
%
% requirements.txt:
%   flask>=3.0
%   requests>=2.31
%   pyswip>=0.2.11  # OR use kanren for pure Python
%
% Pure Python mini-prolog (kanren):
%   from kanren import run, eq, membero, var, conde
%   # Implement CTL as recursive functions with memoization
%
% ============================================================================
% PART 9: ERROR MESSAGES (for consistent UX)
% ============================================================================

error_msg(prolog_parse, 'Failed to parse Prolog: ~w').
error_msg(prolog_discontiguous, 'Predicates must be contiguous. Group all clauses of ~w together.').
error_msg(ctl_timeout, 'CTL check timed out (possible infinite loop in state graph)').
error_msg(ctl_parse, 'Invalid CTL formula: ~w').
error_msg(llm_no_key, 'No API key found. Set ~w environment variable.').
error_msg(llm_timeout, 'LLM request timed out. Try again.').
error_msg(llm_error, 'LLM API error: ~w').
error_msg(no_prolog_block, 'No ```prolog block found in response.').

% ============================================================================
% PART 10: TEST CASES
% ============================================================================

% Load spec, should succeed
test(load_spec, 'POST /api/spec with valid Prolog', expect_success).

% Query transitions
test(query_transitions, 'POST /api/query with "transition(X,Y,Z)."', 
     expect_results_count('>0')).

% CTL check should pass
test(ctl_no_deadlock, 'POST /api/check with "ag(ef(atom(accepting_input)))"',
     expect_satisfied(true)).

% Invalid CTL should return error
test(ctl_invalid, 'POST /api/check with "invalid_formula"',
     expect_error).

% Chat returns Prolog
test(chat_generates, 'POST /api/chat with "design a traffic light"',
     expect_prolog_block).