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
// ties a coordinate axis to its edge/effect `mode`, a human `label`, and the
// `command` that traverses it. The legend, the axis-diff detector, and the
// "how to reach" hint all derive from this one list, so they can never drift
// apart. Order is most-exotic-first — the priority the detector and the hint
// resolve ties in. Physical (same-reality) travel is not a transition and is
// intentionally absent.
export const TRANSITIONS = [
  { mode: "universe",   axis: "Universe",    label: "universe",   command: "universe" },
  { mode: "timeline",   axis: "Timeline",    label: "timeline",   command: "jump" },
  { mode: "quantum",    axis: "Quantum",     label: "quantum",    command: "shift" },
  { mode: "math",       axis: "Mathematics", label: "structure",  command: "structure" },
  { mode: "simulation", axis: "Simulation",  label: "simulation", command: "simulate" },
  { mode: "consensus",  axis: "Consensus",   label: "consensus",  command: "drift" },
  { mode: "observer",   axis: "Observer",    label: "observer",   command: "observe" },
  { mode: "time",       axis: "Time",        label: "time",       command: "time" },
];

// Reality-transition legend for the top-right key, derived from TRANSITIONS so
// every row consistently carries its mode, label, and traversal command.
export const TRANSITION_LEGEND = TRANSITIONS.map((t) => ({
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
export const LAYER_GAP = 110;
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
// REALITY_SPREAD * 0.5 > PHYS_RADIUS.
export const REALITY_SPREAD = 110;
export const PHYS_RADIUS = 45;

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

// layoutTarget is the node's full x/y home: its reality centre plus a fixed
// physical offset. The reality's "Home" sits at the centre; every other place
// rings around it at a stable, name-derived angle so the same place occupies the
// same relative position in every reality.
export function layoutTarget(node) {
  const c = realityCenter(node);
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
// can be pulled right out to fit on screen; the ceiling keeps a single node from
// filling the canvas. Non-finite input falls back to the floor.
export const MIN_SCALE = 0.003;
export const MAX_SCALE = 3;
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
export function fitView(nodes, view, width, height, margin = FIT_MARGIN) {
  const arr = [...nodes];
  if (!arr.length || !(width > 0) || !(height > 0)) {
    return { scale: 1, ox: 0, oy: 0 };
  }
  const cyaw = Math.cos(view.rotY), syaw = Math.sin(view.rotY);
  const cpit = Math.cos(view.rotX), spit = Math.sin(view.rotX);
  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
  let maxPersp = 0;
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
  }
  const midX = (minX + maxX) / 2, midY = (minY + maxY) / 2;
  // Grow the box by the strongest magnification so magnified near nodes still
  // fit; guard against a degenerate/non-finite factor.
  const grow = Number.isFinite(maxPersp) && maxPersp > 1 ? maxPersp : 1;
  const boxW = (maxX - minX) * grow, boxH = (maxY - minY) * grow;
  const availW = width * (1 - margin), availH = height * (1 - margin);
  let scale = 1;
  if (boxW > 0 || boxH > 0) {
    const sx = boxW > 0 ? availW / boxW : Infinity;
    const sy = boxH > 0 ? availH / boxH : Infinity;
    scale = clampScale(Math.min(sx, sy));
  }
  // Centre the orthographic midpoint on the canvas centre.
  return { scale, ox: -midX * scale, oy: -midY * scale };
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
