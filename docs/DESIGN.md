Design notes — Hierarchical realities & navigation
=================================================

The following is a saved copy of the hierarchical-reality concept and CLI progression. It's intended as a future-improvements reference when we expand beyond local navigation.

The concept treats reality as a hierarchy rather than a flat graph. Roughly:

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

1. Start with only local navigation — a filesystem-like model. Keep the CLI small and familiar: `where`, `ls`, `cd`/`travel`, `route`, `look`, `scan`, `cost`.

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

---

# Addressing reality: coordinates, vectors, and readable addresses

These notes capture the idea that a location is more than a point in space — it is an address in a multidimensional reality. They expand on the hierarchical model and offer practical CLI conventions to make coordinates both human-readable and machine-friendly.

Core idea

- A location is an address in reality, analogous to a URL, a filesystem path, or GPS coordinates.
- The hierarchy can be thought of along three orthogonal dimensions:

```
Reality
├── Which reality?   (Universe / Timeline / Branch / Quantum variant)
├── Where in that reality? (Galaxy → System → Planet → Region → City → Location)
└── When?            (timestamp / temporal coordinate)
```

Example hierarchical path (readable):

```
Reality:
Origin
 └─ Timeline 0
     └─ Earth
         └─ Europe
             └─ UK
                 └─ Yorkshire
                     └─ Leeds
                         └─ Home
                             @ 2026-08-02 18:37
```

Compact coordinate (ID-based):

```
R0:T0:U1:G42:S3:P1:L118
```

Filesystem-like path (CLI-friendly):

```
/origin/timeline0/earth/uk/yorkshire/leeds/home
```

Multidimensional structure

The model can be represented as a structured value rather than a single string. Example Go type:

```go
type Coordinate struct {
    Layer        int       // Reality layer (0 = local)
    UniverseID   string
    TimelineID   string
    BranchID     string
    QuantumID    string

    Galaxy       string
    StarSystem   string
    Planet       string

    Region       string
    City         string
    Location     string

    Time         time.Time
}
```

Rendering

The CLI should allow multiple renderings, e.g. `where --short` for compact IDs and `where --full` for an expanded, human-readable view.

Coordinates as vectors

Treat coordinates as vectors in a reality space: each field is an axis, and journeys are vector transformations. This lets the routing algorithm compute lowest-cost paths through a multidimensional graph where different axes have different traversal costs:

- Walk to a station: change only the `Location` axis.
- Shift quantum variant: change `QuantumID`.
- Enter alternate history: change `BranchID`.
- Jump universes: change `UniverseID`.

Costs

Costs arise from which axes change and by how much. Examples (illustrative):

- Local move (Location): cost 1
- Quantum shift: cost 20
- Timeline change: cost 800
- Universe change: cost 30000

This model keeps the CLI surface stable (same commands) while the routing backend interprets edges and costs across axes.

Use these notes as a guide when extending Onto beyond local navigation: they explain the address model, rendering choices, and the vector-based view that unifies walking and exotic reality transitions.
