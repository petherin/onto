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

Instead of treating reality as a single fixed map, Onto models existence as layered coordinates. Each layer is a distinct axis of reality that you can navigate along — from the mundane to the deeply abstract.

### Spatial layer
The baseline. Walking to a station, taking a train, flying to another continent, reaching orbit, warping between stars. All of these are the same operation: physical movement through space at different scales. The coordinate tracks planet, country, region, city, and location. At the largest scales this includes regions beyond our observable horizon — cosmologically distant but still the same universe, with the same physical laws and constants. No new axis is needed; it is just a very long journey. _(Tegmark Level I applies to this extreme end only — regions beyond the observable horizon — not to local travel.)_

### Physical universe layer _(Tegmark Level II)_
Beyond our own bubble, inflationary cosmology suggests other universes exist with different physical constants — a different speed of light, a different gravitational constant, or entirely different fundamental forces. These are not quantum branches of our universe; they are separate universes produced by the same inflationary process. A `universe` shift crosses into one of these.

### Quantum branch layer _(Tegmark Level III)_
Every quantum event that could have gone differently spawns a parallel branch of the universe — the many-worlds interpretation of quantum mechanics. In Onto, `shift` steps you sideways into an adjacent branch. The physics are identical, but small differences have accumulated from that branching point forward. The further you shift, the more things diverge.

### Mathematical layer _(Tegmark Level IV)_
The most abstract navigable space. The mathematical multiverse hypothesis holds that every self-consistent mathematical structure exists as its own reality — not just our physics, but any set of axioms that doesn't contradict itself. A mathematical reality might have different numbers of spatial dimensions, different rules of logic, or laws of nature that bear no resemblance to ours. Crossing into a mathematical reality is not a physical journey; it is a transition into a different formal system.

### Historical / timeline layer
A coarser kind of branching than quantum. A timeline represents a history where a significant event went differently — a war that ended another way, a technology that was never invented, a civilization that collapsed or didn't. Timeline branches are more expensive to cross (cost 800 vs 20 for quantum) because the differences are larger and the distance harder to bridge. `jump` moves you forward into a new alternate history; `jump back` returns you to the one you came from.

### Simulation layer
If a reality can be computed, it can be nested. This layer tracks depth within a simulation stack — whether you are in the base reality, inside a computed world running on top of it, or deeper still. `simulation` entry moves you down into the next layer; the return path leads back out.

### Perceptual / observer layer
Some differences in reality are not about where or when you are, but about the perspective from which you observe. An `observer` shift changes whose frame of reference you occupy — a different conscious viewpoint, a different measuring apparatus, or a different relationship to the events around you.

### Meta / ontological layer
The outermost coordinate. Above all specific models of physics, mathematics, or consciousness lies the question of what kind of existence something has at all — whether it is concrete, abstract, fictional, potential, or something with no name yet. The meta layer is a placeholder for transitions that don't fit any other axis.

---

The CLI does not need to understand every layer immediately. It starts with physical navigation and expands into more exotic modes as commands are implemented.

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

