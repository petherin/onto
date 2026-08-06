package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// runeStr converts candidates back to strings for readable assertions.
func runeStr(candidates [][]rune) []string {
	out := make([]string, len(candidates))
	for i, r := range candidates {
		out[i] = string(r)
	}
	return out
}

func TestCompleter_EmptyLine_OffersAllCommands(t *testing.T) {
	c := NewCompleter(NewApp())
	candidates, length := c.Do([]rune{}, 0)

	assert.Equal(t, 0, length)
	names := runeStr(candidates)
	for _, cmd := range []string{cmdWhere, cmdTravel, cmdShift, cmdJump, cmdExit} {
		assert.Contains(t, names, cmd)
	}
}

func TestCompleter_PartialCommand_FiltersToMatchingCommands(t *testing.T) {
	c := NewCompleter(NewApp())
	// "tr" should match "travel" only among standard commands.
	candidates, length := c.Do([]rune("tr"), 2)

	assert.Equal(t, 2, length)
	names := runeStr(candidates)
	assert.Contains(t, names, "avel")
	assert.NotContains(t, names, cmdWhere)
}

func TestCompleter_FullCommandWithSpace_AfterTravel_OffersLocations(t *testing.T) {
	c := NewCompleter(NewApp())
	candidates, length := c.Do([]rune("travel "), 7)

	assert.Equal(t, 0, length, "no prefix typed yet — length should be 0")
	names := runeStr(candidates)
	assert.Contains(t, names, "station")
	assert.NotContains(t, names, "home")
}

func TestCompleter_OnlyOffersCurrentPhysicalJourneys(t *testing.T) {
	app := NewApp()
	app.Execute("travel station")
	c := NewCompleter(app)

	candidates, _ := c.Do([]rune("travel "), 7)
	names := runeStr(candidates)

	assert.Contains(t, names, "city-centre")
	assert.NotContains(t, names, "home")
	assert.NotContains(t, names, "park")
}

func TestCompleter_PartialArg_AfterTravel_FiltersLocations(t *testing.T) {
	c := NewCompleter(NewApp())
	candidates, length := c.Do([]rune("travel sta"), 10)

	assert.Equal(t, 3, length, "length should be len('sta')")
	names := runeStr(candidates)
	assert.Contains(t, names, "tion")
	assert.NotContains(t, names, "home")
}

func TestCompleter_PartialTravelArgumentReturnsSuffixWithoutRepeatingPrefix(t *testing.T) {
	c := NewCompleter(NewApp())

	candidates, length := c.Do([]rune("travel p"), 8)

	assert.Equal(t, 1, length)
	assert.Contains(t, runeStr(candidates), "ark")
}

func TestCompleter_FullCommandWithSpace_AfterRoute_OffersLocations(t *testing.T) {
	c := NewCompleter(NewApp())
	candidates, _ := c.Do([]rune("route "), 6)

	names := runeStr(candidates)
	assert.Contains(t, names, "station")
	assert.NotContains(t, names, "home")
}

func TestCompleter_AfterShift_OffersBack(t *testing.T) {
	c := NewCompleter(NewApp())
	candidates, _ := c.Do([]rune("shift "), 6)

	names := runeStr(candidates)
	assert.Contains(t, names, "back")
}

func TestCompleter_AfterJump_OffersBack(t *testing.T) {
	c := NewCompleter(NewApp())
	candidates, _ := c.Do([]rune("jump "), 5)

	names := runeStr(candidates)
	assert.Contains(t, names, "back")
}

func TestCompleter_UnknownCommand_ReturnsNoCompletions(t *testing.T) {
	c := NewCompleter(NewApp())
	candidates, _ := c.Do([]rune("unknown "), 8)
	assert.Empty(t, candidates)
}

func TestCompleter_TwoFullArgs_ReturnsNothing(t *testing.T) {
	c := NewCompleter(NewApp())
	// "travel home " — already has two complete words with trailing space
	candidates, _ := c.Do([]rune("travel home "), 12)
	assert.Empty(t, candidates)
}
