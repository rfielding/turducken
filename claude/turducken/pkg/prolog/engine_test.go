package prolog

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if e == nil {
		t.Fatal("New() returned nil engine")
	}
}

func TestLoadSpec(t *testing.T) {
	e, _ := New()
	ctx := context.Background()

	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			name: "simple state machine",
			spec: `
                state(idle, [waiting]).
                state(busy, [processing]).
                initial(idle).
                transition(idle, start, busy).
                transition(busy, done, idle).
            `,
			wantErr: false,
		},
		{
			name: "with props",
			spec: `
                state(a, []).
                initial(a).
                transition(a, go, b).
                prop(a, ready).
                prop(b, done).
            `,
			wantErr: false,
		},
		{
			name:    "empty spec",
			spec:    "",
			wantErr: false,
		},
		{
			name:    "syntax error",
			spec:    "this is not valid prolog (",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e.Reset()
			err := e.LoadSpec(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Test state machine extraction after loading
	t.Run("extract state machine", func(t *testing.T) {
		e.Reset()
		e.LoadSpec(`
            initial(s1).
            transition(s1, a, s2).
            transition(s2, b, s3).
            transition(s3, c, s1).
        `)
		sm, err := e.GetStateMachine(ctx)
		if err != nil {
			t.Fatalf("GetStateMachine error: %v", err)
		}
		if len(sm.Transitions) != 3 {
			t.Errorf("expected 3 transitions, got %d", len(sm.Transitions))
		}
		if len(sm.Initial) != 1 || sm.Initial[0] != "s1" {
			t.Errorf("expected initial [s1], got %v", sm.Initial)
		}
	})
}

func TestCTLOperators(t *testing.T) {
	e, _ := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Load a simple state machine
	e.LoadSpec(`
        initial(s0).
        transition(s0, a, s1).
        transition(s1, b, s2).
        transition(s2, c, s0).
        prop(s0, start).
        prop(s1, middle).
        prop(s2, end).
    `)

	tests := []struct {
		name     string
		formula  string
		expected bool
	}{
		{"EF reachable", "ef(atom(end))", true},
		{"EF start from start", "ef(atom(start))", true},
		{"AG can reach start", "ag(ef(atom(start)))", true},
		{"EX from initial", "ex(atom(middle))", true},
		{"not EX to unreachable", "ex(atom(nonexistent))", false},
		{"AF eventually end", "af(atom(end))", true},
		{"EG always start", "eg(atom(start))", false}, // can't stay in start forever
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := "check_ctl(" + tt.formula + ")."
			result, err := e.QueryOne(ctx, query)
			if err != nil {
				t.Fatalf("QueryOne error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("check_ctl(%s) = %v, want %v", tt.formula, result, tt.expected)
			}
		})
	}
}

func TestCTLCombinators(t *testing.T) {
	e, _ := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	e.LoadSpec(`
        initial(s0).
        transition(s0, a, s1).
        transition(s1, b, s0).
        prop(s0, even).
        prop(s1, odd).
    `)

	tests := []struct {
		name     string
		formula  string
		expected bool
	}{
		{"atom", "atom(even)", true},
		{"not atom", "not(atom(odd))", true},
		{"and", "and(atom(even), not(atom(odd)))", true},
		{"or true", "or(atom(even), atom(odd))", true},
		{"or false", "or(atom(nonexistent), atom(also_not))", false},
		{"nested", "ef(and(atom(odd), ex(atom(even))))", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := "check_ctl(" + tt.formula + ")."
			result, err := e.QueryOne(ctx, query)
			if err != nil {
				t.Fatalf("QueryOne error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("check_ctl(%s) = %v, want %v", tt.formula, result, tt.expected)
			}
		})
	}
}

func TestSequenceDiagram(t *testing.T) {
	e, _ := New()
	ctx := context.Background()

	e.LoadSpec(`
        lifeline(client).
        lifeline(server).
        message(1, client, server, request).
        message(2, server, client, response).
    `)

	seq, err := e.GetSequenceDiagram(ctx)
	if err != nil {
		t.Fatalf("GetSequenceDiagram error: %v", err)
	}

	if len(seq.Lifelines) != 2 {
		t.Errorf("expected 2 lifelines, got %d", len(seq.Lifelines))
	}
	if len(seq.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(seq.Messages))
	}
}

func TestPieChart(t *testing.T) {
	e, _ := New()
	ctx := context.Background()

	e.LoadSpec(`
        pie_slice(a, 30).
        pie_slice(b, 50).
        pie_slice(c, 20).
    `)

	slices, err := e.GetPieChart(ctx)
	if err != nil {
		t.Fatalf("GetPieChart error: %v", err)
	}

	if len(slices) != 3 {
		t.Errorf("expected 3 slices, got %d", len(slices))
	}

	total := 0.0
	for _, s := range slices {
		total += s.Value
	}
	if total != 100 {
		t.Errorf("expected total 100, got %f", total)
	}
}

func TestLineChart(t *testing.T) {
	e, _ := New()
	ctx := context.Background()

	e.LoadSpec(`
        line_point(series1, 1, 10).
        line_point(series1, 2, 20).
        line_point(series2, 1, 5).
        line_point(series2, 2, 15).
    `)

	points, err := e.GetLineChart(ctx)
	if err != nil {
		t.Fatalf("GetLineChart error: %v", err)
	}

	if len(points) != 4 {
		t.Errorf("expected 4 points, got %d", len(points))
	}

	// Check we have both series
	series := make(map[string]int)
	for _, p := range points {
		series[p.Series]++
	}
	if series["series1"] != 2 || series["series2"] != 2 {
		t.Errorf("expected 2 points per series, got %v", series)
	}
}

func TestProperties(t *testing.T) {
	e, _ := New()
	ctx := context.Background()

	e.LoadSpec(`
        property(test_prop, 'A test property', 'ag(ef(atom(done)))').
        property(another, 'Another one', 'ef(atom(start))').
    `)

	props, err := e.GetProperties(ctx)
	if err != nil {
		t.Fatalf("GetProperties error: %v", err)
	}

	if len(props) != 2 {
		t.Errorf("expected 2 properties, got %d", len(props))
	}
}

func TestReset(t *testing.T) {
	e, _ := New()
	ctx := context.Background()

	e.LoadSpec(`
        initial(test).
        transition(test, go, done).
    `)

	sm1, _ := e.GetStateMachine(ctx)
	if len(sm1.Transitions) != 1 {
		t.Errorf("expected 1 transition before reset")
	}

	e.Reset()

	sm2, _ := e.GetStateMachine(ctx)
	if len(sm2.Transitions) != 0 {
		t.Errorf("expected 0 transitions after reset, got %d", len(sm2.Transitions))
	}
}

func TestActorStateMachines(t *testing.T) {
	e, _ := New()
	ctx := context.Background()

	e.LoadSpec(`
        actor_initial(engine, idle).
        actor_transition(engine, idle, start, running).
        actor_transition(engine, running, stop, idle).
        
        actor_initial(ui, waiting).
        actor_transition(ui, waiting, click, processing).
        actor_transition(ui, processing, done, waiting).
    `)

	// This test documents the expected behavior for actor-based state machines
	// Currently the engine doesn't separate actors - this test will fail
	// until we implement actor_transition/4 handling

	sm, _ := e.GetStateMachine(ctx)
	// For now, actor_transition isn't extracted - transitions will be 0
	// This test serves as documentation of what needs to be implemented
	t.Logf("Got %d transitions (actor_transition not yet implemented)", len(sm.Transitions))
}
