// Onto Reality Map — pure, framework-free view logic.
//
// This module holds the parts of the front-end that have no dependency on the
// DOM, the canvas, or wall-clock time: the reality-axis defaults, per-mode edge
// styling, transition detection, node colouring, and the 3D projection maths.
// app.js imports these and wires them to the browser; logic.test.mjs imports the
// same functions and exercises them under Node's built-in test runner, so the
// interesting maths is covered without booting a browser.

export const DEFAULTS = {
  Mathematics: "Classical",
  Universe: "Origin",
  Timeline: "Prime",
  Quantum: "Q0",
  Observer: "Human",
};

// Per-mode edge appearance. All physical modes share a solid blue; each reality
// transition gets its own hue and dash signature, so a quantum hop reads
// differently from a timeline jump or an observer shift. The same RGB triples
// tint the ripple animation played when you make that transition.
export const MODE_STYLE = {
  walk:       { rgb: "110,168,255", dash: [] },
  cycle:      { rgb: "110,168,255", dash: [] },
  drive:      { rgb: "110,168,255", dash: [] },
  rail:       { rgb: "110,168,255", dash: [] },
  flight:     { rgb: "110,168,255", dash: [] },
  orbit:      { rgb: "110,168,255", dash: [] },
  warp:       { rgb: "110,168,255", dash: [] },
  quantum:    { rgb: "255,95,176",  dash: [4, 4] },
  timeline:   { rgb: "255,176,64",  dash: [7, 4] },
  universe:   { rgb: "178,123,255", dash: [2, 5] },
  simulation: { rgb: "87,226,165",  dash: [9, 4] },
  observer:   { rgb: "94,214,255",  dash: [2, 3] },
  consensus:  { rgb: "255,99,99",   dash: [6, 3, 2, 3] },
  time:       { rgb: "240,208,96",  dash: [11, 4] },
  math:       { rgb: "120,220,200", dash: [3, 3] },
};
export const DEFAULT_MODE_STYLE = { rgb: "255,95,176", dash: [4, 4] };
export function modeStyle(mode) { return MODE_STYLE[mode] || DEFAULT_MODE_STYLE; }

// Edge-spring rest lengths for the force layout (app.js tick). Physical edges
// keep a short rest so same-reality locations cluster tightly; reality-transition
// edges get a much longer rest, so a shift/jump/simulate visibly pushes its child
// sub-graph away from the parent and nesting emerges from the layout itself. A
// mode counts as a transition when its style is dashed (physical modes are the
// only solid ones in MODE_STYLE), so unknown modes fall on the transition side.
export const REST_PHYSICAL = 50;
export const REST_TRANSITION = 110;
export function edgeRestLength(mode) {
  return modeStyle(mode).dash.length ? REST_TRANSITION : REST_PHYSICAL;
}

// Edge cost weighting for the base map. Mode owns each edge's hue and dash (see
// MODE_STYLE); cost owns its heft, so an expensive exotic jump *looks* expensive
// at rest — thick and opaque — while a cheap step reads thin and faint. Costs
// span orders of magnitude (an observe shift is ~2 σ, a structure jump ~50000),
// so a linear scale would flatten everything; edgeWeight maps cost through a log
// curve clamped to [EDGE_COST_MIN, EDGE_COST_MAX], normalised to [0,1], then
// interpolated onto the width and alpha ranges below. Cost at or under the floor
// (including 0/undefined) sits at the thin, faint end; cost at or over the cap
// saturates at the thick, opaque end, so one runaway edge can't blow out the map.
export const EDGE_COST_MIN = 1;
export const EDGE_COST_MAX = 50000;
export const EDGE_WIDTH_MIN = 0.75;
export const EDGE_WIDTH_MAX = 4;
export const EDGE_ALPHA_MIN = 0.2;
export const EDGE_ALPHA_MAX = 0.6;
export function edgeWeight(cost) {
  const c = Math.min(EDGE_COST_MAX, Math.max(EDGE_COST_MIN, cost || 0));
  const t = Math.log(c / EDGE_COST_MIN) / Math.log(EDGE_COST_MAX / EDGE_COST_MIN);
  return {
    width: EDGE_WIDTH_MIN + t * (EDGE_WIDTH_MAX - EDGE_WIDTH_MIN),
    alpha: EDGE_ALPHA_MIN + t * (EDGE_ALPHA_MAX - EDGE_ALPHA_MIN),
  };
}

// Each reality transition plays its own character-matched animation rather than
// a single generic ripple, so the *feel* of crossing an axis matches its nature.
// kind selects the canvas renderer in app.js (drawEffects); the colour still
// comes from modeStyle so the key, edges, and effect never drift apart. Physical
// travel and any unknown mode fall back to the plain expanding ripple.
export const EFFECT_KIND = {
  universe:   "fade",          // reality dissolves to black and re-forms as a new bubble
  quantum:    "superposition", // jittering, flickering probability ghosts collapse
  timeline:   "sweep",         // a bright bar jumps sideways along the timeline
  simulation: "glitch",        // the simulation re-renders in torn scanlines
  observer:   "blink",         // eyelids close and reopen — you wake as someone else
  consensus:  "shockwave",     // agreed-upon reality wobbles outward as it drifts
  time:       "clock",         // a clock hand sweeps the dial to the new moment
  math:       "grid",          // the underlying mathematical structure flashes into view
};
export const DEFAULT_EFFECT_KIND = "ripple";

// How long each kind runs (ms). The fade and blink are longer because they take
// over the whole screen; the ripple keeps its original 900ms so plain travel is
// unchanged.
export const EFFECT_DURATION = {
  fade: 1400,
  superposition: 1000,
  sweep: 850,
  glitch: 900,
  blink: 1100,
  shockwave: 1200,
  clock: 1300,
  grid: 1000,
  ripple: 900,
};

export function effectSpec(mode) {
  const kind = EFFECT_KIND[mode] || DEFAULT_EFFECT_KIND;
  return { kind, duration: EFFECT_DURATION[kind] || EFFECT_DURATION.ripple };
}

// Each reality transition also gets its own synthesised cue, sound-designed in the
// spirit of Star Wars / Alien — organic and grungy rather than clean and
// synthetic. The character comes from abused, inharmonic sources: ring modulation
// for clangorous metal, FM at non-integer ratios for struck-metal and creature
// timbres, per-voice distortion for grit, resonant band-passed noise for breath
// and wind, and a little random pitch jitter so no two plays are identical. The
// player (app.js) adds the shared space: a convolution reverb with pre-delay,
// master saturation, and a compressor for glue. No audio assets.
//
// A voice is one source with an envelope:
//   type              "sine" | "triangle" | "sawtooth" | "square" | "noise"
//                     ("noise" is filtered white noise — whooshes, breath, air)
//   freq, freqEnd     pitch start and optional glide target (Hz); ignored for noise
//   gain              peak level 0–1 (scaled by the player's master gain)
//   attack, release   envelope times (s); a long attack makes a reverse-style swell
//   delay             optional start offset (s) so voices stack or arrive late
//   detune            optional detune (cents) for width/beating between layers
//   jitter            optional random detune (± cents), re-rolled each play, for
//                     organic imperfection (pitched voices only)
//   pan               optional stereo position (−1 left … +1 right)
//   drive             optional distortion 0–1 — grit/saturation ("grunge")
//   ring              optional ring modulation { freq, depth? } — metallic, alien
//                     sidebands; depth 1 is true ring mod, below 1 blends to tremolo
//   fm                optional frequency modulation { ratio, depth } — a modulator
//                     at freq×ratio bends pitch by ±depth Hz; non-integer ratios
//                     give inharmonic, struck-metal / creature timbres
//   filter            optional sweep { type?, freq, freqEnd, q } — type defaults to
//                     "lowpass"; "highpass"/"bandpass" shape noise into whooshes
//   lfo               optional modulation { target: "gain" | "pitch" | "filter",
//                     freq, depth } — gain depth 0–1 (tremolo/flicker), pitch depth
//                     in cents (wobble), filter depth in Hz (organic sweep)
// Physical/unknown modes never reach here (only detectTransition arms a sound),
// but DEFAULT_SOUND keeps it safe.
export const SOUND_SPEC = {
  // Reality dissolves and re-forms — the biggest cue. A driven noise swell tears
  // in, then a distorted, ring-modulated saw cluster booms like a groaning hull
  // under a gritty sub drop, and an FM'd shimmer clangs away into the reverb.
  universe: [
    { type: "noise", gain: 0.4, attack: 1.0, release: 0.6, drive: 0.5, filter: { type: "highpass", freq: 200, freqEnd: 5000, q: 0.7 } },
    { type: "sine",     freq: 120, freqEnd: 26, gain: 0.95, attack: 0.03, release: 2.8, drive: 0.4, filter: { freq: 500, freqEnd: 55, q: 6 }, delay: 0.85 },
    { type: "sawtooth", freq: 55,  freqEnd: 52, gain: 0.42, attack: 0.08, release: 2.7, detune: -8, jitter: 12, drive: 0.55, ring: { freq: 47, depth: 0.35 }, filter: { freq: 220, freqEnd: 1300, q: 4 }, delay: 0.85 },
    { type: "sawtooth", freq: 82.5, freqEnd: 80, gain: 0.3, attack: 0.08, release: 2.6, detune: 8, jitter: 12, drive: 0.5, delay: 0.88 },
    { type: "sine",     freq: 900, freqEnd: 240, gain: 0.2, attack: 0.2, release: 1.9, fm: { ratio: 2.7, depth: 220 }, delay: 0.95, pan: 0.3 },
  ],
  // Superposed probability ghosts flicker and collapse: two ring-modulated,
  // jittering voices beat against each other under a fast gain flicker, panned
  // wide, with an FM sparkle and a gritty band-passed noise hissing through.
  quantum: [
    { type: "sine", freq: 660, freqEnd: 990,  gain: 0.4, attack: 0.01, release: 0.9, detune: 11, jitter: 15, pan: -0.4, ring: { freq: 140, depth: 0.7 }, lfo: { target: "gain", freq: 19, depth: 0.6 } },
    { type: "sine", freq: 668, freqEnd: 1000, gain: 0.4, attack: 0.01, release: 0.9, detune: -11, jitter: 15, pan: 0.4, delay: 0.02, ring: { freq: 150, depth: 0.7 }, lfo: { target: "gain", freq: 23, depth: 0.6 } },
    { type: "sine", freq: 1320, freqEnd: 1980, gain: 0.16, attack: 0.02, release: 0.7, fm: { ratio: 3.3, depth: 400 }, delay: 0.05 },
    { type: "noise", gain: 0.12, attack: 0.02, release: 0.6, drive: 0.3, filter: { type: "bandpass", freq: 2600, freqEnd: 5200, q: 4 } },
  ],
  // A bright bar jumps sideways: a driven band-passed noise whoosh rises and
  // sweeps, a gritty jittering saw riser climbs with it, then an FM'd sub thump
  // lands as the new timeline arrives.
  timeline: [
    { type: "noise", gain: 0.34, attack: 0.28, release: 0.34, drive: 0.5, filter: { type: "bandpass", freq: 500, freqEnd: 6000, q: 1.4 } },
    { type: "sawtooth", freq: 200, freqEnd: 1200, gain: 0.4, attack: 0.02, release: 0.6, detune: 6, jitter: 10, drive: 0.55, filter: { freq: 300, freqEnd: 5000, q: 3 } },
    { type: "sine", freq: 170, freqEnd: 74, gain: 0.6, attack: 0.005, release: 0.55, drive: 0.4, fm: { ratio: 1.5, depth: 80 }, delay: 0.32 },
  ],
  // The simulation re-renders in torn scanlines: a heavily driven, ring-modulated
  // square stutters via a fast gain flicker, a hard-distorted square glitches
  // against it over a sub, and a gritty high-passed noise crackles like tearing.
  simulation: [
    { type: "square", freq: 140, freqEnd: 88, gain: 0.36, attack: 0.005, release: 0.6, drive: 0.6, ring: { freq: 90, depth: 0.5 }, filter: { freq: 900, freqEnd: 180, q: 4 }, lfo: { target: "gain", freq: 32, depth: 0.9 } },
    { type: "square", freq: 222, freqEnd: 176, gain: 0.24, attack: 0.005, release: 0.4, detune: 18, jitter: 20, drive: 0.7, delay: 0.06, pan: 0.25 },
    { type: "sine",   freq: 90,  freqEnd: 50, gain: 0.5, attack: 0.005, release: 0.55, drive: 0.35 },
    { type: "noise",  gain: 0.12, attack: 0.005, release: 0.5, drive: 0.4, filter: { type: "highpass", freq: 3000, freqEnd: 4200, q: 0.7 }, lfo: { target: "gain", freq: 60, depth: 0.9 } },
  ],
  // Eyelids close and reopen — you wake as someone else: a breathy noise formant
  // swells in with a slow resonant filter sweep, an FM'd, jittering, vibrato'd
  // body settles over a warm undertone, and an airy top drifts.
  observer: [
    { type: "noise", gain: 0.18, attack: 0.5, release: 0.7, drive: 0.25, filter: { type: "bandpass", freq: 700, freqEnd: 1600, q: 6 }, lfo: { target: "filter", freq: 0.7, depth: 500 } },
    { type: "sine", freq: 520, freqEnd: 300, gain: 0.5, attack: 0.06, release: 1.1, jitter: 14, fm: { ratio: 1.4, depth: 60 }, filter: { freq: 1200, freqEnd: 480, q: 2 }, delay: 0.25, lfo: { target: "pitch", freq: 5, depth: 14 } },
    { type: "triangle", freq: 260, freqEnd: 180, gain: 0.34, attack: 0.1, release: 1.2, jitter: 10, delay: 0.25 },
    { type: "sine", freq: 1040, freqEnd: 700, gain: 0.12, attack: 0.12, release: 0.9, delay: 0.3, pan: -0.3 },
  ],
  // Agreed reality wobbles outward as it drifts: a driven low noise shockwave
  // bursts, a ring-modulated, jittering body groans and wobbles over a gritty
  // sub, and an FM'd overtone rings out — a hull flexing.
  consensus: [
    { type: "noise", gain: 0.24, attack: 0.02, release: 0.95, drive: 0.45, filter: { type: "lowpass", freq: 1200, freqEnd: 200, q: 1 } },
    { type: "triangle", freq: 420, freqEnd: 210, gain: 0.46, attack: 0.03, release: 1.2, detune: -6, jitter: 16, drive: 0.4, ring: { freq: 63, depth: 0.5 }, lfo: { target: "pitch", freq: 6, depth: 30 } },
    { type: "sine", freq: 140, freqEnd: 64, gain: 0.6, attack: 0.02, release: 1.3, drive: 0.4 },
    { type: "triangle", freq: 630, freqEnd: 315, gain: 0.2, attack: 0.03, release: 1.0, fm: { ratio: 2.4, depth: 120 }, delay: 0.05, pan: 0.35 },
  ],
  // A clock hand sweeps the dial: a high reverse-swell rushes up to the strike,
  // two ring-modulated, FM'd metal clangs land (a struck mechanism, not clean
  // ticks), and an inharmonic chime blooms and rings out beneath.
  time: [
    { type: "noise", gain: 0.16, attack: 0.35, release: 0.16, filter: { type: "highpass", freq: 1500, freqEnd: 5000, q: 0.7 } },
    { type: "sine", freq: 1200, gain: 0.5, attack: 0.002, release: 0.18, drive: 0.4, ring: { freq: 430, depth: 0.6 }, fm: { ratio: 2.8, depth: 300 }, delay: 0.36 },
    { type: "sine", freq: 1200, gain: 0.5, attack: 0.002, release: 0.18, drive: 0.4, ring: { freq: 430, depth: 0.6 }, fm: { ratio: 2.8, depth: 300 }, delay: 0.54 },
    { type: "sine", freq: 600,  gain: 0.3, attack: 0.01, release: 1.7, fm: { ratio: 3.1, depth: 180 }, delay: 0.36 },
  ],
  // The underlying mathematical structure flashes into view: a noise swell lifts
  // into a struck-crystal bell chord made inharmonic by FM — sub root, root,
  // fifth, and a shimmering ring-modulated octave — ringing long into the reverb.
  math: [
    { type: "noise", gain: 0.14, attack: 0.4, release: 0.5, filter: { type: "highpass", freq: 2000, freqEnd: 7000, q: 0.7 } },
    { type: "sine", freq: 261,  gain: 0.4,  attack: 0.02, release: 1.8, fm: { ratio: 1.41, depth: 90 }, delay: 0.3 },
    { type: "sine", freq: 523,  gain: 0.5,  attack: 0.01, release: 1.9, fm: { ratio: 2.76, depth: 130 }, ring: { freq: 311, depth: 0.3 }, delay: 0.3 },
    { type: "sine", freq: 784,  gain: 0.4,  attack: 0.01, release: 1.9, fm: { ratio: 3.14, depth: 160 }, delay: 0.34, pan: 0.25 },
    { type: "sine", freq: 1046, gain: 0.22, attack: 0.02, release: 1.8, delay: 0.38, pan: -0.25, lfo: { target: "gain", freq: 7, depth: 0.4 } },
  ],
};
export const DEFAULT_SOUND = [
  { type: "sine", freq: 440, freqEnd: 330, gain: 0.4, attack: 0.01, release: 0.5, filter: { freq: 2000, freqEnd: 600, q: 1 } },
];

// BLOCKED_SOUND is the short, harsh cue played when an action is refused (a
// transition or travel that can't happen — out of budget, already at the base of
// an axis, no route). It is deliberately unlike the transition cues: a curt,
// driven, ring-modulated buzz that falls rather than blooms, so a blocked press
// reads instantly as "no". Kept out of SOUND_SPEC so it never counts as a
// reality transition; soundSpec serves it under the "blocked" mode.
export const BLOCKED_SOUND = [
  { type: "square", freq: 150, freqEnd: 90, gain: 0.5, attack: 0.005, release: 0.18, drive: 0.7, detune: 9, filter: { type: "lowpass", freq: 1200, freqEnd: 480, q: 3 } },
  { type: "square", freq: 72, gain: 0.4, attack: 0.005, release: 0.16, drive: 0.6, ring: { freq: 57, depth: 0.9 } },
];

// TRAVEL_SOUND is the soft cue for ordinary physical travel — a plain location
// change within one reality (the travel command, or clicking a reachable node),
// which detectTransition never arms as a reality transition. It is deliberately
// brief and understated next to the cinematic transition cues, since travel is by
// far the most common move: an airy band-passed noise whoosh (movement through
// space) over a short, gently driven low thump (a footfall landing). Kept out of
// SOUND_SPEC so it never counts as a reality transition; soundSpec serves it
// under the "travel" mode.
export const TRAVEL_SOUND = [
  { type: "noise", gain: 0.16, attack: 0.05, release: 0.24, filter: { type: "bandpass", freq: 650, freqEnd: 1900, q: 1.1 } },
  { type: "sine", freq: 190, freqEnd: 120, gain: 0.24, attack: 0.005, release: 0.2, drive: 0.25 },
];

// soundSpec returns the voices for a mode (falling back to DEFAULT_SOUND) plus
// the total duration (s) the sound occupies — the latest voice's delay + attack
// + release — so the player and any caller agree on when it has finished. The
// "blocked" and "travel" modes are special-cased to their own cues, kept out of
// the reality-transition palette.
export function soundSpec(mode) {
  const voices =
    mode === "blocked" ? BLOCKED_SOUND :
    mode === "travel" ? TRAVEL_SOUND :
    SOUND_SPEC[mode] || DEFAULT_SOUND;
  const duration = voices.reduce(
    (max, v) => Math.max(max, (v.delay || 0) + v.attack + v.release),
    0,
  );
  return { voices, duration };
}

// sessionMoved reports whether a command actually advanced the world: a genuine
// move changes the location or spends budget (every transition and travel costs
// something). When neither changes the action was refused, which the UI uses to
// flash the button and play the blocked cue. With no prior session (the first
// apply) it returns true so the opening state is never mistaken for a block.
export function sessionMoved(prev, next) {
  if (!prev || !next) return true;
  return prev.Location !== next.Location || prev.CumulativeCost !== next.CumulativeCost;
}

// A node that a reality transition has just revealed gets a brief faint halo, so
// the eye is drawn to what changed. spawnHalo turns the time elapsed since the
// node appeared (ms) into that halo for the current frame: alpha fades linearly
// from a faint peak to zero, and grow is the extra radius (in pixels) the sphere
// swells by as it fades. It returns null before the halo starts or once it has
// run its course, so app.js can drop the marker and the halo never lingers.
export const SPAWN_HALO_MS = 2200;
export const SPAWN_HALO_ALPHA = 0.28;
export const SPAWN_HALO_GROW = 26;
export function spawnHalo(elapsed, duration = SPAWN_HALO_MS) {
  if (!(elapsed >= 0) || elapsed >= duration) return null;
  const t = elapsed / duration;
  return { t, alpha: (1 - t) * SPAWN_HALO_ALPHA, grow: 4 + t * SPAWN_HALO_GROW };
}

// TRANSITIONS is the single source of truth for reality transitions: each entry
// ties a coordinate axis to its edge/effect `mode`, a human `label`, the
// `command` that traverses it, a plain-language `what`/`cost`, and any `refs`
// (authoritative further reading) for the help modal. The legend, the axis-diff
// detector, the "how to reach" hint, and the in-app documentation all derive
// from this one list, so they can never drift apart. Order is most-exotic-first
// — the priority the detector and the hint resolve ties in. Physical
// (same-reality) travel is not a transition and is intentionally absent. The
// `cost` strings mirror the per-step constants in internal/domain/universe/edge.go.
export const TRANSITIONS = [
  { mode: "universe",   axis: "Universe",    label: "universe",   command: "universe",  cost: "5,000 / hop",    costValue: 5000,
    what: "A parallel bubble universe with different physical constants (Tegmark Level II).",
    refs: [{ label: "Tegmark multiverse", url: "https://en.wikipedia.org/wiki/Multiverse#Max_Tegmark's_four_levels" }] },
  { mode: "timeline",   axis: "Timeline",    label: "timeline",   command: "jump",      cost: "800 / hop",      costValue: 800,
    what: "An alternate timeline where history diverged — the differences may be subtle or catastrophic." },
  { mode: "quantum",    axis: "Quantum",     label: "quantum",    command: "shift",     cost: "20 / hop",       costValue: 20,
    what: "A neighbouring quantum branch: almost identical, but something is subtly different (Everett many-worlds).",
    refs: [{ label: "Many-worlds interpretation", url: "https://en.wikipedia.org/wiki/Many-worlds_interpretation" }] },
  { mode: "math",       axis: "Mathematics", label: "structure",  command: "structure", cost: "50,000 / hop",   costValue: 50000,
    what: "A different mathematical structure — dimensions, logic, or physical law need not match (Tegmark Level IV).",
    refs: [{ label: "Mathematical universe hypothesis", url: "https://en.wikipedia.org/wiki/Mathematical_universe_hypothesis" }] },
  { mode: "simulation", axis: "Simulation",  label: "simulation", command: "simulate",  cost: "10 in · 50 out", costValue: 10,
    what: "A nested simulation whose rules can be rewritten. Entering is cheap; leaving means finding an exit.",
    refs: [{ label: "Simulation hypothesis", url: "https://en.wikipedia.org/wiki/Simulation_hypothesis" }] },
  { mode: "consensus",  axis: "Consensus",   label: "consensus",  command: "drift",     cost: "5 / level",      costValue: 5,
    what: "A reality that has drifted from shared consensus; its rules may no longer match the world you left.",
    refs: [{ label: "Consensus reality", url: "https://en.wikipedia.org/wiki/Consensus_reality" }] },
  { mode: "observer",   axis: "Observer",    label: "observer",   command: "observe",   cost: "2 / shift",      costValue: 2,
    what: "The same reality perceived through another observer (e.g. Bat, Octopus, Machine).",
    refs: [{ label: "Umwelt", url: "https://en.wikipedia.org/wiki/Umwelt" },
           { label: "What Is It Like to Be a Bat?", url: "https://en.wikipedia.org/wiki/What_Is_It_Like_to_Be_a_Bat%3F" }] },
  { mode: "time",       axis: "Time",        label: "time",       command: "time",      cost: "100 / shift",    costValue: 100,
    what: "The same reality at another point in time." },
];

// TRANSITIONS_BY_COST is the display ordering: the same transitions sorted by
// ascending cost. The user-facing lists (legend and help modal) read from this
// so they always read cheapest-first, while TRANSITIONS keeps its most-exotic
// declaration order for the detector's tie-break.
export const TRANSITIONS_BY_COST = [...TRANSITIONS].sort(
  (a, b) => a.costValue - b.costValue,
);

// Reality-transition legend for the top-right key, derived from the cost-sorted
// view so every row consistently carries its mode, label, and traversal command
// and reads cheapest-first.
export const TRANSITION_LEGEND = TRANSITIONS_BY_COST.map((t) => ({
  mode: t.mode,
  label: t.label,
  command: t.command,
}));

// [axis, mode] pairs for detectTransition, derived from TRANSITIONS. A
// transition is detected by diffing a reality axis between snapshots, so the
// ripple fires on the actual state change (button or typed command) and never
// on a failed command or on plain physical travel. Checked most-exotic-first.
export const AXIS_MODE = TRANSITIONS.map((t) => [t.axis, t.mode]);

export function detectTransition(prev, next) {
  if (!prev || !next) return null;
  for (const [axis, mode] of AXIS_MODE) {
    if (String(prev[axis]) !== String(next[axis])) return mode;
  }
  return null;
}

// requiredTransition reports which reality transition would move `session`'s
// reality to match a target `node`'s — the first axis that differs, checked
// most-exotic-first. Axes the node doesn't carry (e.g. a NodeSnapshot has no
// Time) are skipped, so they never register as a spurious difference. Returns
// the matching TRANSITIONS entry, or null when the realities already match
// (the node is in the same reality, reachable by ordinary travel).
export function requiredTransition(session, node) {
  if (!session || !node) return null;
  for (const t of TRANSITIONS) {
    const want = node[t.axis];
    if (want === undefined || want === null || want === "") continue;
    if (String(session[t.axis]) !== String(want)) return t;
  }
  return null;
}

// ── Physical route preview ─────────────────────────────────────────────────
// The set of travel modes the `travel` command permits — the solid-blue,
// same-reality edges. It mirrors universe.TravelModeVO.IsPhysical() in the Go
// domain, so the client previews exactly the routes the server would let you
// walk. Any other mode (a reality transition, or an unknown one) is not physical.
export const PHYSICAL_MODES = new Set(["walk", "cycle", "drive", "rail", "flight", "orbit", "warp"]);
export function isPhysical(mode) { return PHYSICAL_MODES.has(mode); }

// physicalRoute plans the fewest-hops physical route from `from` to `to` over the
// live EdgeSnapshot list, following only physical edges. It mirrors the domain's
// BFSPathfinder (navigation.FindRoute over physical edges), so the preview a
// hover shows is the same journey a click's `travel` would take. Returns the
// ordered edges of the path, or null when there is no physical route (including
// when from/to are missing or identical).
export function physicalRoute(from, to, edges) {
  if (!from || !to || from === to) return null;
  const adj = new Map();
  for (const e of edges || []) {
    if (!isPhysical(e.Mode)) continue;
    if (!adj.has(e.From)) adj.set(e.From, []);
    adj.get(e.From).push(e);
  }
  const visited = new Set([from]);
  const parentEdge = new Map();
  const queue = [from];
  while (queue.length) {
    const current = queue.shift();
    for (const e of adj.get(current) || []) {
      if (visited.has(e.To)) continue;
      visited.add(e.To);
      parentEdge.set(e.To, e);
      if (e.To === to) {
        const path = [];
        for (let n = to; n !== from; n = parentEdge.get(n).From) path.push(parentEdge.get(n));
        path.reverse();
        return path;
      }
      queue.push(e.To);
    }
  }
  return null;
}

// routeTotals sums a path's cost and distance and counts its hops, mirroring the
// domain's PathCost/PathDistance. A null/empty path totals to zero across the
// board, so callers can render it without a null check.
export function routeTotals(path) {
  const p = path || [];
  let cost = 0, distance = 0;
  for (const e of p) { cost += e.Cost || 0; distance += e.Distance || 0; }
  return { steps: p.length, cost, distance };
}

// Node fills, chosen so the map answers "where can I go?" at a glance:
//   green  — you are here (handled in draw)
//   blue   — reachable now by ordinary travel (click to go)
//   pink   — a different quantum branch (needs 'shift', not travel)
//   grey   — exists, but no travel route from here (needs another jump/route)
export const NODE_REACHABLE = "#6ea8ff";
export const NODE_QUANTUM = "#ff5fb0";
export const NODE_UNREACHABLE = "#535d82";

export function colorFor(node) {
  if (node.reachable) return NODE_REACHABLE;
  if (node.quantum && node.quantum !== DEFAULTS.Quantum) return NODE_QUANTUM;
  return NODE_UNREACHABLE;
}

// A node's reality nesting depth (NodeSnapshot.Depth from the Go facade) maps to
// a fixed z-layer, so deeper-nested realities stack behind base reality instead
// of scattering. layerZ turns that integer depth into the world-space z the
// layout springs toward each frame (app.js), while x/y stay force-directed —
// giving stacked shells of increasing depth. Base reality (depth 0) sits on the
// z = 0 plane; missing or non-finite depth falls back to base so a node is never
// flung out of view.
export const LAYER_GAP = 80;
export function layerZ(depth) {
  const d = Number(depth);
  if (!Number.isFinite(d)) return 0;
  return d * LAYER_GAP;
}

// ── Deterministic x/y layout ───────────────────────────────────────────────
//
// z stacks realities by nesting depth (layerZ); x/y used to be purely
// force-directed, which read as haphazard. layoutTarget gives every node a
// fixed home in the x/y plane so the map is ordered and repeatable: the force
// layout then only nudges nodes off that anchor to avoid overlap.
//
// The home has two parts. First a reality centre: each reality axis fans its
// sub-graph out along a fixed direction, scaled by how far that axis has moved
// from base reality, and a reality two steps out sits twice as far as one step.
// Every axis direction points *downward* (into a cone around straight-down), so
// a new reality is always created below its parent — never above it — while the
// horizontal spread keeps the axes apart (timeline leans right, quantum leans
// left, universe goes straight down). Second a physical offset within that
// reality: each place is pinned to a stable angle on a ring around its reality
// centre (derived from the place name), so the *same* place lands in the *same*
// relative spot in every reality, and the reality's home sits dead centre. The
// result: base reality clusters at the origin and each nested reality is its own
// tidy satellite cluster, always fanning downward.

// hashString is a small deterministic FNV-1a hash → an unsigned 32-bit int, so
// the same string always yields the same layout angle/offset across sessions
// and machines. Kept tiny and dependency-free; it is not cryptographic.
export function hashString(value) {
  const s = String(value);
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

// axisLevel turns a reality-axis value into a signed distance from its base.
// The base value (and empty/undefined) is 0; a plain number or a trailing
// number ("Q1", "T2") is that number; any other label gets a small stable
// offset so distinct named realities still separate instead of colliding.
export function axisLevel(value, base) {
  if (value === undefined || value === null || value === "") return 0;
  if (value === base) return 0;
  const s = String(value);
  const n = Number(s);
  if (Number.isFinite(n)) return n;
  const m = /(\d+)$/.exec(s);
  if (m) return Number(m[1]);
  return 1 + (hashString(s) % 3);
}

// Fixed unit direction per reality axis. All point downward (world +y, which is
// screen-down in every view), spread evenly across a 30°–150° cone so a new
// reality always appears below its parent yet each axis stays visually distinct.
export const AXIS_DIR = {
  timeline:   [0.866, 0.5],    //  30° — down and to the right
  simulation: [0.643, 0.766],  //  50°
  math:       [0.342, 0.940],  //  70°
  universe:   [0, 1],          //  90° — straight down
  consensus:  [-0.342, 0.940], // 110°
  observer:   [-0.643, 0.766], // 130°
  quantum:    [-0.866, 0.5],   // 150° — down and to the left
};
// REALITY_SPREAD sets the distance between reality group centres (per axis
// level); PHYS_RADIUS is the radius of the physical ring within a group. The
// spread must clear the ring on the shallowest downward axis (min downward
// component 0.5, timeline/quantum) so a new reality always lands below base:
// REALITY_SPREAD * 0.5 > PHYS_RADIUS. Kept compact so groups sit close together
// rather than drifting far apart as the map grows.
export const REALITY_SPREAD = 80;
export const PHYS_RADIUS = 34;
// CHAIN_STEP / CHAIN_FAN lay an auto-generated chain out as a downward tree from
// its reality centre. Infinite chaining (each generated node is a leaf that
// spawns the next) would otherwise pile every generated node in a reality onto
// the one PHYS_RADIUS ring — an unclickable heap. Instead each step down a chain
// drops the node CHAIN_STEP further down the screen (world +y) and siblings of
// one parent fan CHAIN_FAN world units apart across the x axis. Because y only
// ever *increases* with chain depth, a child never rises above its parent — new
// nodes are always created downward, matching how realities themselves fan down.
// Both are world-unit distances (unlike the reality-axis directions, which are
// unit vectors scaled by REALITY_SPREAD).
export const CHAIN_STEP = 30;
export const CHAIN_FAN = 40;

// realityCenter sums each axis's direction × its level × REALITY_SPREAD, giving
// the x/y anchor for a node's whole reality. Base reality resolves to {0,0}.
export function realityCenter(node) {
  const levels = {
    timeline: axisLevel(node.Timeline, DEFAULTS.Timeline),
    quantum: axisLevel(node.Quantum, DEFAULTS.Quantum),
    universe: axisLevel(node.Universe, DEFAULTS.Universe),
    math: axisLevel(node.Mathematics, DEFAULTS.Mathematics),
    simulation: axisLevel(node.Simulation, 0),
    consensus: axisLevel(node.Consensus, 0),
    observer: axisLevel(node.Observer, DEFAULTS.Observer),
  };
  let x = 0, y = 0;
  for (const axis in AXIS_DIR) {
    const [dx, dy] = AXIS_DIR[axis];
    x += dx * levels[axis] * REALITY_SPREAD;
    y += dy * levels[axis] * REALITY_SPREAD;
  }
  return { x, y };
}

// chainIndices splits a location ID into its stable anchor (the seed slug, e.g.
// "well" or "kirkstall-abbey") and the run of numeric chain indices an
// auto-generated expansion appends ("well-1-2-3" → indices ["1","2","3"]). Chain
// indices are the only pure-digit segments in an ID: the anchor slug carries no
// digit-only segment and every reality-axis suffix ("q1", "t1", "c1", …) starts
// with a letter, so a simple digit test cleanly separates the three. A node with
// no chain indices is a named seed place, not a generated chain node.
export function chainIndices(id) {
  const anchorSegs = [];
  const indices = [];
  let seenDigit = false;
  for (const seg of String(id).split("-")) {
    if (/^[0-9]+$/.test(seg)) { seenDigit = true; indices.push(seg); }
    else if (!seenDigit) anchorSegs.push(seg);
  }
  return { anchor: anchorSegs.join("-"), indices };
}

// layoutTarget is the node's full x/y home: its reality centre plus a physical
// offset. The reality's "Home" sits at the centre. A named place rings around it
// at a stable, name-derived angle so the same place occupies the same relative
// position in every reality. An auto-generated chain node instead descends as a
// tree from the centre: y grows one CHAIN_STEP per chain level (strictly
// downward, so a child never rises above its parent — new nodes are created
// downward), while x fans siblings CHAIN_FAN apart. A per-anchor x column and a
// small per-parent x meander keep distinct chains and distinct branches from
// overlapping, so a long chain reads as a spreading downward trail of
// individually clickable nodes rather than a heap on the ring.
export function layoutTarget(node) {
  const c = realityCenter(node);
  const { anchor, indices } = chainIndices(node.ID || "");
  if (indices.length > 0) {
    let dx = ((hashString(anchor) % 201) - 100);
    let dy = PHYS_RADIUS;
    let prefix = anchor;
    for (const idx of indices) {
      const meander = (((hashString(prefix) % 61) - 30) / 30) * (CHAIN_FAN * 0.25);
      dx += (Number(idx) - 1) * CHAIN_FAN + meander;
      dy += CHAIN_STEP;
      prefix += "-" + idx;
    }
    return { x: c.x + dx, y: c.y + dy };
  }
  const place = String(node.Location || node.Name || node.ID || "");
  if (place.trim().toLowerCase() === "home") return c;
  const angle = (hashString(place) % 360) * (Math.PI / 180);
  return { x: c.x + Math.cos(angle) * PHYS_RADIUS, y: c.y + Math.sin(angle) * PHYS_RADIUS };
}

// project maps a node's world position to screen space. It rotates about the Y
// (yaw) then X (pitch) axes, then applies a perspective divide so depth reads as
// size. Pulled out of the canvas so the maths is testable in isolation; app.js's
// toScreen is a thin wrapper that passes the live view and canvas dimensions.
export const FOCAL = 900;

export function project(n, view, width, height) {
  const cy = Math.cos(view.rotY), sy = Math.sin(view.rotY);
  const cx = Math.cos(view.rotX), sx = Math.sin(view.rotX);
  const x1 = n.x * cy - n.z * sy;
  const z1 = n.x * sy + n.z * cy;
  const y1 = n.y * cx - z1 * sx;
  const z2 = n.y * sx + z1 * cx;
  const persp = FOCAL / Math.max(FOCAL + z2, 1);
  return {
    x: width / 2 + view.ox + x1 * view.scale * persp,
    y: height / 2 + view.oy + y1 * view.scale * persp,
    depth: z2,
    persp,
  };
}

// ── Camera controls ─────────────────────────────────────────────────────────
// Rotation is free on both axes: a drag turns yaw (rotY) with horizontal motion
// and pitch (rotX) with vertical motion, with no limits, so the map can be spun
// to any orientation the pointer asks for. The drag orbits about the pivot under
// the cursor (see unproject/panToScreen) rather than the canvas centre.
// ROTATE_SPEED converts pointer pixels to radians (lower = finer control).
export const ROTATE_SPEED = 0.006;

// clampScale bounds wheel zoom. The floor is very low so a big, sprawling map
// can be pulled right out to fit on screen; the ceiling is deliberately high so a
// deep, far-flung reality (e.g. M9/U3/Q2/cons:24, whose centre sits thousands of
// world units from the origin and whose nodes cluster tightly there) can be
// zoomed right in on to read and click individual places. Non-finite input falls
// back to the floor.
export const MIN_SCALE = 0.003;
export const MAX_SCALE = 40;
export function clampScale(scale) {
  if (!Number.isFinite(scale)) return MIN_SCALE;
  return scale < MIN_SCALE ? MIN_SCALE : scale > MAX_SCALE ? MAX_SCALE : scale;
}

// zoomOffset re-centres the pan so the world point under the cursor stays put
// while the scale changes — i.e. the wheel zooms toward the pointer instead of
// the canvas centre. Per axis: `o` is the current pan offset, `cursor` the
// pointer position and `center` the canvas centre (both CSS px), and `factor` is
// the scale ratio actually applied (newScale / oldScale, after clamping). It
// returns the new pan offset. factor === 1 leaves the offset untouched.
export function zoomOffset(o, cursor, center, factor) {
  return cursor - center - (cursor - center - o) * factor;
}

// unproject is the inverse of project for points lying on the view plane through
// the world origin — the plane the camera faces, where the rotated depth z2 is 0
// and the perspective factor is exactly 1. Given a screen point it returns the
// world point on that plane, used to find the pivot under the cursor when
// orbiting. It undoes the pan/scale, then the pitch (rotX) and yaw (rotY)
// rotations in reverse of project's order.
export function unproject(sx, sy, view, width, height) {
  const x1 = (sx - width / 2 - view.ox) / view.scale;
  const y1 = (sy - height / 2 - view.oy) / view.scale;
  const cx = Math.cos(view.rotX), sxp = Math.sin(view.rotX);
  const cy = Math.cos(view.rotY), syp = Math.sin(view.rotY);
  // Undo the pitch with z2 pinned to 0: inverse-rotate (y1, 0) back to (wy, z1).
  const wy = y1 * cx;
  const z1 = -y1 * sxp;
  // Undo the yaw: inverse-rotate (x1, z1) back to (wx, wz).
  const wx = x1 * cy + z1 * syp;
  const wz = -x1 * syp + z1 * cy;
  return { x: wx, y: wy, z: wz };
}

// panToScreen returns the pan offset {ox, oy} that makes world point `p` project
// to the screen target (tx, ty) under the given rotation/scale. Orbiting uses it
// to hold the pivot (the point grabbed at drag start) fixed on screen while the
// rotation changes, so the map spins about the cursor rather than the canvas
// centre.
export function panToScreen(p, view, width, height, tx, ty) {
  const proj = project(p, { ...view, ox: 0, oy: 0 }, width, height);
  return { ox: tx - proj.x, oy: ty - proj.y };
}

// fitView computes the scale + pan ({scale, ox, oy}) that frames every node in
// `nodes` within the canvas at the view's *current* rotation, so a "best fit"
// button can show as many nodes as possible without changing the viewing angle.
//
// It deliberately frames on the *orthographic* rotated coordinates (x1, y1 —
// before the perspective divide), not the perspective-projected screen points.
// A big, deep journey pushes some nodes to a large rotated depth (z2); the
// perspective factor FOCAL/(FOCAL+z2) then blows up (or clamps) for a node at or
// behind the focal plane, which used to throw a single wild outlier into the
// bounding box, collapse the scale to its floor, and skew the centre — so the
// map vanished to a sub-pixel dot parked off-screen. Orthographic extents are
// bounded and outlier-free, so every node is guaranteed inside the frame.
//
// Perspective is then accounted for conservatively: the box half-extents are
// grown by the largest perspective magnification present among the nodes (near
// nodes, with z2 < 0, render larger than their orthographic position), so those
// magnified nodes still fit. The result is centred on the orthographic midpoint,
// which is where project() also centres at persp≈1. An empty set leaves the map
// centred at scale 1.
export const FIT_MARGIN = 0.12;
// FIT_GROUP_MARGIN frames a single group (the nodes a move just revealed) as the
// hero: a much larger margin than FIT_MARGIN so the group is centred and sits in
// the middle ~40% of the canvas rather than filling it, leaving the surrounding
// old realities visible around the edges for context. Used by frameToGroup.
export const FIT_GROUP_MARGIN = 0.6;
// FIT_GROUP_MIN_SPAN is the smallest world-space span frameToGroup frames a
// revealed group as. A freshly auto-generated cluster is tiny — one node has a
// zero-size box (which would leave the scale at its default 1, snapping the
// camera far out from wherever you were zoomed), and a 2–3 node cluster sits
// within one PHYS_RADIUS (which would pin the scale to MAX_SCALE, zooming in too
// close). Flooring the box to this span makes such a cluster frame at a steady,
// moderate zoom instead: zoomed in on the new node(s), but never all the way out
// or all the way in. Larger groups (a reality reveal) exceed it, so they still
// fit exactly. Only frameToGroup passes it; the full-map fits use the default 0.
export const FIT_GROUP_MIN_SPAN = 256;
export function fitView(nodes, view, width, height, margin = FIT_MARGIN, minSpan = 0) {
  const arr = [...nodes];
  if (!arr.length || !(width > 0) || !(height > 0)) {
    return { scale: 1, ox: 0, oy: 0 };
  }
  const cyaw = Math.cos(view.rotY), syaw = Math.sin(view.rotY);
  const cpit = Math.cos(view.rotX), spit = Math.sin(view.rotX);
  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
  let maxPersp = 0;
  // Also accumulate the perspective-projected offsets so the frame can centre on
  // where project() actually draws the nodes, not their orthographic midpoint.
  let sumPX = 0, sumPY = 0;
  for (const n of arr) {
    const nx = n.x || 0, ny = n.y || 0, nz = n.z || 0;
    const x1 = nx * cyaw - nz * syaw;
    const z1 = nx * syaw + nz * cyaw;
    const y1 = ny * cpit - z1 * spit;
    const z2 = ny * spit + z1 * cpit;
    if (x1 < minX) minX = x1;
    if (x1 > maxX) maxX = x1;
    if (y1 < minY) minY = y1;
    if (y1 > maxY) maxY = y1;
    const persp = FOCAL / Math.max(FOCAL + z2, 1);
    if (persp > maxPersp) maxPersp = persp;
    sumPX += x1 * persp;
    sumPY += y1 * persp;
  }
  // Grow the box by the strongest magnification so magnified near nodes still
  // fit; guard against a degenerate/non-finite factor. Then floor each extent to
  // minSpan so a tiny or single-node group (frameToGroup) frames at a steady,
  // moderate zoom rather than collapsing to the default scale or maxing out.
  const grow = Number.isFinite(maxPersp) && maxPersp > 1 ? maxPersp : 1;
  const boxW = Math.max((maxX - minX) * grow, minSpan);
  const boxH = Math.max((maxY - minY) * grow, minSpan);
  const availW = width * (1 - margin), availH = height * (1 - margin);
  let scale = 1;
  if (boxW > 0 || boxH > 0) {
    const sx = boxW > 0 ? availW / boxW : Infinity;
    const sy = boxH > 0 ? availH / boxH : Infinity;
    scale = clampScale(Math.min(sx, sy));
  }
  // Centre on the mean *projected* position (perspective included), so a group
  // seen at a steep pitch — where a node's screen-y is y1*scale*persp, not just
  // y1*scale — lands on the canvas centre. Centring on the orthographic midpoint
  // (persp ignored) drifted deeper groups off the bottom as journeys stacked up.
  // A mean is used rather than the box midpoint so no single far node (large,
  // unstable persp) can throw the centre off. Falls back to the orthographic
  // midpoint if the sums are non-finite.
  const px = Number.isFinite(sumPX) ? sumPX / arr.length : (minX + maxX) / 2;
  const py = Number.isFinite(sumPY) ? sumPY / arr.length : (minY + maxY) / 2;
  return { scale, ox: -px * scale, oy: -py * scale };
}

// depthAlpha maps a node's projected depth to an opacity so nodes further from
// the camera fade out, keeping a busy map readable. The fade is *relative* to
// the depth range currently on screen (min/max of the drawn nodes) rather than
// absolute: the raw perspective spread is tiny (the node cloud is shallow next
// to the focal length), so an absolute falloff is imperceptible. The nearest
// node is fully opaque and the farthest is floored at MIN_DEPTH_ALPHA — still
// visible, never gone — with a linear ramp between. A flat set (no spread)
// stays fully opaque.
export const MIN_DEPTH_ALPHA = 0.3;
export function depthAlpha(depth, minDepth, maxDepth) {
  if (!(maxDepth > minDepth)) return 1;
  let t = (depth - minDepth) / (maxDepth - minDepth); // 0 = nearest, 1 = farthest
  t = t < 0 ? 0 : t > 1 ? 1 : t;
  return 1 - t * (1 - MIN_DEPTH_ALPHA);
}

// abbreviateLabel shortens a node name for the map so labels stay compact until
// the node is hovered (app.js reveals the full name then). Names within the
// limit are returned unchanged; longer ones are cut to LABEL_MAX characters with
// a trailing ellipsis (which counts toward the limit) so the map doesn't clutter
// as names grow.
export const LABEL_MAX = 14;
export function abbreviateLabel(name, max = LABEL_MAX) {
  if (typeof name !== "string") return "";
  if (name.length <= max) return name;
  if (max <= 1) return name.slice(0, Math.max(max, 0));
  return name.slice(0, max - 1) + "\u2026";
}

// escapeHtml keeps values that end up in innerHTML — a free-form observer name
// the user typed, or any server-provided label — from being interpreted as
// markup. Non-string input is coerced to a string first.
export const HTML_ESCAPES = { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" };
export function escapeHtml(value) {
  return String(value).replace(/[&<>"']/g, (c) => HTML_ESCAPES[c]);
}
