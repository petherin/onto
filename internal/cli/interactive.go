package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/petherin/onto/internal/reality"
)

// interactiveEnsureOutgoing prompts the user to auto-generate or create a
// custom outgoing location. Returns true when a new location was created.
func (a *App) interactiveEnsureOutgoing(id string) bool {
	if a.interactiveReader == nil {
		return false
	}
	fmt.Printf("No outgoing journeys from %s.\n", a.displayName(id))
	fmt.Println("Options: (a)uto-generate, (s)kip, (c)reate custom")
	fmt.Print("Choose [a/s/c]: ")
	line, _ := a.interactiveReader.ReadString('\n')
	choice := strings.TrimSpace(strings.ToLower(line))
	if choice == "s" || choice == "skip" {
		return false
	}
	if choice == "a" || choice == "auto" {
		// delegate to generator
		created := autoGenerateNearby(a, id)
		if created {
			// announce created node
			for _, edge := range a.universe.Edges[id] {
				fmt.Printf("Auto-generated: %s (%s)\n", a.displayName(edge.To), edge.To)
				break
			}
		}
		return created
	}
	if choice == "c" || choice == "create" {
		fmt.Print("Enter ID (short, lowercase, hyphen allowed) [suggested will be used if empty]: ")
		idLine, _ := a.interactiveReader.ReadString('\n')
		idLine = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(idLine), " ", "-"))
		suggested := ""
		for i := 1; i < 1000; i++ {
			candidate := fmt.Sprintf("%s-%d", id, i)
			if _, ok := a.universe.GetLocation(candidate); !ok {
				suggested = candidate
				break
			}
		}
		if idLine == "" {
			idLine = suggested
			fmt.Printf("Using suggested id: %s\n", idLine)
		}
		if _, exists := a.universe.GetLocation(idLine); exists {
			fmt.Printf("ID %s already exists, aborting.\n", idLine)
			return false
		}
		fmt.Print("Enter display name: ")
		nameLine, _ := a.interactiveReader.ReadString('\n')
		nameLine = strings.TrimSpace(nameLine)
		if nameLine == "" {
			nameLine = idLine
		}
		fmt.Print("Enter description: ")
		descLine, _ := a.interactiveReader.ReadString('\n')
		descLine = strings.TrimSpace(descLine)
		fmt.Print("Enter distance (km) [default 1.0]: ")
		distLine, _ := a.interactiveReader.ReadString('\n')
		distLine = strings.TrimSpace(distLine)
		distance := 1.0
		if distLine != "" {
			if v, err := strconv.ParseFloat(distLine, 64); err == nil {
				distance = v
			}
		}
		fmt.Print("Enter cost (numeric) [default 1]: ")
		costLine, _ := a.interactiveReader.ReadString('\n')
		costLine = strings.TrimSpace(costLine)
		cost := 1.0
		if costLine != "" {
			if v, err := strconv.ParseFloat(costLine, 64); err == nil {
				cost = v
			}
		}
		coord := a.currentCoordinate
		coord.Location = nameLine
		loc := reality.Location{ID: idLine, Name: nameLine, Description: descLine, Coordinate: coord}
		a.universe.AddLocation(loc)
		edge := reality.Edge{From: id, To: idLine, Mode: reality.Walk, Distance: distance, Cost: cost, Description: "User-created path"}
		a.universe.AddEdge(edge)
		reverse := reality.Edge{From: idLine, To: id, Mode: reality.Walk, Distance: distance, Cost: cost, Description: "User-created return path"}
		a.universe.AddEdge(reverse)
		fmt.Printf("Created: %s (%s)\n", nameLine, idLine)
		return true
	}
	return false
}
