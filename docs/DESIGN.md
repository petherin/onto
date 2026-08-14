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

CLI commands (current)

```
where                  — current reality coordinate and cumulative journey cost
look                   — describe the current location
ls                     — adjacent locations and available transitions
route <destination>    — plan a route without moving
travel <destination>   — move to a destination (physical edges only)
home                   — show the plan and cost to return home, confirm, then execute
shift                  — jump forward to the next quantum branch (Q0 → Q1 → …)
shift back             — return to the previous quantum branch
jump                   — jump forward to the next timeline branch (Prime → T1 → …)
jump back              — return to the previous timeline branch
drift                  — enter the next consensus divergence (0 → 1 → …)
align                  — return one level toward shared consensus
observe <observer>     — change observer perspective
observe back           — return to the previous observer perspective
time <RFC3339>         — enter a temporal branch at an absolute timestamp
time back               — return to the previous temporal branch
<number>               — take the corresponding numbered possible journey
cost                   — show travel cost information
help                   — list all commands
exit                   — leave the CLI
```

`travel` enforces physical-only routing — it rejects any path that contains a quantum or timeline edge. Exotic transitions require their own dedicated commands (`shift`, `jump`).

`home` is a composite operation: it shows the full unwind plan (observer return, consensus alignment, timeline jumps back, quantum shifts back, physical travel), displays the estimated cost, and asks for confirmation before executing. Each step uses the same underlying commands and incurs the same cost as doing it manually.

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

Canonical onto:// address (implemented)

Every coordinate serialises to a deterministic, parseable `onto://` address. Two forms are supported:

**Full address** — all axes always present, unset fields rendered as `_`:

```
onto://<meta>.<math>/<universe>/<timeline>/<quantum>/sim:<n>/<galaxy>/<system>/<planet>/<country>/<region>/<city>/<location>@<observer>+<RFC3339>
```

- `sim:<n>` is omitted when simulation depth is 0
- `+<RFC3339>` is omitted when time is the zero value
- Spaces in field values are encoded as `_` (e.g. `United_Kingdom`)

Example (default position):

```
onto://Origin.Classical/Origin/Prime/Q0/Milky_Way/Solar_System/Earth/United_Kingdom/Yorkshire/Leeds/Home@Human
```

**Short address** — only non-default axes included, in order. Used in the CLI prompt and route summaries:

```
onto://Leeds/Home
onto://Q1/Leeds/Station
onto://T2/Leeds/Home@Machine
onto://Andromeda/Kepler-22/Kepler-22b/New_Athens/Home
```

Both forms are parseable with `ParseOntoAddress()`. The full address round-trips exactly; the short address populates whichever axes are present.

Multidimensional structure

The model can be represented as a structured value rather than a single string. The current implementation in `internal/domain/universe/coordinate.go`:

```go
type CoordinateVO struct {
    Meta        string    // Ontological layer
    Mathematics string    // Mathematical framework
    Universe    string    // Which universe
    Timeline    string    // Historical timeline
    Quantum     string    // Quantum branch (e.g. "Q0", "Q1")
    Simulation  int       // Nesting depth within simulations (0 = base reality)
    Galaxy      string
    System      string
    Planet      string
    Country     string
    Region      string
    City        string
    Location    string
    Observer    string    // Perceptual umwelt
    Time        time.Time
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

Observer axis and the umwelt

The `Observer` axis captures the idea that reality is not objective — it is always perceived from a point of view. The concept comes from Jakob von Uexküll's notion of the *umwelt*: the subjective, species-specific perceptual world that every organism inhabits. A bat, a dog, and a human standing in the same room occupy the same spatial location but three entirely different umwelts.

In Onto, two coordinates can share every other axis and still differ on `Observer`. An `ObserverShift` edge transitions between umwelts — from human perception to machine perception, from waking to dreaming, from one interpretive frame to another. This is one of the cheapest exotic transitions (it requires no physical movement) but one of the hardest to reverse, because the origin umwelt may no longer be recognisable from inside the destination one.

Simulation axis

The `Simulation int` axis represents depth within nested simulations. At depth 0 you are in base reality (or what you take to be base reality). Each `SimulationEntry` edge increments the depth by one. Simulations can be arbitrarily nested, and the routing graph can represent them as subgraphs reachable only through a simulation boundary edge.

Cost implications:
- Entering a simulation: low cost (10) — the boundary is intentionally designed to be crossed (`simulate`)
- Exiting a simulation: moderate cost (50) — requires finding or constructing an exit (`simulate back`)
- Determining whether you are inside a simulation: undefined — this is left to the observer

The simulation axis interacts with the observer axis: changing umwelt inside a simulation may be easier than in base reality, because the simulation's rules are malleable. Implementation uses `BranchSimulation` with asymmetric forward/reverse edge costs via `ContextualTransitionSpec.ReverseCost`.

Use these notes as a guide when extending Onto beyond local navigation: they explain the address model, rendering choices, and the vector-based view that unifies walking and exotic reality transitions.
