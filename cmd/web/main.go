// Package main is the Composition Root for the Onto web server. It assembles
// infrastructure and domain via the bootstrap layer, then delegates to the web
// delivery mechanism. The listen address can be overridden with ONTO_WEB_ADDR
// (default ":8090").
package main

import (
	"log"
	"os"

	"github.com/petherin/onto/internal/application/facade"
	"github.com/petherin/onto/internal/bootstrap"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/interface/web"
)

const defaultAddr = ":8090"

func main() {
	cfg := bootstrap.DefaultConfig()
	state, err := bootstrap.Bootstrap(cfg)
	if err != nil {
		log.Fatal(err)
	}

	a, err := facade.New(
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

	addr := os.Getenv("ONTO_WEB_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	if err := web.Run(a, addr); err != nil {
		log.Fatal(err)
	}
}
