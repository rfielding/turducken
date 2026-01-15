% filepath: /home/rfielding/code/turducken/claude/turducken/specs/turducken-bootstrap.pl
% ============================================================================
% TURDUCKEN BOOTSTRAP SPECIFICATION v2
% ============================================================================
% 
% PURPOSE: Complete specification to regenerate turducken in any language.
% Feed this to an LLM to one-shot the entire application.
%
% PROLOG DIALECT: ichiban/prolog (Go) is the reference implementation.
% Can be adapted to SWI-Prolog or GNU Prolog with minor changes:
%   - SWI/GNU support :- discontiguous (ichiban does not)
%   - SWI/GNU support -> operator (ichiban does not)
%   - All dialects: keep predicates contiguous for portability
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

% === SEQUENCE DIAGRAM (derived from channels) ===
channel(ui_http, 1).
channel(http_llm, 1).
channel(http_engine, 1).

send(ui_http, post_chat, ui_idle, ui_chatting).
recv(ui_http, post_chat, server_ready, server_handling).
send(http_llm, send_prompt, server_handling, server_handling).
recv(http_llm, send_prompt, llm_idle, llm_requesting).
send(http_llm, llm_response, llm_requesting, llm_received).
recv(http_llm, llm_response, server_handling, server_handling).
send(ui_http, chat_response, server_handling, server_ready).
recv(ui_http, chat_response, ui_chatting, ui_idle).

send(ui_http, post_spec, ui_applying, ui_applying).
recv(ui_http, post_spec, server_ready, server_handling).
send(http_engine, load_spec, server_handling, server_handling).
recv(http_engine, load_spec, engine_empty, engine_loaded).
send(http_engine, load_result, engine_loaded, engine_loaded).
recv(http_engine, load_result, server_handling, server_ready).
send(ui_http, spec_status, server_ready, server_ready).
recv(ui_http, spec_status, ui_applying, ui_idle).

% ============================================================================
% PART 2: CTL MODEL CHECKING (embed these predicates)
% ============================================================================
%
% sat(S, atom(P)) :- prop(S, P).
% sat(S, not(F)) :- \+ sat(S, F).
% sat(S, and(F, G)) :- sat(S, F), sat(S, G).
% sat(S, or(F, G)) :- sat(S, F).
% sat(S, or(F, G)) :- sat(S, G).
% sat(S, ex(F)) :- transition(S, _, S2), sat(S2, F).
% sat(S, ax(F)) :- forall(transition(S, _, S2), sat(S2, F)).
% sat(S, ef(F)) :- ef_check(S, F, []).
% sat(S, af(F)) :- af_check(S, F, []).
% sat(S, eg(F)) :- eg_check(S, F, []).
% sat(S, ag(F)) :- ag_check(S, F, []).
% check_ctl(Formula) :- forall(initial(S), sat(S, Formula)).
%
% (Full implementation with cycle detection in engine source)

% ============================================================================
% PART 3: HTTP API SPECIFICATION (Swagger-compatible)
% ============================================================================

% --- API INFO ---
api_info(title, 'Turducken API').
api_info(version, '1.0.0').
api_info(description, 'Formal methods tool with Prolog backend and CTL model checking').
api_info(base_path, '/api').

% --- ENDPOINTS ---
api_endpoint('POST', '/spec', 'Load Prolog specification', spec).
api_endpoint('POST', '/query', 'Execute Prolog query', query).
api_endpoint('POST', '/check', 'Verify CTL property', check).
api_endpoint('POST', '/chat', 'Send message to LLM', chat).
api_endpoint('GET', '/visualize', 'Get visualization data', visualize).
api_endpoint('GET', '/metrics', 'Get request counters', metrics).
api_endpoint('POST', '/reset', 'Reset engine to original spec', reset).
api_endpoint('GET', '/properties', 'Get named properties with results', properties).

% --- REQUEST FIELDS ---
% api_request(operation_id, field_name, field_type, required, description)
api_request(spec, source, string, true, 'Prolog source code').
api_request(query, query, string, true, 'Prolog query ending with period').
api_request(check, formula, string, true, 'CTL formula e.g. ag(ef(atom(ready)))').
api_request(chat, message, string, true, 'User message').
api_request(chat, context, string, false, 'Current spec for LLM context').
api_request(visualize, type, string, false, 'One of: stateMachine, sequence, pie, line').

% --- RESPONSE FIELDS ---
% api_response(operation_id, field_name, field_type, description)
api_response(spec, success, boolean, 'Whether spec loaded successfully').
api_response(spec, error, string, 'Error message if failed').
api_response(query, results, array, 'Array of binding objects').
api_response(query, error, string, 'Error message if failed').
api_response(check, satisfied, boolean, 'Whether CTL formula holds').
api_response(check, error, string, 'Error message if failed').
api_response(chat, response, string, 'LLM response text').
api_response(chat, prolog, string, 'Extracted prolog code block or null').
api_response(chat, error, string, 'Error message if failed').
api_response(visualize, stateMachine, object, 'State machine with transitions and initial').
api_response(visualize, sequence, object, 'Sequence diagram with lifelines and messages').
api_response(metrics, counters, object, 'Map of metric name to count').
api_response(metrics, timeSeries, array, 'Historical counter snapshots').
api_response(reset, success, boolean, 'Whether reset succeeded').
api_response(properties, properties, array, 'Array of property objects with satisfied field').

% --- HTTP STATUS CODES ---
api_status(200, 'OK', 'Request succeeded').
api_status(400, 'Bad Request', 'Invalid input').
api_status(500, 'Internal Server Error', 'Server error').

% --- METRICS ---
metric(spec_loads, counter, 'Times spec was loaded').
metric(queries, counter, 'Prolog queries executed').
metric(ctl_checks, counter, 'CTL checks performed').
metric(chat_requests, counter, 'LLM requests made').
metric(errors, counter, 'Errors encountered').

% ============================================================================
% PART 4: LLM INTEGRATION
% ============================================================================

% --- SYSTEM PROMPT PARTS (concatenate with newlines) ---
llm_prompt_part(1, 'You are a Prolog specification generator for a formal methods tool.').
llm_prompt_part(2, 'Generate specifications with: actor/1, actor_initial/2, actor_state/3, actor_transition/4, property/3').
llm_prompt_part(3, 'Include derived predicates: prop/2, initial/1, transition/3').
llm_prompt_part(4, 'Keep all clauses for same predicate contiguous.').
llm_prompt_part(5, 'Use single quotes for atoms. No discontiguous directive. No -> operator.').
llm_prompt_part(6, 'Wrap Prolog code in triple-backtick prolog blocks.').

% --- API PROVIDERS ---
llm_provider(openai, 'https://api.openai.com/v1/chat/completions', 'OPENAI_API_KEY').
llm_provider(anthropic, 'https://api.anthropic.com/v1/messages', 'ANTHROPIC_API_KEY').

% ============================================================================
% PART 5: FRONTEND SPECIFICATION
% ============================================================================

% --- CSS THEME ---
css_var('--bg-primary', '#0d1117').
css_var('--bg-secondary', '#161b22').
css_var('--bg-tertiary', '#21262d').
css_var('--border-color', '#30363d').
css_var('--text-primary', '#e6edf3').
css_var('--text-secondary', '#8b949e').
css_var('--accent-blue', '#58a6ff').
css_var('--accent-green', '#3fb950').
css_var('--accent-red', '#f85149').

% --- TABS ---
ui_tab(visualize, 'Visualize', 'State machines per actor, sequence diagrams').
ui_tab(editor, 'Editor', 'Edit and apply Prolog spec').
ui_tab(query, 'Query', 'Run Prolog queries').
ui_tab(check, 'Check', 'Verify CTL properties').

% --- STATE PREFIX TO ACTOR MAPPING ---
state_prefix('ui_', ui).
state_prefix('server_', http_server).
state_prefix('engine_', prolog_engine).
state_prefix('llm_', llm_client).

% --- MESSAGE ANNOTATIONS (for state diagram labels) ---
msg_annotation(submit_chat, send, http_server).
msg_annotation(chat_response, recv, http_server).
msg_annotation(apply_spec, send, http_server).
msg_annotation(spec_loaded, recv, http_server).
msg_annotation(receive_request, recv, ui).
msg_annotation(send_response, send, ui).
msg_annotation(load_spec, recv, http_server).
msg_annotation(send_prompt, recv, http_server).
msg_annotation(receive_response, recv, external_api).

% ============================================================================
% PART 6: ERROR MESSAGES
% ============================================================================

error_msg(prolog_parse, 'Failed to parse Prolog').
error_msg(prolog_discontiguous, 'Predicates must be contiguous').
error_msg(ctl_timeout, 'CTL check timed out').
error_msg(ctl_parse, 'Invalid CTL formula').
error_msg(llm_no_key, 'No API key found').
error_msg(llm_timeout, 'LLM request timed out').
error_msg(no_prolog_block, 'No prolog block in response').

% ============================================================================
