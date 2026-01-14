% ============================================================================
% TURDUCKEN: Self-Describing Specification
% A formal model of the turducken system itself
% ============================================================================

% === ACTORS ===
% The system consists of these communicating components

actor(user, ui_idle).
actor(http_server, server_ready).
actor(prolog_engine, engine_empty).
actor(llm_client, llm_idle).
actor(visualizer, viz_idle).

% === PROLOG ENGINE STATE MACHINE ===
% The core interpreter that holds specifications

state(engine_empty, [can_load, no_spec]).
state(engine_loaded, [can_query, can_reset, has_spec]).
state(engine_error, [can_reset]).

initial(engine_empty).

transition(engine_empty, load_spec, engine_loaded).
transition(engine_empty, load_spec_error, engine_error).
transition(engine_loaded, reset, engine_empty).
transition(engine_loaded, load_spec, engine_loaded).  % reload
transition(engine_loaded, query, engine_loaded).      % query doesn't change state
transition(engine_error, reset, engine_empty).

prop(engine_empty, can_load).
prop(engine_empty, no_spec).
prop(engine_loaded, can_query).
prop(engine_loaded, can_reset).
prop(engine_loaded, has_spec).
prop(engine_error, can_reset).

% === LLM CLIENT STATE MACHINE ===
% Manages communication with ChatGPT or Claude

state(llm_idle, [ready]).
state(llm_requesting, [busy]).
state(llm_received, [has_response]).
state(llm_error, [failed]).

initial(llm_idle).

transition(llm_idle, send_prompt, llm_requesting).
transition(llm_requesting, receive_response, llm_received).
transition(llm_requesting, timeout, llm_error).
transition(llm_requesting, api_error, llm_error).
transition(llm_received, extract_prolog, llm_idle).
transition(llm_received, no_prolog, llm_idle).
transition(llm_error, retry, llm_idle).

prop(llm_idle, ready).
prop(llm_requesting, busy).
prop(llm_received, has_response).
prop(llm_error, failed).

% === UI STATE MACHINE ===
% Browser-side user interface states

state(ui_idle, [accepting_input]).
state(ui_chatting, [waiting_response]).
state(ui_applying, [loading_spec]).
state(ui_querying, [running_query]).
state(ui_visualizing, [rendering]).

initial(ui_idle).

transition(ui_idle, submit_chat, ui_chatting).
transition(ui_idle, apply_spec, ui_applying).
transition(ui_idle, run_query, ui_querying).
transition(ui_idle, request_viz, ui_visualizing).
transition(ui_chatting, chat_response, ui_idle).
transition(ui_chatting, chat_error, ui_idle).
transition(ui_applying, spec_loaded, ui_idle).
transition(ui_applying, spec_error, ui_idle).
transition(ui_querying, query_result, ui_idle).
transition(ui_visualizing, viz_rendered, ui_idle).

prop(ui_idle, accepting_input).
prop(ui_chatting, waiting_response).
prop(ui_applying, loading_spec).
prop(ui_querying, running_query).
prop(ui_visualizing, rendering).

% === HTTP SERVER STATE MACHINE ===
% Request handling states

state(server_ready, [listening]).
state(server_handling, [processing]).

initial(server_ready).

transition(server_ready, receive_request, server_handling).
transition(server_handling, send_response, server_ready).

prop(server_ready, listening).
prop(server_handling, processing).

% === CSP-STYLE CHANNELS ===
% Communication channels between components

channel(ui_to_server, 10).       % HTTP requests
channel(server_to_ui, 10).       % HTTP responses
channel(server_to_llm, 5).       % LLM API calls
channel(llm_to_server, 5).       % LLM responses
channel(server_to_engine, 10).   % Prolog operations
channel(engine_to_server, 10).   % Query results

% === MESSAGE TYPES ===
% Messages that flow through channels

% Chat flow
send(ui_to_server, chat_request(Msg), ui_idle, ui_chatting).
send(server_to_llm, prompt(Msg, Context), server_handling, server_handling).
send(llm_to_server, response(Text, Prolog), llm_received, llm_idle).
send(server_to_ui, chat_response(Text, Prolog), server_handling, server_ready).

% Spec flow
send(ui_to_server, apply_spec(Source), ui_idle, ui_applying).
send(server_to_engine, load(Source), server_handling, server_handling).
send(engine_to_server, load_result(Success), engine_loaded, engine_loaded).
send(server_to_ui, spec_result(Success), server_handling, server_ready).

% Query flow
send(ui_to_server, query_request(Query), ui_idle, ui_querying).
send(server_to_engine, execute(Query), server_handling, server_handling).
send(engine_to_server, query_result(Result), engine_loaded, engine_loaded).
send(server_to_ui, query_response(Result), server_handling, server_ready).

% Visualization flow
send(ui_to_server, viz_request(Type), ui_idle, ui_visualizing).
send(server_to_engine, extract(Type), server_handling, server_handling).
send(engine_to_server, viz_data(Data), engine_loaded, engine_loaded).
send(server_to_ui, viz_response(Mermaid), server_handling, server_ready).

% === PROCESS ALGEBRA DEFINITIONS ===
% Recursive equations describing component behavior

% User interaction loop
proc(user_process,
    choice(
        prefix(submit_chat, 
            prefix(chat_response, user_process)),
        choice(
            prefix(apply_spec,
                prefix(spec_result, user_process)),
            choice(
                prefix(run_query,
                    prefix(query_result, user_process)),
                prefix(request_viz,
                    prefix(viz_rendered, user_process)))))).

% HTTP server loop
proc(server_process,
    prefix(receive_request,
        prefix(send_response, server_process))).

% Prolog engine loop
proc(engine_process,
    choice(
        prefix(load_spec, engine_process),
        choice(
            prefix(query, engine_process),
            prefix(reset, engine_process)))).

% LLM client loop  
proc(llm_process,
    prefix(send_prompt,
        choice(
            prefix(receive_response,
                prefix(extract_prolog, llm_process)),
            prefix(api_error,
                prefix(retry, llm_process))))).

% Full system composition
proc(turducken,
    parallel(
        parallel(user_process, server_process),
        parallel(engine_process, llm_process))).

% === SEQUENCE DIAGRAMS ===

% Chat interaction flow
lifeline(user).
lifeline(browser).
lifeline(server).
lifeline(llm).
lifeline(engine).

% Main chat-to-visualization flow
message(1, user, browser, 'describe system').
message(2, browser, server, 'POST /api/chat').
message(3, server, llm, 'prompt + context').
message(4, llm, server, 'response + prolog').
message(5, server, browser, 'JSON response').
message(6, browser, browser, 'populate editor').
message(7, user, browser, 'click Apply').
message(8, browser, server, 'POST /api/spec').
message(9, server, engine, 'load(source)').
message(10, engine, server, 'success').
message(11, server, browser, 'spec loaded').
message(12, browser, server, 'GET /api/visualize').
message(13, server, engine, 'extract(statemachine)').
message(14, engine, server, 'viz_data').
message(15, server, browser, 'mermaid code').
message(16, browser, browser, 'render diagram').

% === CTL TEMPORAL PROPERTIES ===
% Properties that turducken should satisfy

% Safety: Engine never loses spec during query
% If we have a spec, querying preserves it
safety_preserve_spec :-
    check_ctl(ag(and(atom(has_spec), ex(atom(running_query))) -> 
                 ax(atom(has_spec)))).

% Liveness: Chat requests eventually get responses
liveness_chat_response :-
    check_ctl(ag(atom(waiting_response) -> af(atom(accepting_input)))).

% No deadlock: UI can always accept input eventually
no_deadlock :-
    check_ctl(ag(ef(atom(accepting_input)))).

% Liveness: From any state, we can reset to empty
can_always_reset :-
    check_ctl(ag(ef(atom(no_spec)))).

% Safety: LLM errors are recoverable
llm_error_recovery :-
    check_ctl(ag(atom(failed) -> ef(atom(ready)))).

% Invariant: Server is always either listening or processing
server_invariant :-
    check_ctl(ag(or(atom(listening), atom(processing)))).

% === VISUALIZATION DATA ===

% Architecture distribution
pie_slice('HTTP Server', 20).
pie_slice('Prolog Engine', 35).
pie_slice('LLM Client', 25).
pie_slice('UI/Visualization', 20).

% Message flow complexity
line_point(messages, 1, 2).   % user -> browser
line_point(messages, 2, 4).   % + browser -> server
line_point(messages, 3, 6).   % + server -> llm
line_point(messages, 4, 8).   % + llm -> server -> browser
line_point(messages, 5, 12).  % + apply flow
line_point(messages, 6, 16).  % + visualize flow

% Component interactions per request type
bar_value('Chat', 5).
bar_value('Apply', 3).
bar_value('Query', 3).
bar_value('Visualize', 4).

% === META: TURDUCKEN VERIFYING ITSELF ===
% If turducken can load this spec and verify these properties,
% it demonstrates the system is working correctly.

% The system is self-consistent if:
% 1. This spec can be loaded without error
% 2. The state machines are well-formed
% 3. The CTL properties can be checked
% 4. The visualizations can be rendered

self_consistent :-
    initial(S),
    check_ctl(ef(atom(has_spec))),
    check_ctl(ag(ef(atom(accepting_input)))).

% ============================================================================
% USAGE:
% 1. Load this spec into turducken
% 2. Visualize the state machine to see turducken's architecture
% 3. View the sequence diagram for the chat flow
% 4. Check the CTL properties to verify system guarantees
% ============================================================================
