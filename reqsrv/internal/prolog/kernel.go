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
	p := new(prolog.Interpreter)
	k := &Kernel{p: p}

	_, _ = k.p.Exec(`
		:- dynamic edge/2.
		:- dynamic holds/2.

		% EF(P, S): exists a path from S to a state where holds(State,P).
		ef(P, S) :- ef(P, S, [S]).
		ef(P, S, _) :- holds(S, P).
		ef(P, S, Vis) :- edge(S, T), \\+ member(T, Vis), ef(P, T, [T|Vis]).

		% Reachability (naive; replace with bounded + memoization later)
		reachable(S, S).
		reachable(S, T) :- edge(S, U), reachable(U, T).

		% AG(P, S): no reachable state violates P (NAF).
		ag(P, S) :- \\+ (reachable(S, T), \\+ holds(T, P)).
	`)
	return k
}

func (k *Kernel) Assertz(ctx context.Context, clause string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	c := strings.TrimSpace(clause)
	if c == "" {
		return fmt.Errorf("empty clause")
	}

	if strings.HasSuffix(c, ".") {
		c = strings.TrimSuffix(c, ".")
	}
	_, err := k.p.Exec(fmt.Sprintf(":- assertz((%s)).", c))
	return err
}

func (k *Kernel) QueryBool(ctx context.Context, q string) (bool, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	query := strings.TrimSpace(q)
	if query == "" {
		return false, fmt.Errorf("empty query")
	}
	if !strings.HasSuffix(query, ".") {
		query += "."
	}

	sols, err := k.p.Query(query)
	if err != nil {
		return false, err
	}
	defer sols.Close()
	return sols.Next(), nil
}

func (k *Kernel) Summary(ctx context.Context) string {
	return "Kernel: dynamic edge/2, holds/2; CTL: ef/2 (cycle-safe), ag/2 (NAF over reachable/2)."
}
