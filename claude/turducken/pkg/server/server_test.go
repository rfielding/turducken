package server

import (
	"testing"

	"github.com/rfielding/turducken/pkg/prolog"
)

func xTestSimulationStateGuardBlocksTransitions(t *testing.T) {
	engine, err := prolog.New()
	if err != nil {
		t.Fatalf("prolog.New error: %v", err)
	}

	if err := engine.LoadSpec(`
        initial(s0).
        transition(s0, go, s1).
        state_guard(s0, never).
        never :- fail.
    `); err != nil {
		t.Fatalf("LoadSpec error: %v", err)
	}

	s := &Server{engine: engine}
	s.runAndCacheSimulation(5)

	if s.cachedSimulation == nil {
		t.Fatalf("expected cached simulation result")
	}
	if s.cachedSimulation.Total != 0 {
		t.Errorf("expected 0 transitions, got %d", s.cachedSimulation.Total)
	}
}

func TestSimulationTransitionGuardAllowsTransition(t *testing.T) {
	engine, err := prolog.New()
	if err != nil {
		t.Fatalf("prolog.New error: %v", err)
	}

	if err := engine.LoadSpec(`
        initial(s0).
        transition(s0, go, s1).
        transition_guard(s0, go, s1, always).
        always :- dice0(0.0, 1.0).
    `); err != nil {
		t.Fatalf("LoadSpec error: %v", err)
	}

	s := &Server{engine: engine}
	s.runAndCacheSimulation(1)

	if s.cachedSimulation == nil {
		t.Fatalf("expected cached simulation result")
	}
	if s.cachedSimulation.Total != 1 {
		t.Errorf("expected 1 transition, got %d", s.cachedSimulation.Total)
	}
}
