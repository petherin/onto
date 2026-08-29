Design notes — Hierarchical realities & navigation
=================================================

These are the design notes behind Onto's reality model: where the
hierarchical-reality idea came from, how the CLI was expected to grow into it,
and the design of the implemented coordinate and `onto://` address system. Much
of the early "future" progression below has since shipped — the notes are kept
as the reasoning behind what now exists.

This document is the "why" behind the coordinate/address design. For the current,
user-facing description of the model (the Tegmark hierarchy, the axes, the
movement rules, and the command reference) see the project README; for how those
concepts map onto the code (layers, aggregates, repositories) see [DDD.md](DDD.md).

The concept treats reality as a hierarchy rather than a flat graph. The original
sketch — since refined into the Tegmark-based model described in the README — was
roughly:

- **Local reality** (walking around normally)
- **Neighbouring realities** (small quantum divergences)
- **Branches** (major historical differences)
- **Universes** (different physical constants)
- **Meta-realities** (collections of universes)
- **Infinite hierarchy**

## Key ideas

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

**Status: not implemented.** The interaction above is speculative. Today `route`
reports only the path, the physical distance in km, and the travel cost (see
`formatRouteResult`). The σ (sigma) cost unit, the risk bar, and the
probability-of-successful-return are design ideas, not current behaviour — see
"Cost unit" and "Future gamification ideas" below.

## Recommended progression

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

## Command reference

The full CLI command set — every command, its cost, forward/back behaviour, the
physical-vs-contextual routing rules, and the `home` unwind — is documented in
the project README ("Contextual transition reference" and "Example CLI
experience"). It isn't duplicated here so the two can't drift apart.

The key UX principle: the same navigation commands work at all layers; only the graph and edge semantics change. This continuity is what makes the interface feel like an "operating system for reality" rather than a collection of separate features.

---

# Addressing reality: coordinates, vectors, and readable addresses

These notes capture the idea that a location is more than a point in space — it is an address in a multidimensional reality. They expand on the hierarchical model and offer practical CLI conventions to make coordinates both human-readable and machine-friendly.

## Core idea

- A location is an address in reality, analogous to a URL, a filesystem path, or GPS coordinates.
- The hierarchy can be thought of along four orthogonal dimensions — not only *where* and *when*, but *which* reality and *how* that reality is experienced:

```
Reality
├── Which reality?    (Meta · Mathematics · Universe · Timeline · Quantum · Simulation · Consensus)
├── Where in it?      (Galaxy → System → Planet → Country → Region → City → Location)
├── When?             (timestamp / temporal coordinate)
└── Experienced how?  (Observer — the umwelt, the point of view reality is perceived from)
```

The fourth dimension is easy to overlook: two coordinates can share every "which / where / when" axis and still differ because they are inhabited from different points of view. The `Observer` axis captures that — see "Observer axis and the umwelt" below.

Example hierarchical path (readable):

```
Reality:
Origin
 └─ Prime
     └─ Milky Way
         └─ Solar System
             └─ Earth
                 └─ United Kingdom
                     └─ Yorkshire
                         └─ Leeds
                             └─ Home
                                 @ Human, 2026-08-02T18:37
```

## Canonical onto:// address (implemented)

Every coordinate serialises to a deterministic, parseable `onto://` address. Two forms are supported:

**Full address** — all axes always present, unset fields rendered as `_`:

```
onto://<meta>.<math>/<universe>/<timeline>/<quantum>/<galaxy>/<system>/<planet>/<country>/<region>/<city>/<location>/sim:<n>/cons:<n>@<observer>+<RFC3339>
```

- `sim:<n>` and `cons:<n>` are omitted when their value is 0
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

## Coordinate value object

These axes map one-to-one onto the `CoordinateVO` value object in
`internal/domain/universe/coordinate.go` — string fields for the named axes
(Meta, Mathematics, Universe, Timeline, Quantum, Galaxy…Location, Observer),
integers for the counted axes (Simulation, Consensus), and a `time.Time` for
*When*. That struct is the source of truth, so it isn't reproduced here.

## Rendering

The CLI should allow multiple renderings, e.g. `where --short` for compact IDs and `where --full` for an expanded, human-readable view.

## Coordinates as vectors

Treat coordinates as vectors in a reality space: each field is an axis, and journeys are vector transformations. This lets the routing algorithm compute lowest-cost paths through a multidimensional graph where different axes have different traversal costs:

- Walk to a station: change only the `Location` axis.
- Shift quantum variant: change `Quantum`.
- Enter alternate history: change `Timeline`.
- Jump universes: change `Universe`.

## Costs

Costs arise from which axes change and by how much: a local step is cheap, a
quantum shift more, a timeline jump more again, and a universe or
mathematical-structure change is extreme. The exact, authoritative
per-transition costs live in the README's "Contextual transition reference"
table — they are not repeated here so the two can't drift.

### Cost unit

In the code, cost is a dimensionless `float64` — `TransitionCost` sums it, the
session accumulates it as `cumulativeCost`, and it is displayed simply as
"Cumulative journey cost" with no unit. Proposed name for that unit: **σ
(sigma)**, read as "reality debt" — the accumulated instability of having moved
away from your origin across one or more reality axes. This is a display-only
naming suggestion: adopting it would mean rendering costs as e.g. `σ 140` in
`where`, route output, and the budget HUD, without changing any of the
underlying numbers. The speculative example at the top of this document already
uses this σ notation.

This model keeps the CLI surface stable (same commands) while the routing backend interprets edges and costs across axes.

## Observer axis and the umwelt

The `Observer` axis captures the idea that reality is not objective — it is always perceived from a point of view. The concept comes from Jakob von Uexküll's notion of the *umwelt*: the subjective, species-specific perceptual world that every organism inhabits. A bat, a dog, and a human standing in the same room occupy the same spatial location but three entirely different umwelts.

In Onto, two coordinates can share every other axis and still differ on `Observer`. An `ObserverShift` edge transitions between umwelts — from human perception to machine perception, from waking to dreaming, from one interpretive frame to another. This is one of the cheapest exotic transitions (it requires no physical movement) but one of the hardest to reverse, because the origin umwelt may no longer be recognisable from inside the destination one.

## Simulation axis

The `Simulation int` axis represents depth within nested simulations. At depth 0 you are in base reality (or what you take to be base reality). Each `SimulationEntry` edge increments the depth by one. Simulations can be arbitrarily nested, and the routing graph can represent them as subgraphs reachable only through a simulation boundary edge.

Cost implications:
- Entering a simulation: low cost (10) — the boundary is intentionally designed to be crossed (`simulate`)
- Exiting a simulation: moderate cost (50) — requires finding or constructing an exit (`simulate back`)
- Determining whether you are inside a simulation: undefined — this is left to the observer

The simulation axis interacts with the observer axis: changing umwelt inside a simulation may be easier than in base reality, because the simulation's rules are malleable. Implementation uses `BranchSimulation` with asymmetric forward/reverse edge costs via `ContextualTransitionSpec.ReverseCost`.

## Future gamification ideas

Game mode today (documented in the README) gives a finite **budget**, an ordered
**quest chain** of round trips, a **par** optimal cost, and a **1–3 star**
efficiency rating. The following build on that base and on the σ / risk /
return-probability sketch at the top of this document — none are implemented yet:

- **Return probability & risk.** Give exotic transitions a chance of failing or
  of stranding the traveller, and surface a per-route risk bar and
  probability-of-successful-return. Deep simulation and mathematical-structure
  jumps would be the riskiest.
- **Reality debt / instability.** Let accumulated σ have consequences: past a
  threshold the traveller is destabilised — random forced drifts, higher exit
  costs, or edges that temporarily close until σ is paid down by going home.
- **Seeded daily quest.** A deterministic quest from the objective pool keyed on
  the date, so every player faces the same chain and can compare scores.
- **Personal bests & leaderboards.** Persist the best star rating and lowest cost
  per quest (or per seed), locally at first.
- **Achievements.** Award badges for milestones: visit every Tegmark level, reach
  simulation depth N, win under par with budget to spare, or restore every axis
  by hand rather than via `home`.
- **Fog of war.** Hide coordinates until first visited, making discovery — not
  just routing — part of the game.
- **Move or time limit.** Cap the number of moves (or wall-clock time) as an
  alternative constraint to the spending budget.
- **Regenerating energy.** Replace or complement the fixed budget with an energy
  pool that refills slowly per move, rewarding patient, efficient routing.

These would slot into the existing facade options (`WithBudget`, `WithTargets`,
`WithObjectivePool`) and the session's cost/goal tracking without changing the
coordinate or routing model.

## Future web UI & navigation ideas

Ideas for enriching the browser Reality Map (the Canvas-2D renderer in
`internal/interface/web/static/`) and the graph-travel model behind it. The
connecting theme is shifting the map from **imperative stepping** (click a node,
travel one edge) toward **plan → preview → commit**, with cost, risk, and
return-probability made visible — the direction the "GPS for existence" framing
above already gestures at. None are implemented except where marked.

Presentation — cheap wins (the data is already sent by the API, just not drawn):

- **Edge cost / distance.** `EdgeSnapshot.Cost` and `.Distance` are sent but not
  rendered; label edges (or show on hover) and vary edge width/opacity by cost so
  expensive exotic jumps *look* expensive. **Shipped** (this session) as an
  on-hover route preview with per-hop cost labels and a totals line; the
  always-on edge weighting is still open.
- **Full objective chain.** `Objectives[]` (with per-waypoint `Reached`) is sent
  but only the current target shows in the HUD; render the whole chain as a
  checklist and mark each waypoint node on the map (a ring/flag) so the quest is
  spatially legible.
- **History trail.** `SessionSnapshot.History` only appears in the log panel;
  draw the recent path as a fading breadcrumb on the graph.

Presentation — interaction, usability, performance:

- **Search / jump-to** a coordinate or node name (important once the graph grows).
- **Minimap / best-fit per reality** — the deterministic layout already clusters
  each reality; a minimap of cluster centres would aid navigation across nested
  realities.
- **Accessibility & mobile** — keyboard-driven node focus/travel and touch
  gestures (the current shift+drag / wheel model is mouse-only).
- **First-run tour** explaining the colour legend and the physical-vs-contextual
  distinction.
- **Layout performance** — the `tick()` repulsion loop is O(n²) over all node
  pairs every frame; a spatial grid / Barnes-Hut approximation would keep it
  smooth as the universe grows.

Enriching the graph-travel model:

- **Make the route the hero, not the node.** A first-class multi-hop route
  planner — pick a destination coordinate, see the full plan (physical legs +
  contextual transitions) with its total cost, then execute step-by-step — fits
  the "GPS for existence" framing far better than click-to-step.
- **Cross-axis routing.** `travel` deliberately refuses any route crossing a
  reality boundary, so a journey like `home → Q2 → sim:1` has no single planned
  path today. A higher-level planner that *composes* physical travel with
  contextual transitions (still executing them as distinct edges) would let you
  route to any coordinate, not just physical ones.
- **Risk & return-probability.** Surface the risk bar and probability-of-return
  from the speculative sketch above on routes/edges, so deep simulation and
  mathematical-structure jumps read as visibly risky.
- **σ / reality-debt as a live pressure.** Visualise accumulated cost as
  instability — the further you drift, the more the map destabilises (jitter,
  desaturation) — with `home` as the pressure-release, turning the cumulative
  cost number into a felt mechanic.
- **Reality "diff" view.** Hovering a node in another reality shows *what differs*
  from your current coordinate (which axes changed, by how much) — the vector
  view from "Coordinates as vectors" made visible.
- **Time-scrubber** for the Time axis (a slider previewing temporal branches)
  rather than a raw RFC3339 input.
- **Alternative routes.** Offer 2–3 candidate paths (cheapest / safest /
  fewest-hops) and let the player choose — classic routing UX applied to reality.

Use these notes as a guide when extending Onto beyond local navigation: they explain the address model, rendering choices, and the vector-based view that unifies walking and exotic reality transitions.
