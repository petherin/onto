package universe

import (
	"regexp"
	"strconv"
	"strings"
)

// axisSuffixes captures the reality-branch axes encoded in a location ID's
// suffix. The zero value represents "no branch on this axis".
type axisSuffixes struct {
	mathematics int    // 0 = no mathematics suffix
	universe    int    // 0 = no universe suffix
	quantum     int    // 0 = no quantum suffix
	timeline    int    // 0 = no timeline suffix
	consensus   int    // 0 = no consensus suffix
	simulation  int    // 0 = no simulation suffix
	time        string // "" = no time suffix; otherwise the raw timestamp token
	observer    string // "" = no observer suffix; otherwise the raw observer token
}

var (
	mathematicsSuffixRe = regexp.MustCompile(`-m(\d+)$`)
	universeSuffixRe    = regexp.MustCompile(`-u(\d+)$`)
	quantumSuffixRe     = regexp.MustCompile(`-q(\d+)$`)
	timelineSuffixRe    = regexp.MustCompile(`-t(\d+)$`)
	consensusSuffixRe   = regexp.MustCompile(`-c(\d+)$`)
	simulationSuffixRe  = regexp.MustCompile(`-s(\d+)$`)
	timeSuffixRe        = regexp.MustCompile(`-at-([0-9tz]+)$`)
	observerSuffixRe    = regexp.MustCompile(`-o-(.+)$`)
)

// parseLocationID splits a location ID into its stable base (the physical
// anchor, e.g. "home", or an opaque nearby-location ID such as "home-1") and
// the reality-branch axes encoded in its suffix. It repeatedly strips known
// axis suffixes from the end of the string, so it recognizes suffixes
// regardless of the order they were appended in — this lets canonical
// reassembly correct for legacy, order-dependent IDs (e.g. "home-c1-q1").
//
// Observer suffixes are always the outermost segment (matched last, greedily
// consuming everything remaining) since observer-perspective nesting order is
// semantically meaningful and intentionally left un-reordered.
func parseLocationID(id string) (base string, ax axisSuffixes) {
	base = id
	for {
		if m := observerSuffixRe.FindStringSubmatch(base); m != nil && ax.observer == "" {
			ax.observer = m[1]
			base = base[:len(base)-len(m[0])]
			continue
		}
		if m := timeSuffixRe.FindStringSubmatch(base); m != nil && ax.time == "" {
			ax.time = m[1]
			base = base[:len(base)-len(m[0])]
			continue
		}
		if m := quantumSuffixRe.FindStringSubmatch(base); m != nil && ax.quantum == 0 {
			ax.quantum = atoiSafe(m[1])
			base = base[:len(base)-len(m[0])]
			continue
		}
		if m := timelineSuffixRe.FindStringSubmatch(base); m != nil && ax.timeline == 0 {
			ax.timeline = atoiSafe(m[1])
			base = base[:len(base)-len(m[0])]
			continue
		}
		if m := consensusSuffixRe.FindStringSubmatch(base); m != nil && ax.consensus == 0 {
			ax.consensus = atoiSafe(m[1])
			base = base[:len(base)-len(m[0])]
			continue
		}
		if m := simulationSuffixRe.FindStringSubmatch(base); m != nil && ax.simulation == 0 {
			ax.simulation = atoiSafe(m[1])
			base = base[:len(base)-len(m[0])]
			continue
		}
		if m := universeSuffixRe.FindStringSubmatch(base); m != nil && ax.universe == 0 {
			ax.universe = atoiSafe(m[1])
			base = base[:len(base)-len(m[0])]
			continue
		}
		if m := mathematicsSuffixRe.FindStringSubmatch(base); m != nil && ax.mathematics == 0 {
			ax.mathematics = atoiSafe(m[1])
			base = base[:len(base)-len(m[0])]
			continue
		}
		break
	}
	return base, ax
}

// buildLocationID reassembles a base and its axes into a canonical location
// ID. The axis order is always mathematics, universe, quantum, timeline,
// consensus, simulation, time, observer — regardless of the order the
// branches were actually taken in — so that reaching the same logical
// coordinate via a different sequence of shifts always produces the same ID.
func buildLocationID(base string, ax axisSuffixes) string {
	id := base
	if ax.mathematics > 0 {
		id += "-m" + itoa(ax.mathematics)
	}
	if ax.universe > 0 {
		id += "-u" + itoa(ax.universe)
	}
	if ax.quantum > 0 {
		id += "-q" + itoa(ax.quantum)
	}
	if ax.timeline > 0 {
		id += "-t" + itoa(ax.timeline)
	}
	if ax.consensus > 0 {
		id += "-c" + itoa(ax.consensus)
	}
	if ax.simulation > 0 {
		id += "-s" + itoa(ax.simulation)
	}
	if ax.time != "" {
		id += "-at-" + ax.time
	}
	if ax.observer != "" {
		id += "-o-" + ax.observer
	}
	return id
}

// ParseLocationID exposes parseLocationID for consumers outside this package
// (e.g. scripts/validate_locations.go) that need to check ID/coordinate
// consistency without duplicating the suffix grammar.
func ParseLocationID(id string) (base string, mathematics, universeLvl, quantum, timeline, consensus, simulation int, time, observer string) {
	b, ax := parseLocationID(id)
	return b, ax.mathematics, ax.universe, ax.quantum, ax.timeline, ax.consensus, ax.simulation, ax.time, ax.observer
}

// CanonicalLocationID rebuilds id so its reality-axis suffixes are taken from
// coord and appear in canonical order, while the physical anchor (the base
// place name plus any numeric nearby indices) is preserved in its original
// order. The ID's own time/observer tokens are kept verbatim, since those
// axes are edge-defined rather than level-encoded.
//
// It repairs IDs whose axis suffixes drifted out of canonical position — most
// importantly a nearby location generated inside a reality branch, where a bare
// "-<index>" was appended after an axis suffix (e.g. "park-u1-1"), leaving the
// ID's encoded axes disagreeing with its coordinate and breaking every *back /
// return-home step. Such an ID is rebuilt to "park-1-u1". A healthy, already
// canonical ID is returned unchanged, so callers can detect corruption simply
// by comparing the result to the original.
func CanonicalLocationID(id string, coord CoordinateVO) string {
	_, idAx := parseLocationID(id)
	ax := axisSuffixes{
		mathematics: coord.MathematicsLevel(),
		universe:    coord.UniverseLevel(),
		quantum:     coord.QuantumLevel(),
		timeline:    coord.TimelineLevel(),
		consensus:   coord.Consensus,
		simulation:  coord.Simulation,
		time:        idAx.time,
		observer:    idAx.observer,
	}
	return buildLocationID(physicalAnchor(id), ax)
}

// LocationIDIsMalformed reports whether id carries a reality-axis segment buried
// where parseLocationID cannot strip it — the corruption class that breaks *back
// navigation. It is true precisely when the physical anchor (axes removed from
// anywhere) differs from the base parseLocationID recovers by stripping only
// canonical trailing suffixes. A nearby location spawned inside a branch, whose
// bare "-<index>" was appended after an axis suffix (e.g. "park-u1-1"), is
// malformed: parse stops at the trailing "-1" leaving "-u1" stranded in the
// base. An ID that merely omits axis suffixes it could carry (e.g. a hand-seeded
// "base" whose coordinate is on Timeline T3) is NOT malformed, so load-time
// repair leaves it untouched rather than renaming a legitimate place. An
// order-shuffled but fully strippable ID (e.g. "home-q2-u1") is likewise not
// malformed — parse recovers its axes regardless of order, so it navigates fine.
func LocationIDIsMalformed(id string) bool {
	base, _ := parseLocationID(id)
	return physicalAnchor(id) != base
}

// physicalAnchor strips every reality-axis segment from id wherever it appears,
// leaving only the base place name and any numeric nearby indices in their
// original order. Unlike parseLocationID (which only strips canonical trailing
// suffixes), this tolerates axis suffixes interleaved with nearby indices, so a
// corrupt "park-u1-1" reduces to the anchor "park-1". An observer marker ("o")
// and everything after it are dropped, since the observer token may itself
// contain hyphens; a time marker ("at") drops the single token that follows.
func physicalAnchor(id string) string {
	parts := strings.Split(id, "-")
	if len(parts) == 0 {
		return id
	}
	anchor := []string{parts[0]}
	for i := 1; i < len(parts); i++ {
		seg := parts[i]
		switch {
		case seg == "o":
			return strings.Join(anchor, "-")
		case seg == "at":
			i++ // skip the timestamp token that follows the marker
		case isNumericAxisSegment(seg):
			// drop a level-encoded axis suffix (m/u/q/t/c/s + digits)
		default:
			anchor = append(anchor, seg) // base word part or numeric nearby index
		}
	}
	return strings.Join(anchor, "-")
}

// isNumericAxisSegment reports whether seg is a level-encoded axis suffix token
// (one of the m/u/q/t/c/s axis letters followed by one or more digits).
func isNumericAxisSegment(seg string) bool {
	if len(seg) < 2 {
		return false
	}
	switch seg[0] {
	case 'm', 'u', 'q', 't', 'c', 's':
	default:
		return false
	}
	for _, r := range seg[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// CanonicalIDWithSimulation parses currentID, overrides its simulation axis,
// and reassembles the ID in canonical order (see CanonicalIDWithQuantum).
func CanonicalIDWithSimulation(currentID string, level int) string {
	base, ax := parseLocationID(currentID)
	ax.simulation = level
	return buildLocationID(base, ax)
}

// CanonicalIDWithMathematics parses currentID, overrides its mathematics axis,
// and reassembles the ID in canonical order (see CanonicalIDWithQuantum).
func CanonicalIDWithMathematics(currentID string, level int) string {
	base, ax := parseLocationID(currentID)
	ax.mathematics = level
	return buildLocationID(base, ax)
}

// CanonicalIDWithUniverse parses currentID, overrides its universe axis, and
// reassembles the ID in canonical order (see CanonicalIDWithQuantum).
func CanonicalIDWithUniverse(currentID string, level int) string {
	base, ax := parseLocationID(currentID)
	ax.universe = level
	return buildLocationID(base, ax)
}

// CanonicalIDWithQuantum parses currentID, overrides its quantum axis, and
// reassembles the ID in canonical order. This is the single choke point used
// by every branch-generating command so IDs never depend on the order
// branches were taken in.
func CanonicalIDWithQuantum(currentID string, level int) string {
	base, ax := parseLocationID(currentID)
	ax.quantum = level
	return buildLocationID(base, ax)
}

// CanonicalIDWithTimeline parses currentID, overrides its timeline axis, and
// reassembles the ID in canonical order (see CanonicalIDWithQuantum).
func CanonicalIDWithTimeline(currentID string, level int) string {
	base, ax := parseLocationID(currentID)
	ax.timeline = level
	return buildLocationID(base, ax)
}

// CanonicalIDWithConsensus parses currentID, overrides its consensus axis,
// and reassembles the ID in canonical order (see CanonicalIDWithQuantum).
func CanonicalIDWithConsensus(currentID string, level int) string {
	base, ax := parseLocationID(currentID)
	ax.consensus = level
	return buildLocationID(base, ax)
}

// CanonicalIDWithTime parses currentID, overrides its time axis, and
// reassembles the ID in canonical order (see CanonicalIDWithQuantum).
func CanonicalIDWithTime(currentID, token string) string {
	base, ax := parseLocationID(currentID)
	ax.time = token
	return buildLocationID(base, ax)
}

// CanonicalIDWithObserver parses currentID, overrides its observer axis, and
// reassembles the ID in canonical order (see CanonicalIDWithQuantum).
func CanonicalIDWithObserver(currentID, token string) string {
	base, ax := parseLocationID(currentID)
	ax.observer = token
	return buildLocationID(base, ax)
}

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
