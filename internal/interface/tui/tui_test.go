package tui

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/require"

	"github.com/petherin/onto/internal/bootstrap"
	"github.com/petherin/onto/internal/application/facade"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
)

// newTestApp builds a *facade.App backed by an isolated, per-test data file so
// dashboard smoke tests never touch the repo's real data/locations.json or
// collide with other tests running in parallel.
func newTestApp(t *testing.T) *facade.App {
	t.Helper()
	dataPath := filepath.Join(t.TempDir(), "locations.json")
	t.Setenv("ONTO_DATA_FILE", dataPath)

	state, err := bootstrap.Bootstrap(bootstrap.DefaultConfig())
	require.NoError(t, err)

	a, err := facade.New(
		state.Universe,
		state.Repo,
		state.StartID,
		navigation.NewBFSPathfinder(),
		universe.NewSequentialLocationGenerator(),
	)
	require.NoError(t, err)
	return a
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

// TestDashboardOverflowIndicatorsVisible fills the log beyond the pane height
// and checks that the rendered view exposes an obvious "more below" cue so
// the user knows content is clipped and scrollable.
func TestDashboardOverflowIndicatorsVisible(t *testing.T) {
	app := newTestApp(t)
	m := New(app)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	for i := 0; i < 40; i++ {
		m.appendLog(fmt.Sprintf("overflow-line-%02d padding padding padding", i))
	}
	m.refreshLog()

	view := m.View()
	require.Contains(t, view, "more below", "expected a visible overflow footer when log content is clipped")
	require.Contains(t, view, "▼+", "expected a compact title marker for hidden lines below")
	require.Contains(t, view, "Tab", "expected scroll focus hint when content overflows")

	// A second render must not keep shrinking the log viewport / changing the
	// overflow math; the cue should remain stable.
	view2 := m.View()
	require.Contains(t, view2, "more below")
}

// TestDashboardLogOverflowAbsentWhenShort ensures a short log does not show
// overflow chrome on the Log pane. Navigation Options may still overflow when
// the default journey list exceeds the nav height cap — that is expected.
func TestDashboardLogOverflowAbsentWhenShort(t *testing.T) {
	app := newTestApp(t)
	m := New(app)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(Model)

	view := m.View()
	for _, line := range strings.Split(view, "\n") {
		if !strings.Contains(line, "Log") {
			continue
		}
		require.NotContains(t, line, "▼+", "short log must not show a below-overflow title marker")
		require.NotContains(t, line, "▲+", "short log must not show an above-overflow title marker")
		require.NotContains(t, line, "more below")
		require.NotContains(t, line, "more above")
	}
}

// TestDashboardNavOverflowIndicatorVisible checks that a long Navigation
// Options list surfaces the same clipped-content cue as the log pane when
// the terminal is short enough that the nav budget cannot fit every journey.
func TestDashboardNavOverflowIndicatorVisible(t *testing.T) {
	app := newTestApp(t)
	m := New(app)

	// Short terminal: top panes + input leave little flex, so the default
	// journey list exceeds the nav body budget and must show overflow cues.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 22})
	m = updated.(Model)

	view := m.View()
	require.Contains(t, view, "Navigation Options")
	require.Contains(t, view, "more below")
	require.Contains(t, view, "▼+")
}

// TestNavBodyBudgetGivesNavMoreThanLog locks the flex split: Navigation
// Options is the primary working surface and must receive a larger share of
// the flexible vertical region than the Log pane.
func TestNavBodyBudgetGivesNavMoreThanLog(t *testing.T) {
	for _, flex := range []int{10, 15, 24, 30, 60} {
		nav := navBodyBudget(flex)
		logShare := flex - nav
		if logShare < minLogBodyHeight {
			logShare = minLogBodyHeight
		}
		require.Greater(t, nav, logShare, "flex=%d nav=%d log=%d", flex, nav, logShare)
	}
}

// TestDashboardNavGetsMoreRowsThanLog renders a tall terminal with enough
// nav content to fill its budget and checks nav is allocated more rows than
// log. Heights are read from layoutBodyHeights because View is a value
// receiver and does not persist viewport sizes onto the caller's Model.
func TestDashboardNavGetsMoreRowsThanLog(t *testing.T) {
	app := newTestApp(t)
	m := New(app)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = updated.(Model)

	// Ensure nav content is long enough to claim its full budget.
	longList := strings.TrimRight(strings.Repeat("journey option line\n", 40), "\n")
	m.navContent = longList
	m.nav.SetContent(longList)

	// Approximate chrome the same way View measures it: top panes + input.
	topHeight := 6
	inputHeight := 1
	navBody, logBody := m.layoutBodyHeights(topHeight, inputHeight)
	require.Greater(t, navBody, logBody,
		"nav body budget %d should exceed log body budget %d", navBody, logBody)

	// Smoke-render so layout panics cannot hide behind the pure height math.
	require.NotContains(t, m.View(), "Loading...")
}
