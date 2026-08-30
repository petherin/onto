package universe

import (
	"fmt"
	"math/rand"
)

// Name corpora for NewNearbyCluster. A display name is composed as
// "<qualifier> <noun>" (e.g. "The Quiet Wharf"), drawn from these pools so
// auto-generated nodes read as distinct places rather than a "Nearby N"
// counter. The pools are kept large enough that qualifiers × nouns far exceeds
// the handful of nodes any single expansion adds, so collisions are rare and
// the uniqueness fallback in generateNearbyName seldom fires. They parallel the
// phrasing pools in description_phrases.go.

// nearbyNameQualifiers open a name with an atmospheric adjective phrase.
var nearbyNameQualifiers = []string{
	"The Quiet", "The Forgotten", "The Hollow", "The Crooked", "The Weathered",
	"The Drifting", "The Silent", "The Hidden", "The Fading", "The Narrow",
	"The Shuttered", "The Sunken", "The Wandering", "The Restless", "The Pale",
	"The Ember", "The Distant", "The Lantern", "The Salt", "The Grey",
	"The Sunless", "The Whispering", "The Broken", "The Frostbitten", "The Amber",
	"The Cinder", "The Moss", "The Iron", "The Ashen", "The Veiled",
	"The Twilight", "The Winding", "The Brackish", "The Threadbare", "The Murmuring",
	"The Vanished", "The Rusted", "The Gloaming", "The Hushed", "The Marsh",
}

// nearbyNameNouns close a name with a place-noun.
var nearbyNameNouns = []string{
	"Wharf", "Passage", "Terrace", "Junction", "Hollow",
	"Landing", "Yard", "Arcade", "Crossing", "Steps",
	"Cutting", "Alley", "Court", "Row", "Quay",
	"Sidings", "Green", "Mews", "Gate", "Wynd",
	"Approach", "Embankment", "Causeway", "Footbridge", "Cloister",
	"Undercroft", "Vault", "Bailey", "Culvert", "Towpath",
	"Ford", "Weir", "Lane", "Close", "Snicket",
	"Ginnel", "Bank", "Reach", "Verge", "Spur",
}

// generateNearbyName composes a display name unique within used, drawing from
// the deterministic rng stream so a given coordinate always yields the same
// names. It re-picks a bounded number of times on a collision, then falls back
// to appending a numeric disambiguator so a name is always produced while
// determinism is preserved. Callers are expected to add the returned name to
// used before requesting the next one.
func generateNearbyName(rng *rand.Rand, used map[string]bool) string {
	const attempts = 20
	compose := func() string {
		return nearbyNameQualifiers[rng.Intn(len(nearbyNameQualifiers))] + " " +
			nearbyNameNouns[rng.Intn(len(nearbyNameNouns))]
	}
	for i := 0; i < attempts; i++ {
		name := compose()
		if !used[name] {
			return name
		}
	}
	base := compose()
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s %d", base, n)
		if !used[candidate] {
			return candidate
		}
	}
}
