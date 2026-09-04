package facade

import (
	"testing"

	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newHyphenApp builds an App whose universe holds a hyphenated location ID whose
// display name differs from that ID, so the fuzzy matcher's compact-ID and
// name-matching branches can be exercised independently of the plain-ID branch.
func newHyphenApp(t *testing.T) *App {
	t.Helper()
	u := universe.NewAggregate()
	base := universe.DefaultCoordinateVO()
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home", Name: "Home", Coordinate: base}))
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "city-centre", Name: "Town Hall", Coordinate: base}))

	repo := mocks.NewMockRepository(t)
	app, err := New(u, repo, "home",
		navigation.NewBFSPathfinder(),
		universe.NewClusterLocationGenerator(),
	)
	require.NoError(t, err)
	return app
}

// suggestDestination is the fuzzy destination matcher behind the "did you mean?"
// hint. It returns a match when the closest location ID/name is within edit
// distance 2 outright, or within 3 for a longer typed target (>6 chars), and ""
// otherwise. These cases pin each of those thresholds.
func TestSuggestDestination(t *testing.T) {
	app := newGameApp(t) // home / station universe

	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "empty target", target: "", want: ""},
		{name: "near typo within 2", target: "statio", want: "station"},
		{name: "single-char slip", target: "hom", want: "home"},
		{name: "far mismatch", target: "abc", want: ""},
		{name: "distance 3 on a long target matches", target: "stationxyz", want: "station"},
		{name: "distance 3 on a short target does not match", target: "homxxx", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, app.suggestDestination(tt.target))
		})
	}
}

// The matcher compacts hyphens/underscores out of IDs and also compares against
// the display name, so a spaced or unhyphenated target still resolves to the
// canonical ID. Both branches are checked here because neither can win on the
// plain home/station universe (whose IDs equal their compacted forms and names).
func TestSuggestDestination_CompactAndNameBranches(t *testing.T) {
	app := newHyphenApp(t)

	assert.Equal(t, "city-centre", app.suggestDestination("citycentre"),
		"an unhyphenated target matches the compacted ID")
	assert.Equal(t, "city-centre", app.suggestDestination("town hall"),
		"a target matches the display name, resolving to its ID")
}
