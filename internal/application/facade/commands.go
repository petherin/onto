package facade

import (
	"errors"
	"fmt"
	"strings"

	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/application/queries"
	"github.com/petherin/onto/internal/domain/universe"
)

// afford gates a move on the session's budget. When the move is affordable it
// returns ("", true) and the caller proceeds; otherwise it returns a rejection
// message and false, and the caller must return that message without moving.
// With no budget in force every move is affordable.
func (a *App) afford(label string, cost float64) (string, bool) {
	if a.session.CanAfford(cost) {
		return "", true
	}
	return fmt.Sprintf("Not enough budget to %s — it costs %.0f but only %.0f remains.",
		label, cost, a.session.RemainingBudget()), false
}

// Travel attempts physical movement to target.
func (a *App) Travel(target string) string {
	cmd := &commands.TravelCommand{
		Universe:   a.univ,
		Session:    a.session,
		Pathfinder: a.pathfinder,
	}
	result, err := cmd.Execute(target)
	if result == nil {
		if errors.Is(err, commands.ErrInsufficientBudget) {
			return err.Error()
		}
		norm := normalise(target)
		if _, known := a.univ.GetLocation(norm); !known {
			if norm == a.session.NextQuantumID() {
				return fmt.Sprintf("%s is a quantum branch — use 'shift' to enter it", target)
			}
			if norm == a.session.NextTimelineID() {
				return fmt.Sprintf("%s is a timeline branch — use 'jump' to enter it", target)
			}
			if norm == a.session.NextUniverseID() {
				return fmt.Sprintf("%s is a bubble universe — use 'universe' to enter it", target)
			}
			if norm == a.session.NextMathematicsID() {
				return fmt.Sprintf("%s is a mathematical structure — use 'structure' to enter it", target)
			}
			if norm == a.session.NextSimulationID() {
				return fmt.Sprintf("%s is a nested simulation — use 'simulate' to enter it", target)
			}
			if suggestion := a.suggestDestination(target); suggestion != "" {
				return fmt.Sprintf("Unknown destination: %s\n\nDid you mean '%s'?", target, suggestion)
			}
			return a.routeUnavailableDiagnostics(target)
		}
		return err.Error()
	}
	output := a.formatTravelResult(result)
	if !result.DeadEndHandled {
		return output
	}
	// A genuine physical sink (no outgoing physical edge at all, e.g. the well)
	// must stay a dead end: expanding it would hand the traveller a walkable way
	// out and defeat the point. Ordinary leaves and nearby chains keep a physical
	// edge (at least back the way they came), so they still auto-expand here.
	if !universe.HasPhysicalExit(a.univ, result.Location.ID) {
		return output
	}
	location, err := (&commands.GenerateNearbyLocationCommand{
		Universe:  a.univ,
		Generator: a.locationGenerator,
		OriginID:  result.Location.ID,
	}).Execute()
	if err != nil {
		return fmt.Sprintf("%s\n\nUnable to generate a nearby location: %v", output, err)
	}
	a.markDirty()
	return fmt.Sprintf("%s\n\nAuto-generated: %s (%s)", output, location.Name, location.ID)
}

// Shift advances the session to the next quantum branch of the current location.
func (a *App) Shift() string {
	if msg, ok := a.afford("shift", universe.QuantumShiftCost); !ok {
		return msg
	}
	cmd := &commands.ShiftCommand{Universe: a.univ, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Shift failed: %v", err)
	}
	a.markDirty()
	return a.maybeGenerateEscape(result.Location, universe.QuantumShiftCost, a.formatShiftResult(result))
}

// ShiftBack returns the session to the previous quantum branch.
func (a *App) ShiftBack() string {
	if msg, ok := a.afford("shift back", universe.QuantumShiftCost); !ok {
		return msg
	}
	cmd := &commands.ShiftCommand{Universe: a.univ, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot shift back: %v", err)
	}
	a.markDirty()
	return a.formatShiftResult(result)
}

// Jump advances the session to the next timeline branch of the current location.
func (a *App) Jump() string {
	if msg, ok := a.afford("jump", universe.TimelineShiftCost); !ok {
		return msg
	}
	cmd := &commands.JumpCommand{Universe: a.univ, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Jump failed: %v", err)
	}
	a.markDirty()
	return a.maybeGenerateEscape(result.Location, universe.TimelineShiftCost, a.formatJumpResult(result))
}

// JumpBack returns the session to the previous timeline branch.
func (a *App) JumpBack() string {
	if msg, ok := a.afford("jump back", universe.TimelineShiftCost); !ok {
		return msg
	}
	cmd := &commands.JumpCommand{Universe: a.univ, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot jump back: %v", err)
	}
	a.markDirty()
	return a.formatJumpResult(result)
}

// Universe shifts the session to the next bubble universe of the current location.
func (a *App) Universe() string {
	if msg, ok := a.afford("shift universe", universe.UniverseShiftCost); !ok {
		return msg
	}
	cmd := &commands.UniverseCommand{Universe: a.univ, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Universe shift failed: %v", err)
	}
	a.markDirty()
	return a.maybeGenerateEscape(result.Location, universe.UniverseShiftCost, a.formatUniverseResult(result))
}

// UniverseBack returns the session to the previous bubble universe.
func (a *App) UniverseBack() string {
	if msg, ok := a.afford("shift universe back", universe.UniverseShiftCost); !ok {
		return msg
	}
	cmd := &commands.UniverseCommand{Universe: a.univ, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot return to previous universe: %v", err)
	}
	a.markDirty()
	return a.formatUniverseResult(result)
}

// Structure shifts the session to the next mathematical structure of the current location.
func (a *App) Structure() string {
	if msg, ok := a.afford("shift structure", universe.MathematicalShiftCost); !ok {
		return msg
	}
	cmd := &commands.StructureCommand{Universe: a.univ, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Mathematical structure shift failed: %v", err)
	}
	a.markDirty()
	return a.maybeGenerateEscape(result.Location, universe.MathematicalShiftCost, a.formatStructureResult(result))
}

// StructureBack returns the session to the previous mathematical structure.
func (a *App) StructureBack() string {
	if msg, ok := a.afford("shift structure back", universe.MathematicalShiftCost); !ok {
		return msg
	}
	cmd := &commands.StructureCommand{Universe: a.univ, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot return to previous mathematical structure: %v", err)
	}
	a.markDirty()
	return a.formatStructureResult(result)
}

// Simulate enters the next nested simulation layer of the current location.
func (a *App) Simulate() string {
	if msg, ok := a.afford("enter a simulation", universe.SimulationEntryCost); !ok {
		return msg
	}
	cmd := &commands.SimulateCommand{Universe: a.univ, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Simulation entry failed: %v", err)
	}
	a.markDirty()
	return a.maybeGenerateEscape(result.Location, universe.SimulationEntryCost, a.formatSimulateResult(result))
}

// SimulateBack exits one simulation layer toward base reality.
func (a *App) SimulateBack() string {
	if msg, ok := a.afford("exit the simulation", universe.SimulationExitCost); !ok {
		return msg
	}
	cmd := &commands.SimulateCommand{Universe: a.univ, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot exit simulation: %v", err)
	}
	a.markDirty()
	return a.formatSimulateResult(result)
}

// Drift enters the next consensus divergence from the current location.
func (a *App) Drift() string {
	if msg, ok := a.afford("drift", universe.ConsensusShiftCost); !ok {
		return msg
	}
	cmd := &commands.DriftCommand{Universe: a.univ, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Drift failed: %v", err)
	}
	a.markDirty()
	return a.maybeGenerateEscape(result.Location, universe.ConsensusShiftCost, a.formatDriftResult(result))
}

// Align returns the session one level toward shared consensus.
func (a *App) Align() string {
	if msg, ok := a.afford("align", universe.ConsensusShiftCost); !ok {
		return msg
	}
	cmd := &commands.DriftCommand{Universe: a.univ, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot align: %v", err)
	}
	a.markDirty()
	return a.formatDriftResult(result)
}

// Observe shifts the session to the given observer perspective.
func (a *App) Observe(observer string) string {
	if msg, ok := a.afford("change observer", universe.ObserverShiftCost); !ok {
		return msg
	}
	cmd := &commands.ObserveCommand{Universe: a.univ, Session: a.session, Observer: observer}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Observer shift failed: %v", err)
	}
	a.markDirty()
	return a.maybeGenerateEscape(result.Location, universe.ObserverShiftCost, a.formatObserveResult(result))
}

// ObserveBack restores the previous observer perspective.
func (a *App) ObserveBack() string {
	if msg, ok := a.afford("restore observer", universe.ObserverShiftCost); !ok {
		return msg
	}
	cmd := &commands.ObserveCommand{Universe: a.univ, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot return observer perspective: %v", err)
	}
	a.markDirty()
	return a.formatObserveResult(result)
}

// Time enters a temporal branch at the given RFC3339 timestamp.
func (a *App) Time(target string) string {
	if msg, ok := a.afford("time shift", universe.TimeShiftCost); !ok {
		return msg
	}
	result, err := (&commands.TimeCommand{Universe: a.univ, Session: a.session, Target: target}).Execute()
	if result == nil {
		return fmt.Sprintf("Time shift failed: %v", err)
	}
	a.markDirty()
	return a.maybeGenerateEscape(result.Location, universe.TimeShiftCost, a.formatTimeResult(result))
}

// TimeBack returns the session through the temporal branch.
func (a *App) TimeBack() string {
	if msg, ok := a.afford("time shift back", universe.TimeShiftCost); !ok {
		return msg
	}
	result, err := (&commands.TimeCommand{Universe: a.univ, Session: a.session, Back: true}).Execute()
	if result == nil {
		return fmt.Sprintf("Cannot return through time: %v", err)
	}
	a.markDirty()
	return a.formatTimeResult(result)
}

// maybeGenerateEscape expands a physical dead end reached by a non-physical
// move (e.g. drifting the well into another reality). Ordinary travel already
// spawns a nearby node whenever it lands on a dead end; a non-physical move does
// not, so without this a mirrored dead end like the well stays a physical sink
// in every reality. Here escapability is gated per reality: HasPhysicalEscape
// rolls a deterministic, coordinate-seeded verdict whose odds scale with
// transitionCost — the σ the arriving move cost — so cheap transitions rarely
// yield a physical way out (a nearby "ladder") while expensive ones usually do.
// The traveller gambles more σ for better odds, can keep trying other realities,
// and is never hard-locked because a non-physical exit always remains. The
// base-reality case is never gated, so existing behaviour there is unchanged. It
// returns the output, augmented with the outcome when the landed location is a
// dead end.
func (a *App) maybeGenerateEscape(landed universe.LocationEntity, transitionCost float64, output string) string {
	if !universe.IsPhysicalDeadEnd(a.univ, landed.ID, "") {
		return output
	}
	if !universe.HasPhysicalEscape(landed.Coordinate, transitionCost) {
		return fmt.Sprintf("%s\n\nNo way out: this reality offers no physical route from %s. Try another reality.",
			output, a.locationName(landed.ID))
	}
	location, err := (&commands.GenerateNearbyLocationCommand{
		Universe:  a.univ,
		Generator: a.locationGenerator,
		OriginID:  landed.ID,
	}).Execute()
	if err != nil {
		return fmt.Sprintf("%s\n\nUnable to generate a nearby location: %v", output, err)
	}
	a.markDirty()
	return fmt.Sprintf("%s\n\nA way out: %s (%s)", output, location.Name, location.ID)
}

func (a *App) markDirty() { a.dirty = true }

// SaveIfDirty persists the universe if dirty, then clears the flag.
func (a *App) SaveIfDirty() error {
	if !a.dirty {
		return nil
	}
	if err := a.repo.Save(a.univ); err != nil {
		return err
	}
	a.dirty = false
	return nil
}

// Save unconditionally persists and returns a confirmation message.
func (a *App) Save() (string, error) {
	if err := a.repo.Save(a.univ); err != nil {
		return "", err
	}
	a.dirty = false
	return "Saved.", nil
}

func normalise(target string) string {
	return strings.ToLower(strings.ReplaceAll(target, " ", "-"))
}

// Route plans a route without moving the session.
func (a *App) Route(target string) string {
	q := &queries.RouteQuery{Universe: a.univ, Session: a.session, Pathfinder: a.pathfinder}
	result, err := q.Execute(target)
	if err != nil {
		if suggestion := a.suggestDestination(target); suggestion != "" {
			return fmt.Sprintf("Unknown destination: %s\n\nDid you mean '%s'?", target, suggestion)
		}
		return fmt.Sprintf("Route unavailable to %s.", target)
	}
	return a.formatRouteResult(result)
}

// Cost returns the session's total running travel cost.
func (a *App) Cost() string {
	return fmt.Sprintf("Total journey cost: %.0f", a.session.CumulativeCost())
}

const (
	// MsgAlreadyHome is returned by GoHome when the session is already at the
	// start location with no context to unwind.
	MsgAlreadyHome = "You are already home."
	// HomeConfirmPrompt terminates every GoHome plan that requires the user to
	// confirm before GoHomeConfirm executes it. Delivery mechanisms detect it
	// via NeedsHomeConfirm rather than string-comparing against MsgAlreadyHome.
	HomeConfirmPrompt = "Proceed? [y/N]:"
)

// NeedsHomeConfirm reports whether a GoHome result is an actionable plan that
// requires a y/N confirmation, as opposed to a terminal message such as
// "already home" or "no route home". Delivery mechanisms use this to decide
// whether to enter their confirm/execute flow.
func NeedsHomeConfirm(plan string) bool {
	return strings.HasSuffix(plan, HomeConfirmPrompt)
}

// GoHome builds the route plan for returning home (no movement occurs).
func (a *App) GoHome() string {
	cmd := &commands.ReturnHomeCommand{
		Universe:   a.univ,
		Session:    a.session,
		Pathfinder: a.pathfinder,
		HomeID:     a.homeID,
	}
	steps, cost := cmd.Plan()
	if len(steps) == 0 {
		// An empty plan is ambiguous: either there is genuinely nothing to do
		// (already home), or the current location has no route back home (a
		// dead-end node). Distinguish the two so a stranded traveller isn't
		// falsely told they are already home.
		if a.session.Location() == a.homeID {
			return MsgAlreadyHome
		}
		return fmt.Sprintf("No route home from %s. There is no path back to %s from here.",
			a.locationName(a.session.Location()), a.locationName(a.homeID))
	}
	var lines []string
	for _, step := range steps {
		line := fmt.Sprintf("  %-10s", step.Action)
		if step.Detail != "" {
			line += " (" + step.Detail + ")"
		}
		if step.Cost != 0 {
			line += fmt.Sprintf("  cost %.0f", step.Cost)
		}
		lines = append(lines, line)
	}
	return fmt.Sprintf("Route home\n%s\n\nEstimated cost: %.0f\n\n%s", strings.Join(lines, "\n"), cost, HomeConfirmPrompt)
}

// GoHomeConfirm executes the home journey produced by GoHome.
func (a *App) GoHomeConfirm() string {
	doneBefore, reachedBefore, wonBefore := a.session.ObjectiveIndex(), a.session.ReachedTarget(), a.session.Won()
	cmd := &commands.ReturnHomeCommand{
		Universe:   a.univ,
		Session:    a.session,
		Pathfinder: a.pathfinder,
		HomeID:     a.homeID,
	}
	steps, err := cmd.Execute()
	if err != nil {
		return fmt.Sprintf("Failed while returning home: %v", err)
	}
	a.markDirty()
	var lines []string
	for _, step := range steps {
		switch step.Action {
		case "observe back":
			lines = append(lines, fmt.Sprintf("Observer return → %s", step.Detail))
		case "align":
			lines = append(lines, fmt.Sprintf("Consensus alignment → level %s", step.Detail))
		default:
			lines = append(lines, fmt.Sprintf("%s %s", step.Action, step.Detail))
		}
	}
	return fmt.Sprintf("Heading home...\n\nSteps taken\n%s\n\nYou are home.\n\nCumulative journey cost\n%.0f",
		strings.Join(lines, "\n"), a.session.CumulativeCost()) + a.goalBanner(doneBefore, reachedBefore, wonBefore)
}
