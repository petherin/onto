package facade

import (
	"math"
	"strings"
)

func (a *App) suggestDestination(target string) string {
	if target == "" {
		return ""
	}
	best := ""
	bestDistance := math.MaxInt
	lowerTarget := strings.ToLower(target)
	compactTarget := strings.ReplaceAll(lowerTarget, " ", "")

	for _, loc := range a.univ.AllLocations() {
		id := loc.ID
		if d := levenshteinDistance(lowerTarget, strings.ToLower(id)); d < bestDistance {
			bestDistance = d
			best = id
		}
		compactID := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(id), "-", ""), "_", "")
		if d := levenshteinDistance(compactTarget, compactID); d < bestDistance {
			bestDistance = d
			best = id
		}
		if d := levenshteinDistance(lowerTarget, strings.ToLower(loc.Name)); d < bestDistance {
			bestDistance = d
			best = id
		}
	}

	if bestDistance <= 2 {
		return best
	}
	if bestDistance <= 3 && len(target) > 6 {
		return best
	}
	return ""
}

var allCommands = []string{
	"help", "where", "look", "ls", "route", "travel", "home", "cost",
	"shift", "jump", "universe", "structure", "simulate", "drift", "align",
	"observe", "time", "save", "exit",
}

func (a *App) suggestCommand(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return ""
	}
	best := ""
	bestDistance := math.MaxInt
	for _, candidate := range allCommands {
		if d := levenshteinDistance(input, candidate); d < bestDistance {
			bestDistance = d
			best = candidate
		}
	}
	if bestDistance <= 2 {
		return best
	}
	return ""
}

// AllCommandNames returns every top-level command name. Exported for the CLI
// tab-completer.
func AllCommandNames() []string {
	return allCommands
}

func levenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = minInt(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func minInt(values ...int) int {
	best := values[0]
	for _, v := range values[1:] {
		if v < best {
			best = v
		}
	}
	return best
}
