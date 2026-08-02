# Onto

Onto is an experimental CLI for navigating reality as a coordinate system.

## Table of contents

- [Vision](#vision)
- [Core idea](#core-idea)
- [Coordinate model](#coordinate-model)
- [Example CLI experience](#example-cli-experience)
- [Current status](#current-status)
- [Getting started](#getting-started)
- [Roadmap](#roadmap)
- [Notes](#notes)
- [Why the name "Onto"](#why-the-name-onto)
- [Design notes](docs/DESIGN.md)


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
history
cost
exit
```

These commands are intended to grow from a simple local-navigation prototype into a much broader reality traversal interface.

## Current status

The repository now contains a basic Go project scaffold with:

- a command entrypoint in cmd/onto
- CLI package structure
- navigation primitives for graph routing and pathfinding
- reality abstractions for coordinates, locations, edges, and travel plans
- starter world and data files for Earth

This is an early foundation, not yet a full implementation.

## Getting started

Run the current entrypoint with:

```bash
go run ./cmd/onto
```

## Roadmap

1. Implement a simple local-world graph for Earth.
2. Add location lookup and basic routing.
3. Introduce travel modes and cost calculations.
4. Expand the coordinate model with more layers.
5. Support speculative routing such as timeline or quantum transitions.
6. Evolve the CLI into a true reality navigator.

## Notes

This project is philosophical in spirit, but practical in implementation. It treats navigation as a general problem: move from one place to another, whether that place is physical, historical, simulated, or otherwise.

## Design notes — Hierarchical realities & navigation

The following is a saved copy of the hierarchical-reality concept and CLI progression you described. It's intended as a future-improvements reference when we expand beyond local navigation.

Yes. The concept treats reality as a hierarchy rather than a flat graph. Roughly:

- **Local reality** (walking around normally)
- **Neighbouring realities** (small quantum divergences)
- **Branches** (major historical differences)
- **Universes** (different physical constants)
- **Meta-realities** (collections of universes)
- **Infinite hierarchy**

Key ideas

- Travelling has a cost (Sigma instability, entropy, or "reality debt"). Some transitions are effectively impossible or very costly.
- The CLI acts like a GPS for existence: users request a `route` and the engine computes a path across layers, with cost, risk, and return probability metadata.

Example interaction (speculative):

```
> where
Φ: Sol.Earth.Europe.UK.Leeds.Home.Office

> route Shakespeare.Alive

Searching...

✓ Route found

1. Leeds.Office
2. Leeds.Outside
3. Earth.Quantum+3
4. Britain.MonarchyDivergence
5. Timeline 447A
6. Shakespeare.Alive

Estimated Cost:
σ 14.7

Risk:
■■□□□□

Probability of successful return:
92%
```

Recommended progression

1. Start with only local navigation — a filesystem-like model (this is exactly the right decision). Keep the CLI small and familiar: `where`, `ls`, `cd`/`travel`, `route`, `look`, `scan`, `cost`.

```
Reality
└── Earth
    └── Europe
        └── UK
            └── Yorkshire
                └── Leeds
                    ├── Station
                    ├── University
                    ├── CityCentre
                    └── Home
```

2. Add coordinates for each location (human-readable and machine-friendly).

3. Navigation: `route <destination>` returns path, distance, cost, sigma/entropy and risk metadata.

4. After local navigation is solid, add higher-dimension transports (quantum shifts, timeline shifts, universe jumps). The commands remain the same; only the underlying graph and edge types change.

CLI commands (early set)

```
where    — current coordinate
look     — describe location
ls       — adjacent nodes
cd       — move (alias: travel)
route    — compute route
scan     — nearby reachable nodes
cost     — show energy/cost
history  — visited
map      — ascii map
```

Later, add `shift` and `route` variants that operate on quantum/historical layers, for example:

```
shift +1           // move to neighbouring quantum branch
route "Roman Empire survives"  // route to an alternate-history node
```

The key UX principle: the same navigation commands work at all layers; only the graph and edge semantics change. This continuity is what makes the interface feel like an "operating system for reality" rather than a collection of separate features.

Use this section as a future roadmap reference when we expand Onto beyond local navigation.

For full design notes, coordinates and the vector model, see [DESIGN.md](DESIGN.md).

## Why the name "Onto"

The name "Onto" is chosen deliberately for two related reasons:

- Ontological: the prefix "onto-" (as in ontology) points to being and existence — the project models different modes of existence (what exists) as layered coordinates.
- Onto-as-motion: colloquially "onto" suggests movement (putting something onto something else). The CLI is about moving "onto" other places, timelines, and perspectives — so the name also reads as an action.

The dual meaning signals the project's intent: it's both a tool for describing kinds of existence (an ontology) and a navigator for moving through them. The UI reflects that by showing your current coordinate and making movement (routing/travel) the central operation.

