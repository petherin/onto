// validate_locations validates a persisted Onto universe graph.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/petherin/onto/internal/domain/universe"
)

type persistedUniverse struct {
	Locations []universe.LocationEntity `json:"locations"`
	Edges     []universe.EdgeVO         `json:"edges"`
}

func main() {
	path := flag.String("file", "data/locations.json", "path to the locations JSON file")
	flag.Parse()

	file, err := os.Open(*path)
	if err != nil {
		if os.IsNotExist(err) {
			// No runtime save file yet is expected/normal — the app falls
			// back to its in-code starter universe until the first save.
			fmt.Printf("%s does not exist yet — nothing to validate (the app will use its built-in starter universe).\n", *path)
			return
		}
		fail("open %s: %v", *path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fail("close %s: %v", *path, err)
		}
	}()

	var saved persistedUniverse
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&saved); err != nil {
		fail("decode %s: %v", *path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		fail("%s contains trailing JSON data", *path)
	}

	locations := make(map[string]universe.LocationEntity, len(saved.Locations))
	var problems []string
	for _, location := range saved.Locations {
		if strings.TrimSpace(location.ID) == "" {
			problems = append(problems, "location has an empty ID")
			continue
		}
		if _, exists := locations[location.ID]; exists {
			problems = append(problems, fmt.Sprintf("duplicate location ID %q", location.ID))
			continue
		}
		locations[location.ID] = location
	}

	for _, location := range saved.Locations {
		problems = append(problems, checkIDMatchesCoordinate(location)...)
	}

	for _, edge := range saved.Edges {
		from, fromExists := locations[edge.From]
		to, toExists := locations[edge.To]
		switch {
		case !fromExists:
			problems = append(problems, fmt.Sprintf("edge source %q does not exist", edge.From))
		case !toExists:
			problems = append(problems, fmt.Sprintf("edge destination %q does not exist", edge.To))
		case edge.Mode.IsPhysical() && !from.Coordinate.SamePhysicalReality(to.Coordinate):
			problems = append(problems, fmt.Sprintf("physical edge %q -> %q crosses a reality boundary", edge.From, edge.To))
		}
	}

	if len(problems) > 0 {
		for _, problem := range problems {
			fmt.Fprintln(os.Stderr, problem)
		}
		os.Exit(1)
	}

	fmt.Printf("%s is valid: %d locations, %d edges\n", *path, len(saved.Locations), len(saved.Edges))
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// checkIDMatchesCoordinate catches the class of corruption this validator
// previously missed: a location's own coordinate fields disagreeing with
// what its ID suffix claims (e.g. an ID with no "-c" segment but a non-zero
// Consensus, or an ID claiming "-q1" while Coordinate.Quantum is still "Q0").
// This is exactly the bug that produced the "physical edge crosses reality
// boundary" data corruption fixed in this repo's history.
//
// Auto-generated nearby/dead-end locations (see internal/domain/universe/nearby.go)
// are intentionally skipped: their IDs use plain sequential numbering (e.g.
// "home-1") rather than reality-branch suffixes, and NewNearbyLocation marks
// them with a generic Coordinate.Location of "Nearby N" — that's the signal
// used here to exclude them from this check.
func checkIDMatchesCoordinate(location universe.LocationEntity) []string {
	if strings.HasPrefix(location.Coordinate.Location, "Nearby ") {
		return nil
	}

	_, idMathematics, idUniverse, idQuantum, idTimeline, idConsensus, idSimulation, idTime, idObserver := universe.ParseLocationID(location.ID)

	var problems []string
	if idMathematics != location.Coordinate.MathematicsLevel() {
		problems = append(problems, fmt.Sprintf(
			"location %q: ID suggests mathematics level %d but Coordinate.Mathematics is %q (level %d)",
			location.ID, idMathematics, location.Coordinate.Mathematics, location.Coordinate.MathematicsLevel()))
	}
	if idUniverse != location.Coordinate.UniverseLevel() {
		problems = append(problems, fmt.Sprintf(
			"location %q: ID suggests universe level %d but Coordinate.Universe is %q (level %d)",
			location.ID, idUniverse, location.Coordinate.Universe, location.Coordinate.UniverseLevel()))
	}
	if idQuantum != location.Coordinate.QuantumLevel() {
		problems = append(problems, fmt.Sprintf(
			"location %q: ID suggests quantum level %d but Coordinate.Quantum is %q (level %d)",
			location.ID, idQuantum, location.Coordinate.Quantum, location.Coordinate.QuantumLevel()))
	}
	if idTimeline != location.Coordinate.TimelineLevel() {
		problems = append(problems, fmt.Sprintf(
			"location %q: ID suggests timeline level %d but Coordinate.Timeline is %q (level %d)",
			location.ID, idTimeline, location.Coordinate.Timeline, location.Coordinate.TimelineLevel()))
	}
	if idConsensus != location.Coordinate.Consensus {
		problems = append(problems, fmt.Sprintf(
			"location %q: ID suggests consensus level %d but Coordinate.Consensus is %d",
			location.ID, idConsensus, location.Coordinate.Consensus))
	}
	if idSimulation != location.Coordinate.Simulation {
		problems = append(problems, fmt.Sprintf(
			"location %q: ID suggests simulation depth %d but Coordinate.Simulation is %d",
			location.ID, idSimulation, location.Coordinate.Simulation))
	}
	if (idTime != "") != !location.Coordinate.Time.IsZero() {
		problems = append(problems, fmt.Sprintf(
			"location %q: ID time suffix presence (%v) does not match whether Coordinate.Time is set (%v)",
			location.ID, idTime != "", !location.Coordinate.Time.IsZero()))
	}
	if (idObserver != "") != (location.Coordinate.Observer != "" && location.Coordinate.Observer != universe.DefaultCoordinateVO().Observer) {
		problems = append(problems, fmt.Sprintf(
			"location %q: ID observer suffix presence (%v) does not match whether Coordinate.Observer (%q) differs from the default observer",
			location.ID, idObserver != "", location.Coordinate.Observer))
	}
	return problems
}
