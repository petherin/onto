package facade

import (
	"fmt"
	"strings"
	"time"

	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/application/queries"
	"github.com/petherin/onto/internal/domain/universe"
)

// Where formats the full reality coordinate, possible journeys, and recent history.
func (a *App) Where() string {
	q := &queries.LookupQuery{Universe: a.univ, Session: a.session}
	r := q.Where()
	coord := r.Coordinate
	return fmt.Sprintf(
		"Reality Coordinate\n%s\n\nMathematics: %s\nUniverse : %s\nTimeline : %s\nQuantum  : %s\nSimulation: %d\nConsensus: %d\nPlanet   : %s\nCountry  : %s\nRegion   : %s\nCity     : %s\nLocation : %s\nObserver : %s\n\nCumulative journey cost\n%.0f%s\n\nPossible journeys\n%s",
		coord.OntoAddress(),
		coord.Mathematics, coord.Universe, coord.Timeline, coord.Quantum, coord.Simulation, coord.Consensus,
		coord.Planet, coord.Country, coord.Region, coord.City, coord.Location, coord.Observer,
		a.session.CumulativeCost(),
		a.objectiveStatus(),
		a.formatEdges(r.Edges),
	)
}

// objectiveStatus formats the budget and objective lines for status displays,
// each as its own labelled block. It returns an empty string when neither a
// budget nor a target is in force, so non-game sessions are unaffected.
func (a *App) objectiveStatus() string {
	s := a.session
	var b strings.Builder
	if s.HasBudget() {
		fmt.Fprintf(&b, "\n\nBudget remaining\n%.0f of %.0f", s.RemainingBudget(), s.Budget())
		if s.RemainingBudget() <= 0 {
			fmt.Fprintf(&b, " — %s", BudgetExhaustedMarker)
		}
	}
	if s.HasTarget() {
		par := a.objectivePar()
		done := s.ObjectiveIndex()
		targets := s.Targets()
		fmt.Fprintf(&b, "\n\nObjective (%d of %d complete)", done, len(targets))
		for i, t := range targets {
			mark := " "
			switch {
			case i < done:
				mark = "x"
			case i == done && s.ReachedTarget():
				mark = "~"
			}
			fmt.Fprintf(&b, "\n  [%s] %d. Reach %s", mark, i+1, t.ShortOntoAddress())
		}
		switch {
		case s.Won():
			b.WriteString("\nComplete — every objective reached and returned home.")
		case s.ReachedTarget():
			b.WriteString("\nObjective reached — return home to complete it.")
		default:
			b.WriteString("\nReach the objective, then return home to complete it.")
		}
		fmt.Fprintf(&b, "\nPar %.0f", par)
		if s.Won() {
			fmt.Fprintf(&b, "\nRating %s (%.0f cost)", starBar(starsForCost(s.CumulativeCost(), par)), s.CumulativeCost())
		}
	}
	return b.String()
}

// Look formats the name and description of the current location.
func (a *App) Look() string {
	q := &queries.LookupQuery{Universe: a.univ, Session: a.session}
	result, ok := q.Look()
	if !ok {
		return "Current location is unknown."
	}
	return fmt.Sprintf("%s\n\n%s", result.Name, result.Description)
}

// List formats the outgoing edges from the current location.
func (a *App) List() string {
	q := &queries.LookupQuery{Universe: a.univ, Session: a.session}
	r := q.List()
	return a.formatEdges(r.Edges)
}

func (a *App) formatTravelResult(r *commands.TravelResult) string {
	return fmt.Sprintf("%s\n\nArrived.\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		travelVerb(r.Path), a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
}

func (a *App) formatShiftResult(r *commands.ShiftResult) string {
	verb := "Quantum branch entered"
	if r.Reversed {
		verb = "Quantum branch exited"
	}
	return fmt.Sprintf("Shifting...\n\n%s: %s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.NextQuantum, r.Location.Description, a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
}

func (a *App) formatJumpResult(r *commands.JumpResult) string {
	verb := "Arrived in a distant Hubble volume"
	if r.Reversed {
		verb = "Returned from the distant Hubble volume"
	}
	return fmt.Sprintf("Jumping (threading a wormhole to a farther Hubble volume)...\n\n%s: %s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.NextTimeline, r.Location.Description, a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
}

func (a *App) formatUniverseResult(r *commands.UniverseResult) string {
	verb := "Bubble universe entered"
	if r.Reversed {
		verb = "Bubble universe exited"
	}
	return fmt.Sprintf("Shifting universes...\n\n%s: %s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.NextUniverse, r.Location.Description, a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
}

func (a *App) formatStructureResult(r *commands.StructureResult) string {
	verb := "Mathematical structure entered"
	if r.Reversed {
		verb = "Mathematical structure exited"
	}
	return fmt.Sprintf("Crossing formal systems...\n\n%s: %s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.NextMathematics, r.Location.Description, a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
}

func (a *App) formatSimulateResult(r *commands.SimulateResult) string {
	verb := "Simulation layer entered"
	if r.Reversed {
		verb = "Simulation layer exited"
	}
	return fmt.Sprintf("Crossing the simulation boundary...\n\n%s: depth %d\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.Simulation, r.Location.Description, a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
}

func (a *App) formatDriftResult(r *commands.DriftResult) string {
	verb := "Consensus divergence entered"
	if r.Reversed {
		verb = "Shared consensus approached"
	}
	return fmt.Sprintf("Drifting...\n\n%s: level %d\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.Consensus, r.Location.Description, a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
}

func (a *App) formatObserveResult(r *commands.ObserveResult) string {
	verb := "Observer perspective entered"
	if r.Reversed {
		verb = "Observer perspective restored"
	}
	return fmt.Sprintf("Observing...\n\n%s: %s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.Observer, r.Location.Description, a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
}

func (a *App) formatTimeResult(r *commands.TimeResult) string {
	verb := "Temporal branch entered"
	if r.Reversed {
		verb = "Temporal branch exited"
	}
	return fmt.Sprintf("Time shifting...\n\n%s: %s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.Time.Format(time.RFC3339), r.Location.Description, a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
}

func (a *App) formatRouteResult(r *queries.RouteResult) string {
	var steps []string
	for _, edge := range r.Steps {
		steps = append(steps, fmt.Sprintf("%s (%s)", a.locationName(edge.To), string(edge.Mode)))
	}
	return fmt.Sprintf("Route\n%s\n\nDistance\n%.1f km\n\nTravel Cost\n%.0f",
		strings.Join(steps, "\n"), r.Distance, r.Cost)
}

// formatEdges renders the outgoing edges as a numbered journey list.
func (a *App) formatEdges(edges []universe.EdgeVO) string {
	options, observerShiftAvailable := a.JourneyOptions(edges)
	lines := make([]string, 0, len(options)+1)
	for i, option := range options {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, option.Description))
	}
	if observerShiftAvailable {
		lines = append(lines, fmt.Sprintf("- observer perspective (%.0f cost — observe <observer>)", universe.ObserverShiftCost))
		lines = append(lines, fmt.Sprintf("- temporal branch (%.0f cost — time <RFC3339>)", universe.TimeShiftCost))
	}
	return strings.Join(lines, "\n")
}

func (a *App) locationName(id string) string {
	if loc, ok := a.univ.GetLocation(id); ok && loc.Name != "" {
		return loc.Name
	}
	return id
}

func (a *App) routeUnavailableDiagnostics(target string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Route unavailable to %s.\n", target)
	fmt.Fprintf(&b, "Current location id: %s\n", a.session.Location())
	if _, ok := a.univ.GetLocation(a.homeID); ok {
		fmt.Fprintf(&b, "%s is present in universe\n", a.homeID)
	} else {
		fmt.Fprintf(&b, "%s is NOT present in universe\n", a.homeID)
	}
	b.WriteString("Outgoing from current location:\n")
	for _, e := range a.univ.EdgesFrom(a.session.Location()) {
		fmt.Fprintf(&b, "- %s (%s)\n", e.To, e.Mode)
	}
	b.WriteString("\nKnown location IDs:\n")
	for _, id := range a.univ.AllLocationIDs() {
		fmt.Fprintf(&b, "- %s\n", id)
	}
	return b.String()
}

func travelVerb(path []universe.EdgeVO) string {
	order := []universe.TravelModeVO{
		universe.Warp, universe.Orbit, universe.Flight,
		universe.Rail, universe.Drive, universe.Cycle, universe.Walk,
	}
	modes := make(map[universe.TravelModeVO]bool, len(path))
	for _, e := range path {
		modes[e.Mode] = true
	}
	for _, m := range order {
		if modes[m] {
			switch m {
			case universe.Walk:
				return "Walking..."
			case universe.Cycle:
				return "Cycling..."
			case universe.Drive:
				return "Driving..."
			case universe.Rail:
				return "Taking the train..."
			case universe.Flight:
				return "Flying..."
			case universe.Orbit:
				return "Entering orbit..."
			case universe.Warp:
				return "Warping..."
			}
		}
	}
	return "Travelling..."
}
