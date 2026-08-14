// Package tui implements an optional multi-pane terminal dashboard for the
// Onto CLI, built with Bubble Tea. It presents location, navigation options,
// and cost information in separate panes alongside a scrollable log and a
// command input line, while delegating all command execution to the same
// *cli.App used by the plain REPL.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/petherin/onto/internal/interface/cli"
)

const (
	msgGoodbye     = "Goodbye."
	msgAlreadyHome = "You are already home."
	cmdHome        = "home"
	cmdList        = "ls"

	// Vertical split between Navigation Options and Log over the flexible
	// region under the top panes. Nav is the primary working surface, so it
	// receives the larger share; Log keeps a usable minority.
	navFlexNumerator   = 2
	navFlexDenominator = 3

	minNavBodyHeight = 4
	minLogBodyHeight = 3
	// scrollPaneChrome is title row + top/bottom border for a rounded pane.
	scrollPaneChrome = 3
)

// focusTarget identifies which scrollable pane (Log or Navigation Options)
// currently receives arrow/page-up/page-down key presses. Tab cycles focus
// between them; the command input always receives text regardless of focus.
type focusTarget int

const (
	focusLog focusTarget = iota
	focusNav
)

var (
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	focusedPaneStyle = paneStyle.
				BorderForeground(lipgloss.Color("205"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("36")).
			Bold(true)

	// overflowStyle highlights clipped-content cues so hidden lines are
	// obvious without relying on the user noticing a quiet title hint.
	overflowStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))
)

// Model is the Bubble Tea model driving the dashboard. It wraps a *cli.App
// and re-renders the location, navigation, and cost panes after every
// executed command.
type Model struct {
	app   *cli.App
	input textinput.Model
	log   viewport.Model
	nav   viewport.Model

	width  int
	height int
	ready  bool

	history []string

	// navContent is the last wrapped Navigation Options body rendered into
	// the nav viewport; refreshNav compares against it so a re-render with
	// unchanged content doesn't reset the user's scroll position.
	navContent string

	// navViewportHeight is the full body budget for the Navigation Options
	// viewport (before any overflow-footer reservation). View resets
	// nav.Height from this each frame so a footer row cannot shrink the
	// pane across successive renders.
	navViewportHeight int

	// focus selects which scrollable pane (Log or Navigation Options)
	// receives arrow/page-up/page-down keys; Tab toggles it.
	focus focusTarget

	// awaitingHomeConfirm is set while the user is being asked to confirm the
	// 'home' journey plan produced by App.GoHome.
	awaitingHomeConfirm bool
}

// New builds a Model ready to be run with tea.NewProgram.
func New(app *cli.App) Model {
	ti := textinput.New()
	ti.Placeholder = "type a command, e.g. help, look, travel <destination>"
	ti.Focus()
	ti.CharLimit = 256

	m := Model{
		app:   app,
		input: ti,
	}
	m.appendLog(fmt.Sprintf("%s\n\nType 'help' to see the available commands.", cliAppVersion()))
	return m
}

func cliAppVersion() string {
	return cli.AppVersion
}

// Run starts the dashboard program on the given app and blocks until the
// user exits.
func Run(app *cli.App) error {
	p := tea.NewProgram(New(app), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.log = viewport.New(m.logContentWidth(), 3)
			m.log.SetContent(m.wrappedHistory())
			m.nav = viewport.New(m.logContentWidth(), 1)
			m.ready = true
		} else {
			m.log.Width = m.logContentWidth()
			m.log.SetContent(m.wrappedHistory())
			m.nav.Width = m.logContentWidth()
		}
		m.refreshNav()
		m.input.Width = m.width - 4
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m.quit()
		case tea.KeyTab:
			if m.focus == focusLog {
				m.focus = focusNav
			} else {
				m.focus = focusLog
			}
			return m, nil
		case tea.KeyEnter:
			line := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			return m.handleLine(line)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	var cmd2 tea.Cmd
	if m.focus == focusNav {
		m.nav, cmd2 = m.nav.Update(msg)
	} else {
		m.log, cmd2 = m.log.Update(msg)
	}
	return m, tea.Batch(cmd, cmd2)
}

// handleLine processes one submitted line of input, including the two-step
// 'home' confirm/execute flow that App.GoHome/App.GoHomeConfirm implement.
func (m Model) handleLine(line string) (tea.Model, tea.Cmd) {
	if line == "" {
		return m, nil
	}

	if m.awaitingHomeConfirm {
		m.awaitingHomeConfirm = false
		if strings.ToLower(line) == "y" {
			m.appendLog(m.app.GoHomeConfirm())
		} else {
			m.appendLog("Cancelled.")
		}
		m.refreshLog()
		m.refreshNav()
		return m, nil
	}

	fields := strings.Fields(line)
	if len(fields) > 0 && fields[0] == cmdHome {
		plan := m.app.GoHome()
		// GoHome's returned plan string already ends with "Proceed? [y/N]:",
		// so it must not be appended again here.
		m.appendLog(plan)
		if plan != msgAlreadyHome {
			m.awaitingHomeConfirm = true
		}
		m.refreshLog()
		m.refreshNav()
		return m, nil
	}

	response := m.app.Execute(line)
	if response == msgGoodbye {
		m.appendLog(response)
		m.refreshLog()
		return m.quit()
	}
	// The 'ls' command's response, and the "Possible journeys" section that
	// several other commands (travel, shift, jump, drift, observe, time)
	// append to their own output, just repeat the Navigation Options pane,
	// which is always visible, so don't also clutter the Log with them.
	if response != "" && fields[0] != cmdList {
		m.appendLog(stripPossibleJourneys(response))
	}
	m.refreshLog()
	m.refreshNav()
	return m, nil
}

// stripPossibleJourneys removes a trailing "Possible journeys" section (and
// its blank-line separator) from a command's response text, if present.
func stripPossibleJourneys(s string) string {
	if idx := strings.Index(s, "\n\nPossible journeys"); idx != -1 {
		return strings.TrimRight(s[:idx], "\n")
	}
	return s
}

func (m Model) quit() (tea.Model, tea.Cmd) {
	if err := m.app.SaveIfDirty(); err != nil {
		m.appendLog(fmt.Sprintf("Warning: failed to save before exit: %v", err))
		m.refreshLog()
	}
	return m, tea.Quit
}

func (m *Model) appendLog(s string) {
	m.history = append(m.history, s)
}

// History returns the sequence of rendered log entries. It exists primarily
// so tests can assert on command output deterministically, independent of
// the scrollable viewport's current visible window.
func (m Model) History() []string {
	return m.history
}

func (m *Model) refreshLog() {
	if !m.ready {
		return
	}
	wrapped := m.wrappedHistoryEntries()
	// Scroll to the top of the most recently appended entry rather than the
	// absolute bottom of the log. Jumping straight to the bottom can hide the
	// start of long responses (e.g. a multi-step 'home' route plan) behind
	// whatever short line was appended after it (like a confirmation prompt).
	newestStart := 0
	if len(wrapped) > 1 {
		prior := strings.Join(wrapped[:len(wrapped)-1], "\n\n")
		newestStart = strings.Count(prior, "\n") + 2 // prior's lines, plus the "\n\n" separator
	}
	m.log.SetContent(strings.Join(wrapped, "\n\n"))
	m.log.SetYOffset(newestStart)
}

// refreshNav re-renders the Navigation Options pane's content into its
// viewport. Vertical sizing is applied in View from the current terminal
// budget (nav gets a larger share than log). Scroll position is only reset
// when the content actually changed, so refreshing after an unrelated
// command doesn't jump the user back to the top if they had scrolled.
func (m *Model) refreshNav() {
	if !m.ready {
		return
	}
	content := lipgloss.NewStyle().Width(m.logContentWidth()).Render(m.app.List())
	if content == m.navContent {
		return
	}
	m.navContent = content
	m.nav.SetContent(content)
	m.nav.GotoTop()
}

func (m Model) navContentLines() int {
	if m.navContent == "" {
		return 1
	}
	return strings.Count(m.navContent, "\n") + 1
}

// navBodyBudget returns how many content rows Navigation Options may use,
// given the flexible vertical space shared with the Log pane. Nav receives
// the larger fraction; short option lists still size to their content.
func navBodyBudget(flex int) int {
	if flex < minNavBodyHeight+minLogBodyHeight {
		flex = minNavBodyHeight + minLogBodyHeight
	}
	navShare := (flex * navFlexNumerator) / navFlexDenominator
	if navShare < minNavBodyHeight {
		navShare = minNavBodyHeight
	}
	if flex-navShare < minLogBodyHeight {
		navShare = flex - minLogBodyHeight
	}
	if navShare < 1 {
		navShare = 1
	}
	return navShare
}

// wrappedHistoryEntries returns each log entry word-wrapped to the log
// pane's current content width, so long lines fold instead of being
// truncated by the viewport.
func (m Model) wrappedHistoryEntries() []string {
	w := m.logContentWidth()
	wrapped := make([]string, len(m.history))
	for i, s := range m.history {
		wrapped[i] = lipgloss.NewStyle().Width(w).Render(s)
	}
	return wrapped
}

// wrappedHistory returns the full log history as a single word-wrapped
// string, joining entries with a blank line between them.
func (m Model) wrappedHistory() string {
	return strings.Join(m.wrappedHistoryEntries(), "\n\n")
}

// layoutBodyHeights returns the Navigation Options and Log content-row
// budgets for the current terminal size and nav content. Nav receives the
// larger flex share; short journey lists still collapse to their content so
// unused nav space falls through to the log. Safe to call from tests —
// View is a value receiver and does not persist these heights on Model.
func (m Model) layoutBodyHeights(topHeight, inputHeight int) (navBody, logBody int) {
	flex := m.height - topHeight - inputHeight - 2*scrollPaneChrome
	navCap := navBodyBudget(flex)
	navBody = m.navContentLines()
	if navBody > navCap {
		navBody = navCap
	}
	if navBody < 1 {
		navBody = 1
	}

	// Estimate log rows from the flex remainder after nav claims its body +
	// chrome. Matches the post-measure fallback mins used in View.
	logBody = flex - navBody
	if logBody < minLogBodyHeight {
		logBody = minLogBodyHeight
	}
	return navBody, logBody
}

// View implements tea.Model, laying out the location, navigation, and cost
// panes above a scrollable log and the command input line. The log pane's
// height is recomputed on every render from the actual measured height of
// the other panes, so the total layout always fits within the terminal
// height regardless of how long the location/cost/navigation text is.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	topWidth := m.width - 2
	if topWidth < 20 {
		topWidth = 20
	}
	leftWidth := topWidth / 2
	rightWidth := topWidth - leftWidth

	locationPane := m.pane("Location", m.app.Look(), leftWidth, false)
	costPane := m.pane("Cost", m.app.Cost(), rightWidth, false)
	top := lipgloss.JoinHorizontal(lipgloss.Top, locationPane, costPane)
	inputLine := promptStyle.Render(m.app.Prompt()) + m.input.View()

	topHeight := lipgloss.Height(top)
	inputHeight := lipgloss.Height(inputLine)
	navBodyHeight, _ := m.layoutBodyHeights(topHeight, inputHeight)
	m.navViewportHeight = navBodyHeight
	m.nav.Width = m.logContentWidth()
	m.nav.Height = navBodyHeight
	// If the nav viewport is clipped, reserve one row inside the pane for a
	// sticky overflow footer so the cue stays visible while scrolling.
	navOverflow := viewportOverflows(m.nav)
	if navOverflow && m.nav.Height > 1 {
		m.nav.Height = navBodyHeight - 1
	}
	navPane := m.scrollablePane(
		"Navigation Options",
		m.nav,
		topWidth,
		m.focus == focusNav,
		navOverflow,
	)

	// Log fills whatever rows remain after the (now larger-capable) nav pane.
	reserved := topHeight + lipgloss.Height(navPane) + inputHeight
	logHeight := m.height - reserved - scrollPaneChrome
	if logHeight < minLogBodyHeight {
		logHeight = minLogBodyHeight
	}
	m.log.Width = m.logContentWidth()
	m.log.Height = logHeight
	// Re-check after assigning the real log height; reserve a footer row when
	// content still overflows so the indicator never covers the last line.
	logOverflow := viewportOverflows(m.log)
	if logOverflow && m.log.Height > 1 {
		m.log.Height--
	}
	logPane := m.scrollablePane(
		"Log",
		m.log,
		topWidth,
		m.focus == focusLog,
		logOverflow,
	)

	return lipgloss.JoinVertical(lipgloss.Left, top, navPane, logPane, inputLine)
}

// scrollablePane renders a titled viewport pane, and when content is clipped
// appends a high-visibility footer describing how much is hidden and how to
// scroll (Tab to focus when unfocused, ↑/↓ when focused).
func (m Model) scrollablePane(title string, vp viewport.Model, width int, focused, showOverflow bool) string {
	style := paneStyle
	if focused {
		style = focusedPaneStyle
		title += " (↑/↓ scroll, Tab to switch)"
	} else if showOverflow {
		title += " (Tab · ↑/↓ to scroll)"
	}

	body := vp.View()
	if showOverflow {
		if status := scrollStatus(vp); status != "" {
			title += " " + overflowStyle.Render(status)
			body += "\n" + overflowStyle.Render(scrollFooter(vp, focused))
		}
	}

	return style.Width(width).Render(titleStyle.Render(title) + "\n" + body)
}

// viewportOverflows reports whether vp has more lines than its visible height.
func viewportOverflows(vp viewport.Model) bool {
	return vp.TotalLineCount() > vp.Height && vp.Height > 0
}

// scrollStatus is a compact title suffix like "▼+5" / "▲+2 ▼+3".
func scrollStatus(vp viewport.Model) string {
	above, below := hiddenLines(vp)
	if above == 0 && below == 0 {
		return ""
	}
	var parts []string
	if above > 0 {
		parts = append(parts, fmt.Sprintf("▲+%d", above))
	}
	if below > 0 {
		parts = append(parts, fmt.Sprintf("▼+%d", below))
	}
	return strings.Join(parts, " ")
}

// scrollFooter is a full-width sticky cue under clipped viewport content.
func scrollFooter(vp viewport.Model, focused bool) string {
	above, below := hiddenLines(vp)
	var parts []string
	if above > 0 {
		parts = append(parts, fmt.Sprintf("▲ %d more above", above))
	}
	if below > 0 {
		parts = append(parts, fmt.Sprintf("▼ %d more below", below))
	}
	hint := "Tab then ↑/↓ to scroll"
	if focused {
		hint = "↑/↓ to scroll"
	}
	if len(parts) == 0 {
		return hint
	}
	return strings.Join(parts, " · ") + " · " + hint
}

func hiddenLines(vp viewport.Model) (above, below int) {
	total := vp.TotalLineCount()
	if total <= vp.Height {
		return 0, 0
	}
	above = vp.YOffset
	if above < 0 {
		above = 0
	}
	below = total - vp.Height - vp.YOffset
	if below < 0 {
		below = 0
	}
	return above, below
}

func (m Model) pane(title, body string, width int, focused bool) string {
	if width < 10 {
		width = 10
	}
	style := paneStyle
	if focused {
		style = focusedPaneStyle
	}
	return style.Width(width - 2).Render(titleStyle.Render(title) + "\n" + body)
}

func (m Model) logContentWidth() int {
	w := m.width - 4
	if w < 10 {
		w = 10
	}
	return w
}
