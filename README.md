# Onto

![alt text](/docs/watermarked_img_55746261856063020.jpg "Title")

Onto is an experimental CLI for navigating reality as a coordinate system.

## Table of contents

- [Vision](#vision)
- [Core idea](#core-idea)
- [Coordinate model](#coordinate-model)
- [Example CLI experience](#example-cli-experience)
- [Game mode](#game-mode)
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

Instead of treating reality as a single fixed map, Onto models existence as a coordinate system with nested and orthogonal axes. At the foundation: **which exclusive universe you inhabit** (the strict Tegmark hierarchy), and woven through it: **how you exist within that universe** (orthogonal dimensions of branching and perception).

---

### Part 1 — Which universe? The strict Tegmark hierarchy

The Tegmark hierarchy is exclusively nested: you belong to exactly one mathematical structure, and within it, one bubble universe, and within that, you navigate two orthogonal dimensions of location (space and quantum branch). Moving up this hierarchy is extremely expensive because it means leaving your entire universe.

```
Level IV: Mathematical structures (most exotic)
    ↓ contains
Level II: Bubble universes (different physical constants)
    ↓ contains
Level I + Level III: Spatial regions with quantum branching (same physics, different locations and outcomes)
```

**The key insight:** Level III (quantum branches) is not a separate level in the hierarchy above Level I. Instead, Level III and Level I are **orthogonal dimensions of the same universe**. Every location in space (Level I) has quantum branches threading through it (Level III). You do not move to "Level III space" — rather, you shift to a different quantum branch *at your current location*.

#### Spatial regions _(Tegmark Level I)_
Distant regions of our own universe beyond the observable horizon. Same physical laws, same constants — just unreachably far away. Local travel (walking, driving, flying) operates entirely within this level. `travel` is the command.

#### Bubble universes _(Tegmark Level II)_
Other universes produced by the same inflationary process as ours, but with different physical constants — a different speed of light, different fundamental forces. Not quantum branches; entirely separate bubbles. Each bubble has its own Level I spatial structure, and quantum branches (Level III) thread through all of it. A `universe` shift moves between bubbles; `universe back` returns to the previous one. (Very expensive — see the [contextual transition reference](#contextual-transition-reference) for exact costs.)

#### Quantum branches _(Tegmark Level III — orthogonal to Level I, not hierarchically above it)_
Every quantum event that could have gone differently spawns a parallel branch — the many-worlds interpretation. **Level III branches exist within the same physical universe as Level I regions.** You can occupy the same spatial location (Level I) in different quantum branches (Level III), each with a different history of outcomes. Physics identical, branching trajectories from quantum decisions. `shift` steps into an adjacent branch; `shift back` returns.

#### Mathematical structures _(Tegmark Level IV)_
Every self-consistent mathematical structure exists as its own reality. Different numbers of spatial dimensions, different rules of logic, laws of nature unrecognisable from ours. Each structure contains all its bubbles and all their Level I regions and Level III branches. Crossing here is not a physical journey; it is a transition into a fundamentally different formal system. `structure` moves forward into the next mathematical structure; `structure back` returns. This is the most expensive implemented transition — see the [contextual transition reference](#contextual-transition-reference).

---

### Part 2 — How you exist within that universe (orthogonal branching and perception axes)

You have identified your bubble and mathematical structure (Tegmark II and IV). Within that bubble's Level I universe, you occupy:
- A location in space (Level I)
- A quantum branch (Level III)

On top of these, four more independent dimensions allow you to navigate *how* you exist:

#### Timeline
A coarser historical fork than quantum mechanics — a macroscopic divergence point. A timeline marks where a significant historical event went differently — a war, a technology, a civilisation. Multiple timelines can exist within a single quantum branch. `jump` moves forward into an alternate history; `jump back` returns. _(Not a Tegmark level: it branches within a single Level I + III address.)_

#### Time
Every universe has a temporal dimension. Navigating to a different point in time within the same location, branch, and timeline — past or future — is cheaper than most transitions, but increasingly expensive the further you travel.

#### Simulation depth
If a reality can be computed, it can be nested. Simulation depth tracks whether you are in base reality or inside a computed world running on top of it. A simulated reality contains its own complete universe: its own Level I spatial regions, Level III quantum branches, and all the branching/temporal axes within. `simulate` enters one layer deeper; `simulate back` exits one layer toward base reality — and it is deliberately harder to leave than to enter (see the [contextual transition reference](#contextual-transition-reference)).

#### Observer (umwelt)
Reality is never perceived directly — it is filtered through the senses and cognition of an observer. Two observers in the same location, same quantum branch, same timeline, same time, same simulation depth can inhabit entirely different experienced worlds. A bat, a human, and an AI standing in the same room share the same full coordinate but different umwelts. An observer shift changes whose perceptual frame you occupy. Cost: low, but hard to reverse.

#### Consensus divergence
Reality exists on a spectrum between collective consensus (everyone agrees on the rules) and individual divergence (your world operates by different rules). Consensus divergence tracks whether you are in the shared consensus reality (0) or inside a state where the rules are different (1+): dreams, delusions, hallucinations, altered consciousness, psychosis, fiction, mythology, fantasy. These are not simulated (not computational) and not merely perceptual (the rules themselves change, not just what you sense). A delusion is as real-to-you as waking reality is, but it's divergent from consensus. Dreams follow dream-logic. Hallucinations are sensory divergences. Altered states (meditation, intoxication) change how causality itself works. `drift` enters a divergent state; `align` returns to consensus. Cost: low, but dangerous—deeper divergences make consensus reality harder to access. Nesting possible: dream within a hallucination, psychotic episode within a dream, story within a delusion. **Note:** Consensus divergence applies only to observers with subjective experience. Non-conscious entities (rocks, algorithms without experience) remain at consensus=0 by definition—they have no inner world to diverge.

---

### How it all hangs together

**Your address has three layers:**

1. **Exclusive hierarchy** (you occupy exactly one position):
   - Which mathematical structure (Level IV)?
   - Which bubble universe (Level II)?
   - These determine your fundamental physics.

2. **Universe coordinates** (two orthogonal dimensions of the same universe):
   - Which spatial region (Level I)?
   - Which quantum branch within that region (Level III)?
   - These determine your *where* and *which outcome*.

3. **Experience overlays** (five independent axes you navigate within your universe):
   - Which timeline (macroscopic history branch)?
   - Which time (temporal position)?
   - Which simulation depth (base or nested, computational)?
   - Which consensus divergence (consensus reality or divergent state)?
   - Which observer (perceptual frame)?

**How they interact:**

- **Levels IV and II are permanent anchors.** Moving between them is expensive and rare. Once you commit to a bubble universe, you stay within it unless you pay the extreme cost.

- **Level I and Level III are orthogonal coordinates of the same space.** Shifting quantum branches doesn't move you physically; moving physically doesn't change your quantum branch. A "position" requires both: "Earth, Quantum branch Q3" is different from "Earth, Quantum branch Q1," but both are still Earth.

- **Timeline, Time, Simulation, Consensus divergence, and Observer are independent overlays.** You can:
  - Shift quantum branches without changing your timeline
  - Jump to a different timeline without changing your location or branch
  - Move through time without leaving your timeline
  - Enter a simulation while staying in the same branch, location, and timeline (the simulation contains its own nested Level I/III structure)
  - Drift into a divergent state (dream, delusion, hallucination, psychosis) without leaving your simulation, location, or timeline (you can dream while awake, hallucinate while in a simulation, be delusional within a dream)
  - Change observer without any physical or temporal movement

- **Local coordinates (galaxy, planet, city, location) are always nested within your current Level I region**, regardless of which quantum branch, timeline, time, or simulation depth you occupy.

**Cost hierarchy (cheapest to most expensive):**
- **Cheapest**: physical travel (walking/driving within one region)
- **Very cheap**: observer shifts (changing perceptual perspective), imagination transitions (entering/exiting dreams, stories)
- **Low**: time travel (short distances), simulation boundary crossings
- **Moderate**: timeline jumps (macroscopic history shifts)
- **Expensive**: quantum shifts (Level III branches)
- **Very expensive**: universe shifts (Level II bubbles)
- **Extreme**: mathematical structure transitions (Level IV)

**Why this model works:**

The design unifies all navigation under a single principle: *routing through a graph*. Whether you're walking to a store, shifting to a different quantum outcome, jumping to an alternate history, or entering a simulation, the CLI treats it as the same operation: move from coordinate A to coordinate B. The difference is the edge type (physical, quantum, temporal, etc.) and its cost.

---

### Coordinate diagram

Here's the complete coordinate space visualized:

```
═══════════════════════════════════════════════════════════════════════════════
                    THE TEGMARK HIERARCHY (Exclusive)
═══════════════════════════════════════════════════════════════════════════════

                        ┌──────────────────────┐
                        │  Level IV: Which     │
                        │  Mathematical        │
                        │  Structure?          │
                        │  (extreme cost)      │
                        └──────────┬───────────┘
                                   │
                        ┌──────────┴───────────┐
                        │  Level II: Which     │
                        │  Bubble Universe?    │
                        │  (very high cost)    │
                        └──────────┬───────────┘
                                   │
    ═══════════════════════════════╩══════════════════════════════════════════════

          AT THIS POINT, YOU ARE IN A SPECIFIC UNIVERSE.
    YOU MUST NOW LOCATE YOURSELF IN TWO ORTHOGONAL DIMENSIONS:

    ┌─────────────────────────────────┬──────────────────────────────────┐
    │   Level I: Which Spatial Region?│ Level III: Which Quantum Branch? │
    │   (within that universe)         │ (within that location)           │
    │   e.g., "Earth"                 │ e.g., "Q3" (Schrödinger branch) │
    │                                 │                                  │
    │   Galaxy → System → Planet       │ Outcome of quantum branching     │
    │   (local hierarchy below)        │ at this location and time        │
    └─────────────────────────────────┴──────────────────────────────────┘
                         ⊕ (orthogonal to each other)

    ═══════════════════════════════════╦═════════════════════════════════════════

          NOW, LAYER THESE FIVE INDEPENDENT EXPERIENCE AXES:

         ┌──────────────┬──────────────┬─────────────┬──────────────┬────────────────┐
         │  Timeline    │  Time        │  Simulation │ Consensus    │  Observer      │
         │  Which hist? │  When?       │  Depth?     │ Divergence?  │  Whose eyes?   │
         │  (cost 800)  │  (cost var)  │  (cost low) │  (cost low)  │  (cost low)    │
         │              │              │             │              │                │
         │  T1, T2, T3  │  timestamp   │  0, 1, 2... │  0, 1, 2...  │  human, bat,   │
         │  (branches)  │  (position)  │  (nesting)  │  (consensus  │  ai, ...       │
         │              │              │             │   to dream)  │                │
         └──────────────┴──────────────┴─────────────┴──────────────┴────────────────┘
              ⊕            ⊕               ⊕              ⊕              ⊕
         (all orthogonal, mix and match independently)

═══════════════════════════════════════════════════════════════════════════════

EXAMPLE COMPLETE COORDINATES:

  [MathA / BubbleX / Earth / Q3 / T2 / 2025-08-06 / sim=0 / cons=0 / human]
  "Earth in MathStructure A's Bubble X, quantum branch Q3, alternate timeline T2,
   August 6th 2025, base reality, consensus reality, perceived through human senses"

  [MathA / BubbleX / Earth / Q3 / T2 / 2025-08-06 / sim=0 / cons=1 / human]
  "Same location and timestamp, but in a DREAM (cons=1): the rules of waking physics
   don't apply. Flying is possible, physics bends, dream-logic governs."

  [MathA / BubbleX / Earth / Q3 / T2 / 2025-08-06 / sim=0 / cons=2 / human]
  "Same location and timestamp, but in PSYCHOSIS or SEVERE DELUSION (cons=2): the
   world is fundamentally distorted. Walls may breathe, thoughts seem external,
   causality is broken. More divergent from consensus than a dream."

  [MathA / BubbleX / Earth / Q3 / T2 / 2025-08-06 / sim=0 / cons=1-halluc / human]
  "Consensus divergence through HALLUCINATION: you're still in consensus reality
   (cons=1) but experiencing sensory events no one else perceives. You see things
   that aren't there, but the world's rules are still normal."

  [MathA / BubbleX / Earth / Q1 / T2 / 2024-01-01 / sim=1 / cons=2 / human]
  "Nested divergences: inside a simulation (sim=1), in a psychotic episode (cons=2),
   in a different quantum branch and timeline. Dream-logic stacks on simulation-logic."

═══════════════════════════════════════════════════════════════════════════════

LOCAL COORDINATES (nested within Level I region)

                        Your current region (e.g., "Earth")
                                   │
                    ┌──────────────┼──────────────┐
                    │              │              │
                 Galaxy A       Galaxy B       Galaxy C
                    │
                 System 1
                    │
                 Planet 1
                    │
          ┌─────────┼─────────┐
          │         │         │
       Country1  Country2  Country3
          │
        Region A
          │
        City 1
          │
      Location: Home
           ↓
      `[Earth/Country/Region/City/Home] >` (appears in CLI prompt)

═══════════════════════════════════════════════════════════════════════════════
```

**Reading this diagram:**

- The **top box** (Level IV → Level II) is the exclusive hierarchy: you pick a mathematical structure and a bubble universe, and you stay there.

- **Level I ⊕ Level III** are orthogonal: "Earth in Q3" is different from "Earth in Q1," but both are Earth. Shifting branches doesn't move you spatially.

- **The five experience axes** (Timeline, Time, Simulation, Consensus divergence, Observer) can all change independently. You can be at the same location in multiple different combinations of these. Consensus divergence is particularly flexible: you can dream while physically awake, hallucinate within a simulation, fall into psychosis within a timeline, or stack divergences (a delusion within a dream).

- **The local hierarchy** (Galaxy → Location) exists entirely within your Level I region and is the same across all quantum branches, timelines, and observer perspectives (only your perception changes).

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
    Consensus   int
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

The current implementation navigates the physical layers (planet through location), quantum, timeline, consensus, observer, simulation, universe (Tegmark Level II), and mathematical-structure (Tegmark Level IV) axes.

### Two kinds of movement: physical travel vs. contextual transitions

Every move in Onto is an edge in the same graph, but those edges come in exactly two families — and **everything except physical travel is contextual**:

- **Physical travel** — the `travel` command (and the final leg of `home`). This is the *only* kind of movement that is **not** contextual. It moves you around the *local hierarchy* (galaxy → system → planet → country → region → city → location) **without changing any reality axis**. You stay in the same quantum branch, timeline, simulation depth, consensus level, universe, structure, and observer. These physical edges are the only ones `travel` is allowed to cross.

- **Contextual transitions** — *everything else*. A contextual transition changes exactly one **non-spatial reality axis** while keeping your physical location fixed: it changes your *context*, not your whereabouts. This one label covers the whole range, from the cheap experience overlays right up to the exotic Tegmark levels: quantum branch (`shift`, Tegmark III), timeline (`jump`), consensus divergence (`drift` / `align`), simulation depth (`simulate`), observer / umwelt (`observe`), time (`time`), bubble universe (`universe`, Tegmark II), and mathematical structure (`structure`, Tegmark IV). Their exact costs are in the reference table below.

So to answer the obvious question directly: **yes** — changing your Tegmark bubble universe or mathematical structure is contextual, and so are observer, timeline, simulation, and consensus shifts. They all share the same machinery. Physical `travel` is the one thing that is *not* contextual.

"Contextual" is simply shorthand for "changes a reality-context axis, not a physical one." Whenever you make a contextual transition, Onto materializes coordinate-matched copies of every physical location reachable from where you stand, so ordinary local `travel` — and the route back — keeps working inside the new context.

#### How the two kinds are the same

- Both are ordinary graph edges with a cost; routing treats them identically — move from coordinate A to coordinate B.
- Both leave every axis they *don't* touch untouched.
- Both are reversible: physical travel by walking back, contextual transitions via a paired return edge (`shift back`, `universe back`, `align`, and so on).

#### How they differ

- Physical travel changes only the local hierarchy; a contextual transition changes only its single reality axis and never your local position.
- The two families never mix within one move: `travel` refuses any route that would cross a contextual edge, and each contextual command only ever crosses its own axis.
- Only contextual transitions rematerialize a coordinate-matched physical map in the destination context.

#### Contextual transition reference

Contextual transitions **stack**: each one changes only its own axis and preserves every other active context. That is what lets you be, for example, in an alternate timeline, inside a nested simulation, and viewed through a non-default observer all at once — nothing gets reset until you explicitly unwind it.

| Transition | Command (forward / back) | Axis changed | Cost (forward / reverse) |
|---|---|---|---|
| Quantum shift | `shift` / `shift back` | Quantum branch (Tegmark III) | 20 / 20 |
| Timeline jump | `jump` / `jump back` | Timeline (macro history) | 800 / 800 |
| Consensus drift | `drift` / `align` | Consensus divergence | 5 / 5 |
| Simulation entry | `simulate` / `simulate back` | Simulation depth | 10 / 50 |
| Observer shift | `observe <observer>` / `observe back` | Observer (umwelt) | 2 / 2 |
| Time travel | `time <RFC3339>` / `time back` | Time | 100 / 100 |
| Universe shift | `universe` / `universe back` | Bubble universe (Tegmark II) | 5000 / 5000 |
| Structure shift | `structure` / `structure back` | Mathematical structure (Tegmark IV) | 50000 / 50000 |

Every transition above preserves all other axes and resets nothing on its own —
simulation entry is the only one whose reverse costs more than its forward
(harder to leave than to enter). `home` is the explicit unwind operation: it
returns consensus, simulation, time, timeline, quantum, universe, and
mathematical-structure axes to their base levels and restores the default
observer before travelling physically back to the start location.

## Example CLI experience

The prompt reflects your current position, including non-default quantum, timeline, and consensus levels:

```text
[Leeds/Home] >
[Q1/Leeds/Station] >
[T2/Q1/Leeds/Station] >
[cons:1/Leeds/Station] >
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
universe               Shift forward to the next bubble universe (cost 5000)
universe back          Return to the previous bubble universe
structure              Shift forward to the next mathematical structure (cost 50000)
structure back         Return to the previous mathematical structure
simulate               Enter the next nested simulation layer (cost 10)
simulate back          Exit one simulation layer toward base reality (cost 50)
drift                  Enter the next consensus divergence (cost 5)
align                  Return one level toward shared consensus (cost 5)
observe <observer>     Change observer perspective (cost 2)
observe back           Return to the previous observer perspective (cost 2)
time <RFC3339>         Enter a temporal branch (cost 100)
time back              Return to the previous temporal branch (cost 100)
save                   Persist the current universe graph to disk
<number>               Take the corresponding numbered possible journey
cost                   Show travel cost information
help                   List all commands
exit                   Leave the CLI
```

## Game mode

The explorer can run as a small game. When enabled (the default for both `cmd/cli` and `cmd/web`), two rules apply on top of ordinary navigation:

- **Budget.** You start with a finite spending pool (`1000` by default). Every move spends its cost against the pool — physical travel costs the sum of its edges, and each contextual transition costs the amount shown in the [contextual transition reference](#contextual-transition-reference) (e.g. `shift` 20, `jump` 800). A move you cannot afford is **refused and spends nothing**, so the session is never left half-moved. The most expensive transitions (`universe`, `structure`) are out of reach on the starting budget, so the budget is felt.
- **Objective.** You are given a **quest chain**: an ordered list of target coordinates, each completed as its own **round trip**. The default chain has two waypoints on different reality axes — first the second quantum branch of home (`Q2`), then one simulation layer deep (`sim:1`). You reach the current waypoint, then return home to complete that objective before the next begins. Reaching a waypoint announces the return-home step ("Objective reached — now return home to complete it."); returning home banks it and names the next ("Objective 1 of 2 complete — next: reach …"); the last return home, once every objective is done, announces "You win!". A single-objective quest is just a chain of length one.
- **Rating.** Each quest has a **par** — the optimal cost, summed over every objective's round trip out and back (`140` for the default chain: two `shift`s out and two back for `Q2`, then `simulate` in and out for `sim:1`). On winning you earn a **1–3 star efficiency rating**: three stars for finishing at or under par, two within twice par, and one for any slower win. Par is shown from the start so you know the score to beat.

`where` shows your **budget remaining**, the **objective checklist** (each objective ticked once completed, with the current one marked as reached while you head home), your progress and **par** (plus your rating once won) at every step, and the browser HUD shows a budget chip (red when depleted) and an objective badge (per-objective progress `N/M`: reach the current waypoint → return home to complete it → won, with par and your star rating on the win).

Returning home is **always allowed**, even when the walk home costs more than the budget covers — an over-budget return completes rather than stranding you, and the remaining budget simply reports as empty (`0`) rather than going negative.

Game mode is off when no budget is set (a budget of `0` means unlimited spending and no objective), which is how the delivery-layer tests exercise navigation without the game rules.

## Current status

The app is functional. It includes:

- a command entrypoint in `cmd/cli` (plain REPL) and a browser-based Reality Map in `cmd/web` (interactive 3D graph with reachability colouring and per-transition edge styles — see [Reality Map (browser)](#reality-map-browser))
- the full command set listed under [Example CLI experience](#example-cli-experience)
- BFS-based graph routing across locations with travel modes (walk, rail, etc.)
- contextual navigation along all eight non-physical axes — quantum (`shift`), timeline (`jump`), bubble universe (`universe`), mathematical structure (`structure`), simulation depth (`simulate`), consensus divergence (`drift` / `align`), observer (`observe`), and time (`time`) — each with a paired reverse; see the [contextual transition reference](#contextual-transition-reference) for costs and behaviour
- each contextual transition creates coordinate-matched physical locations and return edges, so local travel and returning remain available throughout a branch
- `travel` rejects routes that cross any reality boundary — physical and contextual travel are kept separate
- `home` command: shows the full plan and estimated cost to unwind every contextual axis (restore observer, align consensus, exit simulation, and reverse temporal, timeline, quantum, universe, and structure shifts), then travel back to the start location before asking for confirmation
- cumulative journey cost tracked across the session and shown in `where` output and after every move
- an optional [game mode](#game-mode) (on by default): a finite budget that refuses unaffordable moves, an ordered multi-objective quest chain of round trips (reach each waypoint in order, returning home after each; the last return home wins), and a par-based 1–3 star efficiency rating on winning, surfaced in `where` and the browser HUD
- a full coordinate model covering mathematical structure, universe, timeline, quantum, simulation, consensus, physical location, observer, and time
- location and edge data loaded from `data/locations.json`, with a built-in fallback map
- `make validate-locations` checks saved location IDs, edge references, and physical reality boundaries before committing graph data
- `make toc` regenerates this README's table of contents
- interactive prompting to create new locations when arriving at a dead-end node
- universe graph mutations accumulate in memory during a session and are written to `data/locations.json` when you run `save` or exit cleanly from the plain CLI

## Architecture

Five layers, each importing only inward:

| Layer | Package path | Role |
|---|---|---|
| Domain | `internal/domain/` | Business rules, aggregates, entities, value objects, repository interfaces — no I/O |
| Application | `internal/application/commands/`, `internal/application/queries/` | Use-case orchestration; mutates state or reads it, never touches I/O |
| Application facade | `internal/application/facade/` | Delivery-agnostic entry point; dispatches input strings to commands/queries and formats results as strings |
| Infrastructure | `internal/infrastructure/` | JSON persistence — implements the domain repository interface |
| Bootstrap | `internal/bootstrap/` | Wires infrastructure to domain at startup; only `cmd/` entry points import this |
| Interface | `internal/interface/cli/`, `internal/interface/web/` | Thin delivery wrappers — a readline REPL and a browser Reality Map served over a JSON API; all delegate every command to the facade |

The domain defines the types and interfaces; every other layer depends on it, never the reverse. The interface packages depend only on the application facade, not on each other. See [docs/DDD.md](docs/DDD.md) for how DDD patterns are applied here.

## Getting started

**Natively** (requires Go):

```bash
make run
# or directly:
go run ./cmd/cli
```

### Reality Map (browser)

A browser-based delivery mechanism renders the universe as an interactive 3D
force-directed graph, backed by the same facade as the CLI:

```bash
make web
# or directly:
go run ./cmd/web
```

Then open the printed address (default `http://localhost:8090`; override with
`ONTO_WEB_ADDR`). The map is a thin client over a small JSON API (`/api/state`
and `/api/execute`) — all navigation logic stays in the facade and domain.

**Reading the map** — nodes and edges are colour-coded so it answers "where can
I go?" at a glance:

| Node colour | Meaning |
|---|---|
| 🟢 green | you are here |
| 🔵 blue | reachable now by ordinary `travel` — click to go |
| 🩷 pink | a different quantum branch — needs `shift`, not travel |
| ⚫ grey | exists, but no physical route from here |

Reachability is computed on the backend (`navigation.ReachableFrom`) and exposed
per node, so the colouring reflects real routing rather than a client-side guess.
Edges are **solid blue** for ordinary physical travel; every contextual
transition (quantum, timeline, universe, simulation, consensus, observer, time,
structure) gets its own hue and dash pattern, keyed in the top-right legend. When
a transition lands, a colour-matched ripple radiates from your current location.

**Controls** — click a reachable (blue) node to travel there (the cursor turns to
a pointer over reachable nodes); scroll to zoom; drag to pan; **Shift**+drag to
rotate the graph in 3D. The right-hand panel mirrors every command as a button,
plus free-text entry, an observer picker, and a time picker.

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

Validate the saved universe graph at any time:

```bash
make validate-locations
```

Environment variables can be set in `.env` (copy `.env.example` to get started) or overridden inline:

| Variable | Default | Description |
|---|---|---|
| `ONTO_DATA_FILE` | `data/locations.json` | Path to the universe JSON file |
| `ONTO_START_LOCATION` | `home` | Location ID the app starts at |
| `ONTO_WEB_ADDR` | `:8090` | Listen address for the browser Reality Map (`make web`) |
| `ONTO_GAME` | `1` (on) | [Game mode](#game-mode) toggle; set to `0`/`false`/`off` to disable the budget and objective |
| `ONTO_BUDGET` | `1000` | Starting budget when game mode is on; a positive value overrides the default pool |

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

**Continuous Integration & Tagging:**

Pushes to `main` automatically run unit tests, linting, and saved-universe validation via GitHub Actions. Upon success, semantic version tags (`vX.Y.Z`) are automatically generated and pushed based on [Conventional Commit](https://www.conventionalcommits.org/) messages: `BREAKING CHANGE` triggers a major bump, `feat:` triggers a minor bump, and `fix:` triggers a patch bump. Other commits use the configured patch fallback.

## Roadmap

1. ~~Implement a simple local-world graph for Earth.~~ ✓
2. ~~Add location lookup and basic routing.~~ ✓
3. ~~Track cumulative journey cost across the session.~~ ✓
4. ~~Support quantum transitions.~~ ✓ (`shift` / `shift back`)
5. ~~Support timeline transitions.~~ ✓ (`jump` / `jump back`)
6. ~~Add `home` command to return to start, unwinding all branches with confirmation.~~ ✓
7. ~~Support consensus divergence transitions.~~ ✓ (`drift` / `align`)
8. ~~Support universe transitions (higher-cost exotic modes).~~ ✓ (`universe` / `universe back`)
9. ~~Support mathematical-structure transitions (Tegmark Level IV).~~ ✓ (`structure` / `structure back`)
10. ~~Expand navigation with simulation depth.~~ ✓ (`simulate` / `simulate back`)
11. Evolve the CLI into a true reality navigator.

## Notes

This project is philosophical in spirit, but practical in implementation. It treats navigation as a general problem: move from one place to another, whether that place is physical, historical, simulated, or otherwise.

## Design notes

For the hierarchical reality model, coordinate system, vector costs, and future CLI progression, see [DESIGN.md](docs/DESIGN.md).

## Why the name "Onto"

The name "Onto" is chosen deliberately for two related reasons:

- Ontological: the prefix "onto-" (as in ontology) points to being and existence — the project models different modes of existence (what exists) as layered coordinates.
- Onto-as-motion: colloquially "onto" suggests movement (putting something onto something else). The CLI is about moving "onto" other places, timelines, and perspectives — so the name also reads as an action.

The dual meaning signals the project's intent: it's both a tool for describing kinds of existence (an ontology) and a navigator for moving through them. The UI reflects that by showing your current coordinate and making movement (routing/travel) the central operation.
