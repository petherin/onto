package universe

import (
	"regexp"
	"strconv"
)

// axisSuffixes captures the reality-branch axes encoded in a location ID's
// suffix. The zero value represents "no branch on this axis".
type axisSuffixes struct {
	quantum   int    // 0 = no quantum suffix
	timeline  int    // 0 = no timeline suffix
	consensus int    // 0 = no consensus suffix
	time      string // "" = no time suffix; otherwise the raw timestamp token
	observer  string // "" = no observer suffix; otherwise the raw observer token
}

var (
	quantumSuffixRe   = regexp.MustCompile(`-q(\d+)$`)
	timelineSuffixRe  = regexp.MustCompile(`-t(\d+)$`)
	consensusSuffixRe = regexp.MustCompile(`-c(\d+)$`)
	timeSuffixRe      = regexp.MustCompile(`-at-([0-9tz]+)$`)
	observerSuffixRe  = regexp.MustCompile(`-o-(.+)$`)
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
		break
	}
	return base, ax
}

// buildLocationID reassembles a base and its axes into a canonical location
// ID. The axis order is always quantum, timeline, consensus, time, observer —
// regardless of the order the branches were actually taken in — so that
// reaching the same logical coordinate via a different sequence of shifts
// always produces the same ID.
func buildLocationID(base string, ax axisSuffixes) string {
	id := base
	if ax.quantum > 0 {
		id += "-q" + itoa(ax.quantum)
	}
	if ax.timeline > 0 {
		id += "-t" + itoa(ax.timeline)
	}
	if ax.consensus > 0 {
		id += "-c" + itoa(ax.consensus)
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
func ParseLocationID(id string) (base string, quantum, timeline, consensus int, time, observer string) {
	b, ax := parseLocationID(id)
	return b, ax.quantum, ax.timeline, ax.consensus, ax.time, ax.observer
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
