package universe

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CoordinateVO is a value object representing a full reality position vector.
// Each field narrows scope from the widest (Meta, Universe) down to the most
// local (Location, Observer). Zero values are valid — a CoordinateVO with only
// Planet and City set still describes a meaningful position within a physical world.
type CoordinateVO struct {
	Meta        string
	Mathematics string
	Universe    string
	Timeline    string
	Quantum     string
	Simulation  int
	Consensus   int
	Galaxy      string
	System      string
	Planet      string
	Country     string
	Region      string
	City        string
	Location    string
	Observer    string
	Time        time.Time
}

// DefaultCoordinateVO returns the default starting CoordinateVO: Earth, United
// Kingdom, Yorkshire, Leeds, Home, Observer: Human, Prime timeline, Q0 quantum level,
// consensus reality (0).
func DefaultCoordinateVO() CoordinateVO {
	return CoordinateVO{
		Meta:        "Origin",
		Mathematics: "Classical",
		Universe:    "Origin",
		Timeline:    "Prime",
		Quantum:     "Q0",
		Simulation:  0,
		Consensus:   0,
		Galaxy:      "Milky Way",
		System:      "Solar System",
		Planet:      "Earth",
		Country:     "United Kingdom",
		Region:      "Yorkshire",
		City:        "Leeds",
		Location:    "Home",
		Observer:    "Human",
	}
}

// QuantumLevel returns the numeric quantum level encoded in the Quantum field (Q0 → 0, Q1 → 1, …).
func (c CoordinateVO) QuantumLevel() int {
	if len(c.Quantum) > 1 && c.Quantum[0] == 'Q' {
		if n, err := strconv.Atoi(c.Quantum[1:]); err == nil {
			return n
		}
	}
	return 0
}

// TimelineLevel returns the numeric timeline level ("Prime" → 0, "T1" → 1, "T2" → 2, …).
func (c CoordinateVO) TimelineLevel() int {
	if c.Timeline == "Prime" || c.Timeline == "" {
		return 0
	}
	if len(c.Timeline) > 1 && c.Timeline[0] == 'T' {
		if n, err := strconv.Atoi(c.Timeline[1:]); err == nil {
			return n
		}
	}
	return 0
}

// UniverseLevel returns the numeric bubble-universe level encoded in the
// Universe field ("Origin" → 0, "U1" → 1, "U2" → 2, …).
func (c CoordinateVO) UniverseLevel() int {
	if c.Universe == "Origin" || c.Universe == "" {
		return 0
	}
	if len(c.Universe) > 1 && c.Universe[0] == 'U' {
		if n, err := strconv.Atoi(c.Universe[1:]); err == nil {
			return n
		}
	}
	return 0
}

// MathematicsLevel returns the numeric mathematical-structure level encoded in
// the Mathematics field ("Classical" → 0, "M1" → 1, "M2" → 2, …).
func (c CoordinateVO) MathematicsLevel() int {
	if c.Mathematics == "Classical" || c.Mathematics == "" {
		return 0
	}
	if len(c.Mathematics) > 1 && c.Mathematics[0] == 'M' {
		if n, err := strconv.Atoi(c.Mathematics[1:]); err == nil {
			return n
		}
	}
	return 0
}

// NestingDepth reports how many reality transitions separate this coordinate
// from base reality: the sum of every axis's nesting level. It counts the
// numeric axes (universe, timeline, quantum, mathematics, simulation, and
// consensus) plus one each for a non-default observer and a set (non-zero)
// time. The default coordinate — base reality — has depth 0. It gives GUIs a
// single measure of "how deep" a location sits so they can lay nested realities
// out by depth rather than scattering them.
func (c CoordinateVO) NestingDepth() int {
	depth := c.UniverseLevel() + c.TimelineLevel() + c.QuantumLevel() +
		c.MathematicsLevel() + c.Simulation + c.Consensus
	if c.Observer != "" && c.Observer != DefaultCoordinateVO().Observer {
		depth++
	}
	if !c.Time.IsZero() {
		depth++
	}
	return depth
}

// SamePhysicalReality reports whether two coordinates differ only in their
// spatial position. Physical travel must not cross any reality boundary.
func (c CoordinateVO) SamePhysicalReality(other CoordinateVO) bool {
	return c.Meta == other.Meta &&
		c.Mathematics == other.Mathematics &&
		c.Universe == other.Universe &&
		c.Timeline == other.Timeline &&
		c.Quantum == other.Quantum &&
		c.Simulation == other.Simulation &&
		c.Consensus == other.Consensus &&
		c.Observer == other.Observer &&
		c.Time.Equal(other.Time)
}

// OntoAddress returns the full canonical Onto Address for the coordinate.
// The Onto Address is the addressing system for this project — a deterministic,
// parseable string that uniquely identifies any position across all realities
// and modes of existence. All axes are always included; empty string fields are
// rendered as "_" so the segment count is fixed and the address is unambiguous.
// Spaces within field values are encoded as "_".
//
// Format:
//
//	onto://<meta>.<math>/<universe>/<timeline>/<quantum>/<galaxy>/<system>/<planet>/<country>/<region>/<city>/<location>/sim:<n>/cons:<n>@<observer>+<time>
//
// sim:<n> and cons:<n> are omitted when n == 0.
// The +<time> suffix is omitted when Time is the zero value.
func (c CoordinateVO) OntoAddress() string {
	var b strings.Builder
	fmt.Fprintf(&b, "onto://%s.%s/%s/%s/%s/%s/%s/%s/%s/%s/%s/%s",
		segEncode(c.Meta), segEncode(c.Mathematics),
		segEncode(c.Universe), segEncode(c.Timeline), segEncode(c.Quantum),
		segEncode(c.Galaxy), segEncode(c.System), segEncode(c.Planet),
		segEncode(c.Country), segEncode(c.Region), segEncode(c.City), segEncode(c.Location),
	)
	if c.Simulation != 0 {
		fmt.Fprintf(&b, "/sim:%d", c.Simulation)
	}
	if c.Consensus != 0 {
		fmt.Fprintf(&b, "/cons:%d", c.Consensus)
	}
	fmt.Fprintf(&b, "@%s", segEncode(c.Observer))
	if !c.Time.IsZero() {
		fmt.Fprintf(&b, "+%s", c.Time.UTC().Format(time.RFC3339))
	}
	return b.String()
}

// ShortOntoAddress returns a compact Onto Address that omits axes whose value
// matches the default (Origin, Classical, Prime, Q0, Human, Milky Way, etc.)
// or is empty. Useful for prompts and route summaries where brevity matters.
func (c CoordinateVO) ShortOntoAddress() string {
	d := DefaultCoordinateVO()
	var parts []string

	reality := []string{}
	if c.Meta != "" && c.Meta != d.Meta {
		reality = append(reality, segEncode(c.Meta))
	}
	if c.Mathematics != "" && c.Mathematics != d.Mathematics {
		reality = append(reality, segEncode(c.Mathematics))
	}
	if len(reality) > 0 {
		parts = append(parts, strings.Join(reality, "."))
	}
	if c.Universe != "" && c.Universe != d.Universe {
		parts = append(parts, segEncode(c.Universe))
	}
	if c.Timeline != "" && c.Timeline != d.Timeline {
		parts = append(parts, segEncode(c.Timeline))
	}
	if c.Quantum != "" && c.Quantum != d.Quantum {
		parts = append(parts, segEncode(c.Quantum))
	}
	if c.Simulation != 0 {
		parts = append(parts, fmt.Sprintf("sim:%d", c.Simulation))
	}
	if c.Consensus != 0 {
		parts = append(parts, fmt.Sprintf("cons:%d", c.Consensus))
	}
	if c.Galaxy != "" && c.Galaxy != d.Galaxy {
		parts = append(parts, segEncode(c.Galaxy))
	}
	if c.System != "" && c.System != d.System {
		parts = append(parts, segEncode(c.System))
	}
	if c.Planet != "" && c.Planet != d.Planet {
		parts = append(parts, segEncode(c.Planet))
	}
	if c.Country != "" && c.Country != d.Country {
		parts = append(parts, segEncode(c.Country))
	}
	if c.Region != "" && c.Region != d.Region {
		parts = append(parts, segEncode(c.Region))
	}
	if c.City != "" {
		parts = append(parts, segEncode(c.City))
	}
	if c.Location != "" {
		parts = append(parts, segEncode(c.Location))
	}

	addr := "onto://" + strings.Join(parts, "/")

	if c.Observer != "" && c.Observer != d.Observer {
		addr += "@" + segEncode(c.Observer)
	}
	if !c.Time.IsZero() {
		addr += "+" + c.Time.UTC().Format(time.RFC3339)
	}
	return addr
}

// ParseOntoAddress parses a canonical or short Onto Address back into a
// CoordinateVO. It is the inverse of OntoAddress() for full addresses. Short
// addresses round-trip through ShortOntoAddress() → ParseOntoAddress() but
// will only populate the fields that were present.
func ParseOntoAddress(addr string) (CoordinateVO, error) {
	addr = strings.TrimPrefix(addr, "onto://")

	// Split off time suffix (+RFC3339).
	var c CoordinateVO
	if idx := strings.LastIndex(addr, "+"); idx != -1 {
		t, err := time.Parse(time.RFC3339, addr[idx+1:])
		if err == nil {
			c.Time = t
			addr = addr[:idx]
		}
	}

	// Split off observer suffix (@observer).
	if idx := strings.LastIndex(addr, "@"); idx != -1 {
		c.Observer = segDecode(addr[idx+1:])
		addr = addr[:idx]
	}

	segments := strings.Split(addr, "/")

	// Detect full vs short by whether segment[0] is a meta.math pair and there
	// are at least the 11 fixed spatial segments; otherwise it is a short
	// address carrying only the non-default fields.
	if len(segments) >= 11 && strings.Contains(segments[0], ".") {
		return parseFullOntoAddress(c, segments), nil
	}
	return parseShortOntoAddress(c, segments), nil
}

// parseFullOntoAddress populates c from a full canonical address' segments,
// whose positions are fixed: [0] meta.math, [1] universe, [2] timeline,
// [3] quantum, [4..10] galaxy…location, and [11+] the sim:n / cons:n suffixes.
func parseFullOntoAddress(c CoordinateVO, segments []string) CoordinateVO {
	dot := strings.SplitN(segments[0], ".", 2)
	c.Meta = segDecode(dot[0])
	c.Mathematics = segDecode(dot[1])
	c.Universe = segDecode(segments[1])
	c.Timeline = segDecode(segments[2])
	c.Quantum = segDecode(segments[3])
	c.Galaxy = segDecode(segments[4])
	c.System = segDecode(segments[5])
	c.Planet = segDecode(segments[6])
	c.Country = segDecode(segments[7])
	c.Region = segDecode(segments[8])
	c.City = segDecode(segments[9])
	c.Location = segDecode(segments[10])
	for i := 11; i < len(segments); i++ {
		if strings.HasPrefix(segments[i], "sim:") {
			n, err := strconv.Atoi(strings.TrimPrefix(segments[i], "sim:"))
			if err == nil {
				c.Simulation = n
			}
		} else if strings.HasPrefix(segments[i], "cons:") {
			n, err := strconv.Atoi(strings.TrimPrefix(segments[i], "cons:"))
			if err == nil {
				c.Consensus = n
			}
		}
	}
	return c
}

// parseShortOntoAddress populates c from a short address, where only the
// non-default fields are present and in spatial order. It is intentionally
// best-effort: sim:, cons:, and meta.math segments are recognised explicitly,
// and every other segment fills the first still-empty spatial field in order.
func parseShortOntoAddress(c CoordinateVO, segments []string) CoordinateVO {
	for _, s := range segments {
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, "sim:") {
			n, err := strconv.Atoi(strings.TrimPrefix(s, "sim:"))
			if err == nil {
				c.Simulation = n
			}
			continue
		}
		if strings.HasPrefix(s, "cons:") {
			n, err := strconv.Atoi(strings.TrimPrefix(s, "cons:"))
			if err == nil {
				c.Consensus = n
			}
			continue
		}
		if strings.Contains(s, ".") {
			dot := strings.SplitN(s, ".", 2)
			c.Meta = segDecode(dot[0])
			c.Mathematics = segDecode(dot[1])
			continue
		}
		c = assignFirstEmptySpatialField(c, segDecode(s))
	}
	return c
}

// assignFirstEmptySpatialField writes val into the first still-empty spatial
// field of c, in canonical order from universe down to location, and returns
// the updated coordinate. It is the field-placement half of short-address
// parsing, where segment positions are not fixed.
func assignFirstEmptySpatialField(c CoordinateVO, val string) CoordinateVO {
	switch {
	case c.Universe == "":
		c.Universe = val
	case c.Timeline == "":
		c.Timeline = val
	case c.Quantum == "":
		c.Quantum = val
	case c.Galaxy == "":
		c.Galaxy = val
	case c.System == "":
		c.System = val
	case c.Planet == "":
		c.Planet = val
	case c.Country == "":
		c.Country = val
	case c.Region == "":
		c.Region = val
	case c.City == "":
		c.City = val
	default:
		c.Location = val
	}
	return c
}

// segEncode encodes a coordinate field for use in an Onto Address segment:
// empty strings become "_" and spaces are replaced with "_".
func segEncode(s string) string {
	if s == "" {
		return "_"
	}
	return strings.ReplaceAll(s, " ", "_")
}

// segDecode is the inverse of segEncode: "_" sentinel becomes "" and "_" within
// a non-sentinel value is decoded back to a space.
func segDecode(s string) string {
	if s == "_" {
		return ""
	}
	return strings.ReplaceAll(s, "_", " ")
}
