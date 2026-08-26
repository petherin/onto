package universe

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- QuantumLevel / TimelineLevel ---

func TestQuantumLevel(t *testing.T) {
	tests := []struct {
		name    string
		quantum string
		want    int
	}{
		{"default Q0", "Q0", 0},
		{"Q3", "Q3", 3},
		{"unrecognised", "unknown", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := CoordinateVO{Quantum: tc.quantum}
			assert.Equal(t, tc.want, c.QuantumLevel())
		})
	}
}

func TestTimelineLevel(t *testing.T) {
	tests := []struct {
		name     string
		timeline string
		want     int
	}{
		{"Prime", "Prime", 0},
		{"T2", "T2", 2},
		{"unrecognised", "unknown", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := CoordinateVO{Timeline: tc.timeline}
			assert.Equal(t, tc.want, c.TimelineLevel())
		})
	}
}

func TestNestingDepth(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		coord CoordinateVO
		want  int
	}{
		{"default coordinate is base reality", DefaultCoordinateVO(), 0},
		{"zero value is base reality", CoordinateVO{}, 0},
		{"single numeric axis", CoordinateVO{Quantum: "Q3"}, 3},
		{
			"numeric axes sum",
			CoordinateVO{Universe: "U2", Timeline: "T1", Quantum: "Q1", Mathematics: "M1", Simulation: 2, Consensus: 1},
			8,
		},
		{"non-default observer adds one", CoordinateVO{Observer: "Bat"}, 1},
		{"default observer adds nothing", CoordinateVO{Observer: "Human"}, 0},
		{"set time adds one", CoordinateVO{Time: base}, 1},
		{
			"everything combined",
			CoordinateVO{Universe: "U1", Simulation: 1, Observer: "Bat", Time: base},
			4,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.coord.NestingDepth())
		})
	}
}

func TestUniverseLevel(t *testing.T) {
	tests := []struct {
		name     string
		universe string
		want     int
	}{
		{"default Origin", "Origin", 0},
		{"U4", "U4", 4},
		{"unrecognised", "unknown", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := CoordinateVO{Universe: tc.universe}
			assert.Equal(t, tc.want, c.UniverseLevel())
		})
	}
}

func TestMathematicsLevel(t *testing.T) {
	tests := []struct {
		name        string
		mathematics string
		want        int
	}{
		{"default Classical", "Classical", 0},
		{"M3", "M3", 3},
		{"unrecognised", "unknown", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := CoordinateVO{Mathematics: tc.mathematics}
			assert.Equal(t, tc.want, c.MathematicsLevel())
		})
	}
}

// --- OntoAddress() ---

func TestOntoAddress_FullCoordinate(t *testing.T) {
	c := DefaultCoordinateVO()
	addr := c.OntoAddress()

	assert.True(t, len(addr) > 7 && addr[:7] == "onto://", "Onto Address should start with onto://")
	assert.Contains(t, addr, "Origin.Classical")
	assert.Contains(t, addr, "/Origin/")    // Universe
	assert.Contains(t, addr, "/Prime/")     // Timeline
	assert.Contains(t, addr, "/Q0/")        // Quantum
	assert.Contains(t, addr, "/Milky_Way/") // Galaxy (space → _)
	assert.Contains(t, addr, "/Earth/")
	assert.Contains(t, addr, "/United_Kingdom/")
	assert.Contains(t, addr, "/Yorkshire/")
	assert.Contains(t, addr, "/Leeds/")
	assert.Contains(t, addr, "/Home@Human")
}

func TestOntoAddress_EmptyFieldsRenderedAsUnderscore(t *testing.T) {
	c := CoordinateVO{City: "Leeds", Location: "Home"}
	addr := c.OntoAddress()

	// Unset string fields become "_" in the full Onto Address.
	assert.Contains(t, addr, "_._/") // meta.math
	assert.Contains(t, addr, "@_")   // observer
}

func TestOntoAddress_SimulationDepthIncluded(t *testing.T) {
	c := DefaultCoordinateVO()
	c.Simulation = 3
	addr := c.OntoAddress()

	assert.Contains(t, addr, "/sim:3@")
}

func TestOntoAddress_SimulationZeroOmitted(t *testing.T) {
	c := DefaultCoordinateVO()
	addr := c.OntoAddress()

	assert.NotContains(t, addr, "sim:")
}

func TestOntoAddress_ConsensusDepthIncluded(t *testing.T) {
	c := DefaultCoordinateVO()
	c.Consensus = 2
	addr := c.OntoAddress()

	assert.Contains(t, addr, "/cons:2@")
}

func TestOntoAddress_ConsensusZeroOmitted(t *testing.T) {
	c := DefaultCoordinateVO()
	addr := c.OntoAddress()

	assert.NotContains(t, addr, "cons:")
}

func TestCoordinateVOSamePhysicalReality(t *testing.T) {
	base := DefaultCoordinateVO()
	spatialMove := base
	spatialMove.Location = "Station"
	assert.True(t, base.SamePhysicalReality(spatialMove))

	divergent := base
	divergent.Consensus = 1
	assert.False(t, base.SamePhysicalReality(divergent))
}

func TestOntoAddress_SimulationAndConsensusIncluded(t *testing.T) {
	c := DefaultCoordinateVO()
	c.Simulation = 1
	c.Consensus = 2
	addr := c.OntoAddress()

	assert.Contains(t, addr, "/sim:1/cons:2@")
}

func TestOntoAddress_TimeIncluded(t *testing.T) {
	c := DefaultCoordinateVO()
	c.Time = time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	addr := c.OntoAddress()

	assert.Contains(t, addr, "+2026-08-04T12:00:00Z")
}

func TestOntoAddress_ZeroTimeOmitted(t *testing.T) {
	c := DefaultCoordinateVO()
	addr := c.OntoAddress()

	assert.NotContains(t, addr, "+")
}

// --- ShortOntoAddress() ---

func TestShortOntoAddress(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*CoordinateVO)
		wantExact   string // non-empty → assert exact equality
		mustContain []string
		mustAbsent  []string
	}{
		{
			name:      "defaults omitted — city and location only",
			modify:    func(*CoordinateVO) {},
			wantExact: "onto://Leeds/Home",
		},
		{
			name:        "non-default quantum included",
			modify:      func(c *CoordinateVO) { c.Quantum = "Q2" },
			mustContain: []string{"Q2", "Leeds"},
		},
		{
			name:        "non-default timeline included",
			modify:      func(c *CoordinateVO) { c.Timeline = "T3" },
			mustContain: []string{"T3"},
		},
		{
			name:        "non-default observer included",
			modify:      func(c *CoordinateVO) { c.Observer = "Machine" },
			mustContain: []string{"@Machine"},
		},
		{
			name:        "non-default planet included, default Earth omitted",
			modify:      func(c *CoordinateVO) { c.Planet = "Mars" },
			mustContain: []string{"Mars"},
			mustAbsent:  []string{"Earth"},
		},
		{
			name:        "spaces encoded as underscore",
			modify:      func(c *CoordinateVO) { c.City = "New York" },
			mustContain: []string{"New_York"},
			mustAbsent:  []string{"New York"},
		},
		{
			name:        "non-default simulation and consensus included",
			modify:      func(c *CoordinateVO) { c.Simulation = 1; c.Consensus = 2 },
			mustContain: []string{"sim:1", "cons:2"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := DefaultCoordinateVO()
			tc.modify(&c)
			addr := c.ShortOntoAddress()

			if tc.wantExact != "" {
				assert.Equal(t, tc.wantExact, addr)
			}
			for _, s := range tc.mustContain {
				assert.Contains(t, addr, s)
			}
			for _, s := range tc.mustAbsent {
				assert.NotContains(t, addr, s)
			}
		})
	}
}

// --- ParseOntoAddress() ---

func TestParseOntoAddress_RoundTrip_FullAddress(t *testing.T) {
	original := DefaultCoordinateVO()
	original.Simulation = 2
	original.Consensus = 1
	original.Time = time.Date(2026, 8, 4, 9, 0, 0, 0, time.UTC)

	addr := original.OntoAddress()
	parsed, err := ParseOntoAddress(addr)

	require.NoError(t, err)
	assert.Equal(t, original.Meta, parsed.Meta)
	assert.Equal(t, original.Mathematics, parsed.Mathematics)
	assert.Equal(t, original.Universe, parsed.Universe)
	assert.Equal(t, original.Timeline, parsed.Timeline)
	assert.Equal(t, original.Quantum, parsed.Quantum)
	assert.Equal(t, original.Simulation, parsed.Simulation)
	assert.Equal(t, original.Consensus, parsed.Consensus)
	assert.Equal(t, original.Galaxy, parsed.Galaxy)
	assert.Equal(t, original.System, parsed.System)
	assert.Equal(t, original.Planet, parsed.Planet)
	assert.Equal(t, original.Country, parsed.Country)
	assert.Equal(t, original.Region, parsed.Region)
	assert.Equal(t, original.City, parsed.City)
	assert.Equal(t, original.Location, parsed.Location)
	assert.Equal(t, original.Observer, parsed.Observer)
	assert.Equal(t, original.Time.UTC(), parsed.Time.UTC())
}

func TestParseOntoAddress_SpacesDecodedFromUnderscore(t *testing.T) {
	c := DefaultCoordinateVO()
	addr := c.OntoAddress()
	parsed, err := ParseOntoAddress(addr)

	require.NoError(t, err)
	assert.Equal(t, "Milky Way", parsed.Galaxy)
	assert.Equal(t, "Solar System", parsed.System)
	assert.Equal(t, "United Kingdom", parsed.Country)
}

func TestParseOntoAddress_SchemeOptional(t *testing.T) {
	c := DefaultCoordinateVO()
	addr := c.OntoAddress()

	// Should parse with or without the onto:// prefix.
	parsed, err := ParseOntoAddress(addr)
	require.NoError(t, err)
	assert.Equal(t, c.Planet, parsed.Planet)
}
