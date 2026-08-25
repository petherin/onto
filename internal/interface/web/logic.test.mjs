// Unit tests for the pure front-end logic in static/logic.js. Run with Node's
// built-in test runner (no external dependencies): `node --test` from this
// directory, or `make test-js` from the repo root. Only the framework-free
// maths lives in logic.js, so these cover it without a DOM or a browser.

import test from "node:test";
import assert from "node:assert/strict";

import {
  DEFAULTS,
  modeStyle,
  DEFAULT_MODE_STYLE,
  detectTransition,
  colorFor,
  NODE_REACHABLE,
  NODE_QUANTUM,
  NODE_UNREACHABLE,
  project,
  FOCAL,
} from "./static/logic.js";

test("modeStyle returns solid blue for physical modes", () => {
  const st = modeStyle("walk");
  assert.equal(st.rgb, "110,168,255");
  assert.deepEqual(st.dash, []);
});

test("modeStyle returns a distinct dashed style per transition", () => {
  assert.deepEqual(modeStyle("quantum").dash, [4, 4]);
  assert.deepEqual(modeStyle("timeline").dash, [7, 4]);
  assert.notEqual(modeStyle("quantum").rgb, modeStyle("timeline").rgb);
});

test("modeStyle falls back for unknown modes", () => {
  assert.equal(modeStyle("nonsense"), DEFAULT_MODE_STYLE);
  assert.equal(modeStyle(undefined), DEFAULT_MODE_STYLE);
});

test("detectTransition returns null when a snapshot is missing", () => {
  assert.equal(detectTransition(null, { Quantum: "Q1" }), null);
  assert.equal(detectTransition({ Quantum: "Q0" }, null), null);
});

test("detectTransition returns null when no reality axis changed", () => {
  const s = { Quantum: "Q0", Universe: "Origin", Location: "home" };
  // A pure location change is not a reality transition.
  assert.equal(detectTransition(s, { ...s, Location: "park" }), null);
});

test("detectTransition names the changed axis", () => {
  const base = { Quantum: "Q0", Timeline: "Prime", Observer: "Human", Time: "t0" };
  assert.equal(detectTransition(base, { ...base, Quantum: "Q1" }), "quantum");
  assert.equal(detectTransition(base, { ...base, Observer: "Bat" }), "observer");
  assert.equal(detectTransition(base, { ...base, Time: "t1" }), "time");
});

test("detectTransition prefers the most-exotic axis when several change", () => {
  const base = { Universe: "Origin", Quantum: "Q0" };
  // Universe is checked before Quantum, so it wins.
  const next = { Universe: "Alt", Quantum: "Q1" };
  assert.equal(detectTransition(base, next), "universe");
});

test("colorFor: reachable nodes are blue", () => {
  assert.equal(colorFor({ reachable: true, quantum: "Q9" }), NODE_REACHABLE);
});

test("colorFor: unreachable non-default quantum branch is pink", () => {
  assert.equal(colorFor({ reachable: false, quantum: "Q1" }), NODE_QUANTUM);
});

test("colorFor: unreachable default/undefined quantum is grey", () => {
  assert.equal(colorFor({ reachable: false, quantum: DEFAULTS.Quantum }), NODE_UNREACHABLE);
  assert.equal(colorFor({ reachable: false }), NODE_UNREACHABLE);
});

const identityView = { scale: 1, ox: 0, oy: 0, rotX: 0, rotY: 0 };
const approx = (a, b, eps = 1e-9) => assert.ok(Math.abs(a - b) <= eps, `${a} !~= ${b}`);

test("project: origin node maps to the canvas centre plus offsets", () => {
  const p = project({ x: 0, y: 0, z: 0 }, identityView, 800, 600);
  approx(p.x, 400);
  approx(p.y, 300);
  approx(p.depth, 0);
  approx(p.persp, 1);
});

test("project: perspective divides x-offset by depth", () => {
  // At z = FOCAL the perspective factor is exactly 0.5.
  const p = project({ x: 100, y: 0, z: FOCAL }, identityView, 800, 600);
  approx(p.persp, 0.5);
  approx(p.depth, FOCAL);
  approx(p.x, 400 + 100 * 0.5);
});

test("project: a 90° yaw swaps x into depth", () => {
  const view = { ...identityView, rotY: Math.PI / 2 };
  // x becomes z1 (depth); the on-screen x contribution collapses to ~0.
  const p = project({ x: 100, y: 0, z: 0 }, view, 800, 600);
  approx(p.depth, 100, 1e-6);
  approx(p.x, 400, 1e-6);
});

test("project: pan offset and scale shift/scale about the canvas centre", () => {
  // Locks the contract the startup-centring fix relies on: the origin lands at
  // width/2+ox, height/2+oy, and displacement scales by view.scale. toScreen
  // must pass the CSS-pixel size so this centre matches the dpr-scaled context.
  const view = { scale: 2, ox: 30, oy: -20, rotX: 0, rotY: 0 };
  const origin = project({ x: 0, y: 0, z: 0 }, view, 800, 600);
  approx(origin.x, 400 + 30);
  approx(origin.y, 300 - 20);

  const off = project({ x: 10, y: 5, z: 0 }, view, 800, 600);
  approx(off.x, 400 + 30 + 10 * 2);
  approx(off.y, 300 - 20 + 5 * 2);
});
