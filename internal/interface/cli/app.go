// Package cli is the readline delivery mechanism for the Onto application. It
// owns the interactive terminal run loop and tab-completion. All application
// logic lives in internal/application/facade; this package only handles I/O.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/petherin/onto/internal/application/facade"
)

// App is the CLI delivery wrapper. It holds a facade.App and owns the
// readline/bufio run loop; it has no application logic of its own.
type App struct {
	app               *facade.App
	interactiveReader *bufio.Reader
}

// New wraps a fully-wired facade.App in the CLI delivery mechanism.
func New(app *facade.App) *App {
	return &App{app: app}
}

// Execute delegates to the facade.
func (a *App) Execute(input string) string { return a.app.Execute(input) }

// GoHome delegates to the facade.
func (a *App) GoHome() string { return a.app.GoHome() }

// GoHomeConfirm delegates to the facade.
func (a *App) GoHomeConfirm() string { return a.app.GoHomeConfirm() }

// SaveIfDirty delegates to the facade.
func (a *App) SaveIfDirty() error { return a.app.SaveIfDirty() }

func (a *App) warnIfSaveBeforeExitFails() {
	if err := a.app.SaveIfDirty(); err != nil {
		fmt.Printf(fmtExitSaveWarning+"\n", err)
	}
}

// handleHome runs the two-step 'home' confirmation flow shared by both REPLs.
// readConfirm reads the user's y/n answer; the two loops differ only in how they
// read it (readline vs bufio), so that is passed in as a closure.
func (a *App) handleHome(readConfirm func() string) {
	plan := a.app.GoHome()
	fmt.Println(plan)
	if !facade.NeedsHomeConfirm(plan) {
		return
	}
	if strings.ToLower(strings.TrimSpace(readConfirm())) == "y" {
		fmt.Println(a.app.GoHomeConfirm())
	} else {
		fmt.Println("Cancelled.")
	}
}

// runCommand executes one command line and prints its response. It returns true
// when the REPL should exit (the user asked to quit).
func (a *App) runCommand(trimmed string) bool {
	response := a.app.Execute(trimmed)
	if response == msgGoodbye {
		a.warnIfSaveBeforeExitFails()
		fmt.Println(response)
		return true
	}
	if response != "" {
		fmt.Println(response)
	}
	return false
}

// Run starts the interactive readline REPL. Falls back to plain bufio when
// stdout is not a TTY (e.g. in tests or pipes).
func (a *App) Run() {
	a.printWelcome()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          a.Prompt(),
		AutoComplete:    NewCompleter(a),
		HistoryFile:     os.TempDir() + "/onto_history",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		a.runPlain()
		return
	}
	defer func() { _ = rl.Close() }()

	a.interactiveReader = bufio.NewReader(os.Stdin)

	for {
		rl.SetPrompt(a.Prompt())
		line, err := rl.Readline()
		if err != nil {
			a.warnIfSaveBeforeExitFails()
			break
		}
		trimmed := strings.TrimSpace(line)

		if fields := strings.Fields(trimmed); len(fields) > 0 && fields[0] == "home" {
			a.handleHome(func() string {
				rl.SetPrompt("")
				confirm, _ := rl.Readline()
				return confirm
			})
			continue
		}

		if a.runCommand(trimmed) {
			break
		}
	}
}

// runPlain is the non-readline fallback REPL used when stdin is not a TTY.
func (a *App) runPlain() {
	reader := bufio.NewReader(os.Stdin)
	a.interactiveReader = reader
	for {
		fmt.Print(a.Prompt())
		input, err := reader.ReadString('\n')
		if err != nil {
			a.warnIfSaveBeforeExitFails()
			break
		}
		trimmed := strings.TrimSpace(input)

		if fields := strings.Fields(trimmed); len(fields) > 0 && fields[0] == "home" {
			a.handleHome(func() string {
				confirm, _ := reader.ReadString('\n')
				return confirm
			})
			continue
		}

		if a.runCommand(trimmed) {
			break
		}
	}
}

func (a *App) printWelcome() {
	fmt.Println(AppVersion)
	fmt.Println()
	fmt.Println("Type 'help' to see the available commands. Press Tab to complete commands and destinations.")
	fmt.Println()
	fmt.Println("Current Position")
	fmt.Println(a.app.Where())
	fmt.Println()
}
