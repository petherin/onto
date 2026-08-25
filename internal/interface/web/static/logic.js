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

// Reality-transition legend: [mode, label] for the top-right key. Physical modes
// are intentionally omitted — every solid-blue edge is ordinary travel.
export const TRANSITION_LEGEND = [
  ["quantum", "quantum · shift"],
  ["timeline", "timeline · jump"],
  ["universe", "universe"],
  ["simulation", "simulation"],
  ["consensus", "consensus · drift"],
  ["observer", "observer"],
  ["time", "time"],
  ["math", "structure"],
];

// A transition is detected by diffing a reality axis between snapshots, so the
// ripple fires on the actual state change (button or typed command) and never
// on a failed command or on plain physical travel. Checked most-exotic-first.
export const AXIS_MODE = [
  ["Universe", "universe"],
  ["Timeline", "timeline"],
  ["Quantum", "quantum"],
  ["Mathematics", "math"],
  ["Simulation", "simulation"],
  ["Consensus", "consensus"],
  ["Observer", "observer"],
  ["Time", "time"],
];

export function detectTransition(prev, next) {
  if (!prev || !next) return null;
  for (const [axis, mode] of AXIS_MODE) {
    if (String(prev[axis]) !== String(next[axis])) return mode;
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
