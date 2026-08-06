package cli

import (
	"strings"

	"github.com/chzyer/readline"
)

// ontoCompleter implements readline.AutoCompleter with context-aware
// completion for commands and destination IDs.
type ontoCompleter struct {
	app *App
}

// NewCompleter returns a readline.AutoCompleter backed by the given App.
func NewCompleter(app *App) readline.AutoCompleter {
	return &ontoCompleter{app: app}
}

// Do implements readline.AutoCompleter. It parses the current line prefix and
// returns candidates appropriate for the cursor position.
//
//   - First word: complete from command names.
//   - After "travel" or "route": complete from known location IDs.
//   - After "shift" or "jump": offer "back".
func (c *ontoCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	prefix := string(line[:pos])
	parts := strings.Fields(prefix)
	endsWithSpace := strings.HasSuffix(prefix, " ")

	switch {
	case len(parts) == 0:
		// Empty line — offer all commands.
		return completionsFor(allCommandNames(), "")

	case len(parts) == 1 && !endsWithSpace:
		// Partial command word.
		return completionsFor(allCommandNames(), parts[0])

	case len(parts) == 1 && endsWithSpace:
		// Full command, no argument typed yet.
		return c.completeArg(strings.ToLower(parts[0]), "")

	case len(parts) == 2 && !endsWithSpace:
		// Partial argument.
		return c.completeArg(strings.ToLower(parts[0]), parts[1])
	}

	return nil, 0
}

// completeArg returns completions for the argument of cmd given the typed prefix.
func (c *ontoCompleter) completeArg(cmd, argPrefix string) ([][]rune, int) {
	switch cmd {
	case cmdTravel, cmdRoute:
		return completionsFor(c.travelDestinationIDs(), argPrefix)
	case cmdShift, cmdJump:
		return completionsFor([]string{argBack}, argPrefix)
	}
	return nil, 0
}

// travelDestinationIDs returns physical destinations that are immediately
// reachable from the current location and therefore shown by the CLI.
func (c *ontoCompleter) travelDestinationIDs() []string {
	current := c.app.session.Coordinate()
	var ids []string
	for _, edge := range c.app.universe.EdgesFrom(c.app.session.Location()) {
		if !edge.Mode.IsPhysical() {
			continue
		}
		dest, ok := c.app.universe.GetLocation(edge.To)
		if ok && current.SamePhysicalReality(dest.Coordinate) {
			ids = append(ids, dest.ID)
		}
	}
	return ids
}

// allCommandNames returns every top-level command name.
func allCommandNames() []string {
	return []string{
		cmdHelp, cmdWhere, cmdLook, cmdList,
		cmdRoute, cmdTravel, cmdHome, cmdCost,
		cmdShift, cmdJump, cmdDrift, cmdAlign, cmdExit,
	}
}

// completionsFor returns readline-style completions: each entry is the
// untyped suffix to append, and length is the number of already typed runes.
func completionsFor(candidates []string, prefix string) ([][]rune, int) {
	lower := strings.ToLower(prefix)
	prefixLen := len([]rune(prefix))
	var matches [][]rune
	for _, candidate := range candidates {
		if strings.HasPrefix(strings.ToLower(candidate), lower) {
			matches = append(matches, []rune(candidate)[prefixLen:])
		}
	}
	return matches, prefixLen
}
