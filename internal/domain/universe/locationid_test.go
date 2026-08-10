package universe

import "testing"

// TestCanonicalIDs_OrderIndependent guards against the order-dependent ID
// collision bug: reaching the same logical coordinate via different branch
// orderings (e.g. shift-then-drift vs. drift-then-shift) must always produce
// the same location ID.
func TestCanonicalIDs_OrderIndependent(t *testing.T) {
	cases := []struct {
		name string
		a, b string
	}{
		{
			name: "shift then drift vs drift then shift",
			a:    CanonicalIDWithConsensus(CanonicalIDWithQuantum("home", 1), 1),
			b:    CanonicalIDWithQuantum(CanonicalIDWithConsensus("home", 1), 1),
		},
		{
			name: "jump then shift vs shift then jump",
			a:    CanonicalIDWithQuantum(CanonicalIDWithTimeline("home", 1), 1),
			b:    CanonicalIDWithTimeline(CanonicalIDWithQuantum("home", 1), 1),
		},
		{
			name: "drift then jump vs jump then drift",
			a:    CanonicalIDWithTimeline(CanonicalIDWithConsensus("home", 1), 1),
			b:    CanonicalIDWithConsensus(CanonicalIDWithTimeline("home", 1), 1),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.a != tc.b {
				t.Fatalf("expected same ID regardless of order, got %q vs %q", tc.a, tc.b)
			}
		})
	}
}

func TestCanonicalID_RepeatedAxisDoesNotDuplicateSuffix(t *testing.T) {
	id := CanonicalIDWithQuantum("home", 1)
	id = CanonicalIDWithQuantum(id, 2)
	if id != "home-q2" {
		t.Fatalf("expected home-q2, got %q", id)
	}
}

func TestParseLocationID_RoundTrip(t *testing.T) {
	id := "home-q1-t2-c3-at-20250101t000000z-o-mirror"
	base, ax := parseLocationID(id)
	if base != "home" {
		t.Fatalf("expected base 'home', got %q", base)
	}
	if ax.quantum != 1 || ax.timeline != 2 || ax.consensus != 3 || ax.time != "20250101t000000z" || ax.observer != "mirror" {
		t.Fatalf("unexpected axes: %+v", ax)
	}
	if got := buildLocationID(base, ax); got != id {
		t.Fatalf("round trip mismatch: got %q, want %q", got, id)
	}
}

func TestParseLocationID_NearbyIDPreservedAsOpaqueBase(t *testing.T) {
	// Nearby dead-end locations use plain numeric suffixes (no axis letter),
	// so they must be preserved verbatim as part of the base rather than
	// misparsed as an axis suffix.
	base, ax := parseLocationID("home-1")
	if base != "home-1" {
		t.Fatalf("expected nearby ID preserved as base, got %q", base)
	}
	if ax != (axisSuffixes{}) {
		t.Fatalf("expected no axes parsed from nearby ID, got %+v", ax)
	}

	// Further branching off a nearby location still canonicalizes correctly.
	branched := CanonicalIDWithQuantum("home-1", 1)
	if branched != "home-1-q1" {
		t.Fatalf("expected home-1-q1, got %q", branched)
	}
}
