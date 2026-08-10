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
)

var (
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("36")).
			Bold(true)
)

// Model is the Bubble Tea model driving the dashboard. It wraps a *cli.App
// and re-renders the location, navigation, and cost panes after every
// executed command.
type Model struct {
	app   *cli.App
	input textinput.Model
	log   viewport.Model

	width  int
	height int
	ready  bool

	history []string

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
			m.log.SetContent(strings.Join(m.history, "\n\n"))
			m.ready = true
		} else {
			m.log.Width = m.logContentWidth()
		}
		m.input.Width = m.width - 4
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m.quit()
		case tea.KeyEnter:
			line := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			return m.handleLine(line)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	var cmd2 tea.Cmd
	m.log, cmd2 = m.log.Update(msg)
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
		return m, nil
	}

	response := m.app.Execute(line)
	if response == msgGoodbye {
		m.appendLog(response)
		m.refreshLog()
		return m.quit()
	}
	if response != "" {
		m.appendLog(response)
	}
	m.refreshLog()
	return m, nil
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
	// Scroll to the top of the most recently appended entry rather than the
	// absolute bottom of the log. Jumping straight to the bottom can hide the
	// start of long responses (e.g. a multi-step 'home' route plan) behind
	// whatever short line was appended after it (like a confirmation prompt).
	newestStart := 0
	if len(m.history) > 1 {
		prior := strings.Join(m.history[:len(m.history)-1], "\n\n")
		newestStart = strings.Count(prior, "\n") + 2 // prior's lines, plus the "\n\n" separator
	}
	m.log.SetContent(strings.Join(m.history, "\n\n"))
	m.log.SetYOffset(newestStart)
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

	locationPane := m.pane("Location", m.app.Look(), leftWidth)
	costPane := m.pane("Cost", m.app.Cost(), rightWidth)
	navPane := m.pane("Navigation Options", m.app.List(), topWidth)

	top := lipgloss.JoinHorizontal(lipgloss.Top, locationPane, costPane)

	inputLine := promptStyle.Render(m.app.Prompt()) + m.input.View()

	// Reserve space for everything except the log pane, then give the log
	// pane whatever rows remain (minimum 3 for its border + one content line).
	reserved := lipgloss.Height(top) + lipgloss.Height(navPane) + lipgloss.Height(inputLine)
	logHeight := m.height - reserved - 2 // -2 for the log pane's own border
	if logHeight < 1 {
		logHeight = 1
	}
	m.log.Width = m.logContentWidth()
	m.log.Height = logHeight

	logPane := paneStyle.Width(topWidth).Render(
		titleStyle.Render("Log") + "\n" + m.log.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left, top, navPane, logPane, inputLine)
}

func (m Model) pane(title, body string, width int) string {
	if width < 10 {
		width = 10
	}
	return paneStyle.Width(width - 2).Render(titleStyle.Render(title) + "\n" + body)
}

func (m Model) logContentWidth() int {
	w := m.width - 4
	if w < 10 {
		w = 10
	}
	return w
}
