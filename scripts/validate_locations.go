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
