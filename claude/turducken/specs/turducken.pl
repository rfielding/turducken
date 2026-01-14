% filepath: /home/rfielding/code/turducken/claude/turducken/specs/turducken.pl
% ============================================================================
% TURDUCKEN: Self-Describing Specification
% A formal model of the turducken system itself
% ============================================================================

% === PROJECT DOCUMENTATION ===
% Use doc/2 for prose descriptions that aren't formalized

doc(project, "Turducken is a formal specification tool that combines:
- Prolog for logic and state machine definitions
- CTL model checking for temporal property verification
- LLM integration for natural language to formal spec translation
- Mermaid.js visualization of state machines, sequences, and charts

The name 'turducken' reflects the layered architecture: an LLM wrapped
around a Prolog engine, which itself reasons about state machines and
temporal logic.").

doc(architecture, "The system consists of four main actors:
1. UI (browser) - handles user interaction and visualization
2. HTTP Server - routes requests and manages state
3. Prolog Engine - loads specs, runs queries, checks properties
4. LLM Client - translates natural language to Prolog specs").

doc(usage, "To use turducken:
1. Describe your system in the Chat panel
2. The LLM generates a Prolog specification
3. Apply the spec to load it into the Prolog engine
4. Visualize state machines, sequences, and charts
5. Check temporal properties using CTL formulas").

% === NAMED PROPERTIES (for Check tab) ===
% property(Name, Description, Formula) - named CTL properties

property(safety_preserve_spec, 
    "If spec is loaded and query runs, spec remains loaded",
    ag(or(not(and(atom(has_spec), ex(atom(running_query)))), ax(atom(has_spec))))).

property(liveness_chat_response,
    "Chat responses are always eventually received", 
    ag(or(not(atom(waiting_response)), af(atom(accepting_input))))).

property(no_deadlock,
    "System can always eventually accept input",
    ag(ef(atom(accepting_input)))).

property(can_always_reset,
    "System can always eventually be reset",
    ag(ef(atom(no_spec)))).

property(llm_error_recovery,
    "LLM errors can always be recovered from",
    ag(or(not(atom(failed)), ef(atom(ready))))).

property(server_invariant,
    "Server is always listening or processing",
    ag(or(atom(listening), atom(processing)))).

% === ACTORS ===
actor(user, ui_idle).
actor(http_server, server_ready).
actor(prolog_engine, engine_empty).
actor(llm_client, llm_idle).

% === STATES ===
state(engine_empty, [can_load, no_spec]).
state(engine_loaded, [can_query, can_reset, has_spec]).
state(engine_error, [can_reset]).
state(llm_idle, [ready]).
state(llm_requesting, [busy]).
state(llm_received, [has_response]).
state(llm_error, [failed]).
state(ui_idle, [accepting_input]).
state(ui_chatting, [waiting_response]).
state(ui_applying, [loading_spec]).
state(ui_querying, [running_query]).
state(server_ready, [listening]).
state(server_handling, [processing]).

% === INITIAL STATES ===
initial(engine_empty).
initial(llm_idle).
initial(ui_idle).
initial(server_ready).

% === TRANSITIONS ===
transition(engine_empty, load_spec, engine_loaded).
transition(engine_empty, load_spec_error, engine_error).
transition(engine_loaded, reset, engine_empty).
transition(engine_loaded, load_spec, engine_loaded).
transition(engine_loaded, query, engine_loaded).
transition(engine_error, reset, engine_empty).
transition(llm_idle, send_prompt, llm_requesting).
transition(llm_requesting, receive_response, llm_received).
transition(llm_requesting, timeout, llm_error).
transition(llm_requesting, api_error, llm_error).
transition(llm_received, extract_prolog, llm_idle).
transition(llm_received, no_prolog, llm_idle).
transition(llm_error, retry, llm_idle).
transition(ui_idle, submit_chat, ui_chatting).
transition(ui_idle, apply_spec, ui_applying).
transition(ui_idle, run_query, ui_querying).
transition(ui_chatting, chat_response, ui_idle).
transition(ui_chatting, chat_error, ui_idle).
transition(ui_applying, spec_loaded, ui_idle).
transition(ui_applying, spec_error, ui_idle).
transition(ui_querying, query_result, ui_idle).
transition(server_ready, receive_request, server_handling).
transition(server_handling, send_response, server_ready).

% === PROPS ===
prop(engine_empty, can_load).
prop(engine_empty, no_spec).
prop(engine_loaded, can_query).
prop(engine_loaded, can_reset).
prop(engine_loaded, has_spec).
prop(engine_error, can_reset).
prop(llm_idle, ready).
prop(llm_requesting, busy).
prop(llm_received, has_response).
prop(llm_error, failed).
prop(ui_idle, accepting_input).
prop(ui_chatting, waiting_response).
prop(ui_applying, loading_spec).
prop(ui_querying, running_query).
prop(server_ready, listening).
prop(server_handling, processing).

% === SEQUENCE DIAGRAM ===
lifeline(user).
lifeline(browser).
lifeline(server).
lifeline(llm).
lifeline(engine).

message(1, user, browser, describe_system).
message(2, browser, server, post_chat).
message(3, server, llm, prompt_context).
message(4, llm, server, response_prolog).
message(5, server, browser, json_response).
message(6, browser, browser, populate_editor).
message(7, user, browser, click_apply).
message(8, browser, server, post_spec).
message(9, server, engine, load_source).
message(10, engine, server, success).
message(11, server, browser, spec_loaded).

% === PIE CHART ===
pie_slice(http_server, 20).
pie_slice(prolog_engine, 35).
pie_slice(llm_client, 25).
pie_slice(ui_visualization, 20).

% === LINE CHART ===
line_point(messages, 1, 2).
line_point(messages, 2, 4).
line_point(messages, 3, 6).
line_point(messages, 4, 8).
line_point(messages, 5, 12).
line_point(messages, 6, 16).