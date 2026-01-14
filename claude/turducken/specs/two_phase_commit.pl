% ============================================================================
% EXAMPLE: Two-Phase Commit Protocol
% ============================================================================
% This specification models a simplified two-phase commit protocol
% between a coordinator and two participants.

% --- State Machine ---
state(init, [ready]).
state(preparing, []).
state(prepared, [can_commit]).
state(committed, [done]).
state(aborted, [done]).

initial(init).
accepting(committed).
accepting(aborted).

% Phase 1: Prepare
transition(init, prepare, preparing).
transition(preparing, vote_yes, prepared).
transition(preparing, vote_no, aborted).

% Phase 2: Commit or Abort
transition(prepared, commit, committed).
transition(prepared, abort, aborted).

% Properties for CTL
prop(init, ready).
prop(prepared, can_commit).
prop(committed, done).
prop(aborted, done).

% --- Actors ---
actor(coordinator, init).
actor(participant1, init).
actor(participant2, init).

% --- Channels (CSP-style) ---
channel(coord_to_p1, 1).
channel(coord_to_p2, 1).
channel(p1_to_coord, 1).
channel(p2_to_coord, 1).

% --- Sequence Diagram ---
lifeline(coordinator).
lifeline(participant1).
lifeline(participant2).

message(1, coordinator, participant1, prepare).
message(2, coordinator, participant2, prepare).
message(3, participant1, coordinator, vote_yes).
message(4, participant2, coordinator, vote_yes).
message(5, coordinator, participant1, commit).
message(6, coordinator, participant2, commit).

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
