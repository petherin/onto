package universe

import (
	"fmt"
	"math/rand"
)

// TrapType is a value object identifying which trap archetype a location is, if
// any. Like the sink/dead-end distinction, a trap is structural — the type is
// carried here on the location, never inferred from its ID or name. Almost all
// traps are generated in nested realities, but the hand-placed well in base
// reality also carries one (TrapSealedVault). The empty value (NoTrap) marks an
// ordinary node.
type TrapType string

const (
	// NoTrap marks an ordinary (non-trap) location.
	NoTrap TrapType = ""
	// TrapSealedVault has no physical exit at all: you walk in but can only
	// leave by a non-physical drift, the escape gamble, or home. The hand-placed
	// well in base reality is the canonical instance of this archetype.
	TrapSealedVault TrapType = "sealed-vault"
	// TrapTarPit has physical exits, but each costs escalating σ, so walking
	// out is futile and a non-physical move is the real exit.
	TrapTarPit TrapType = "tar-pit"
	// TrapMobiusMaze has several physical exits, all but one silently looping
	// back to the hub; finding the true exit (or drifting out) is the puzzle.
	TrapMobiusMaze TrapType = "mobius-maze"
	// TrapOneWaySink drops you into a physical pocket with no walkable route
	// back to where you came from; only a non-physical drift or home leaves.
	TrapOneWaySink TrapType = "one-way-sink"
)

// IsKnown reports whether t is a supported trap type (NoTrap included).
func (t TrapType) IsKnown() bool {
	switch t {
	case NoTrap, TrapSealedVault, TrapTarPit, TrapMobiusMaze, TrapOneWaySink:
		return true
	}
	return false
}

// Label returns a human-readable name for the trap type.
func (t TrapType) Label() string {
	switch t {
	case NoTrap:
		return "ordinary place"
	case TrapSealedVault:
		return "sealed vault"
	case TrapTarPit:
		return "tar pit"
	case TrapMobiusMaze:
		return "möbius maze"
	case TrapOneWaySink:
		return "one-way sink"
	}
	return "unknown trap"
}

// Trap-generation tuning. The trap decision mirrors the cost-scaled escape
// gamble (HasPhysicalEscape): it is seeded from the destination coordinate so it
// is reproducible across reloads and in tests, yet varies reality-to-reality.
const (
	// TrapProbability is the per-dead-end chance (in a nested reality) that an
	// expansion spawns a trap instead of an ordinary cluster.
	TrapProbability = 0.08
	// trapSeedSalt keeps the trap decision on an independent deterministic
	// stream from the ordinary cluster's names/size, so a coordinate that rolls
	// no trap generates byte-identically to before.
	trapSeedSalt = 0x74726170736c7421 // "traps!" salt
	// tarPitExitCost is the base σ a tar-pit walk costs; it escalates per node.
	tarPitExitCost = 40.0
)

// edgeWiringTraps are the archetypes implemented as pure edge wiring (no
// movement-command changes). SelectTrap draws uniformly from this set.
var edgeWiringTraps = []TrapType{TrapSealedVault, TrapTarPit, TrapMobiusMaze, TrapOneWaySink}

// SelectTrap decides, deterministically from coord, whether a dead end expanding
// in this reality spawns a trap and, if so, which archetype. Base reality
// (nesting depth 0) is never trapped, matching the escape gamble — the starter
// world stays gentle and traps are confined to the nested realities the
// traveller chose to drift into.
func SelectTrap(coord CoordinateVO) (TrapType, bool) {
	if coord.NestingDepth() == 0 {
		return NoTrap, false
	}
	rng := rand.New(rand.NewSource(coordinateSeed(coord) ^ trapSeedSalt))
	if rng.Float64() >= TrapProbability {
		return NoTrap, false
	}
	return edgeWiringTraps[rng.Intn(len(edgeWiringTraps))], true
}

// GenerateTrap expands originID into the given trap archetype, returning the new
// locations and the edges wiring them. Like NewNearbyCluster it performs no
// persistence. Every archetype preserves the no-hard-lock invariant: home can
// always plan a route out, either by a physical walk (tar pit, möbius maze) or
// via FindRoute across a non-physical escape edge back toward the origin (sealed
// vault, one-way sink), exactly as the well's ConsensusShift drift works.
func GenerateTrap(u *Aggregate, originID string, coord CoordinateVO, trap TrapType) ([]LocationEntity, []EdgeVO, error) {
	if _, exists := u.GetLocation(originID); !exists {
		return nil, nil, fmt.Errorf("%w: %s", ErrUnknownEdgeEndpoint, originID)
	}
	rng := rand.New(rand.NewSource(coordinateSeed(coord)))
	used := usedNames(u)
	switch trap {
	case TrapSealedVault:
		return wireSealedVault(u, originID, coord, rng, used)
	case TrapTarPit:
		return wireTarPit(u, originID, coord, rng, used)
	case TrapMobiusMaze:
		return wireMobiusMaze(u, originID, coord, rng, used)
	case TrapOneWaySink:
		return wireOneWaySink(u, originID, coord, rng, used)
	}
	return nil, nil, fmt.Errorf("%w: unknown trap %q", ErrInvalidLocation, trap)
}

// trapNode builds a generated trap location numbered onto id, named name, with a
// coordinate anchored on name so its description reads as a distinct place.
func trapNode(id string, coord CoordinateVO, name string, trap TrapType) LocationEntity {
	c := coord
	c.Location = name
	return LocationEntity{
		ID:          id,
		Name:        name,
		Description: GenerateDescription(c),
		Coordinate:  c,
		Generated:   true,
		Trap:        trap,
	}
}

// escapeEdge is the non-physical way out of a sealed trap node: a ConsensusShift
// drift back to the origin, mirroring the well's seed exit. It is non-physical,
// so travel and physical pathfinding ignore it (the node stays a real dead end),
// but FindRoute — and therefore home — can traverse it, so the trap is never a
// hard-lock.
func escapeEdge(fromID, originID string) EdgeVO {
	return EdgeVO{From: fromID, To: originID, Mode: ConsensusShift, Cost: ConsensusShiftCost, Description: "Drift through a crack back toward the surface"}
}

// walkEdge builds a physical Walk edge with matching distance and cost.
func walkEdge(fromID, toID string, cost float64, desc string) EdgeVO {
	return EdgeVO{From: fromID, To: toID, Mode: Walk, Distance: cost, Cost: cost, Description: desc}
}

// wireSealedVault expands originID into a single sealed node: you walk in, but it
// has no outgoing physical edge, so HasPhysicalExit is false and travel refuses
// to auto-expand it. Its only seed exit is a non-physical drift back to the
// origin, so home (via FindRoute) always gets you out.
func wireSealedVault(u *Aggregate, originID string, coord CoordinateVO, rng *rand.Rand, used map[string]bool) ([]LocationEntity, []EdgeVO, error) {
	ids, err := allocateNearbyIDs(u, originID, 1)
	if err != nil {
		return nil, nil, err
	}
	name := generateNearbyName(rng, used)
	used[name] = true
	vault := trapNode(ids[0], coord, name, TrapSealedVault)
	edges := []EdgeVO{
		walkEdge(originID, vault.ID, 1, "Drop into the sealed vault"),
		escapeEdge(vault.ID, originID),
	}
	return []LocationEntity{vault}, edges, nil
}

// wireTarPit expands originID into a 1–3 node star like an ordinary cluster, but
// every walk edge costs escalating σ, so walking out is futile and a
// non-physical move is the real exit. Each node keeps a physical edge back to the
// origin, so HasPhysicalExit stays true (nodes still auto-expand) and home routes
// out by an ordinary — if expensive — physical walk.
func wireTarPit(u *Aggregate, originID string, coord CoordinateVO, rng *rand.Rand, used map[string]bool) ([]LocationEntity, []EdgeVO, error) {
	count := 1 + rng.Intn(3)
	ids, err := allocateNearbyIDs(u, originID, count)
	if err != nil {
		return nil, nil, err
	}
	locations := make([]LocationEntity, 0, count)
	var edges []EdgeVO
	for i, id := range ids {
		name := generateNearbyName(rng, used)
		used[name] = true
		locations = append(locations, trapNode(id, coord, name, TrapTarPit))
		cost := tarPitExitCost * float64(i+1)
		edges = append(edges,
			walkEdge(originID, id, cost, "Wade deeper into the tar"),
			walkEdge(id, originID, cost, "Drag yourself back through the tar"),
		)
	}
	return locations, edges, nil
}

// wireMobiusMaze expands originID into a hub plus two decoys. The hub's only
// non-looping exit is back to the origin (the true exit); both decoys, and the
// link between them, silently loop back to the hub. Every node keeps a physical
// exit that is not merely the way it was entered, so none is a dead end that
// auto-expands, and the hub's physical edge to the origin keeps home routable.
func wireMobiusMaze(u *Aggregate, originID string, coord CoordinateVO, rng *rand.Rand, used map[string]bool) ([]LocationEntity, []EdgeVO, error) {
	ids, err := allocateNearbyIDs(u, originID, 3)
	if err != nil {
		return nil, nil, err
	}
	names := make([]string, 3)
	for i := range names {
		names[i] = generateNearbyName(rng, used)
		used[names[i]] = true
	}
	hub := trapNode(ids[0], coord, names[0], TrapMobiusMaze)
	left := trapNode(ids[1], coord, names[1], TrapMobiusMaze)
	right := trapNode(ids[2], coord, names[2], TrapMobiusMaze)
	edges := []EdgeVO{
		walkEdge(originID, hub.ID, 1, "Step into the maze"),
		walkEdge(hub.ID, originID, 1, "The one passage that truly leaves"),
		walkEdge(hub.ID, left.ID, 1, "A passage that seems to lead out"),
		walkEdge(left.ID, hub.ID, 1, "The passage curves back"),
		walkEdge(hub.ID, right.ID, 1, "Another promising passage"),
		walkEdge(right.ID, hub.ID, 1, "This one curves back too"),
		walkEdge(left.ID, right.ID, 1, "A connecting passage"),
		walkEdge(right.ID, left.ID, 1, "A connecting passage"),
	}
	return []LocationEntity{hub, left, right}, edges, nil
}

// wireOneWaySink drops you through a one-way trapdoor into a two-node physical
// pocket: you can wander between the pair but there is no walkable route back to
// the origin. The entry node keeps a non-physical drift back to the origin, so
// home (via FindRoute) always leaves even though walking never does.
func wireOneWaySink(u *Aggregate, originID string, coord CoordinateVO, rng *rand.Rand, used map[string]bool) ([]LocationEntity, []EdgeVO, error) {
	ids, err := allocateNearbyIDs(u, originID, 2)
	if err != nil {
		return nil, nil, err
	}
	names := make([]string, 2)
	for i := range names {
		names[i] = generateNearbyName(rng, used)
		used[names[i]] = true
	}
	mouth := trapNode(ids[0], coord, names[0], TrapOneWaySink)
	pit := trapNode(ids[1], coord, names[1], TrapOneWaySink)
	edges := []EdgeVO{
		walkEdge(originID, mouth.ID, 1, "Fall through the trapdoor"),
		walkEdge(mouth.ID, pit.ID, 1, "Wander deeper into the pocket"),
		walkEdge(pit.ID, mouth.ID, 1, "Wander back toward the trapdoor"),
		escapeEdge(mouth.ID, originID),
	}
	return []LocationEntity{mouth, pit}, edges, nil
}
