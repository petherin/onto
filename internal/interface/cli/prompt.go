package cli

import "fmt"

// Prompt returns the context-aware CLI prompt string derived from the
// short Onto Address of the current coordinate.
func (a *App) Prompt() string {
	addr := a.session.Coordinate().ShortOntoAddress()
	// Strip the onto:// scheme for compactness in the prompt.
	addr = addr[len("onto://"):]
	return fmt.Sprintf("[%s] > ", addr)
}
