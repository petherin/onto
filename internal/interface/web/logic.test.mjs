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
  TRANSITIONS,
  TRANSITION_LEGEND,
  AXIS_MODE,
  requiredTransition,
  colorFor,
  NODE_REACHABLE,
  NODE_QUANTUM,
  NODE_UNREACHABLE,
  layerZ,
  LAYER_GAP,
  clampScale,
  MIN_SCALE,
  MAX_SCALE,
  zoomOffset,
  unproject,
  panToScreen,
  hashString,
  axisLevel,
  realityCenter,
  layoutTarget,
  AXIS_DIR,
  REALITY_SPREAD,
  PHYS_RADIUS,
  edgeRestLength,
  REST_PHYSICAL,
  REST_TRANSITION,
  spawnHalo,
  SPAWN_HALO_MS,
  SPAWN_HALO_ALPHA,
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

test("AXIS_MODE and TRANSITION_LEGEND derive from the one TRANSITIONS source", () => {
  // Same length and order as the single source of truth.
  assert.equal(AXIS_MODE.length, TRANSITIONS.length);
  assert.equal(TRANSITION_LEGEND.length, TRANSITIONS.length);
  TRANSITIONS.forEach((t, i) => {
    assert.deepEqual(AXIS_MODE[i], [t.axis, t.mode]);
    assert.deepEqual(TRANSITION_LEGEND[i], { mode: t.mode, label: t.label, command: t.command });
  });
});

test("every legend row carries a command, so the legend is consistent", () => {
  // The bug this fixes: some legend rows showed the traversal command, some
  // didn't. Now every transition has a non-empty command.
  for (const row of TRANSITION_LEGEND) {
    assert.ok(row.command && row.command.length > 0, `missing command for ${row.mode}`);
    assert.ok(row.label && row.label.length > 0, `missing label for ${row.mode}`);
  }
});

test("requiredTransition returns null when a snapshot is missing", () => {
  assert.equal(requiredTransition(null, { Quantum: "Q1" }), null);
  assert.equal(requiredTransition({ Quantum: "Q0" }, null), null);
});

test("requiredTransition returns null when the node is in the same reality", () => {
  const sess = { Universe: "Origin", Timeline: "Prime", Quantum: "Q0", Mathematics: "Classical", Simulation: 0, Consensus: 0, Observer: "Human" };
  // Same reality, different physical place — reachable by ordinary travel.
  const node = { ...sess, Location: "park" };
  assert.equal(requiredTransition(sess, node), null);
});

test("requiredTransition names the transition and command for the differing axis", () => {
  const sess = { Quantum: "Q0", Observer: "Human", Simulation: 0 };
  assert.equal(requiredTransition(sess, { Quantum: "Q1" }).command, "shift");
  assert.equal(requiredTransition(sess, { Observer: "Bat" }).command, "observe");
  assert.equal(requiredTransition(sess, { Simulation: 1 }).mode, "simulation");
});

test("requiredTransition skips axes the node does not carry", () => {
  // A NodeSnapshot has no Time field, so a Time-only session difference must
  // not register as a required transition.
  const sess = { Quantum: "Q0", Time: "t5" };
  assert.equal(requiredTransition(sess, { Quantum: "Q0" }), null);
});

test("requiredTransition prefers the most-exotic axis when several differ", () => {
  const sess = { Universe: "Origin", Quantum: "Q0" };
  const node = { Universe: "Alt", Quantum: "Q1" };
  assert.equal(requiredTransition(sess, node).mode, "universe");
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

test("layerZ: base reality (depth 0) sits on the z=0 plane", () => {
  assert.equal(layerZ(0), 0);
});

test("layerZ: each depth is one LAYER_GAP deeper than the last", () => {
  assert.equal(layerZ(1), LAYER_GAP);
  assert.equal(layerZ(3), 3 * LAYER_GAP);
  assert.equal(layerZ(2) - layerZ(1), LAYER_GAP);
});

test("layerZ: missing or non-finite depth falls back to base reality", () => {
  assert.equal(layerZ(undefined), 0);
  assert.equal(layerZ(null), 0);
  assert.equal(layerZ(NaN), 0);
  assert.equal(layerZ("nonsense"), 0);
});

test("edgeRestLength: physical modes rest short so a reality's locations cluster", () => {
  assert.equal(edgeRestLength("walk"), REST_PHYSICAL);
  assert.equal(edgeRestLength("flight"), REST_PHYSICAL);
});

test("edgeRestLength: reality transitions rest longer, pushing child sub-graphs out", () => {
  assert.equal(edgeRestLength("quantum"), REST_TRANSITION);
  assert.equal(edgeRestLength("timeline"), REST_TRANSITION);
  assert.ok(REST_TRANSITION > REST_PHYSICAL);
});

test("edgeRestLength: unknown modes fall on the transition side", () => {
  assert.equal(edgeRestLength("nonsense"), REST_TRANSITION);
  assert.equal(edgeRestLength(undefined), REST_TRANSITION);
});

test("spawnHalo: is null before it starts and once it has run its course", () => {
  assert.equal(spawnHalo(-1), null);
  assert.equal(spawnHalo(SPAWN_HALO_MS), null);
  assert.equal(spawnHalo(SPAWN_HALO_MS + 100), null);
});

test("spawnHalo: starts at its peak alpha and fades to zero as it grows", () => {
  const start = spawnHalo(0);
  const mid = spawnHalo(SPAWN_HALO_MS / 2);
  assert.equal(start.alpha, SPAWN_HALO_ALPHA);
  assert.ok(start.alpha > mid.alpha, "alpha must fade over time");
  assert.ok(mid.grow > start.grow, "sphere must swell over time");
});

test("spawnHalo: honours an explicit duration", () => {
  assert.equal(spawnHalo(500, 1000).t, 0.5);
  assert.equal(spawnHalo(1000, 1000), null);
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

test("clampScale: bounds zoom and allows pulling far out", () => {
  assert.equal(clampScale(1), 1);
  assert.equal(clampScale(0), MIN_SCALE);
  assert.equal(clampScale(0.0001), MIN_SCALE);
  assert.equal(clampScale(99), MAX_SCALE);
  assert.equal(clampScale(NaN), MIN_SCALE);
  assert.ok(MIN_SCALE <= 0.003, "the zoom-out floor is low enough to fit a sprawling map");
});

test("zoomOffset: is a no-op when the scale doesn't change", () => {
  assert.equal(zoomOffset(30, 550, 400, 1), 30);
  assert.equal(zoomOffset(-80, 120, 400, 1), -80);
});

test("zoomOffset: keeps the world point under the cursor fixed on screen", () => {
  // Reconstruct the projection's 1-D screen mapping: screen = center + o + X*scale.
  const center = 400, cursor = 550, o = 30, oldScale = 1, newScale = 2.5;
  const content = (cursor - center - o) / oldScale; // world X under the cursor
  const newO = zoomOffset(o, cursor, center, newScale / oldScale);
  const screenAfter = center + newO + content * newScale;
  assert.ok(Math.abs(screenAfter - cursor) < 1e-9, "content under the cursor must not move");
});

test("zoomOffset: works when zooming out as well", () => {
  const center = 300, cursor = 90, o = 15, oldScale = 2, newScale = 0.5;
  const content = (cursor - center - o) / oldScale;
  const newO = zoomOffset(o, cursor, center, newScale / oldScale);
  assert.ok(Math.abs((center + newO + content * newScale) - cursor) < 1e-9);
});

test("unproject: round-trips with project (screen → world → screen)", () => {
  // unproject returns the world point on the origin view plane; projecting it
  // back must land on the original screen point.
  const view = { scale: 1.4, ox: 25, oy: -18, rotX: 0.6, rotY: -0.4 };
  const w = unproject(510, 220, view, 800, 600);
  const p = project(w, view, 800, 600);
  assert.ok(Math.abs(p.x - 510) < 1e-9 && Math.abs(p.y - 220) < 1e-9);
});

test("panToScreen: places a world point exactly under the screen target", () => {
  const view = { scale: 2, ox: 0, oy: 0, rotX: 0.3, rotY: 0.9 };
  const p = { x: 12, y: -7, z: 40 };
  const pan = panToScreen(p, view, 800, 600, 510, 220);
  const proj = project(p, { ...view, ...pan }, 800, 600);
  assert.ok(Math.abs(proj.x - 510) < 1e-9 && Math.abs(proj.y - 220) < 1e-9);
});

test("orbit: rotating about the cursor keeps the grabbed pivot under the pointer", () => {
  // Grab the point under the cursor, rotate, re-pan via panToScreen, and the
  // pivot must still project to the original cursor position.
  const cursorX = 520, cursorY = 250;
  const view = { scale: 1.2, ox: 40, oy: -30, rotX: 0.5, rotY: 0.35 };
  const pivot = unproject(cursorX, cursorY, view, 800, 600);
  const rotated = { ...view, rotY: view.rotY + 0.4, rotX: view.rotX - 0.25 };
  const pan = panToScreen(pivot, rotated, 800, 600, cursorX, cursorY);
  const proj = project(pivot, { ...rotated, ...pan }, 800, 600);
  assert.ok(Math.abs(proj.x - cursorX) < 1e-9 && Math.abs(proj.y - cursorY) < 1e-9);
});

test("hashString is deterministic and non-negative", () => {
  assert.equal(hashString("Park"), hashString("Park"));
  assert.notEqual(hashString("Park"), hashString("Station"));
  assert.ok(hashString("anything") >= 0);
});

test("axisLevel: base value and empty resolve to zero", () => {
  assert.equal(axisLevel("Prime", DEFAULTS.Timeline), 0);
  assert.equal(axisLevel("Q0", DEFAULTS.Quantum), 0);
  assert.equal(axisLevel("", DEFAULTS.Universe), 0);
  assert.equal(axisLevel(undefined, DEFAULTS.Observer), 0);
  assert.equal(axisLevel(0, 0), 0);
});

test("axisLevel: numbers and trailing-number labels read as their level", () => {
  assert.equal(axisLevel("T1", DEFAULTS.Timeline), 1);
  assert.equal(axisLevel("Q3", DEFAULTS.Quantum), 3);
  assert.equal(axisLevel(2, 0), 2); // simulation / consensus ints
});

test("axisLevel: arbitrary labels get a small stable non-zero offset", () => {
  const a = axisLevel("Dream", DEFAULTS.Universe);
  assert.equal(a, axisLevel("Dream", DEFAULTS.Universe));
  assert.ok(a >= 1 && a <= 3);
});

test("realityCenter: base reality sits at the origin", () => {
  const c = realityCenter({ Timeline: "Prime", Quantum: "Q0", Universe: "Origin", Mathematics: "Classical", Simulation: 0, Consensus: 0, Observer: "Human" });
  assert.equal(c.x, 0);
  assert.equal(c.y, 0);
});

test("realityCenter: each axis fans out in its own fixed direction and scales with level", () => {
  const t1 = realityCenter({ Timeline: "T1" });
  assert.deepEqual([t1.x, t1.y], [AXIS_DIR.timeline[0] * REALITY_SPREAD, AXIS_DIR.timeline[1] * REALITY_SPREAD]);
  // A quantum shift leans left, a timeline jump leans right.
  const q1 = realityCenter({ Quantum: "Q1" });
  assert.ok(q1.x < 0 && t1.x > 0);
  // Two steps out sits twice as far as one.
  const t2 = realityCenter({ Timeline: "T2" });
  assert.equal(t2.x, t1.x * 2);
});

test("every axis direction points downward, so a new reality never appears above its parent", () => {
  for (const axis in AXIS_DIR) {
    const [, dy] = AXIS_DIR[axis];
    assert.ok(dy > 0, `${axis} must fan downward (dy > 0), got ${dy}`);
  }
});

test("realityCenter: every single-axis reality sits below base reality", () => {
  const cases = [
    { Timeline: "T1" },
    { Quantum: "Q1" },
    { Universe: "Elsewhere" },
    { Mathematics: "Intuitionist" },
    { Simulation: 1 },
    { Consensus: 1 },
    { Observer: "Cat" },
  ];
  for (const node of cases) {
    assert.ok(realityCenter(node).y > 0, `expected a downward centre for ${JSON.stringify(node)}`);
  }
});

test("layoutTarget: even a non-home place in a new reality lands below base reality", () => {
  // The physical ring can nudge a node upward within its reality, but the
  // downward reality offset must always win so nodes are created downward.
  for (const place of ["Park", "Station", "City Centre", "Museum", "Harbour"]) {
    assert.ok(layoutTarget({ Location: place, Timeline: "T1" }).y > 0, `${place} should sit below base`);
    assert.ok(layoutTarget({ Location: place, Quantum: "Q1" }).y > 0, `${place} should sit below base`);
  }
});

test("layoutTarget: a reality's Home sits dead centre on its reality centre", () => {
  const node = { Location: "Home", Quantum: "Q1" };
  assert.deepEqual(layoutTarget(node), realityCenter(node));
});

test("layoutTarget: the same place lands at the same relative offset in every reality", () => {
  const base = layoutTarget({ Location: "Park" });
  const q1 = layoutTarget({ Location: "Park", Quantum: "Q1" });
  const q1Center = realityCenter({ Quantum: "Q1" });
  // The offset from each reality's centre is identical for the same place.
  assert.ok(Math.abs((base.x - 0) - (q1.x - q1Center.x)) < 1e-9);
  assert.ok(Math.abs((base.y - 0) - (q1.y - q1Center.y)) < 1e-9);
});

test("layoutTarget: non-home places sit exactly PHYS_RADIUS from their reality centre", () => {
  const node = { Location: "Station", Timeline: "T1" };
  const c = realityCenter(node);
  const p = layoutTarget(node);
  const r = Math.hypot(p.x - c.x, p.y - c.y);
  assert.ok(Math.abs(r - PHYS_RADIUS) < 1e-9);
});

test("layoutTarget is deterministic for the same node", () => {
  const node = { Location: "City Centre", Quantum: "Q2", Timeline: "T1" };
  assert.deepEqual(layoutTarget(node), layoutTarget(node));
});
