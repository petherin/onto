// Package main is the Composition Root for the Onto CLI. It reads config,
// calls the bootstrap layer to assemble infrastructure and domain, then hands
// the fully-wired state to the CLI delivery mechanism.
package main

import (
	"log"

	"github.com/petherin/onto/internal/application/facade"
	"github.com/petherin/onto/internal/bootstrap"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/interface/cli"
)

func main() {
	state, err := bootstrap.Bootstrap(bootstrap.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	f, err := facade.New(
		state.Universe,
		state.Repo,
		state.StartID,
		navigation.NewBFSPathfinder(),
		universe.NewSequentialLocationGenerator(),
		gameOptions(state)...,
	)
	if err != nil {
		log.Fatal(err)
	}

	cli.New(f).Run()
}

// gameOptions configures the standard game: a starting budget and an objective
// derived from the start coordinate. If the start location is missing (it is
// validated again inside facade.New) only the budget is applied.
func gameOptions(state bootstrap.State) []facade.Option {
	opts := []facade.Option{facade.WithBudget(facade.DefaultBudget)}
	if loc, ok := state.Universe.GetLocation(state.StartID); ok {
		opts = append(opts, facade.WithTarget(facade.DefaultTarget(loc.Coordinate)))
	}
	return opts
}
