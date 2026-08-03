package cli

import (
	"fmt"
	"strings"
)

// Prompt returns the context-aware CLI prompt string, showing the current
// physical location and appending non-default Quantum and Timeline levels.
func (a *App) Prompt() string {
	c := a.session.CurrentCoordinate
	parts := []string{}
	for _, part := range []string{c.Planet, c.Country, c.City, c.Location} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	if c.Quantum != "" && c.Quantum != "Q0" {
		parts = append(parts, c.Quantum)
	}
	if c.Timeline != "" && c.Timeline != "Prime" {
		parts = append(parts, c.Timeline)
	}
	return fmt.Sprintf("[%s] > ", strings.Join(parts, "/"))
}
