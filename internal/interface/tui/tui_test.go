package tui

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/require"

	"github.com/petherin/onto/internal/interface/cli"
)

// newTestApp builds a *cli.App backed by an isolated, per-test data file so
// dashboard smoke tests never touch the repo's real data/locations.json or
// collide with other tests running in parallel.
func newTestApp(t *testing.T) *cli.App {
	t.Helper()
	dataPath := filepath.Join(t.TempDir(), "locations.json")
	t.Setenv("ONTO_DATA_FILE", dataPath)

	app, err := cli.NewAppWithError()
	require.NoError(t, err)
	return app
}

// TestDashboardStartsAndRendersPanes verifies the Bubble Tea program boots,
// lays out its panes (Location, Cost, Navigation Options, Log) inside a
// terminal-sized viewport, and can be torn down cleanly.
func TestDashboardStartsAndRendersPanes(t *testing.T) {
	app := newTestApp(t)
	m := New(app)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))

	teatest.WaitFor(
		t,
		tm.Output(),
		func(b []byte) bool {
			return bytes.Contains(b, []byte("Location")) &&
				bytes.Contains(b, []byte("Cost")) &&
				bytes.Contains(b, []byte("Navigation Options")) &&
				bytes.Contains(b, []byte("Log"))
		},
		teatest.WithDuration(5*time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// TestDashboardExecutesCommand types a command into the input line, submits
// it, and checks the response is recorded in the model's log history —
// verifying the dashboard actually delegates to the shared *cli.App rather
// than just rendering static panes. The log's on-screen viewport can scroll
// past older entries, so history is asserted directly on the final model
// rather than by scanning rendered screen output.
func TestDashboardExecutesCommand(t *testing.T) {
	app := newTestApp(t)
	m := New(app)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))

	teatest.WaitFor(
		t,
		tm.Output(),
		func(b []byte) bool { return bytes.Contains(b, []byte("Log")) },
		teatest.WithDuration(5*time.Second),
	)

	tm.Type("where")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final := tm.FinalModel(t).(Model)
	history := final.History()
	require.NotEmpty(t, history)
	require.Contains(t, history[len(history)-1], "Reality Coordinate")
}

// TestDashboardExitCommand verifies typing 'exit' quits the program on its
// own, without requiring Ctrl+C.
func TestDashboardExitCommand(t *testing.T) {
	app := newTestApp(t)
	m := New(app)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))

	teatest.WaitFor(
		t,
		tm.Output(),
		func(b []byte) bool { return bytes.Contains(b, []byte("Log")) },
		teatest.WithDuration(5*time.Second),
	)

	tm.Type("exit")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	out, err := io.ReadAll(tm.FinalOutput(t))
	require.NoError(t, err)
	require.Contains(t, string(out), "Goodbye.")
}

// TestDashboardHomePlanIsVisibleAfterScroll reproduces a regression where the
// log pane always auto-scrolled to the absolute bottom on every new entry.
// A long 'home' route plan, immediately followed by the "Proceed? [y/N]:"
// confirmation, would get scrolled out of view entirely — only the short
// confirmation line remained visible. The log must instead scroll to the
// top of each newly appended entry so the plan itself stays on screen.
func TestDashboardHomePlanIsVisibleAfterScroll(t *testing.T) {
	app := newTestApp(t)
	m := New(app)

	// Use a small terminal so the log pane is short enough that the bug
	// (scrolling straight to the bottom) would actually hide the plan.
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 30))

	teatest.WaitFor(
		t,
		tm.Output(),
		func(b []byte) bool { return bytes.Contains(b, []byte("Log")) },
		teatest.WithDuration(5*time.Second),
	)

	// Move away from home so the subsequent 'home' command has a non-trivial
	// route plan to display.
	tm.Type("travel station")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	tm.Type("home")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(
		t,
		tm.Output(),
		func(b []byte) bool { return bytes.Contains(b, []byte("Route home")) },
		teatest.WithDuration(5*time.Second),
	)

	tm.Type("y")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final := tm.FinalModel(t).(Model)
	history := final.History()
	require.NotEmpty(t, history)

	// The plan must appear exactly once; the confirmation prompt is part of
	// GoHome's own returned string and must not be duplicated as a separate
	// log entry.
	var planCount, proceedCount int
	for _, entry := range history {
		if strings.Contains(entry, "Route home") {
			planCount++
		}
		proceedCount += strings.Count(entry, "Proceed? [y/N]:")
	}
	require.Equal(t, 1, planCount, "expected exactly one 'home' plan entry")
	require.Equal(t, 1, proceedCount, "expected the confirmation prompt to appear exactly once")
	require.Contains(t, history[len(history)-1], "You are home.")
}
