package cli

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/infrastructure/generator"
)

// InteractiveHandler implements universe.LocationGeneratorService by prompting
// the user to auto-generate, skip, or manually create a new outgoing location.
type InteractiveHandler struct {
	reader *bufio.Reader
	gen    *generator.NearbyGenerator
}

// Handle prompts the user to auto-generate, skip, or manually create a new
// outgoing location when a dead end is reached.
func (h *InteractiveHandler) Handle(u *universe.Aggregate, id string, coord universe.CoordinateVO) bool {
	name := locationDisplayName(u, id)
	fmt.Printf("No outgoing journeys from %s.\n", name)
	fmt.Println("Options: (a)uto-generate, (s)kip, (c)reate custom")
	fmt.Print("Choose [a/s/c]: ")

	line, _ := h.reader.ReadString('\n')
	choice := strings.TrimSpace(strings.ToLower(line))

	switch choice {
	case "s", "skip":
		return false
	case "a", "auto":
		created := h.gen.Handle(u, id, coord)
		if created {
			for _, edge := range u.EdgesFrom(id) {
				fmt.Printf("Auto-generated: %s (%s)\n", locationDisplayName(u, edge.To), edge.To)
				break
			}
		}
		return created
	case "c", "create":
		return h.createCustomLocation(u, id, coord)
	}
	return false
}

func (h *InteractiveHandler) createCustomLocation(u *universe.Aggregate, id string, coord universe.CoordinateVO) bool {
	suggested := ""
	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d", id, i)
		if _, ok := u.GetLocation(candidate); !ok {
			suggested = candidate
			break
		}
	}

	fmt.Print("Enter ID (short, lowercase, hyphen allowed) [suggested will be used if empty]: ")
	idLine, _ := h.reader.ReadString('\n')
	idLine = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(idLine), " ", "-"))
	if idLine == "" {
		idLine = suggested
		fmt.Printf("Using suggested id: %s\n", idLine)
	}
	if _, exists := u.GetLocation(idLine); exists {
		fmt.Printf("ID %s already exists, aborting.\n", idLine)
		return false
	}

	fmt.Print("Enter display name: ")
	nameLine, _ := h.reader.ReadString('\n')
	nameLine = strings.TrimSpace(nameLine)
	if nameLine == "" {
		nameLine = idLine
	}

	fmt.Print("Enter description: ")
	descLine, _ := h.reader.ReadString('\n')
	descLine = strings.TrimSpace(descLine)

	fmt.Print("Enter distance (km) [default 1.0]: ")
	distLine, _ := h.reader.ReadString('\n')
	distance := 1.0
	if v, err := strconv.ParseFloat(strings.TrimSpace(distLine), 64); err == nil {
		distance = v
	}

	fmt.Print("Enter cost (numeric) [default 1]: ")
	costLine, _ := h.reader.ReadString('\n')
	cost := 1.0
	if v, err := strconv.ParseFloat(strings.TrimSpace(costLine), 64); err == nil {
		cost = v
	}

	c := coord
	c.Location = nameLine
	u.AddLocation(universe.LocationEntity{ID: idLine, Name: nameLine, Description: descLine, Coordinate: c})
	u.AddEdge(universe.EdgeVO{From: id, To: idLine, Mode: universe.Walk, Distance: distance, Cost: cost, Description: "User-created path"})
	u.AddEdge(universe.EdgeVO{From: idLine, To: id, Mode: universe.Walk, Distance: distance, Cost: cost, Description: "User-created return path"})
	fmt.Printf("Created: %s (%s)\n", nameLine, idLine)
	return true
}

func locationDisplayName(u *universe.Aggregate, id string) string {
	if loc, ok := u.GetLocation(id); ok && loc.Name != "" {
		return loc.Name
	}
	return id
}
