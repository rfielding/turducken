% filepath: /home/rfielding/code/turducken/claude/turducken/specs/turducken.pl
% ============================================================================
% TURDUCKEN: Self-Describing Specification
% A formal model of the turducken system itself
% ============================================================================

% === PROJECT DOCUMENTATION ===
doc(project, "Turducken is a formal specification tool that combines:
- Prolog for logic and state machine definitions
- CTL model checking for temporal property verification
- LLM integration for natural language to formal spec translation
- Mermaid.js visualization of state machines, sequences, and charts

The name 'turducken' reflects the layered architecture: an LLM wrapped
around a Prolog engine, which itself reasons about state machines and
temporal logic.").

doc(architecture, "The system consists of four actors, each with its own state machine:
1. ui - handles user interaction (browser)
2. http_server - routes requests and manages state
3. prolog_engine - loads specs, runs queries, checks properties
4. llm_client - translates natural language to Prolog specs

Actors communicate via synchronous message passing (see sequence diagram).").

% === NAMED PROPERTIES ===
property(no_deadlock, 'System can always eventually accept input', 'ag(ef(atom(accepting_input)))').
property(can_always_reset, 'System can always eventually be reset', 'ag(ef(atom(no_spec)))').
property(llm_error_recovery, 'LLM errors can always be recovered from', 'ag(or(not(atom(failed)), ef(atom(ready))))').
property(server_invariant, 'Server is always listening or processing', 'ag(or(atom(listening), atom(processing)))').

% === ACTORS ===
actor(ui).
actor(http_server).
actor(prolog_engine).
actor(llm_client).

% === ACTOR INITIAL STATES (all together) ===
actor_initial(ui, ui_idle).
actor_initial(http_server, server_ready).
actor_initial(prolog_engine, engine_empty).
actor_initial(llm_client, llm_idle).

% === ACTOR STATES (all together) ===
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

% === ACTOR TRANSITIONS (all together) ===
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
actor_transition(prolog_engine, engine_loaded, load_spec, engine_loaded).
actor_transition(prolog_engine, engine_loaded, query, engine_loaded).
actor_transition(prolog_engine, engine_error, reset, engine_empty).
actor_transition(llm_client, llm_idle, send_prompt, llm_requesting).
actor_transition(llm_client, llm_requesting, receive_response, llm_received).
actor_transition(llm_client, llm_requesting, timeout, llm_error).
actor_transition(llm_client, llm_requesting, api_error, llm_error).
actor_transition(llm_client, llm_received, extract_prolog, llm_idle).
actor_transition(llm_client, llm_received, no_prolog, llm_idle).
actor_transition(llm_client, llm_error, retry, llm_idle).

% ============================================================================
% DERIVED PREDICATES
% ============================================================================
prop(State, Prop) :- actor_state(_, State, Props), member(Prop, Props).
initial(S) :- actor_initial(_, S).
transition(From, Label, To) :- actor_transition(_, From, Label, To).

% ============================================================================
% INTERACTION DIAGRAM (derived from channels)
% ============================================================================
channel(ui_http, 1).
channel(http_llm, 1).
channel(http_engine, 1).

send(ui_http, post_chat, ui_idle, ui_chatting).
send(http_llm, send_prompt, server_handling, server_handling).
send(http_llm, receive_response, llm_requesting, llm_received).
send(ui_http, chat_response, server_handling, server_ready).

send(ui_http, post_spec, ui_applying, ui_applying).
send(http_engine, load_spec, server_handling, server_handling).
send(http_engine, spec_loaded, engine_loaded, engine_loaded).
send(ui_http, spec_loaded, server_ready, server_ready).

send(ui_http, post_query, ui_querying, ui_querying).
send(http_engine, query, server_handling, server_handling).
send(http_engine, query_result, engine_loaded, engine_loaded).
send(ui_http, query_result, server_ready, server_ready).

recv(ui_http, post_chat, server_ready, server_handling).
recv(http_llm, send_prompt, llm_idle, llm_requesting).
recv(http_llm, receive_response, server_handling, server_handling).
recv(ui_http, chat_response, ui_chatting, ui_idle).

recv(ui_http, post_spec, server_ready, server_handling).
recv(http_engine, load_spec, engine_empty, engine_loaded).
recv(http_engine, spec_loaded, server_handling, server_ready).
recv(ui_http, spec_loaded, ui_applying, ui_idle).

recv(ui_http, post_query, server_ready, server_handling).
recv(http_engine, query, engine_loaded, engine_loaded).
recv(http_engine, query_result, server_handling, server_ready).
recv(ui_http, query_result, ui_querying, ui_idle).

% ============================================================================
% SYNCHRONIZATION POINTS
% ============================================================================
sync(ui, http_server, submit_chat, receive_request).
sync(http_server, llm_client, send_prompt, send_prompt).
sync(llm_client, http_server, receive_response, send_response).
sync(http_server, ui, chat_response, chat_response).
sync(ui, http_server, apply_spec, receive_request).
sync(http_server, prolog_engine, load_spec, load_spec).
sync(prolog_engine, http_server, spec_loaded, send_response).
