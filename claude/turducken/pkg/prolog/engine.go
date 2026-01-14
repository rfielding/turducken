package prolog

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ichiban/prolog"
)

// Engine wraps ichiban/prolog interpreter with turducken-specific functionality
type Engine struct {
	mu          sync.RWMutex
	interpreter *prolog.Interpreter
	specSource  string
}

// New creates a new Prolog engine with the core turducken predicates loaded
func New() (*Engine, error) {
	e := &Engine{
		interpreter: prolog.New(nil, nil),
	}
	
	// Load core predicates for CTL, CSP, and visualization
	if err := e.loadCore(); err != nil {
		return nil, fmt.Errorf("loading core predicates: %w", err)
	}
	
	return e, nil
}

// loadCore loads the built-in predicates for CTL model checking and CSP
func (e *Engine) loadCore() error {
	core := `
% ============================================================================
% TURDUCKEN CORE PREDICATES
% CTL Model Checking + CSP Message Passing + Visualization Extraction
% ============================================================================

% --- State Machine Representation ---
% state(Name, Props) - declares a state with properties
% transition(From, Label, To) - declares a labeled transition
% initial(State) - marks initial state
% accepting(State) - marks accepting state

% --- CTL Operators (Kripke structure based) ---
% The model is defined by: state/2, transition/3, prop/2

% prop(State, Prop) - State satisfies atomic proposition Prop

% EX(Phi) - exists next state satisfying Phi
ctl_ex(State, Phi) :-
    transition(State, _, Next),
    ctl_sat(Next, Phi).

% AX(Phi) - all next states satisfy Phi  
ctl_ax(State, Phi) :-
    findall(Next, transition(State, _, Next), Nexts),
    Nexts \= [],
    forall(member(N, Nexts), ctl_sat(N, Phi)).

% EF(Phi) - exists path to state satisfying Phi (reachability)
ctl_ef(State, Phi) :-
    ctl_ef(State, Phi, []).

ctl_ef(State, Phi, _Visited) :-
    ctl_sat(State, Phi).
ctl_ef(State, Phi, Visited) :-
    \+ member(State, Visited),
    transition(State, _, Next),
    ctl_ef(Next, Phi, [State|Visited]).

% AF(Phi) - all paths lead to state satisfying Phi
ctl_af(State, Phi) :-
    ctl_af(State, Phi, []).

ctl_af(State, Phi, _Visited) :-
    ctl_sat(State, Phi).
ctl_af(State, Phi, Visited) :-
    \+ member(State, Visited),
    findall(Next, transition(State, _, Next), Nexts),
    Nexts \= [],
    forall(member(N, Nexts), ctl_af(N, Phi, [State|Visited])).

% EG(Phi) - exists infinite path where Phi always holds
ctl_eg(State, Phi) :-
    ctl_eg(State, Phi, []).

ctl_eg(State, Phi, Visited) :-
    ctl_sat(State, Phi),
    (member(State, Visited) -> true ;
     (transition(State, _, Next),
      ctl_eg(Next, Phi, [State|Visited]))).

% AG(Phi) - Phi holds globally on all paths
ctl_ag(State, Phi) :-
    ctl_ag(State, Phi, []).

ctl_ag(State, Phi, Visited) :-
    ctl_sat(State, Phi),
    (member(State, Visited) -> true ;
     (findall(Next, transition(State, _, Next), Nexts),
      forall(member(N, Nexts), ctl_ag(N, Phi, [State|Visited])))).

% E[Phi U Psi] - exists path where Phi until Psi
ctl_eu(State, _Phi, Psi, _Visited) :-
    ctl_sat(State, Psi).
ctl_eu(State, Phi, Psi, Visited) :-
    \+ member(State, Visited),
    ctl_sat(State, Phi),
    transition(State, _, Next),
    ctl_eu(Next, Phi, Psi, [State|Visited]).

% A[Phi U Psi] - all paths: Phi until Psi
ctl_au(State, _Phi, Psi, _Visited) :-
    ctl_sat(State, Psi).
ctl_au(State, Phi, Psi, Visited) :-
    \+ member(State, Visited),
    ctl_sat(State, Phi),
    findall(Next, transition(State, _, Next), Nexts),
    Nexts \= [],
    forall(member(N, Nexts), ctl_au(N, Phi, Psi, [State|Visited])).

% Satisfaction relation
ctl_sat(State, atom(P)) :- prop(State, P).
ctl_sat(State, not(Phi)) :- \+ ctl_sat(State, Phi).
ctl_sat(State, and(Phi, Psi)) :- ctl_sat(State, Phi), ctl_sat(State, Psi).
ctl_sat(State, or(Phi, Psi)) :- (ctl_sat(State, Phi) ; ctl_sat(State, Psi)).
ctl_sat(State, ex(Phi)) :- ctl_ex(State, Phi).
ctl_sat(State, ax(Phi)) :- ctl_ax(State, Phi).
ctl_sat(State, ef(Phi)) :- ctl_ef(State, Phi).
ctl_sat(State, af(Phi)) :- ctl_af(State, Phi).
ctl_sat(State, eg(Phi)) :- ctl_eg(State, Phi).
ctl_sat(State, ag(Phi)) :- ctl_ag(State, Phi).
ctl_sat(State, eu(Phi, Psi)) :- ctl_eu(State, Phi, Psi, []).
ctl_sat(State, au(Phi, Psi)) :- ctl_au(State, Phi, Psi, []).

% Check property from initial state
check_ctl(Phi) :-
    initial(S),
    ctl_sat(S, Phi).

% --- CSP-Style Message Passing ---
% channel(Name, Capacity) - buffered channel with capacity
% send(Channel, Msg, FromState, ToState) - send message
% recv(Channel, Msg, FromState, ToState) - receive message

% Channel state representation
% channel_state(Channel, Messages) - current buffer contents

% --- Actors ---
% actor(Name, InitialState) - declares an actor
% actor_transition(Actor, FromState, Event, ToState) - actor state machine

% --- Visualization Extraction ---

% Get all states for state machine diagram
all_states(States) :-
    findall(S, (state(S, _) ; transition(S, _, _) ; transition(_, _, S)), Bag),
    sort(Bag, States).

% Get all transitions for state machine diagram  
all_transitions(Transitions) :-
    findall(t(From, Label, To), transition(From, Label, To), Transitions).

% Get initial states
all_initial(States) :-
    findall(S, initial(S), States).

% Get accepting states
all_accepting(States) :-
    findall(S, accepting(S), States).

% --- Interaction Diagram (Sequence) ---
% message(Seq, From, To, Label) - message in sequence diagram
% lifeline(Actor) - participant in sequence

all_lifelines(Lifelines) :-
    findall(L, lifeline(L), Lifelines).

all_messages(Messages) :-
    findall(m(Seq, From, To, Label), message(Seq, From, To, Label), Bag),
    sort(Bag, Messages).

% --- Charts ---
% pie_slice(Label, Value) - slice of pie chart
% line_point(Series, X, Y) - point on line chart
% bar_value(Label, Value) - bar chart value

all_pie_slices(Slices) :-
    findall(slice(L, V), pie_slice(L, V), Slices).

all_line_points(Series, Points) :-
    findall(point(X, Y), line_point(Series, X, Y), Points).

all_bar_values(Values) :-
    findall(bar(L, V), bar_value(L, V), Values).

% --- Requirements as Recursive Equations ---
% These can be defined in spec files like:
% proc(Name, Def) where Def can use:
%   prefix(Event, Continuation)
%   choice(P1, P2)
%   parallel(P1, P2)
%   stop
%   skip

expand_proc(Name, Expanded) :-
    proc(Name, Def),
    expand_def(Def, Expanded).

expand_def(stop, stop).
expand_def(skip, skip).
expand_def(prefix(E, P), prefix(E, Expanded)) :-
    expand_proc(P, Expanded).
expand_def(choice(P1, P2), choice(E1, E2)) :-
    expand_proc(P1, E1),
    expand_proc(P2, E2).
expand_def(parallel(P1, P2), parallel(E1, E2)) :-
    expand_proc(P1, E1),
    expand_proc(P2, E2).

% --- Utility predicates ---
member(X, [X|_]).
member(X, [_|T]) :- member(X, T).

append([], L, L).
append([H|T], L, [H|R]) :- append(T, L, R).

length([], 0).
length([_|T], N) :- length(T, N1), N is N1 + 1.
`
	return e.interpreter.Exec(core)
}

// LoadSpec loads a Prolog specification from a string
func (e *Engine) LoadSpec(source string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.specSource = source
	return e.interpreter.Exec(source)
}

// LoadSpecFile loads a Prolog specification from a file
func (e *Engine) LoadSpecFile(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if err := e.interpreter.Exec(fmt.Sprintf(`:- consult('%s').`, path)); err != nil {
		return err
	}
	return nil
}

// Query executes a Prolog query and returns solutions
func (e *Engine) Query(ctx context.Context, query string) ([]map[string]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	sols, err := e.interpreter.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer sols.Close()
	
	var results []map[string]string
	for sols.Next() {
		// For now, just record that we found a solution
		// ichiban/prolog requires knowing variable names to scan
		result := make(map[string]string)
		result["_solution"] = "true"
		results = append(results, result)
	}
	
	return results, sols.Err()
}

// QueryOne executes a query expecting at most one solution
func (e *Engine) QueryOne(ctx context.Context, query string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	sols, err := e.interpreter.QueryContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer sols.Close()
	
	return sols.Next(), sols.Err()
}

// GetStateMachine extracts state machine data from the loaded spec
func (e *Engine) GetStateMachine(ctx context.Context) (*StateMachine, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	sm := &StateMachine{
		States:      []string{},
		Transitions: []Transition{},
		Initial:     []string{},
		Accepting:   []string{},
	}
	
	// Get states
	sols, err := e.interpreter.QueryContext(ctx, "all_states(States).")
	if err != nil {
		return nil, err
	}
	if sols.Next() {
		// Extract states from solution
	}
	sols.Close()
	
	// Get transitions
	sols, err = e.interpreter.QueryContext(ctx, "transition(From, Label, To).")
	if err != nil {
		return nil, err
	}
	for sols.Next() {
		// Extract transitions
	}
	sols.Close()
	
	return sm, nil
}

// StateMachine represents extracted state machine data
type StateMachine struct {
	States      []string     `json:"states"`
	Transitions []Transition `json:"transitions"`
	Initial     []string     `json:"initial"`
	Accepting   []string     `json:"accepting"`
}

// Transition represents a state machine transition
type Transition struct {
	From  string `json:"from"`
	Label string `json:"label"`
	To    string `json:"to"`
}

// GetSource returns the current specification source
func (e *Engine) GetSource() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.specSource
}

// Reset clears all dynamic predicates and reloads core
func (e *Engine) Reset() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.interpreter = prolog.New(nil, nil)
	e.specSource = ""
	return e.loadCore()
}

// RawQuery returns raw string output from a query (for debugging)
func (e *Engine) RawQuery(ctx context.Context, query string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	sols, err := e.interpreter.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer sols.Close()
	
	var results []string
	for sols.Next() {
		results = append(results, "true")
	}
	
	if len(results) == 0 {
		return "false", nil
	}
	return strings.Join(results, "\n"), sols.Err()
}
