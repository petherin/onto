// Package main is the Composition Root for the Onto dashboard. It assembles
// infrastructure and domain via the bootstrap layer, then delegates to the TUI
// delivery mechanism.
package main

import (
	"log"

	"github.com/petherin/onto/internal/application/facade"
	"github.com/petherin/onto/internal/bootstrap"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/interface/tui"
)

func main() {
	state, err := bootstrap.Bootstrap(bootstrap.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	a, err := facade.New(
		state.Universe,
		state.Repo,
		state.StartID,
		navigation.NewBFSPathfinder(),
		universe.NewSequentialLocationGenerator(),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := tui.Run(a); err != nil {
		log.Fatal(err)
	}
}
