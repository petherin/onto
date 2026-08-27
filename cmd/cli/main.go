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
	cfg := bootstrap.DefaultConfig()
	state, err := bootstrap.Bootstrap(cfg)
	if err != nil {
		log.Fatal(err)
	}

	f, err := facade.New(
		state.Universe,
		state.Repo,
		state.StartID,
		navigation.NewBFSPathfinder(),
		universe.NewSequentialLocationGenerator(),
		bootstrap.GameOptions(cfg, state)...,
	)
	if err != nil {
		log.Fatal(err)
	}

	cli.New(f).Run()
}
