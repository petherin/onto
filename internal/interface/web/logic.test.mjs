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
  soundSpec,
  SOUND_SPEC,
  DEFAULT_SOUND,
  BLOCKED_SOUND,
  sessionMoved,
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
  fitView,
  FIT_MARGIN,
  FIT_GROUP_MARGIN,
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

test("soundSpec gives each reality transition its own character", () => {
  // Cues lead with a noise texture (whoosh/swell) then stack their tonal body,
  // so the tonal character lives in the layers rather than the first voice.
  assert.ok(soundSpec("timeline").voices.some((v) => v.type === "sawtooth"), "timeline has a saw riser");
  assert.ok(soundSpec("simulation").voices.some((v) => v.type === "square"), "simulation is buzzy squares");
  // The clock still lands two crisp ticks at the same pitch, one after the other.
  const ticks = soundSpec("time").voices.filter((v) => v.type === "sine" && v.freq === 1200);
  assert.equal(ticks.length, 2, "the clock strikes twice");
  assert.ok(ticks[1].delay > ticks[0].delay, "the second tick arrives after the first");
});

test("every reality transition is layered like a film cue, not a bare tone", () => {
  // Each named transition stacks multiple voices (texture + sub + body + shimmer)
  // so it reads as cinematic rather than a single oscillator.
  for (const mode of Object.keys(SOUND_SPEC)) {
    assert.ok(soundSpec(mode).voices.length >= 2, `${mode} should be layered`);
    // Every cue opens with a noise texture leading into the tonal hit.
    assert.ok(soundSpec(mode).voices.some((v) => v.type === "noise"), `${mode} needs a noise texture`);
  }
  // Cinematic weight: the universe reset carries a deep sub-bass drop.
  const universe = soundSpec("universe").voices;
  assert.ok(universe.some((v) => v.freqEnd !== undefined && v.freqEnd <= 60), "universe needs a sub drop");
});

test("soundSpec falls back to the default sound for unknown modes", () => {
  assert.equal(soundSpec("walk").voices, DEFAULT_SOUND);
  assert.equal(soundSpec(undefined).voices, DEFAULT_SOUND);
});

test("soundSpec duration spans the latest-finishing voice's delay, attack and release", () => {
  // The chime tail under the clock ticks outlasts the ticks themselves, so the
  // reported duration follows whichever voice finishes last, not the array order.
  for (const mode of [...Object.keys(SOUND_SPEC), "walk"]) {
    const { voices, duration } = soundSpec(mode);
    const end = Math.max(...voices.map((v) => (v.delay || 0) + v.attack + v.release));
    assert.equal(duration, end, `${mode} duration must span its longest voice`);
  }
});

test("every declared sound voice has a sane, playable envelope", () => {
  const specs = [...Object.values(SOUND_SPEC), DEFAULT_SOUND, BLOCKED_SOUND];
  for (const voices of specs) {
    for (const v of voices) {
      // Pitched voices carry a frequency; noise voices are broadband, so they
      // shape their character through a filter instead of a pitch.
      if (v.type === "noise") {
        assert.ok(v.filter, "a noise voice must be shaped by a filter");
      } else {
        assert.ok(v.freq > 0, `freq must be positive, got ${v.freq}`);
      }
      assert.ok(v.gain > 0 && v.gain <= 1, `gain must be in (0,1], got ${v.gain}`);
      assert.ok(v.attack > 0 && v.release > 0, "attack and release must be positive");
      if (v.freqEnd !== undefined) assert.ok(v.freqEnd > 0, "freqEnd must be positive for a glide");
      if (v.pan !== undefined) assert.ok(v.pan >= -1 && v.pan <= 1, `pan must be in [-1,1], got ${v.pan}`);
      if (v.filter) {
        assert.ok(v.filter.freq > 0, "filter start cutoff must be positive");
        if (v.filter.freqEnd !== undefined) assert.ok(v.filter.freqEnd > 0, "filter sweep target must be positive");
      }
      if (v.lfo) {
        assert.ok(["gain", "pitch", "filter"].includes(v.lfo.target), `lfo target must be gain, pitch or filter, got ${v.lfo.target}`);
        assert.ok(v.lfo.freq > 0, "lfo rate must be positive");
        assert.ok(v.lfo.depth > 0, "lfo depth must be positive");
      }
      // Grunge/organic shaping fields must stay in sane ranges.
      if (v.drive !== undefined) assert.ok(v.drive > 0 && v.drive <= 1, `drive must be in (0,1], got ${v.drive}`);
      if (v.jitter !== undefined) assert.ok(v.jitter > 0, `jitter must be positive cents, got ${v.jitter}`);
      if (v.ring) {
        assert.ok(v.ring.freq > 0, "ring frequency must be positive");
        if (v.ring.depth !== undefined) assert.ok(v.ring.depth > 0 && v.ring.depth <= 1, `ring depth must be in (0,1], got ${v.ring.depth}`);
      }
      if (v.fm) {
        assert.ok(v.fm.ratio > 0, "fm ratio must be positive");
        assert.ok(v.fm.depth > 0, "fm depth must be positive");
      }
    }
  }
});

test("the palette leans on organic, grungy techniques, not clean oscillators", () => {
  // The Star Wars / Alien character comes from abused sources — distortion,
  // ring modulation, inharmonic FM, and per-play jitter — so at least some voice
  // must reach for each, or we've drifted back to a plain synthesiser.
  const all = Object.values(SOUND_SPEC).flat();
  assert.ok(all.some((v) => v.drive), "some voice should be driven/distorted for grit");
  assert.ok(all.some((v) => v.ring), "some voice should be ring-modulated for metallic character");
  assert.ok(all.some((v) => v.fm), "some voice should use FM for inharmonic, organic timbres");
  assert.ok(all.some((v) => v.jitter), "some voice should jitter for organic imperfection");
});

test("soundSpec serves the blocked cue and it stays out of the transition palette", () => {
  // A refused press must sound distinct from any transition, so the blocked cue
  // is its own spec, reached only through the "blocked" mode.
  const { voices, duration } = soundSpec("blocked");
  assert.equal(voices, BLOCKED_SOUND, "the blocked mode must serve BLOCKED_SOUND");
  assert.ok(duration > 0, "the blocked cue must have a positive duration");
  assert.ok(!Object.values(SOUND_SPEC).includes(BLOCKED_SOUND), "the blocked cue must not be a transition sound");
});

test("sessionMoved tells a real move from a refused one", () => {
  // A genuine move changes the location or spends budget; when neither changes
  // the action was blocked. The very first apply (no prior session) counts as a
  // move so the opening state is never mistaken for a block.
  const base = { Location: "a", CumulativeCost: 10 };
  assert.equal(sessionMoved(null, base), true, "first apply is never a block");
  assert.equal(sessionMoved(base, base), false, "an identical snapshot is a block");
  assert.equal(sessionMoved(base, { Location: "b", CumulativeCost: 10 }), true, "a new location is a move");
  assert.equal(sessionMoved(base, { Location: "a", CumulativeCost: 12 }), true, "spent budget is a move");
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

const IDENTITY_VIEW = { rotX: 0, rotY: 0, scale: 1, ox: 0, oy: 0 };

test("fitView returns a centred, unit-scale frame for an empty set", () => {
  assert.deepEqual(fitView([], IDENTITY_VIEW, 800, 600), { scale: 1, ox: 0, oy: 0 });
});

test("fitView centres the bounding box of the nodes", () => {
  // Two nodes symmetric about the origin on a flat (z=0) map, at identity
  // rotation, already centre on the canvas — so the pan is zero.
  const nodes = [
    { x: -100, y: -50, z: 0 },
    { x: 100, y: 50, z: 0 },
  ];
  const fit = fitView(nodes, IDENTITY_VIEW, 800, 600);
  assert.ok(Math.abs(fit.ox) < 1e-9);
  assert.ok(Math.abs(fit.oy) < 1e-9);
  assert.ok(fit.scale > 0);
});

test("fitView pans an off-centre cluster back to the middle", () => {
  const nodes = [
    { x: 200, y: 200, z: 0 },
    { x: 260, y: 240, z: 0 },
  ];
  const w = 800, h = 600;
  const fit = fitView(nodes, IDENTITY_VIEW, w, h);
  // Applying the returned frame, the cluster's midpoint should land on centre.
  const framed = { ...IDENTITY_VIEW, scale: fit.scale, ox: fit.ox, oy: fit.oy };
  const pa = project(nodes[0], framed, w, h);
  const pb = project(nodes[1], framed, w, h);
  const midX = (pa.x + pb.x) / 2, midY = (pa.y + pb.y) / 2;
  assert.ok(Math.abs(midX - w / 2) < 1e-6);
  assert.ok(Math.abs(midY - h / 2) < 1e-6);
});

test("fitView frames a wide spread within the margin", () => {
  const nodes = [
    { x: -4000, y: 0, z: 0 },
    { x: 4000, y: 0, z: 0 },
  ];
  const w = 800, h = 600;
  const fit = fitView(nodes, IDENTITY_VIEW, w, h);
  const framed = { ...IDENTITY_VIEW, scale: fit.scale, ox: fit.ox, oy: fit.oy };
  const pa = project(nodes[0], framed, w, h);
  const pb = project(nodes[1], framed, w, h);
  const spanX = Math.abs(pb.x - pa.x);
  // The span fits inside the available width (canvas minus the margin padding).
  assert.ok(spanX <= w * (1 - FIT_MARGIN) + 1e-6);
});

test("fitView keeps a deep, rotated journey on screen (regression)", () => {
  // Mimics a long journey: nodes spread across x/y and deep in z (nesting), seen
  // through a tilted, yawed view. Some rotate to a large depth where the naive
  // perspective-projected fit produced a wild outlier, collapsed the scale, and
  // parked the whole map off-screen. Every node must land inside the canvas.
  const w = 900, h = 640;
  const view = { rotX: 0.5, rotY: 0.35, scale: 1, ox: 0, oy: 0 };
  const nodes = [];
  for (let i = 0; i < 26; i++) {
    nodes.push({
      x: (i - 13) * 160 + (i % 3) * 40,
      y: ((i * 37) % 400) - 200,
      z: (i % 8) * 110, // layerZ-style nesting depth up to ~770
    });
  }
  const fit = fitView(nodes, view, w, h);
  assert.ok(fit.scale > 0 && Number.isFinite(fit.scale));
  const framed = { ...view, scale: fit.scale, ox: fit.ox, oy: fit.oy };
  for (const n of nodes) {
    const p = project(n, framed, w, h);
    assert.ok(p.x >= -1 && p.x <= w + 1, `x on screen: ${p.x}`);
    assert.ok(p.y >= -1 && p.y <= h + 1, `y on screen: ${p.y}`);
  }
});

test("FIT_GROUP_MARGIN frames a group as a centred hero, leaving room around it", () => {
  // Framing a just-revealed group (frameToGroup uses fitView with the larger
  // FIT_GROUP_MARGIN) must centre the group and keep it well inside the canvas,
  // so the surrounding older realities stay visible around the edges.
  assert.ok(FIT_GROUP_MARGIN > FIT_MARGIN, "the group margin must exceed the fit-everything margin");
  const group = [
    { x: -60, y: -40, z: 0 },
    { x: 60, y: 40, z: 0 },
  ];
  const w = 800, h = 600;
  const fit = fitView(group, IDENTITY_VIEW, w, h, FIT_GROUP_MARGIN);
  const framed = { ...IDENTITY_VIEW, scale: fit.scale, ox: fit.ox, oy: fit.oy };
  const pa = project(group[0], framed, w, h);
  const pb = project(group[1], framed, w, h);
  // Centred on the canvas.
  assert.ok(Math.abs((pa.x + pb.x) / 2 - w / 2) < 1e-6, "group must centre horizontally");
  assert.ok(Math.abs((pa.y + pb.y) / 2 - h / 2) < 1e-6, "group must centre vertically");
  // The group spans only the inner (1 - margin) of the canvas, leaving context
  // room around it — so it reads as a focused hero, not a full-bleed fill.
  const spanX = Math.abs(pb.x - pa.x);
  assert.ok(spanX <= w * (1 - FIT_GROUP_MARGIN) + 1e-6, "group must sit within the group margin");
});

test("fitView centres a deep group at a steep pitch on the canvas (no bottom drift)", () => {
  // Regression: at the near-top-down vertical view, screen-y is y1*scale*persp,
  // not just y1*scale. Centring on the orthographic midpoint (persp ignored) left
  // deeper groups sitting progressively lower until, after a few journeys, they
  // slid off the bottom. A group deep in z, framed at that pitch, must still land
  // with its projected centre on the canvas centre.
  const w = 900, h = 640;
  const view = { rotX: -1.42, rotY: 0, scale: 1, ox: 0, oy: 0 };
  const group = [
    { x: -40, y: 600, z: 520 },
    { x: 40, y: 660, z: 520 },
    { x: 0, y: 630, z: 560 },
  ];
  const fit = fitView(group, view, w, h, FIT_GROUP_MARGIN);
  const framed = { ...view, scale: fit.scale, ox: fit.ox, oy: fit.oy };
  let cx = 0, cy = 0;
  for (const n of group) { const p = project(n, framed, w, h); cx += p.x; cy += p.y; }
  cx /= group.length; cy /= group.length;
  assert.ok(Math.abs(cx - w / 2) < 1e-6, `deep group must centre horizontally, got ${cx}`);
  assert.ok(Math.abs(cy - h / 2) < 1e-6, `deep group must centre vertically, got ${cy}`);
});
