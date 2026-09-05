package cli

import "fmt"

// Prompt returns the context-aware CLI prompt string, wrapping the facade's
// location label in the REPL's "[…] > " chrome.
func (a *App) Prompt() string { return fmt.Sprintf("[%s] > ", a.app.LocationLabel()) }
