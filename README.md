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
- [Cloud deployment](#cloud-deployment)
- [Roadmap](#roadmap)
- [Notes](#notes)
- [Design notes](#design-notes)
- [Why the name "Onto"](#why-the-name-onto)


## Core idea

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
**Level I is infinite Hubble volumes, and every one of them is another universe** — not in the Level II sense of different physics, but in the plainest sense: each Hubble volume is a whole observable-universe-sized region of *our* space, running the *same* physical laws, constants, and particle types, but holding a different arrangement of matter and therefore a different history. There are infinitely many of them, so every history that *can* happen already plays out in one of them somewhere. Onto addresses *which* Hubble volume you occupy on the [`Timeline`](#timeline) axis.

Distant regions of our own universe beyond the observable horizon. Same physical laws, same constants, same particle types — just unreachably far away, with different initial arrangements of that matter. Space is (effectively) infinite but the ways a finite volume of particles can be arranged are not (a Hubble volume — the size of our observable universe, radius ~4.6×10²⁶ m — holds ~10¹¹⁸ particles, and the pigeonhole principle guarantees every arrangement of them recurs), so every arrangement recurs somewhere: past roughly 10^(10^29) metres, and thus that many Hubble volumes, sits an identical copy of you; past roughly 10^(10^115) metres sits an identical copy of the entire observable universe. Every distance short of that holds a *near*-match — including copies of Earth where history unfolded differently, a war lost instead of won, a coin landing the other way. Crucially, this is all still ordinary space and ordinary travel, just at a distance no engine we model can cover except the dedicated `jump`. Local travel (walking, driving, flying) operates within reach of any physical mode. `travel` is the command. (For the scale and reasoning behind these numbers, see Max Tegmark's ["Parallel Universes"](https://www.scientificamerican.com/article/parallel-universes/), *Scientific American*, 2003.)

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
A macroscopic divergence point — a war, a technology, a civilisation that went differently. This isn't a mechanism of its own; it's Tegmark Level I made navigable, and **it is still physical travel, not a different kind of reality**. In the coordinate model the `Timeline` axis *is* the Hubble-volume address: each distinct timeline value names one of Level I's infinitely many Hubble volumes — an observable-universe-sized region of our own space with the same physics but a different history. A timeline is another real Level I region, one that happens to share your local geography (same Earth, same city) but sits an astronomical distance away — many Hubble volumes out — in that (effectively) infinite space, where the accumulated distance meant history had room to diverge. `jump` is a journey to one of those regions — you don't rewrite history, you travel to the address where it already went the other way. No amount of speed gets you there, though: even `warp` still crosses every metre of intervening space, and the distance is too vast for that to ever finish. What makes `jump` a separate command instead of a very long `travel` is a dedicated jump drive that threads a wormhole straight to the target Hubble volume — a shortcut through spacetime, not a faster crossing of it. `jump back` returns. Multiple timelines can exist within a single quantum branch, since Level I addresses and Level III branches are orthogonal.

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
   - Which timeline — i.e. which Level I **Hubble volume** (a journey to an observable-universe-sized region of our own space where history diverged)?
   - Which time (temporal position)?
   - Which simulation depth (base or nested, computational)?
   - Which consensus divergence (consensus reality or divergent state)?
   - Which observer (perceptual frame)?

**How they interact:**

- **Levels IV and II are permanent anchors.** Moving between them is expensive and rare. Once you commit to a bubble universe, you stay within it unless you pay the extreme cost.

- **Level I and Level III are orthogonal coordinates of the same space.** Shifting quantum branches doesn't move you physically; moving physically doesn't change your quantum branch. A "position" requires both: "Earth, Quantum branch Q3" is different from "Earth, Quantum branch Q1," but both are still Earth.

- **Timeline, Time, Simulation, Consensus divergence, and Observer are independent overlays.** You can:
  - Shift quantum branches without changing your timeline
  - Jump to a different timeline (a Level I journey to a region sharing your local geography but not your history) without changing your quantum branch
  - Move through time without leaving your timeline
  - Enter a simulation while staying in the same branch, location, and timeline (the simulation contains its own nested Level I/III structure)
  - Drift into a divergent state (dream, delusion, hallucination, psychosis) without leaving your simulation, location, or timeline (you can dream while awake, hallucinate while in a simulation, be delusional within a dream)
  - Change observer without any physical or temporal movement

- **Local coordinates (galaxy, planet, city, location) are always nested within your current Level I region**, regardless of which quantum branch, timeline, time, or simulation depth you occupy.

**Cost hierarchy (cheapest to most expensive):**
- **Cheapest**: physical travel — walking/driving within one region (1–2 σ)
- **Very cheap**: observer shifts — changing perceptual perspective (2 σ), imagination transitions — entering/exiting dreams, stories (5 σ)
- **Low**: simulation boundary crossings (10 σ in / 50 σ out), quantum shifts — Level III branches (20 σ)
- **Moderate**: time travel (100 σ)
- **High**: timeline jumps — threading a wormhole to a distant Hubble volume with diverged history (800 σ)
- **Very expensive**: universe shifts — Level II bubbles (5000 σ)
- **Extreme**: mathematical structure transitions — Level IV (50000 σ)

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
         │  (cost 800)  │  (cost 100)  │  (cost low) │  (cost low)  │  (cost low)    │
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

- **Timeline is Level I, abstracted — and still just travel.** T1, T2, T3 aren't a separate kind of branch — they're labels for distinct, real Level I regions that happen to share your local geography. Level I is (effectively) infinite, and the pigeonhole principle guarantees every historical divergence already exists somewhere out there — identical copies of you within ~10^(10^29) m, identical copies of the whole observable universe within ~10^(10^115) m. `jump` is shorthand for travelling to one of the nearer, near-identical regions; it's grouped with the exotic transitions only because reaching that distance takes a jump drive that threads a wormhole there, not because it's a different kind of reality.

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

The current implementation navigates the physical layers (planet through location), quantum, timeline (Tegmark Level I — the Hubble-volume coordinate), consensus, observer, simulation, universe (Tegmark Level II), and mathematical-structure (Tegmark Level IV) axes. The `Timeline` field is the **Hubble-volume** coordinate: which of Level I's infinitely many observable-universe-sized regions (same physics, different history) you occupy, changed by `jump`.

### Two kinds of movement: physical travel vs. contextual transitions

Every move in Onto is an edge in the same graph, but those edges come in exactly two families — and **everything except physical travel is contextual**:

- **Physical travel** — the `travel` command (and the final leg of `home`). This is the *only* kind of movement that is **not** contextual. It moves you around the *local hierarchy* (galaxy → system → planet → country → region → city → location) **without changing any reality axis**. You stay in the same quantum branch, timeline, simulation depth, consensus level, universe, structure, and observer. These physical edges are the only ones `travel` is allowed to cross.

- **Contextual transitions** — *everything else*. A contextual transition changes exactly one **non-spatial reality axis** while keeping your physical location fixed: it changes your *context*, not your whereabouts. This one label covers the whole range, from the cheap experience overlays right up to the exotic Tegmark levels: quantum branch (`shift`, Tegmark III), timeline (`jump`, a Tegmark I journey), consensus divergence (`drift` / `align`), simulation depth (`simulate`), observer / umwelt (`observe`), time (`time`), bubble universe (`universe`, Tegmark II), and mathematical structure (`structure`, Tegmark IV). Their exact costs are in the reference table below.

So to answer the obvious question directly: **yes** — changing your Tegmark bubble universe or mathematical structure is contextual, and so are observer, timeline, simulation, and consensus shifts. They all share the same machinery. Physical `travel` is the one thing that is *not* contextual.

**Timeline is the one exception worth calling out.** Quantum, universe, and mathematical-structure shifts are contextual for a *conceptual* reason — they are not journeys through space at all (a different outcome, a different physics, a different formal system). Timeline is contextual only for an *engineering* reason: `jump` is, physically, still travel through ordinary space to a real, distant Level I region — it is just travel across a distance so extreme (many Hubble volumes) that no modeled physical mode, not even `warp`, can make the trip in finite time (you would still be crossing every metre of intervening space). It is grouped with the contextual commands because reaching it takes a dedicated jump drive that threads a wormhole straight to the target Hubble volume — a shortcut through spacetime, not a faster crossing of it — not because the destination is a different kind of reality.

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
| Observer shift | `observe <observer>` / `observe back` | Observer (umwelt) | 2 / 2 |
| Consensus drift | `drift` / `align` | Consensus divergence | 5 / 5 |
| Simulation entry | `simulate` / `simulate back` | Simulation depth | 10 / 50 |
| Quantum shift | `shift` / `shift back` | Quantum branch (Tegmark III) | 20 / 20 |
| Time travel | `time <RFC3339>` / `time back` | Time | 100 / 100 |
| Timeline jump | `jump` / `jump back` | Timeline / Hubble volume (Tegmark I — selects a different Level I Hubble volume; a jump drive threads a wormhole to it) | 800 / 800 |
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
docker compose -f deploy/cli/docker-compose.yml --project-directory . run --rm cli
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

## Cloud deployment

Onto can run as a split cloud stack — the static SPA on **S3**, the Go API on
**ECS/Fargate** behind an **ALB**, fronted by **Route53** on the domain
`onto.world` — described entirely in Terraform under `deploy/terraform/`:

```
onto.world      → SPA       (S3, via CloudFront on AWS / nginx edge locally)
api.onto.world  → JSON API  (ALB → ECS task running cmd/web)
```

The SPA and API live on different origins, so the API sends permissive CORS
headers (`internal/interface/web`) and the SPA reads its API base from a
generated `config.js`. One shared module drives two environments:

```
deploy/terraform/
├── modules/onto/   # SHARED — S3, ECS, ALB, Route53 (all parameterised)
├── envs/local/     # MiniStack: endpoint overrides + run-task/edge shims
└── envs/aws/       # real AWS: HTTPS/ACM, CloudFront, IAM role, security groups
```

Everything declarative lives in the shared module; the two environments differ
only where MiniStack and AWS genuinely diverge:

| Concern | Local (MiniStack) | AWS |
|---|---|---|
| Provider | endpoints → `localhost:4566`, fake creds | real credentials |
| ECS runtime | `run-task` + register-target shim (its service is phantom) | native `aws_ecs_service` |
| DNS / TLS | nginx edge, plain HTTP | existing Route53 zone, HTTPS via ACM (80→443) |
| SPA delivery | edge proxies S3 path-style | CloudFront + Origin Access Control |
| Network / IAM | default VPC, none | default VPC + security groups + Fargate role |

### Local (MiniStack)

Requires Docker and Terraform. [MiniStack](https://ministack.org) (a local AWS
emulator) plus a small nginx edge stand in for the two things AWS gives for free
that it can't: a DNS resolver and host-based routing. A
[StackPort](https://stackport.cloud) UI is bundled in for browsing what MiniStack
holds (S3 objects, the ECS task, the ALB, the Route53 zone).

`make ministack` is the whole thing in **one command** — it maps the domains into
`/etc/hosts` if they aren't already (sudo), starts MiniStack + StackPort + the
edge, builds the API image, and `terraform apply`s the S3/ECS/ALB/Route53 stack:

```bash
make ministack        # one command: hosts + up + build + apply
# http://onto.world/              — SPA + API (via the nginx edge)
# http://api.onto.world/api/state — JSON API directly
# http://localhost:8080/          — StackPort (browse MiniStack's resources)
# http://localhost:4566           — MiniStack emulator endpoint
make ministack-down   # terraform destroy + stop everything
```

The `/etc/hosts` step needs sudo, and it's the first thing `make ministack` does —
if it fails (no sudo rights, or you cancel the prompt) the run stops before
anything starts. It only affects browser access to the two split-domain names: add
them yourself (`127.0.0.1 onto.world` and `127.0.0.1 api.onto.world`) or re-run
`make ministack-hosts`. StackPort (`:8080`) and MiniStack (`:4566`) go through
`localhost` and work regardless, and the single-origin dev path (`make web` on
`http://localhost:8090`) needs no `/etc/hosts` changes at all.

### Real AWS

Scope is deliberately minimal: it consumes the account's default VPC/subnets and
creates only the app resources, two security groups, and a Fargate execution
role. The **only manual step is registering `onto.world`** in the Route53
console — that auto-creates the hosted zone, which Terraform then consumes (via a
data source) and issues + DNS-validates both ACM certificates against (a regional
cert for the ALB and, since CloudFront requires it, a us-east-1 cert for the
apex). After registering, supply just an ECR image and apply:

```bash
cp deploy/terraform/envs/aws/terraform.tfvars.example \
   deploy/terraform/envs/aws/terraform.tfvars      # set image
make tf-aws-init && make tf-aws-plan && make tf-aws-apply
```

Everything after the domain purchase is hands-off. The first apply can take a few
minutes (ACM waiting on DNS validation, and CloudFront distribution creation is
slow by nature).

### Deployment checklist — remaining work to go live

The Terraform under `deploy/terraform/envs/aws/` already describes the whole
runtime stack (S3 + CloudFront + Origin Access Control, ALB + ECS/Fargate
service, both ACM certificates, the Route53 records, the security groups, and the
`onto-ecs-execution` Fargate role). What's left splits into a **one-time
bootstrap** you do by hand and the **ongoing build + deploy** that GitHub Actions
runs on every merge to `main`.

**1. Bootstrap once (manual)**

The chicken-and-egg pieces CI can't create before it can authenticate or store
state:
- [ ] **AWS account + region.** Have an account; `region` defaults to `eu-west-1`.
- [ ] **Domain.** Register `onto.world` in the **Route53 console** (inherently
      manual). Registration auto-creates the hosted zone that
      `data "aws_route53_zone" "onto"` consumes and ACM validates against. Using a
      different domain? Register it and set `domain` / `api_domain` /
      `api_base_url`; if it's registered outside Route53, create a hosted zone and
      point the registrar's NS records at it.
- [ ] **Remote Terraform state.** Local state won't work from ephemeral runners —
      create an S3 state bucket + DynamoDB lock table and add a `backend "s3"`
      block to `providers.tf`.
- [ ] **GitHub OIDC + deploy role.** Add a GitHub OIDC provider and an IAM role
      trusted for this repo, with permissions for S3, CloudFront, ACM, Route53,
      ECS, EC2 (VPC/subnet reads + security groups), ELB, ECR, and IAM (it creates
      `onto-ecs-execution`). Actions assumes it via
      `aws-actions/configure-aws-credentials` — no long-lived keys.
- [ ] **ECR repository.** CI pushes here before the first apply, so it must exist
      first. Terraform does **not** create it today — either add an
      `aws_ecr_repository` resource or create `onto-api` by hand in bootstrap.

**2. Ongoing — GitHub Actions (build + deploy on merge to `main`)**

The current workflow (`.github/workflows/ci.yml`) only tests, lints, and tags.
Add a `deploy` job, gated on the existing `test` / `lint` jobs, that:
- [ ] assumes the deploy role via OIDC (`aws-actions/configure-aws-credentials`
      with `role-to-assume`);
- [ ] logs into ECR (`aws-actions/amazon-ecr-login`), builds `--platform
      linux/amd64` (Fargate's default arch), and pushes the image tagged with the
      commit SHA;
- [ ] applies Terraform against `envs/aws`, passing the image as
      `-var image=<repo>:<sha>` (so the tag isn't committed to `terraform.tfvars`)
      — **or** just rolls the service with
      `aws ecs update-service --force-new-deployment` when only the image changed;
- [ ] on SPA changes, `aws s3 sync`s the static assets and issues a CloudFront
      invalidation (`/*`).
- [ ] Store non-secret config as repo **variables** (account ID, region, role ARN,
      ECR repo); OIDC removes the need for any AWS keys as secrets.

**3. First apply**
- [ ] The very first `terraform apply` (run locally against the new remote
      backend, or as a one-off CI run) is slow — ACM waits on DNS validation and
      CloudFront creation takes minutes. Confirm the `urls` output resolves over
      HTTPS.

**4. Local fallback (no CI)**

Everything above can still be driven by hand — useful for the first apply or
break-glass:
- [ ] Make credentials available via the standard chain (`aws configure` /
      `AWS_PROFILE`); `providers.tf` hard-codes none.
- [ ] Build + push the image (`docker build --platform linux/amd64 -f
      deploy/ministack/Dockerfile.api -t
      <acct>.dkr.ecr.<region>.amazonaws.com/onto-api:<tag> .`, then `docker
      push`), set `image` in `terraform.tfvars`, and run `make tf-aws-init &&
      make tf-aws-plan && make tf-aws-apply`.

**5. Operational odds and ends**
- [ ] No `make tf-aws-destroy` target yet — tear down with
      `cd deploy/terraform/envs/aws && terraform destroy` (add a target if useful).
- [ ] Budget: the ALB, CloudFront, the Fargate task, and the Route53 hosted zone
      bill continuously even at idle; ACM certificates are free.
- [ ] Before real traffic, consider raising `desired_count` and adding ECS
      autoscaling (defaults to a single task).

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

For the design reasoning behind the reality model — the origin of the hierarchical
concept, the `onto://` address format, and the coordinate/cost design — see
[DESIGN.md](docs/DESIGN.md). For how those concepts map onto the code, see
[DDD.md](docs/DDD.md).

## Why the name "Onto"

The name "Onto" is chosen deliberately for two related reasons:

- Ontological: the prefix "onto-" (as in ontology) points to being and existence — the project models different modes of existence (what exists) as layered coordinates.
- Onto-as-motion: colloquially "onto" suggests movement (putting something onto something else). The CLI is about moving "onto" other places, timelines, and perspectives — so the name also reads as an action.

The dual meaning signals the project's intent: it's both a tool for describing kinds of existence (an ontology) and a navigator for moving through them. The UI reflects that by showing your current coordinate and making movement (routing/travel) the central operation.
