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
  effectSpec,
  EFFECT_KIND,
  EFFECT_DURATION,
  DEFAULT_EFFECT_KIND,
  detectTransition,
  colorFor,
  NODE_REACHABLE,
  NODE_QUANTUM,
  NODE_UNREACHABLE,
  project,
  FOCAL,
  depthAlpha,
  MIN_DEPTH_ALPHA,
  abbreviateLabel,
  LABEL_MAX,
  escapeHtml,
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

test("effectSpec gives each reality transition its own animation kind", () => {
  assert.equal(effectSpec("universe").kind, "fade");
  assert.equal(effectSpec("quantum").kind, "superposition");
  assert.equal(effectSpec("observer").kind, "blink");
  // Distinct kinds so no two transitions look the same.
  const kinds = Object.values(EFFECT_KIND);
  assert.equal(new Set(kinds).size, kinds.length);
});

test("effectSpec falls back to the ripple for physical/unknown modes", () => {
  assert.equal(effectSpec("walk").kind, DEFAULT_EFFECT_KIND);
  assert.equal(effectSpec(undefined).kind, DEFAULT_EFFECT_KIND);
  assert.equal(effectSpec("walk").kind, "ripple");
});

test("effectSpec returns the duration for its kind", () => {
  assert.equal(effectSpec("universe").duration, EFFECT_DURATION.fade);
  assert.equal(effectSpec("walk").duration, EFFECT_DURATION.ripple);
  // Every declared kind has a positive duration.
  for (const kind of new Set([...Object.values(EFFECT_KIND), DEFAULT_EFFECT_KIND])) {
    assert.ok(EFFECT_DURATION[kind] > 0, `missing duration for ${kind}`);
  }
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

test("depthAlpha: the nearest node is fully opaque, the farthest is floored", () => {
  approx(depthAlpha(-80, -80, 80), 1);
  approx(depthAlpha(80, -80, 80), MIN_DEPTH_ALPHA);
});

test("depthAlpha: the midpoint fades halfway toward the floor", () => {
  approx(depthAlpha(0, -80, 80), 1 - 0.5 * (1 - MIN_DEPTH_ALPHA));
});

test("depthAlpha: a flat set (no depth spread) stays fully opaque", () => {
  approx(depthAlpha(42, 42, 42), 1);
});

test("depthAlpha: a distant node is dimmer than a near one across the same range", () => {
  const near = project({ x: 0, y: 0, z: -80 }, identityView, 800, 600);
  const far = project({ x: 0, y: 0, z: 80 }, identityView, 800, 600);
  const lo = Math.min(near.depth, far.depth), hi = Math.max(near.depth, far.depth);
  assert.ok(depthAlpha(far.depth, lo, hi) < depthAlpha(near.depth, lo, hi));
});

test("abbreviateLabel: names within the limit are returned unchanged", () => {
  assert.equal(abbreviateLabel("Prime"), "Prime");
  const exact = "x".repeat(LABEL_MAX);
  assert.equal(abbreviateLabel(exact), exact);
});

test("abbreviateLabel: long names are cut to the limit with a trailing ellipsis", () => {
  const long = "SupermassiveBlackHole";
  const out = abbreviateLabel(long);
  assert.equal(out.length, LABEL_MAX);
  assert.ok(out.endsWith("\u2026"));
  assert.equal(out, long.slice(0, LABEL_MAX - 1) + "\u2026");
});

test("abbreviateLabel: honours an explicit max", () => {
  assert.equal(abbreviateLabel("HelloWorld", 5), "Hell\u2026");
});

test("abbreviateLabel: non-string input yields an empty string", () => {
  assert.equal(abbreviateLabel(undefined), "");
  assert.equal(abbreviateLabel(null), "");
});

test("escapeHtml: leaves ordinary text untouched", () => {
  assert.equal(escapeHtml("Bat"), "Bat");
  assert.equal(escapeHtml("Timeline Prime-2"), "Timeline Prime-2");
});

test("escapeHtml: neutralises markup so values can't inject HTML", () => {
  assert.equal(
    escapeHtml('<img src=x onerror="alert(1)">'),
    "&lt;img src=x onerror=&quot;alert(1)&quot;&gt;",
  );
  assert.equal(escapeHtml("a & b"), "a &amp; b");
  assert.equal(escapeHtml("it's"), "it&#39;s");
});

test("escapeHtml: coerces non-string input to a string", () => {
  assert.equal(escapeHtml(42), "42");
  assert.equal(escapeHtml(0), "0");
});
