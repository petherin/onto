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
		{
			name: "universe then shift vs shift then universe",
			a:    CanonicalIDWithQuantum(CanonicalIDWithUniverse("home", 1), 1),
			b:    CanonicalIDWithUniverse(CanonicalIDWithQuantum("home", 1), 1),
		},
		{
			name: "universe then jump vs jump then universe",
			a:    CanonicalIDWithTimeline(CanonicalIDWithUniverse("home", 1), 1),
			b:    CanonicalIDWithUniverse(CanonicalIDWithTimeline("home", 1), 1),
		},
		{
			name: "mathematical then universe vs universe then mathematical",
			a:    CanonicalIDWithUniverse(CanonicalIDWithMathematics("home", 1), 1),
			b:    CanonicalIDWithMathematics(CanonicalIDWithUniverse("home", 1), 1),
		},
		{
			name: "mathematical then shift vs shift then mathematical",
			a:    CanonicalIDWithQuantum(CanonicalIDWithMathematics("home", 1), 1),
			b:    CanonicalIDWithMathematics(CanonicalIDWithQuantum("home", 1), 1),
		},
		{
			name: "simulate then drift vs drift then simulate",
			a:    CanonicalIDWithConsensus(CanonicalIDWithSimulation("home", 1), 1),
			b:    CanonicalIDWithSimulation(CanonicalIDWithConsensus("home", 1), 1),
		},
		{
			name: "simulate then shift vs shift then simulate",
			a:    CanonicalIDWithQuantum(CanonicalIDWithSimulation("home", 1), 1),
			b:    CanonicalIDWithSimulation(CanonicalIDWithQuantum("home", 1), 1),
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

func TestCanonicalIDWithUniverse_RepeatedAxisDoesNotDuplicateSuffix(t *testing.T) {
	id := CanonicalIDWithUniverse("home", 1)
	id = CanonicalIDWithUniverse(id, 2)
	if id != "home-u2" {
		t.Fatalf("expected home-u2, got %q", id)
	}
}

func TestCanonicalIDWithMathematics_RepeatedAxisDoesNotDuplicateSuffix(t *testing.T) {
	id := CanonicalIDWithMathematics("home", 1)
	id = CanonicalIDWithMathematics(id, 2)
	if id != "home-m2" {
		t.Fatalf("expected home-m2, got %q", id)
	}
}

func TestCanonicalIDWithSimulation_RepeatedAxisDoesNotDuplicateSuffix(t *testing.T) {
	id := CanonicalIDWithSimulation("home", 1)
	id = CanonicalIDWithSimulation(id, 2)
	if id != "home-s2" {
		t.Fatalf("expected home-s2, got %q", id)
	}
}

func TestParseLocationID_RoundTrip(t *testing.T) {
	id := "home-m4-u5-q1-t2-c3-s2-at-20250101t000000z-o-mirror"
	base, ax := parseLocationID(id)
	if base != "home" {
		t.Fatalf("expected base 'home', got %q", base)
	}
	if ax.mathematics != 4 || ax.universe != 5 || ax.quantum != 1 || ax.timeline != 2 || ax.consensus != 3 || ax.simulation != 2 || ax.time != "20250101t000000z" || ax.observer != "mirror" {
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

func TestCanonicalLocationID(t *testing.T) {
	u1 := DefaultCoordinateVO()
	u1.Universe = "U1"

	u1q2 := DefaultCoordinateVO()
	u1q2.Universe = "U1"
	u1q2.Quantum = "Q2"

	withTime := DefaultCoordinateVO()
	withTime.Universe = "U1"

	withObserver := DefaultCoordinateVO()
	withObserver.Quantum = "Q1"

	cases := []struct {
		name  string
		id    string
		coord CoordinateVO
		want  string
	}{
		{"base id untouched", "home", DefaultCoordinateVO(), "home"},
		{"healthy canonical untouched", "park-1-u1", u1, "park-1-u1"},
		{"healthy hyphenated base untouched", "city-centre-q1", withObserver, "city-centre-q1"},
		{"buried nearby index repaired", "park-u1-1", u1, "park-1-u1"},
		{"deeply buried nearby index repaired", "park-1-1-u1-1", u1, "park-1-1-1-u1"},
		{"multi-axis buried index repaired", "park-u1-1-q2", u1q2, "park-1-u1-q2"},
		{"order shuffle normalised", "home-q2-u1", u1q2, "home-u1-q2"},
		{"time token preserved", "park-u1-1-at-20250101t000000z", withTime, "park-1-u1-at-20250101t000000z"},
		{"observer token preserved", "park-1-q1-o-mirror", withObserver, "park-1-q1-o-mirror"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanonicalLocationID(tc.id, tc.coord); got != tc.want {
				t.Fatalf("CanonicalLocationID(%q) = %q, want %q", tc.id, got, tc.want)
			}
		})
	}
}

func TestLocationIDIsMalformed(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"home", false},         // plain base
		{"park-1", false},       // nearby index, no axis
		{"park-1-u1", false},    // healthy canonical
		{"home-q2-u1", false},   // order-shuffled but fully strippable
		{"base", false},         // omits axes it could carry — not corruption
		{"park-u1-1", true},     // index buried after axis suffix
		{"park-1-1-u1-1", true}, // deeply buried index
		{"park-u1-1-q2", true},  // buried index amid multiple axes
	}
	for _, tc := range cases {
		if got := LocationIDIsMalformed(tc.id); got != tc.want {
			t.Errorf("LocationIDIsMalformed(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}
