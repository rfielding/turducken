% ============================================================================
% PAXOS CONSENSUS ALGORITHM
% ============================================================================
% Based on Leslie Lamport's TLA+ specification
% Simplified single-decree Paxos (consensus on one value)
%
% Roles:
%   - Proposers: propose values
%   - Acceptors: vote on proposals  
%   - Learners: learn decided values (implicit in this spec)
%
% Phases:
%   Phase 1a: Proposer sends Prepare(n) to acceptors
%   Phase 1b: Acceptor responds with Promise(n, accepted_value)
%   Phase 2a: Proposer sends Accept(n, v) if majority promised
%   Phase 2b: Acceptor accepts if n >= promised ballot
%
% ============================================================================

% === ACTORS ===
actor(proposer).
actor(acceptor).

% === ACTOR INITIAL STATES ===
actor_initial(proposer, proposer_idle).
actor_initial(acceptor, acceptor_idle).

% === ACTOR STATES ===
actor_state(proposer, proposer_idle, [can_propose]).
actor_state(proposer, proposer_preparing, [waiting_promises]).
actor_state(proposer, proposer_proposing, [has_quorum, waiting_accepts]).
actor_state(proposer, proposer_decided, [consensus_reached]).
actor_state(proposer, proposer_preempted, [higher_ballot_seen]).
actor_state(acceptor, acceptor_idle, [can_promise]).
actor_state(acceptor, acceptor_promised, [has_promise, can_accept]).
actor_state(acceptor, acceptor_accepted, [has_accepted, has_promise]).

% === ACTOR TRANSITIONS ===
actor_transition(proposer, proposer_idle, prepare, proposer_preparing).
actor_transition(proposer, proposer_preparing, quorum_promised, proposer_proposing).
actor_transition(proposer, proposer_preparing, higher_ballot, proposer_preempted).
actor_transition(proposer, proposer_preparing, timeout, proposer_idle).
actor_transition(proposer, proposer_proposing, send_accept, proposer_proposing).
actor_transition(proposer, proposer_proposing, quorum_accepted, proposer_decided).
actor_transition(proposer, proposer_proposing, higher_ballot, proposer_preempted).
actor_transition(proposer, proposer_preempted, retry_higher, proposer_idle).
actor_transition(proposer, proposer_decided, new_instance, proposer_idle).
actor_transition(acceptor, acceptor_idle, promise, acceptor_promised).
actor_transition(acceptor, acceptor_idle, reject_prepare, acceptor_idle).
actor_transition(acceptor, acceptor_promised, accept_value, acceptor_accepted).
actor_transition(acceptor, acceptor_promised, reject_accept, acceptor_promised).
actor_transition(acceptor, acceptor_promised, higher_promise, acceptor_promised).
actor_transition(acceptor, acceptor_accepted, higher_promise, acceptor_accepted).
actor_transition(acceptor, acceptor_accepted, accept_same_ballot, acceptor_accepted).

% === DERIVED PREDICATES ===
prop(State, Prop) :- actor_state(_, State, Props), member(Prop, Props).
initial(S) :- actor_initial(_, S).
transition(From, Label, To) :- actor_transition(_, From, Label, To).

% === PAXOS SAFETY PROPERTIES ===
property(agreement, 'Once consensus is reached it persists', 'ag(or(not(atom(consensus_reached)), ag(atom(consensus_reached))))').
property(progress, 'Can always eventually reach consensus', 'ag(ef(atom(consensus_reached)))').
property(acceptor_safety, 'Acceptor must promise before accepting', 'ag(or(not(atom(has_accepted)), atom(has_promise)))').
property(preemption_recovery, 'Preempted proposer can retry', 'ag(or(not(atom(higher_ballot_seen)), ef(atom(can_propose))))').

% === SEQUENCE DIAGRAM: Successful Paxos Round ===
lifeline(proposer1).
lifeline(acceptor1).
lifeline(acceptor2).
lifeline(acceptor3).

message(1, proposer1, acceptor1, prepare_n).
message(2, proposer1, acceptor2, prepare_n).
message(3, proposer1, acceptor3, prepare_n).
message(4, acceptor1, proposer1, promise_n).
message(5, acceptor2, proposer1, promise_n).
message(6, acceptor3, proposer1, promise_n).
message(7, proposer1, acceptor1, accept_n_v).
message(8, proposer1, acceptor2, accept_n_v).
message(9, proposer1, acceptor3, accept_n_v).
message(10, acceptor1, proposer1, accepted_n_v).
message(11, acceptor2, proposer1, accepted_n_v).
message(12, acceptor3, proposer1, accepted_n_v).

% === API INFO ===
api_info(title, 'Paxos Consensus').
api_info(version, '1.0.0').
api_info(description, 'Single-decree Paxos consensus algorithm').

% === DOCUMENTATION ===
doc(algorithm, 'Paxos achieves consensus among distributed nodes despite failures').
doc(phase1, 'Prepare/Promise: Proposer acquires promises from majority').
doc(phase2, 'Accept/Accepted: Proposer gets majority to accept value').
doc(quorum, 'Majority quorum ensures any two quorums overlap').
doc(ballot, 'Ballot numbers are unique and totally ordered').
doc(preemption, 'Higher ballot preempts lower - ensures progress').