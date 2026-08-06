// Package main is the entry point for the Onto CLI. It wires together the
// infrastructure, application, and interface layers and hands control to the
// run loop in the cli package.
package main

import (
	"log"

	"github.com/petherin/onto/internal/interface/cli"
)

func main() {
	app, err := cli.NewAppWithError()
	if err != nil {
		log.Fatal(err)
	}
	app.Run()
}
