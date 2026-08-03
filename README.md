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
- a working CLI with `where`, `look`, `ls`, `route`, `travel`, `cost`, and `exit` commands
- BFS-based graph routing across locations with travel modes (walk, rail, etc.)
- a full coordinate model covering universe, timeline, quantum, planet, country, region, city, and location
- location and edge data loaded from `data/locations.json`, with a built-in fallback map
- interactive prompting to create new locations when arriving at a dead-end node
- auto-save of the universe graph back to `data/locations.json` after travel

## Getting started

Run the current entrypoint with:

```bash
go run ./cmd/onto
```

## Roadmap

1. ~~Implement a simple local-world graph for Earth.~~ ✓
2. ~~Add location lookup and basic routing.~~ ✓
3. Introduce full cost calculations (currently a stub).
4. Expand the coordinate model with more layers.
5. Support speculative routing such as timeline or quantum transitions.
6. Evolve the CLI into a true reality navigator.

## Notes

This project is philosophical in spirit, but practical in implementation. It treats navigation as a general problem: move from one place to another, whether that place is physical, historical, simulated, or otherwise.

## Design notes

For the hierarchical reality model, coordinate system, vector costs, and future CLI progression, see [DESIGN.md](docs/DESIGN.md).

## Why the name "Onto"

The name "Onto" is chosen deliberately for two related reasons:

- Ontological: the prefix "onto-" (as in ontology) points to being and existence — the project models different modes of existence (what exists) as layered coordinates.
- Onto-as-motion: colloquially "onto" suggests movement (putting something onto something else). The CLI is about moving "onto" other places, timelines, and perspectives — so the name also reads as an action.

The dual meaning signals the project's intent: it's both a tool for describing kinds of existence (an ontology) and a navigator for moving through them. The UI reflects that by showing your current coordinate and making movement (routing/travel) the central operation.

