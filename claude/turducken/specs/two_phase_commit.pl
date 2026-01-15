% ============================================================================
% EXAMPLE: Two-Phase Commit Protocol
% ============================================================================
% This specification models a simplified two-phase commit protocol
% between a coordinator and two participants.

% --- State Machine (per actor) ---
actor(coordinator).
actor(participant1).
actor(participant2).

actor_initial(coordinator, coord_init).
actor_initial(participant1, p1_init).
actor_initial(participant2, p2_init).

actor_state(coordinator, coord_init, [ready]).
actor_state(coordinator, coord_preparing, []).
actor_state(coordinator, coord_prepared, [can_commit]).
actor_state(coordinator, coord_committed, [done]).
actor_state(coordinator, coord_aborted, [done]).

actor_state(participant1, p1_init, [ready]).
actor_state(participant1, p1_prepared, [can_commit]).
actor_state(participant1, p1_committed, [done]).
actor_state(participant1, p1_aborted, [done]).

actor_state(participant2, p2_init, [ready]).
actor_state(participant2, p2_prepared, [can_commit]).
actor_state(participant2, p2_committed, [done]).
actor_state(participant2, p2_aborted, [done]).

% Phase 1: Prepare
actor_transition(coordinator, coord_init, prepare, coord_preparing).
actor_transition(coordinator, coord_preparing, vote_yes, coord_prepared).
actor_transition(coordinator, coord_preparing, vote_no, coord_aborted).

% Phase 2: Commit or Abort
actor_transition(coordinator, coord_prepared, commit, coord_committed).
actor_transition(coordinator, coord_prepared, abort, coord_aborted).

actor_transition(participant1, p1_init, recv_prepare, p1_prepared).
actor_transition(participant1, p1_prepared, recv_commit, p1_committed).
actor_transition(participant1, p1_prepared, recv_abort, p1_aborted).

actor_transition(participant2, p2_init, recv_prepare, p2_prepared).
actor_transition(participant2, p2_prepared, recv_commit, p2_committed).
actor_transition(participant2, p2_prepared, recv_abort, p2_aborted).

% Properties for CTL
prop(State, Prop) :- actor_state(_, State, Props), member(Prop, Props).
initial(S) :- actor_initial(_, S).
transition(From, Label, To) :- actor_transition(_, From, Label, To).

% --- Channels (CSP-style) ---
channel(coord_to_p1, 1).
channel(coord_to_p2, 1).
channel(p1_to_coord, 1).
channel(p2_to_coord, 1).

% --- Sequence Diagram (derived from channels) ---
send(coord_to_p1, prepare, coord_init, coord_preparing).
send(coord_to_p2, prepare, coord_init, coord_preparing).

send(p1_to_coord, vote_yes, p1_prepared, p1_prepared).
send(p2_to_coord, vote_yes, p2_prepared, p2_prepared).

send(coord_to_p1, commit, coord_prepared, coord_committed).
send(coord_to_p2, commit, coord_prepared, coord_committed).

recv(coord_to_p1, prepare, p1_init, p1_prepared).
recv(coord_to_p2, prepare, p2_init, p2_prepared).
recv(p1_to_coord, vote_yes, coord_preparing, coord_prepared).
recv(p2_to_coord, vote_yes, coord_preparing, coord_prepared).
recv(coord_to_p1, commit, p1_prepared, p1_committed).
recv(coord_to_p2, commit, p2_prepared, p2_committed).

% --- Process Definitions (Recursive Equations) ---
% Coordinator process
proc(coordinator, 
    prefix(send_prepare, 
        prefix(recv_votes,
            choice(
                prefix(send_commit, skip),
                prefix(send_abort, skip)
            )))).

% Participant process
proc(participant,
    prefix(recv_prepare,
        choice(
            prefix(vote_yes, prefix(recv_decision, skip)),
            prefix(vote_no, skip)
        ))).

% --- Chart Data (for visualization) ---
pie_slice(committed, 85).
pie_slice(aborted, 15).

line_point(latency, 1, 10).
line_point(latency, 2, 15).
line_point(latency, 3, 12).
line_point(latency, 4, 18).
line_point(latency, 5, 14).

bar_value(phase1, 50).
bar_value(phase2, 30).
bar_value(cleanup, 20).

% ============================================================================
% CTL PROPERTIES TO CHECK
% ============================================================================

% Safety: if we reach committed, we stay committed (no rollback)
% check_ctl(ag(or(not(atom(done)), atom(done)))).

% Liveness: from init, we eventually reach done
% check_ctl(af(atom(done))).

% No deadlock from prepared state
% check_ctl(ag(or(not(atom(can_commit)), ef(atom(done))))).
