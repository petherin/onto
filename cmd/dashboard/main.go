// Package main is the entry point for the Onto dashboard: a multi-pane
// terminal UI (built with Bubble Tea) showing location, navigation options,
// and cost alongside a scrollable log and command input, backed by the same
// application wiring as the plain cmd/cli REPL.
package main

import (
	"log"

	"github.com/petherin/onto/internal/interface/cli"
	"github.com/petherin/onto/internal/interface/tui"
)

func main() {
	app, err := cli.NewAppWithError()
	if err != nil {
		log.Fatal(err)
	}
	if err := tui.Run(app); err != nil {
		log.Fatal(err)
	}
}
