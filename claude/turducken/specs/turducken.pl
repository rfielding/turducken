% filepath: /home/rfielding/code/turducken/claude/turducken/specs/turducken.pl
% ============================================================================
% TURDUCKEN: Self-Describing Specification
% A formal model of the turducken system itself
% ============================================================================

% === ACTORS ===
actor(user, ui_idle).
actor(http_server, server_ready).
actor(prolog_engine, engine_empty).
actor(llm_client, llm_idle).
actor(visualizer, viz_idle).

% === ALL STATES (grouped together) ===
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
state(ui_visualizing, [rendering]).
state(server_ready, [listening]).
state(server_handling, [processing]).

% === ALL INITIAL STATES (grouped together) ===
initial(engine_empty).
initial(llm_idle).
initial(ui_idle).
initial(server_ready).

% === ALL TRANSITIONS (grouped together) ===
% Prolog engine transitions
transition(engine_empty, load_spec, engine_loaded).
transition(engine_empty, load_spec_error, engine_error).
transition(engine_loaded, reset, engine_empty).
transition(engine_loaded, load_spec, engine_loaded).
transition(engine_loaded, query, engine_loaded).
transition(engine_error, reset, engine_empty).
% LLM client transitions
transition(llm_idle, send_prompt, llm_requesting).
transition(llm_requesting, receive_response, llm_received).
transition(llm_requesting, timeout, llm_error).
transition(llm_requesting, api_error, llm_error).
transition(llm_received, extract_prolog, llm_idle).
transition(llm_received, no_prolog, llm_idle).
transition(llm_error, retry, llm_idle).
% UI transitions
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
% Server transitions
transition(server_ready, receive_request, server_handling).
transition(server_handling, send_response, server_ready).

% === ALL PROPS (grouped together) ===
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
prop(ui_visualizing, rendering).
prop(server_ready, listening).
prop(server_handling, processing).

% === CSP-STYLE CHANNELS ===
channel(ui_to_server, 10).
channel(server_to_ui, 10).
channel(server_to_llm, 5).
channel(llm_to_server, 5).
channel(server_to_engine, 10).
channel(engine_to_server, 10).

% === MESSAGE TYPES ===
send(ui_to_server, chat_request(Msg), ui_idle, ui_chatting).
send(server_to_llm, prompt(Msg, Context), server_handling, server_handling).
send(llm_to_server, response(Text, Prolog), llm_received, llm_idle).
send(server_to_ui, chat_response(Text, Prolog), server_handling, server_ready).
send(ui_to_server, apply_spec(Source), ui_idle, ui_applying).
send(server_to_engine, load(Source), server_handling, server_handling).
send(engine_to_server, load_result(Success), engine_loaded, engine_loaded).
send(server_to_ui, spec_result(Success), server_handling, server_ready).
send(ui_to_server, query_request(Query), ui_idle, ui_querying).
send(server_to_engine, execute(Query), server_handling, server_handling).
send(engine_to_server, query_result(Result), engine_loaded, engine_loaded).
send(server_to_ui, query_response(Result), server_handling, server_ready).
send(ui_to_server, viz_request(Type), ui_idle, ui_visualizing).
send(server_to_engine, extract(Type), server_handling, server_handling).
send(engine_to_server, viz_data(Data), engine_loaded, engine_loaded).
send(server_to_ui, viz_response(Mermaid), server_handling, server_ready).

% === PROCESS ALGEBRA DEFINITIONS ===
proc(user_process,
    choice(
        prefix(submit_chat, prefix(chat_response, user_process)),
        choice(
            prefix(apply_spec, prefix(spec_result, user_process)),
            choice(
                prefix(run_query, prefix(query_result, user_process)),
                prefix(request_viz, prefix(viz_rendered, user_process)))))).
proc(server_process,
    prefix(receive_request, prefix(send_response, server_process))).
proc(engine_process,
    choice(
        prefix(load_spec, engine_process),
        choice(prefix(query, engine_process), prefix(reset, engine_process)))).
proc(llm_process,
    prefix(send_prompt,
        choice(
            prefix(receive_response, prefix(extract_prolog, llm_process)),
            prefix(api_error, prefix(retry, llm_process))))).
proc(turducken,
    parallel(
        parallel(user_process, server_process),
        parallel(engine_process, llm_process))).

% === SEQUENCE DIAGRAMS ===
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
message(12, browser, server, get_visualize).
message(13, server, engine, extract_statemachine).
message(14, engine, server, viz_data).
message(15, server, browser, mermaid_code).
message(16, browser, browser, render_diagram).

% === VISUALIZATION DATA ===
pie_slice(http_server, 20).
pie_slice(prolog_engine, 35).
pie_slice(llm_client, 25).
pie_slice(ui_visualization, 20).

line_point(messages, 1, 2).
line_point(messages, 2, 4).
line_point(messages, 3, 6).
line_point(messages, 4, 8).
line_point(messages, 5, 12).
line_point(messages, 6, 16).

bar_value(chat, 5).
bar_value(apply, 3).
bar_value(query, 3).
bar_value(visualize, 4).

% === CTL TEMPORAL PROPERTIES ===
% Using or(not(A), B) instead of A -> B since ichiban/prolog doesn't support ->

safety_preserve_spec :-
    check_ctl(ag(or(not(and(atom(has_spec), ex(atom(running_query)))), ax(atom(has_spec))))).

liveness_chat_response :-
    check_ctl(ag(or(not(atom(waiting_response)), af(atom(accepting_input))))).

no_deadlock :-
    check_ctl(ag(ef(atom(accepting_input)))).

can_always_reset :-
    check_ctl(ag(ef(atom(no_spec)))).

llm_error_recovery :-
    check_ctl(ag(or(not(atom(failed)), ef(atom(ready))))).

server_invariant :-
    check_ctl(ag(or(atom(listening), atom(processing)))).

% === META ===
self_consistent :-
    initial(S),
    check_ctl(ef(atom(has_spec))),
    check_ctl(ag(ef(atom(accepting_input)))).