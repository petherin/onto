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

Instead of treating reality as a single fixed map, Onto models existence as layered coordinates:

- Meta / ontological layer
- Mathematical layer
- Physical universe layer
- Historical / timeline layer
- Quantum branch layer
- Simulation layer
- Perceptual / observer layer
- Spatial layer
- Temporal layer

The CLI does not need to understand every layer immediately. It can start with local physical navigation and later expand into more exotic modes.

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

The project is building toward a coordinate-based representation like this:

```go
type Coordinate struct {
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

For the current version, the scaffold focuses on the physical and local layers, with Earth and simple location data as a starting point.

## Example CLI experience

The prompt should eventually feel like this:

```text
[Earth/UK/Leeds/Home] >
```

And commands such as:

```text
where
look
ls
route station
travel station
cost
exit
```

These commands are intended to grow from a simple local-navigation prototype into a much broader reality traversal interface.

## Current status

The app is functional. It includes:

- a command entrypoint in `cmd/onto`
- a working CLI with `where`, `look`, `ls`, `route`, `travel`, `cost`, `shift`, `shift back`, `jump`, `jump back`, and `exit` commands
- BFS-based graph routing across locations with travel modes (walk, rail, etc.)
- quantum branch navigation: `shift` jumps forward to the next branch (cost 20); `shift back` returns to the previous one
- timeline branch navigation: `jump` jumps forward to the next alternate history (cost 800); `jump back` returns to the previous one
- `travel` rejects routes that cross quantum or timeline boundaries — physical and non-physical travel are kept separate
- a full coordinate model covering universe, timeline, quantum, planet, country, region, city, and location
- location and edge data loaded from `data/locations.json`, with a built-in fallback map
- interactive prompting to create new locations when arriving at a dead-end node
- auto-save of the universe graph back to `data/locations.json` after travel

## Architecture

The codebase follows **Domain-Driven Design (DDD)** with a strict layered structure. Each layer may only import inward — never outward.

```
cmd/onto/               Entry point — wires everything together and calls Run()

internal/
  domain/               The heart of the software. No imports from other internal layers.
    universe/           Core domain: Location (entity), Coordinate, Edge, TravelMode
                        (value objects), Universe (aggregate root with unexported maps
                        and public accessor methods), Repository + LocationGenerator
                        interfaces, and the BranchQuantum domain service.
    navigation/         Pathfinder interface + pure BFS functions (FindRoute,
                        PathDistance, PathCost). No concrete struct here.
    exploration/        Session entity — tracks current position, travel history,
                        and quantum state. Owned by the user, not the universe.

  application/          Orchestrates use cases. Imports domain only.
    commands/           Write operations that change state:
                          TravelCommand — moves the session to a destination.
                          ShiftCommand  — jumps forward or back through quantum branches.
    queries/            Read operations that never change state:
                          LookupQuery   — Where, Look, List.
                          RouteQuery    — plans a route without travelling it.

  infrastructure/       Technical implementations of domain interfaces.
    persistence/        JSONRepository — loads and saves Universe to a JSON file.
    generator/          NearbyGenerator — auto-creates locations at dead ends.
    navigation/         BFSPathfinder — concrete implementation of navigation.Pathfinder.

  interface/
    cli/                Delivery mechanism. Knows about the terminal; the domain
                        does not know the CLI exists.
                          App             — command dispatcher and run loop.
                          display.go      — formats application results as strings.
                          interactive.go  — InteractiveHandler (prompts user at dead ends).
                          fuzzy.go        — Levenshtein-based command/destination suggestions.

  mocks/                Generated test doubles (mockery). Never edit by hand.
                          fixtures.go     — NewTestUniverse() shared test helper.
```

### Layer rules

| Layer | May import | Must not import |
|---|---|---|
| `domain` | standard library only | anything in `internal/` |
| `application` | `domain` | `infrastructure`, `interface` |
| `infrastructure` | `domain` | `application`, `interface` |
| `interface/cli` | `domain`, `application`, `infrastructure` | — |

### Key patterns

- **Aggregate root** — `Universe` owns all `Location` entities and `Edge` value objects. The internal maps are unexported; all access goes through methods (`GetLocation`, `EdgesFrom`, `AllLocations`, `AllEdgesFlat`, etc.) so invariants are enforced by the struct itself.
- **Repository interface** — defined in `domain/universe`, implemented in `infrastructure/persistence`. The domain never references a file or database.
- **LocationGenerator interface** — also defined in the domain. Two implementations exist: `NearbyGenerator` (auto) and `InteractiveHandler` (prompts user). The `TravelCommand` accepts either without knowing which it has.
- **BranchQuantum / BranchTimeline domain services** — branch creation logic lives in `domain/universe/quantum.go` and `timeline.go`, not in the application commands. `ShiftCommand` and `JumpCommand` call these services rather than building locations and edges themselves.
- **Commands vs Queries (CQRS)** — commands (`Travel`, `Shift`) mutate session and universe state and persist the result. Queries (`Where`, `Look`, `List`, `Route`) are pure reads with no side effects.

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
3. Introduce full cost calculations (currently a stub).
4. Expand the coordinate model with more layers.
5. ~~Support quantum transitions.~~ ✓ (`shift` / `shift back` implemented)
6. ~~Support timeline transitions.~~ ✓ (`jump` / `jump back` implemented)
7. Support universe transitions (higher-cost exotic modes).
7. Evolve the CLI into a true reality navigator.

## Notes

This project is philosophical in spirit, but practical in implementation. It treats navigation as a general problem: move from one place to another, whether that place is physical, historical, simulated, or otherwise.

## Design notes

For the hierarchical reality model, coordinate system, vector costs, and future CLI progression, see [DESIGN.md](docs/DESIGN.md).

## Why the name "Onto"

The name "Onto" is chosen deliberately for two related reasons:

- Ontological: the prefix "onto-" (as in ontology) points to being and existence — the project models different modes of existence (what exists) as layered coordinates.
- Onto-as-motion: colloquially "onto" suggests movement (putting something onto something else). The CLI is about moving "onto" other places, timelines, and perspectives — so the name also reads as an action.

The dual meaning signals the project's intent: it's both a tool for describing kinds of existence (an ontology) and a navigator for moving through them. The UI reflects that by showing your current coordinate and making movement (routing/travel) the central operation.

