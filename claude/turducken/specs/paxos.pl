% ============================================================================
% PAXOS CONSENSUS ALGORITHM
% ============================================================================
% Based on Leslie Lamport's TLA+ specification
% Simplified single-decree Paxos (consensus on one value)
% ============================================================================

% === OVERVIEW DOCUMENTATION ===
doc(title, 'Paxos Consensus Algorithm').
doc(version, '1.0.0').
doc(author, 'Based on Leslie Lamport TLA+ specification').

doc(overview, 'Paxos is a family of protocols for solving consensus in a network of unreliable processors. Consensus is the process of agreeing on one result among a group of participants.').

doc(description, 'This specification models single-decree Paxos, which achieves consensus on a single value. The protocol proceeds in two phases: Prepare/Promise and Accept/Accepted.').

% === ROLE DOCUMENTATION ===
doc(role_proposer, 'Proposers are responsible for initiating consensus rounds. They send Prepare messages with ballot numbers and, upon receiving promises from a quorum, send Accept messages with a value.').

doc(role_acceptor, 'Acceptors vote on proposals. They respond to Prepare messages with Promises (if the ballot is highest seen) and accept values in Accept messages (if ballot matches their promise).').

doc(role_learner, 'Learners discover the decided value once a quorum of acceptors has accepted. In this simplified spec, learners are implicit.').

% === PHASE DOCUMENTATION ===
doc(phase1a, 'Phase 1a (Prepare): A proposer selects a ballot number n and sends Prepare(n) to a majority of acceptors.').

doc(phase1b, 'Phase 1b (Promise): An acceptor receiving Prepare(n) responds with Promise(n) if n is greater than any ballot it has seen. The promise includes any value the acceptor has already accepted.').

doc(phase2a, 'Phase 2a (Accept): If the proposer receives promises from a majority, it sends Accept(n, v) where v is either a value from a promise or a new value if no acceptor had accepted anything.').

doc(phase2b, 'Phase 2b (Accepted): An acceptor receiving Accept(n, v) accepts it if n equals its promised ballot, and notifies learners.').

% === SAFETY DOCUMENTATION ===
doc(safety_agreement, 'Agreement: Only a single value can be chosen. Once a value is chosen, processes can only learn that value.').

doc(safety_validity, 'Validity: Only a value that has been proposed can be chosen.').

doc(safety_termination, 'Termination: Eventually a value is chosen (under fair scheduling and if failures stop).').

% === VISUALIZATION GUIDE ===
doc(viz_statemachine, 'The State Machine view shows two actors: proposer and acceptor. Each has states reflecting their role in the protocol. Transitions show the actions that move between states.').

doc(viz_sequence, 'The Sequence Diagram shows a successful Paxos round with one proposer and three acceptors. Messages flow: Prepare -> Promise -> Accept -> Accepted.').

doc(viz_pie, 'The Pie Charts show distribution of transitions by type, source actor, and destination actor from a 1000-step simulation random walk.').

doc(viz_line, 'The Line Chart shows cumulative transition counts over simulation steps, revealing which transitions dominate the protocol execution.').

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

% --- Visualization Data ---
{
  "timeline": [
    { "step": 0, "label": "...", "from": "...", "to": "..." },
    ...
  ],
  "byType": { ... },
  "bySrc": { ... },
  "byDst": { ... },
  "total": ...,
  "steps": ...
}