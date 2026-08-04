# Onto

Onto is an experimental CLI for navigating reality as a coordinate system.

## Table of contents

- [Vision](#vision)
- [Core idea](#core-idea)
- [Coordinate model](#coordinate-model)
- [Example CLI experience](#example-cli-experience)
- [Current status](#current-status)
- [Architecture](#architecture)
- [Getting started](#getting-started)
- [Roadmap](#roadmap)
- [Notes](#notes)
- [Why the name "Onto"](#why-the-name-onto)
- [Design notes](#design-notes)


The long-term idea is simple: whether you are walking to a station, shifting to a different timeline, entering a simulation, or moving between imagined worlds, the interface should feel like the same operation: routing through a graph of possible locations.

This project begins small, but it is designed to grow into a reality navigator rather than a traditional filesystem or command-line tool.

## Vision

Instead of treating reality as a single fixed map, Onto models existence as a coordinate system with two distinct kinds of axis: **which reality you are in**, and **how you exist within it**.

---

### Part 1 — Which reality? (Tegmark levels)

The Tegmark hierarchy classifies realities by what produces them. Each level is strictly more exotic than the last, and each requires a more expensive transition to cross.

#### Spatial regions _(Tegmark Level I)_
Distant regions of our own universe beyond the observable horizon. Same physical laws, same constants — just unreachably far away. Local travel (walking, driving, flying) operates entirely within this level. `travel` is the command.

#### Bubble universes _(Tegmark Level II)_
Other universes produced by the same inflationary process as ours, but with different physical constants — a different speed of light, different fundamental forces. Not quantum branches; entirely separate bubbles. A `universe` shift crosses into one. Cost: very high.

#### Quantum branches _(Tegmark Level III)_
Every quantum event that could have gone differently spawns a parallel branch — the many-worlds interpretation. Physics identical, history diverging from a single branching point. `shift` steps into an adjacent branch; `shift back` returns. Cost: 20.

#### Mathematical structures _(Tegmark Level IV)_
Every self-consistent mathematical structure exists as its own reality. Different numbers of spatial dimensions, different rules of logic, laws of nature unrecognisable from ours. Crossing here is not a physical journey; it is a transition into a different formal system. Cost: extreme.

---

### Part 2 — How you exist within a reality (modes of existence)

The Tegmark level tells you *which* reality you are in. It says nothing about *how* you exist inside it. These axes are independent overlays — you can change them without changing which Tegmark reality you are in.

#### Timeline
A coarser kind of branching than quantum. A timeline marks where a significant historical event went differently — a war, a technology, a civilisation. `jump` moves forward into an alternate history; `jump back` returns. Cost: 800. _(A timeline is not a Tegmark level — it is a branch within a Level I or III reality, not a separate universe.)_

#### Time
Every reality has its own timeline of events. Navigating to a different point in time within the same reality — past or future — is cheaper than changing realities, but increasingly expensive the further you travel.

#### Simulation depth
If a reality can be computed, it can be nested. Simulation depth tracks whether you are in base reality or inside a computed world running on top of it. Entering a simulation is relatively cheap; the boundary is designed to be crossed. Exiting requires finding or constructing a way out.

#### Observer (umwelt)
Reality is never perceived directly — it is filtered through the senses and cognition of an observer. Two observers in the same physical location can inhabit entirely different experienced worlds. A bat, a human, and an AI standing in the same room share the same Tegmark address but different umwelts. An observer shift changes whose perceptual frame you occupy. Cost: low, but hard to reverse.

---

### The full coordinate

A complete position in Onto combines both kinds of axis:

```
Which reality:   Mathematical structure → Bubble universe → Quantum branch → Spatial region
How you exist:   Timeline → Point in time → Simulation depth → Observer
Where locally:   Galaxy → System → Planet → Country → Region → City → Location
```

Some commands move you within a reality (`travel`, `jump`, `home`). Others move you between realities (`shift`, `universe`). The CLI surface stays the same; only the edge types and costs change.

The CLI does not implement every axis immediately. It starts with physical navigation and expands outward.

## Core idea

A location is not just a place. It is a coordinate in a larger structure.

At the center of the design is a navigation engine that computes routes between points, regardless of whether those routes are:

- walking
- driving
- rail travel
- flight
- orbital travel
- quantum shifts
- timeline shifts
- universe shifts
- simulation entry
- observer shifts

From the CLI's point of view, these are all just different edge types in a graph.

## Coordinate model

The full position vector is implemented as `universe.CoordinateVO` in `internal/domain/universe/coordinate.go`:

```go
type CoordinateVO struct {
    Meta        string
    Mathematics string
    Universe    string
    Timeline    string
    Quantum     string
    Simulation  int
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
```

The current implementation populates the physical layers (planet through location) and the quantum and timeline axes. The higher axes (mathematics, universe, simulation) are present in the struct and ready to be navigated once higher-cost edge types are introduced.

## Example CLI experience

The prompt reflects your current position, including any non-default quantum or timeline level:

```text
[Earth/United Kingdom/Leeds/Home] >
[Earth/United Kingdom/Leeds/Station/Q1] >
[Earth/United Kingdom/Leeds/Station/Q1/T2] >
```

Current commands:

```text
where                  Show your current reality coordinate and cumulative journey cost
look                   Describe your current location
ls                     List connected locations and available transitions
route <destination>    Plan a route without moving
travel <destination>   Move to a destination (physical edges only)
home                   Show the route home and estimated cost, then confirm before executing
shift                  Jump to the next quantum branch (cost 20)
shift back             Return to the previous quantum branch
jump                   Jump to the next timeline branch (cost 800)
jump back              Return to the previous timeline branch
cost                   Show travel cost information
help                   List all commands
exit                   Leave the CLI
```

## Current status

The app is functional. It includes:

- a command entrypoint in `cmd/onto`
- a working CLI with `where`, `look`, `ls`, `route`, `travel`, `home`, `cost`, `shift`, `shift back`, `jump`, `jump back`, and `exit` commands
- BFS-based graph routing across locations with travel modes (walk, rail, etc.)
- quantum branch navigation: `shift` jumps forward to the next branch (cost 20); `shift back` returns to the previous one
- timeline branch navigation: `jump` jumps forward to the next alternate history (cost 800); `jump back` returns to the previous one
- `travel` rejects routes that cross quantum or timeline boundaries — physical and non-physical travel are kept separate
- `home` command: shows the full plan and estimated cost to unwind all timeline jumps, quantum shifts, and physical travel back to the start location, then asks for confirmation before executing
- cumulative journey cost tracked across the session and shown in `where` output and after every move
- a full coordinate model covering universe, timeline, quantum, planet, country, region, city, and location
- location and edge data loaded from `data/locations.json`, with a built-in fallback map
- interactive prompting to create new locations when arriving at a dead-end node
- auto-save of the universe graph back to `data/locations.json` after travel

## Architecture

Four layers, each importing only inward:

| Layer | Package path | Role |
|---|---|---|
| Domain | `internal/domain/` | Business rules, no I/O |
| Application | `internal/application/` | Use-case orchestration |
| Infrastructure | `internal/infrastructure/` | File I/O, graph algorithms |
| Interface | `internal/interface/cli/` | Terminal delivery |

The domain defines the types and interfaces; every other layer depends on it, never the reverse. See [docs/DDD.md](docs/DDD.md) for how DDD patterns are applied here.

## Getting started

**Natively** (requires Go):

```bash
make run
# or directly:
go run ./cmd/onto
```

**In Docker** (requires Docker):

```bash
make docker-run
# or directly:
docker compose run --rm onto
```

The `data/` directory is mounted as a bind mount, so locations you create or travel to persist on the host between runs. To clean up any dangling containers or anonymous volumes:

```bash
make docker-clean
```

Environment variables can be set in `.env` (copy `.env.example` to get started) or overridden inline:

| Variable | Default | Description |
|---|---|---|
| `ONTO_DATA_FILE` | `data/locations.json` | Path to the universe JSON file |
| `ONTO_START_LOCATION` | `home` | Location ID the app starts at |

```bash
ONTO_START_LOCATION=station make docker-run
```

**Running tests:**

```bash
make test
```

**Regenerating mocks** (after changing a domain interface):

```bash
make mocks
```

## Roadmap

1. ~~Implement a simple local-world graph for Earth.~~ ✓
2. ~~Add location lookup and basic routing.~~ ✓
3. ~~Track cumulative journey cost across the session.~~ ✓
4. ~~Support quantum transitions.~~ ✓ (`shift` / `shift back`)
5. ~~Support timeline transitions.~~ ✓ (`jump` / `jump back`)
6. ~~Add `home` command to return to start, unwinding all branches with confirmation.~~ ✓
7. Support universe transitions (higher-cost exotic modes).
8. Expand the coordinate model with more layers (simulation depth, observer shifts).
9. Evolve the CLI into a true reality navigator.

## Notes

This project is philosophical in spirit, but practical in implementation. It treats navigation as a general problem: move from one place to another, whether that place is physical, historical, simulated, or otherwise.

## Design notes

For the hierarchical reality model, coordinate system, vector costs, and future CLI progression, see [DESIGN.md](docs/DESIGN.md).

## Why the name "Onto"

The name "Onto" is chosen deliberately for two related reasons:

- Ontological: the prefix "onto-" (as in ontology) points to being and existence — the project models different modes of existence (what exists) as layered coordinates.
- Onto-as-motion: colloquially "onto" suggests movement (putting something onto something else). The CLI is about moving "onto" other places, timelines, and perspectives — so the name also reads as an action.

The dual meaning signals the project's intent: it's both a tool for describing kinds of existence (an ontology) and a navigator for moving through them. The UI reflects that by showing your current coordinate and making movement (routing/travel) the central operation.

