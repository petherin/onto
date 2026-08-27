package universe

import (
	"hash/fnv"
	"math/rand"
	"strconv"
	"strings"
)

// GenerateDescription composes a rich, varied description for a location from
// its full reality coordinate. It replaces flat placeholder text (e.g. the old
// "Auto-generated nearby location") so every generated node reads as a distinct
// place: a spatial opener anchors the setting, then each active non-default axis
// — quantum, timeline, universe, mathematics, simulation, consensus, observer,
// time — layers on an atmospheric clause describing how that reality frame
// colours it.
//
// Generation is deterministic: the coordinate's canonical Onto Address seeds the
// phrasing, so the same coordinate always yields the same description. That keeps
// descriptions stable across reloads and reproducible in tests, without any
// external service or dependency.
func GenerateDescription(coord CoordinateVO) string {
	rng := rand.New(rand.NewSource(coordinateSeed(coord)))
	pick := func(options []string) string { return options[rng.Intn(len(options))] }

	place := firstNonEmpty(coord.City, coord.Region, coord.Planet)

	var sentences []string
	if place == "" {
		sentences = append(sentences, pick(placelessOpeners))
	} else {
		sentences = append(sentences, strings.ReplaceAll(pick(spatialOpeners), "%s", place))
	}

	sentences = append(sentences, axisClauses(coord, pick)...)

	// A plain base-reality node has no axis clauses; give it a closing beat so it
	// still reads as more than a bare location.
	if len(sentences) == 1 {
		sentences = append(sentences, pick(closers))
	}

	return strings.Join(sentences, " ")
}

// axisClauses returns one atmospheric sentence for each reality axis that
// departs from base reality, in outermost-to-innermost order, choosing phrasing
// via pick so the result varies deterministically with the coordinate.
func axisClauses(coord CoordinateVO, pick func([]string) string) []string {
	var out []string
	add := func(options []string, token string) {
		out = append(out, strings.ReplaceAll(pick(options), "%s", token))
	}

	if coord.MathematicsLevel() > 0 {
		add(mathematicsClauses, coord.Mathematics)
	}
	if coord.UniverseLevel() > 0 {
		add(universeClauses, coord.Universe)
	}
	if coord.TimelineLevel() > 0 {
		add(timelineClauses, coord.Timeline)
	}
	if coord.QuantumLevel() > 0 {
		add(quantumClauses, coord.Quantum)
	}
	if coord.Simulation > 0 {
		add(simulationClauses, ordinal(coord.Simulation))
	}
	if coord.Consensus > 0 {
		add(consensusClauses, ordinal(coord.Consensus))
	}
	if coord.Observer != "" && coord.Observer != DefaultCoordinateVO().Observer {
		add(observerClauses, coord.Observer)
	}
	if !coord.Time.IsZero() {
		add(timeClauses, coord.Time.UTC().Format("2006-01-02 15:04 MST"))
	}
	return out
}

// coordinateSeed derives a stable 63-bit seed from the coordinate's canonical
// Onto Address, which uniquely identifies the position (including its location
// name), so distinct nodes get distinct — but reproducible — phrasing.
func coordinateSeed(coord CoordinateVO) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(coord.OntoAddress()))
	return int64(h.Sum64() & 0x7fffffffffffffff)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// ordinal renders small positive integers as words for readable clauses
// ("first", "second", ...); it falls back to the decimal number for larger depths.
func ordinal(n int) string {
	words := map[int]string{1: "first", 2: "second", 3: "third", 4: "fourth", 5: "fifth"}
	if w, ok := words[n]; ok {
		return w
	}
	return strconv.Itoa(n)
}
