package prolog

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ichiban/prolog"
)

type Kernel struct {
	mu sync.Mutex
	p  *prolog.Interpreter
}

func NewKernel() *Kernel {
	// Sandbox interpreter: no builtins by default.
	p := new(prolog.Interpreter)
	k := &Kernel{p: p}

	// Minimal CTL starter library (you’ll replace/expand this).
	_ = k.mustExec(`
		% Graph
		:- dynamic edge/2.
		:- dynamic holds/2.

		% EF(p, s): exists a path from s reaching a state where holds(s,p).
		ef(P, S) :- ef(P, S, [S]).
		ef(P, S, _) :- holds(S, P).
		ef(P, S, Vis) :- edge(S, T), \+ member(T, Vis), ef(P, T, [T|Vis]).

		% AG(p, s): for all reachable states from s, p holds.
		% (This is a simple closed-world / NAF-style check; you will likely replace with a bounded or cycle-aware version.)
		ag(P, S) :- \+ ( reachable(S, T), \+ holds(T, P) ).
		reachable(S, S).
		reachable(S, T) :- edge(S, U), reachable(U, T).
	`)
	return k
}

func (k *Kernel) mustExec(src string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.p.Exec(src)
}

func (k *Kernel) Assertz(ctx context.Context, clause string) error {
	_ = ctx // reserved for future cancellation hooks; ichiban/prolog calls are synchronous.
	k.mu.Lock()
	defer k.mu.Unlock()

	// Wrap as assertz((...)).
	return k.p.Exec(fmt.Sprintf(":- assertz((%s)).", strings.TrimSpace(clause)))
}

func (k *Kernel) QueryBool(ctx context.Context, q string) (bool, error) {
	_ = ctx
	k.mu.Lock()
	defer k.mu.Unlock()

	sols, err := k.p.Query(q + ".")
	if err != nil {
		return false, err
	}
	defer sols.Close()

	// Any solution means true.
	return sols.Next(), nil
}

func (k *Kernel) Summary(ctx context.Context) string {
	_ = ctx
	return "Prolog kernel loaded (edge/2, holds/2, ef/2, ag/2)."
}

