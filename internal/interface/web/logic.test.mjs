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
  TRAVEL_SOUND,
  sessionMoved,
  detectTransition,
  transitionIntensity,
  dramaSpec,
  impactVoices,
  SHATTER_MIN_INTENSITY,
  TRANSITIONS,
  TRANSITIONS_BY_COST,
  TRANSITION_LEGEND,
  AXIS_MODE,
  requiredTransition,
  isPhysical,
  PHYSICAL_MODES,
  physicalRoute,
  routeTotals,
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
  panToScreenOrbit,
  hashString,
  axisLevel,
  realityCenter,
  layoutTarget,
  AXIS_DIR,
  REALITY_SPREAD,
  PHYS_RADIUS,
  CHAIN_STEP,
  CHAIN_FAN,
  chainIndices,
  realityShells,
  SHELL_STYLE,
  SHELL_PAD,
  SHELL_GAP,
  shellStyle,
  edgeRestLength,
  REST_PHYSICAL,
  REST_TRANSITION,
  edgeWeight,
  EDGE_COST_MIN,
  EDGE_COST_MAX,
  EDGE_WIDTH_MIN,
  EDGE_WIDTH_MAX,
  EDGE_ALPHA_MIN,
  EDGE_ALPHA_MAX,
  spawnHalo,
  SPAWN_HALO_MS,
  SPAWN_HALO_ALPHA,
  project,
  FOCAL,
  fitView,
  FIT_MARGIN,
  FIT_GROUP_MARGIN,
  FIT_GROUP_MIN_SPAN,
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
  const specs = [...Object.values(SOUND_SPEC), DEFAULT_SOUND, BLOCKED_SOUND, TRAVEL_SOUND];
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

test("soundSpec serves the travel cue and it stays out of the transition palette", () => {
  // Ordinary physical travel is a plain location change, not a reality shift, so
  // it has its own soft cue reached only through the "travel" mode.
  const { voices, duration } = soundSpec("travel");
  assert.equal(voices, TRAVEL_SOUND, "the travel mode must serve TRAVEL_SOUND");
  assert.ok(duration > 0, "the travel cue must have a positive duration");
  assert.ok(!Object.values(SOUND_SPEC).includes(TRAVEL_SOUND), "the travel cue must not be a transition sound");
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

test("detectTransition names the mathematical axis with the shared 'math' mode", () => {
  // Regression: the mode string detectTransition returns must match the key the
  // style, effect, and sound tables use (all "math"), so a structure jump gets
  // its own teal grid + crystal chord instead of falling back to the ripple and
  // the default tone. The bug was a "maths" typo in the TRANSITIONS entry.
  const base = { Mathematics: "Classical" };
  const mode = detectTransition(base, { ...base, Mathematics: "NonEuclidean" });
  assert.equal(mode, "math");
  assert.equal(effectSpec(mode).kind, "grid");
  assert.equal(soundSpec(mode).voices, SOUND_SPEC.math);
  assert.notEqual(modeStyle(mode), DEFAULT_MODE_STYLE);
});

test("transitionIntensity rises with cost, is 0 for the cheapest, 1 for the dearest", () => {
  // Log-scaled drama: the cheapest transition (observer, 2 σ) sits at 0 and the
  // dearest (mathematical, 50,000 σ) at 1, with the rest strictly increasing by
  // cost in between, so shake/storm/sound all track what a move costs.
  assert.equal(transitionIntensity("observer"), 0);
  assert.equal(transitionIntensity("math"), 1);
  const ordered = ["observer", "consensus", "simulation", "quantum", "time", "timeline", "universe", "math"];
  for (let k = 1; k < ordered.length; k++) {
    assert.ok(
      transitionIntensity(ordered[k]) > transitionIntensity(ordered[k - 1]),
      `intensity should increase from ${ordered[k - 1]} to ${ordered[k]}`,
    );
  }
  // Physical/unknown modes carry no cost, so they add no drama.
  assert.equal(transitionIntensity("walk"), 0);
  assert.equal(transitionIntensity(undefined), 0);
});

test("dramaSpec suppresses the number storm below the threshold and scales it above", () => {
  // Cheap shifts keep their quiet character — a faint shake, no glyphs — while
  // costly ones burst into a longer, wider storm with more, bigger numbers.
  const cheap = dramaSpec(transitionIntensity("observer"));
  assert.equal(cheap.shatter, false);
  assert.equal(cheap.glyphs, 0);
  assert.equal(cheap.shakeAmp, 0);

  const big = dramaSpec(transitionIntensity("math"));
  assert.equal(big.shatter, true);
  assert.ok(big.glyphs > 0, "a structure jump throws glyphs");
  assert.ok(big.shakeAmp > cheap.shakeAmp, "a dearer move shakes harder");
  assert.ok(big.duration > dramaSpec(SHATTER_MIN_INTENSITY).duration, "and runs longer");

  // Intensity is clamped to [0,1], so out-of-range inputs never blow up the storm.
  assert.equal(dramaSpec(-5).intensity, 0);
  assert.equal(dramaSpec(9).intensity, 1);
});

test("impactVoices adds a scaled low-end wallop in the ordinary voice schema", () => {
  // The extra impact must be renderable by the same player path as any voice
  // (type + gain + attack + release), and hit harder for a dearer move.
  const soft = impactVoices(0.4);
  const hard = impactVoices(1);
  for (const v of [...soft, ...hard]) {
    assert.ok(typeof v.type === "string" && v.gain > 0 && v.attack >= 0 && v.release > 0);
  }
  assert.ok(hard[0].gain > soft[0].gain, "the sub boom is louder for a bigger move");
});

test("AXIS_MODE derives from TRANSITIONS in its most-exotic detection order", () => {
  // AXIS_MODE drives detectTransition's tie-break, so it must preserve the
  // declaration order of the single source of truth.
  assert.equal(AXIS_MODE.length, TRANSITIONS.length);
  TRANSITIONS.forEach((t, i) => {
    assert.deepEqual(AXIS_MODE[i], [t.axis, t.mode]);
  });
});

test("TRANSITION_LEGEND derives from the cost-sorted view, cheapest-first", () => {
  // The visible legend/modal read cheapest-first, independent of the detection
  // order, so they derive from TRANSITIONS_BY_COST.
  assert.equal(TRANSITIONS_BY_COST.length, TRANSITIONS.length);
  assert.equal(TRANSITION_LEGEND.length, TRANSITIONS.length);
  TRANSITIONS_BY_COST.forEach((t, i) => {
    assert.deepEqual(TRANSITION_LEGEND[i], { mode: t.mode, label: t.label, command: t.command });
  });
  // Costs are non-decreasing across the sorted view.
  for (let i = 1; i < TRANSITIONS_BY_COST.length; i++) {
    assert.ok(
      TRANSITIONS_BY_COST[i - 1].costValue <= TRANSITIONS_BY_COST[i].costValue,
      `legend not ascending by cost at index ${i}`,
    );
  }
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

test("isPhysical: the travel-command modes are physical, transitions are not", () => {
  // Mirrors universe.TravelModeVO.IsPhysical() — the solid-blue travel modes.
  for (const mode of ["walk", "cycle", "drive", "rail", "flight", "orbit", "warp"]) {
    assert.equal(isPhysical(mode), true, `${mode} should be physical`);
    assert.ok(PHYSICAL_MODES.has(mode), `${mode} should be in PHYSICAL_MODES`);
  }
  for (const mode of ["quantum", "timeline", "universe", "observer", "time", undefined, "nonsense"]) {
    assert.equal(isPhysical(mode), false, `${String(mode)} should not be physical`);
  }
});

test("physicalRoute: returns null for missing or identical endpoints", () => {
  const edges = [{ From: "home", To: "park", Mode: "walk", Cost: 1, Distance: 1 }];
  assert.equal(physicalRoute(null, "park", edges), null);
  assert.equal(physicalRoute("home", null, edges), null);
  assert.equal(physicalRoute("home", "home", edges), null);
});

test("physicalRoute: finds a multi-hop physical path and preserves order", () => {
  const edges = [
    { From: "home", To: "station", Mode: "walk", Cost: 1, Distance: 1.6 },
    { From: "station", To: "city", Mode: "rail", Cost: 3, Distance: 3.0 },
  ];
  const path = physicalRoute("home", "city", edges);
  assert.equal(path.length, 2);
  assert.deepEqual(path.map((e) => e.To), ["station", "city"]);
});

test("physicalRoute: follows only physical edges, never a reality transition", () => {
  // branch is reachable only across a quantum hop, so there is no physical route.
  const edges = [
    { From: "home", To: "branch", Mode: "quantum", Cost: 20, Distance: 0 },
    { From: "branch", To: "far", Mode: "walk", Cost: 1, Distance: 1 },
  ];
  assert.equal(physicalRoute("home", "far", edges), null);
});

test("physicalRoute: prefers the fewest-hops route, mirroring the BFS pathfinder", () => {
  // A direct drive and a two-hop walk both reach city; BFS returns the shorter.
  const edges = [
    { From: "home", To: "mid", Mode: "walk", Cost: 1, Distance: 1 },
    { From: "mid", To: "city", Mode: "walk", Cost: 1, Distance: 1 },
    { From: "home", To: "city", Mode: "drive", Cost: 5, Distance: 8 },
  ];
  const path = physicalRoute("home", "city", edges);
  assert.equal(path.length, 1);
  assert.equal(path[0].Mode, "drive");
});

test("physicalRoute: returns null when there is no route at all", () => {
  const edges = [{ From: "home", To: "park", Mode: "walk", Cost: 1, Distance: 1 }];
  assert.equal(physicalRoute("home", "island", edges), null);
});

test("routeTotals: sums cost and distance and counts hops", () => {
  const path = [
    { From: "home", To: "station", Mode: "walk", Cost: 1, Distance: 1.6 },
    { From: "station", To: "city", Mode: "rail", Cost: 3, Distance: 3.0 },
  ];
  const totals = routeTotals(path);
  assert.equal(totals.steps, 2);
  assert.equal(totals.cost, 4);
  assert.ok(Math.abs(totals.distance - 4.6) < 1e-9);
});

test("routeTotals: a null or empty path totals to zero", () => {
  assert.deepEqual(routeTotals(null), { steps: 0, cost: 0, distance: 0 });
  assert.deepEqual(routeTotals([]), { steps: 0, cost: 0, distance: 0 });
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

test("edgeWeight: cost at/under the floor sits at the thin, faint end", () => {
  const floor = edgeWeight(EDGE_COST_MIN);
  assert.ok(Math.abs(floor.width - EDGE_WIDTH_MIN) < 1e-9);
  assert.ok(Math.abs(floor.alpha - EDGE_ALPHA_MIN) < 1e-9);
  // 0, undefined, and sub-floor costs all clamp to the same thin, faint end.
  assert.deepEqual(edgeWeight(0), floor);
  assert.deepEqual(edgeWeight(undefined), floor);
  assert.deepEqual(edgeWeight(-100), floor);
});

test("edgeWeight: cost at/over the cap saturates at the thick, opaque end", () => {
  const cap = edgeWeight(EDGE_COST_MAX);
  assert.ok(Math.abs(cap.width - EDGE_WIDTH_MAX) < 1e-9);
  assert.ok(Math.abs(cap.alpha - EDGE_ALPHA_MAX) < 1e-9);
  // A runaway cost past the cap can't blow the map out — it stays clamped.
  assert.deepEqual(edgeWeight(EDGE_COST_MAX * 1000), cap);
});

test("edgeWeight: width and alpha rise monotonically with cost", () => {
  const cheap = edgeWeight(2);       // an observe shift
  const mid = edgeWeight(500);
  const dear = edgeWeight(50000);    // a mathematical jump
  assert.ok(cheap.width < mid.width && mid.width < dear.width);
  assert.ok(cheap.alpha < mid.alpha && mid.alpha < dear.alpha);
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

test("panToScreenOrbit: at the drag start it lands the grabbed pivot exactly under the cursor", () => {
  // The orbit anchor. At drag start the rotation is unchanged, so the pivot (from
  // unproject) still has rotated depth 0 and perspective factor 1 — the orthographic
  // anchor and the perspective projection agree, so the pivot sits exactly under the
  // cursor. Tested at the near-edge-on Ladder angle, where the fly-away used to bite.
  const cursorX = 520, cursorY = 250;
  const view = { scale: 1.2, ox: 40, oy: -30, rotX: -1.42, rotY: 0 };
  const pivot = unproject(cursorX, cursorY, view, 800, 600);
  const pan = panToScreenOrbit(pivot, view, 800, 600, cursorX, cursorY);
  const proj = project(pivot, { ...view, ...pan }, 800, 600);
  assert.ok(Math.abs(proj.x - cursorX) < 1e-9 && Math.abs(proj.y - cursorY) < 1e-9);
});

test("panToScreenOrbit: the pan stays bounded through a pitch sweep that sends the perspective pan flying", () => {
  // The "flies away when I rotate" regression. Zoomed out (small scale) at the
  // grazing Ladder angle, a pivot grabbed away from centre sits deep in world space.
  // Sweeping the pitch swings it across the focal plane, where the perspective
  // panToScreen's factor (FOCAL/(FOCAL+z2)) blows up and hurls the pan tens of
  // thousands of pixels off screen. The orthographic panToScreenOrbit is bounded by
  // the on-screen extent, so it never runs away.
  const W = 800, H = 600;
  const view = { scale: 0.05, ox: 40, oy: -30, rotX: -1.42, rotY: 0 };
  const cursorX = 700, cursorY = 120;
  const pivot = unproject(cursorX, cursorY, view, W, H);
  let orbitMax = 0, perspMax = 0;
  for (let d = -1.5; d <= 1.5 + 1e-9; d += 0.05) {
    const rot = { ...view, rotX: view.rotX + d };
    const o = panToScreenOrbit(pivot, rot, W, H, cursorX, cursorY);
    const p = panToScreen(pivot, rot, W, H, cursorX, cursorY);
    orbitMax = Math.max(orbitMax, Math.abs(o.ox), Math.abs(o.oy));
    perspMax = Math.max(perspMax, Math.abs(p.ox), Math.abs(p.oy));
  }
  assert.ok(orbitMax < 5000, `orbit pan stayed bounded (was ${orbitMax})`);
  assert.ok(perspMax > 50000, `perspective pan blew up (was ${perspMax})`);
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

test("chainIndices: splits a generated ID into its anchor and numeric chain", () => {
  // A base-reality chain and a branched chain both keep the seed slug as anchor
  // (hyphenated slugs included) and expose only the pure-digit chain segments;
  // reality-axis suffixes (q1/t1) are neither anchor nor chain.
  assert.deepEqual(chainIndices("well-1-2-3"), { anchor: "well", indices: ["1", "2", "3"] });
  assert.deepEqual(chainIndices("well-1-1-q1-t1"), { anchor: "well", indices: ["1", "1"] });
  assert.deepEqual(chainIndices("kirkstall-abbey-1"), { anchor: "kirkstall-abbey", indices: ["1"] });
});

test("chainIndices: a named seed place (no chain, or a reality suffix only) has no indices", () => {
  // Empty indices is what marks a node as a non-generated seed place; the anchor
  // is only consumed when indices are present, so only the indices are asserted.
  assert.deepEqual(chainIndices("kirkstall-abbey").indices, []);
  assert.deepEqual(chainIndices("kirkstall-abbey-c1").indices, []);
});

test("layoutTarget: a generated chain node descends one CHAIN_STEP per depth", () => {
  // A chain is laid out as a strictly-downward tree: each step down the chain
  // drops the node exactly one CHAIN_STEP further down the screen (world +y), so
  // depth d sits at y = centre + PHYS_RADIUS + d*CHAIN_STEP. Because y only ever
  // increases with depth, a child is always below its parent — new nodes are
  // created downward and never come back up.
  const c = realityCenter({ Quantum: "Q1", Timeline: "T1" });
  let lastY = -Infinity;
  for (let d = 1; d <= 4; d++) {
    const id = "well-" + Array(d).fill("1").join("-") + "-q1-t1";
    const p = layoutTarget({ ID: id, Location: "Node", Quantum: "Q1", Timeline: "T1" });
    assert.ok(Math.abs((p.y - c.y) - (PHYS_RADIUS + d * CHAIN_STEP)) < 1e-9, `depth ${d} y offset`);
    assert.ok(p.y > lastY, `depth ${d} sits strictly below depth ${d - 1}`);
    lastY = p.y;
  }
});

test("layoutTarget: a generated chain child never rises above its parent", () => {
  // The core "new nodes come back up" regression: walking any chain from its
  // shallowest node to its deepest, y must be monotonically non-decreasing, for
  // several distinct anchors and branch indices — no step ever moves upward.
  const chains = [
    ["well-1-q1", "well-1-1-q1", "well-1-1-1-q1", "well-1-1-1-1-q1"],
    ["snicket-2-t1", "snicket-2-1-t1", "snicket-2-1-3-t1"],
    ["ginnel-3-2-q2-t1", "ginnel-3-2-1-q2-t1"],
  ];
  for (const chain of chains) {
    let lastY = -Infinity;
    for (const id of chain) {
      const p = layoutTarget({ ID: id, Location: "Node", Quantum: "Q1", Timeline: "T1" });
      assert.ok(p.y >= lastY - 1e-9, `${id} must not rise above its parent (y ${p.y} < ${lastY})`);
      lastY = p.y;
    }
  }
});

test("layoutTarget: sibling generated nodes separate horizontally instead of stacking", () => {
  // Same anchor and depth but a different final chain index must fan apart across
  // x (at the same y row), so a cluster's siblings are individually clickable
  // rather than collapsed onto one point.
  const base = { Quantum: "Q1", Timeline: "T1", Location: "Node" };
  const a = layoutTarget({ ...base, ID: "well-1-1-q1-t1" });
  const b = layoutTarget({ ...base, ID: "well-1-2-q1-t1" });
  // Siblings share a depth, so they share a row.
  assert.ok(Math.abs(a.y - b.y) < 1e-9, "siblings sit on the same downward row");
  // Adjacent indices fan a full CHAIN_FAN apart — enough to hit-test each one.
  assert.ok(Math.abs(a.x - b.x) >= CHAIN_FAN - 1e-9, `siblings should fan apart, got ${Math.abs(a.x - b.x)}`);
});

test("layoutTarget: a generated chain node is deterministic", () => {
  const node = { ID: "well-1-2-1-q1-t1", Location: "Node", Quantum: "Q1", Timeline: "T1" };
  assert.deepEqual(layoutTarget(node), layoutTarget(node));
});

test("realityShells: an empty set yields no shells", () => {
  assert.deepEqual(realityShells([]), []);
});

test("realityShells: base-only nodes nest one shell per level, outermost first", () => {
  // All base (Classical/Origin/Prime), so the three levels share the same member
  // cloud: one math shell (Level IV), one universe shell (Level II) and one
  // timeline shell (Level I). They arrive outermost-first (math, universe,
  // timeline) so the caller paints big faint shells behind the nested ones.
  const shells = realityShells([
    { ID: "a", Location: "Alpha" },
    { ID: "b", Location: "Beta" },
    { ID: "c", Location: "Gamma" },
  ]);
  assert.deepEqual(shells.map((s) => s.kind), ["math", "universe", "timeline"]);
  for (const s of shells) assert.equal(s.count, 3);
});

test("realityShells: a nested shell's radius sits inside its parent's", () => {
  // One member cloud shared across all three levels, so the levels share a centroid
  // and raw radius. Each parent is then grown to enclose its child by SHELL_GAP
  // (encloseChildren), so math ⊃ universe ⊃ timeline holds by exactly one gap each.
  const [math, universe, timeline] = realityShells([
    { ID: "a", Location: "Alpha" },
    { ID: "b", Location: "Beta" },
  ]);
  assert.ok(math.radius > universe.radius, "math shell encloses the universe shell");
  assert.ok(universe.radius > timeline.radius, "universe shell encloses the timeline shell");
  assert.ok(Math.abs((math.radius - universe.radius) - SHELL_GAP) < 1e-9);
  assert.ok(Math.abs((universe.radius - timeline.radius) - SHELL_GAP) < 1e-9);
});

test("realityShells: a shell's centre is the centroid of its members' homes", () => {
  const nodes = [{ ID: "a", Location: "Alpha" }, { ID: "b", Location: "Beta" }];
  const homes = nodes.map((n) => layoutTarget(n));
  const cx = (homes[0].x + homes[1].x) / 2;
  const cy = (homes[0].y + homes[1].y) / 2;
  const math = realityShells(nodes).find((s) => s.kind === "math");
  assert.ok(Math.abs(math.cx - cx) < 1e-9);
  assert.ok(Math.abs(math.cy - cy) < 1e-9);
});

test("realityShells: two universes under one structure share a math shell but split the universe shell", () => {
  // Both nodes keep Classical maths, so there is a single math shell enclosing
  // both; they differ by Universe, so each gets its own universe shell (and its
  // own timeline shell, since a timeline is nested inside a universe).
  const shells = realityShells([
    { ID: "a", Location: "Home" },
    { ID: "b", Location: "Home", Universe: "U1" },
  ]);
  const math = shells.filter((s) => s.kind === "math");
  const universe = shells.filter((s) => s.kind === "universe");
  const timeline = shells.filter((s) => s.kind === "timeline");
  assert.equal(math.length, 1);
  assert.equal(math[0].count, 2);
  assert.equal(universe.length, 2);
  assert.equal(timeline.length, 2);
});

test("realityShells: a quantum branch adds no shell of its own", () => {
  // The quantum axis (Tegmark Level III) is deliberately not a shell level, so
  // two nodes differing only by Quantum still share one shell at every level —
  // exactly three shells, each enclosing both nodes.
  const shells = realityShells([
    { ID: "a", Location: "Home" },
    { ID: "b", Location: "Home", Quantum: "Q1" },
  ]);
  assert.deepEqual(shells.map((s) => s.kind), ["math", "universe", "timeline"]);
  for (const s of shells) assert.equal(s.count, 2);
});

test("realityShells: a distinct mathematical structure gets its own math shell", () => {
  const shells = realityShells([
    { ID: "a", Location: "Home" },
    { ID: "b", Location: "Home", Mathematics: "M1" },
  ]);
  const math = shells.filter((s) => s.kind === "math");
  assert.equal(math.length, 2);
  assert.deepEqual(math.map((s) => s.count).sort(), [1, 1]);
});

test("realityShells is deterministic for the same nodes", () => {
  const nodes = [
    { ID: "a", Location: "Alpha", Universe: "U1", Timeline: "T1" },
    { ID: "b", Location: "Beta", Quantum: "Q2" },
  ];
  assert.deepEqual(realityShells(nodes), realityShells(nodes));
});

test("realityShells: grows a shell to enclose the live positions it is given", () => {
  // With a posOf that reports a node far from its layout home, the shell stretches
  // to wrap that live position (plus padding) rather than the static home — so a
  // shell grows to contain generated nodes wherever the layout settles them.
  const nodes = [{ ID: "a", Location: "Home" }, { ID: "b", Location: "Home" }];
  const live = { a: { x: 0, y: 0, z: 0 }, b: { x: 400, y: 0, z: 0 } };
  const math = realityShells(nodes, (n) => live[n.ID]).find((s) => s.kind === "math");
  // Centre is the midpoint (200,0); the far member sits 200 out, plus SHELL_PAD.
  // These base-only nodes also share a nested universe and timeline shell at the
  // same centroid, so the math shell is grown two SHELL_GAP steps to enclose them.
  assert.ok(Math.abs(math.cx - 200) < 1e-9);
  assert.ok(Math.abs(math.radius - (200 + SHELL_PAD + 2 * SHELL_GAP)) < 1e-9);
});

test("realityShells: an outlying universe stays inside its structure shell", () => {
  // Two universes under one structure, one far from the structure centroid. The
  // math shell is grown (encloseChildren) so it fully contains the distant universe
  // shell — nesting holds by construction, not by the clusters happening to align.
  const nodes = [{ ID: "a", Location: "Home" }, { ID: "b", Location: "Home", Universe: "U1" }];
  const live = { a: { x: 0, y: 0, z: 0 }, b: { x: 900, y: 0, z: 0 } };
  const shells = realityShells(nodes, (n) => live[n.ID]);
  const math = shells.find((s) => s.kind === "math");
  for (const u of shells.filter((s) => s.kind === "universe")) {
    const reach = Math.hypot(u.cx - math.cx, u.cy - math.cy) + u.radius;
    assert.ok(reach <= math.radius + 1e-9, "universe shell is fully inside the math shell");
  }
});

// ── Shell containment / nesting invariants ───────────────────────────────────
// The map's central promise is that no node ever renders outside its shell and no
// shell escapes its parent. app.js secures this by computing the shells from the
// nodes' already-projected screen positions (drawRealityShells passes a screen-
// space resolver into realityShells), so the ring is measured from exactly where
// the dots land. These tests exercise that contract directly: they hand
// realityShells arbitrary positions (standing in for force-settled / projected
// coordinates) and assert both invariants hold whatever those positions are.

// keyOf mirrors shellAxisValue in logic.js: an empty/missing axis groups with its
// base value, everything else by its own string.
function keyOf(value, base) {
  return value === undefined || value === null || value === "" ? String(base) : String(value);
}

// nodeShells finds the three shells a node belongs to (its math / universe /
// timeline), matched by the same axis keys realityShells groups on.
function nodeShells(shells, n) {
  const m = keyOf(n.Mathematics, DEFAULTS.Mathematics);
  const u = keyOf(n.Universe, DEFAULTS.Universe);
  const t = keyOf(n.Timeline, DEFAULTS.Timeline);
  return {
    math: shells.find((s) => s.kind === "math" && s.math === m),
    universe: shells.find((s) => s.kind === "universe" && s.math === m && s.universe === u),
    timeline: shells.find((s) => s.kind === "timeline" && s.math === m && s.universe === u && s.timeline === t),
  };
}

// A spread of nodes across several structures, universes and timelines (plus a
// quantum branch that adds no shell), used by the containment/nesting tests.
const SHELL_NODES = [
  { ID: "n0", Location: "Home" },
  { ID: "n1", Location: "Home", Universe: "U1" },
  { ID: "n2", Location: "Home", Universe: "U1", Timeline: "T1" },
  { ID: "n3", Location: "Home", Mathematics: "M1" },
  { ID: "n4", Location: "Home", Mathematics: "M1", Universe: "U2", Timeline: "T2" },
  { ID: "n5", Location: "Home", Quantum: "Q1" },
];

// scatter maps an index to a deterministic, widely-spread position, standing in
// for wherever the force layout / projection happens to settle a node.
function scatter(i) {
  const a = (i * 2654435761) % 4096;
  const b = (i * 40503 + 977) % 4096;
  return { x: (a - 2048) * 0.5, y: (b - 2048) * 0.5, z: (i % 7) * 25 - 75 };
}

test("realityShells: every node sits inside each of its shells, wherever it is placed", () => {
  const pos = new Map(SHELL_NODES.map((n, i) => [n.ID, scatter(i)]));
  const shells = realityShells(SHELL_NODES, (n) => pos.get(n.ID));
  for (const n of SHELL_NODES) {
    const p = pos.get(n.ID);
    const own = nodeShells(shells, n);
    for (const kind of ["math", "universe", "timeline"]) {
      const s = own[kind];
      assert.ok(s, `node ${n.ID} has a ${kind} shell`);
      const d = Math.hypot(p.x - s.cx, p.y - s.cy);
      assert.ok(d <= s.radius + 1e-9, `node ${n.ID} is inside its ${kind} shell (d=${d}, r=${s.radius})`);
    }
  }
});

test("realityShells: every child shell is fully enclosed by its parent, wherever nodes are placed", () => {
  const pos = new Map(SHELL_NODES.map((n, i) => [n.ID, scatter(i)]));
  const shells = realityShells(SHELL_NODES, (n) => pos.get(n.ID));
  const maths = shells.filter((s) => s.kind === "math");
  const universes = shells.filter((s) => s.kind === "universe");
  for (const u of universes) {
    const parent = maths.find((s) => s.math === u.math);
    assert.ok(parent, `universe shell ${u.math}/${u.universe} has a math parent`);
    const reach = Math.hypot(u.cx - parent.cx, u.cy - parent.cy) + u.radius;
    assert.ok(reach <= parent.radius + 1e-9, "universe shell is inside its math shell");
  }
  for (const t of shells.filter((s) => s.kind === "timeline")) {
    const parent = universes.find((s) => s.math === t.math && s.universe === t.universe);
    assert.ok(parent, `timeline shell ${t.math}/${t.universe}/${t.timeline} has a universe parent`);
    const reach = Math.hypot(t.cx - parent.cx, t.cy - parent.cy) + t.radius;
    assert.ok(reach <= parent.radius + 1e-9, "timeline shell is inside its universe shell");
  }
});

test("realityShells: computed from projected screen positions, nodes stay inside their shells under rotation", () => {
  // The exact failure the screen-space rework fixes: two universes under one
  // structure whose nodes sit at different depths, so a flat world-space ring at
  // one averaged depth would tip an off-depth node out under perspective. Feeding
  // realityShells the projected screen positions (as drawRealityShells does) makes
  // every node land inside its ring by construction, whatever the rotation.
  const view = { rotX: 0.6, rotY: 1.1, scale: 1.4, ox: 0, oy: 0 };
  const W = 800, H = 600;
  const world = new Map([
    ["a", { x: 0, y: 0, z: 0 }],
    ["b", { x: 40, y: 20, z: 160 }],
    ["c", { x: 300, y: 10, z: -120 }],
    ["d", { x: 330, y: 60, z: 240 }],
  ]);
  const nodes = [
    { ID: "a", Location: "Home" },
    { ID: "b", Location: "Home" },
    { ID: "c", Location: "Home", Universe: "U1" },
    { ID: "d", Location: "Home", Universe: "U1" },
  ];
  const screen = new Map([...world].map(([id, w]) => [id, project(w, view, W, H)]));
  const shells = realityShells(nodes, (n) => {
    const p = screen.get(n.ID);
    return { x: p.x, y: p.y, z: p.depth };
  });
  for (const n of nodes) {
    const p = screen.get(n.ID);
    const own = nodeShells(shells, n);
    for (const kind of ["math", "universe", "timeline"]) {
      const s = own[kind];
      assert.ok(s, `node ${n.ID} has a ${kind} shell`);
      const d = Math.hypot(p.x - s.cx, p.y - s.cy);
      assert.ok(d <= s.radius + 1e-9, `node ${n.ID} inside its ${kind} shell on screen`);
    }
  }
});

test("realityShells: each shell is labelled with the reality version it denotes", () => {
  const shells = realityShells([
    { ID: "a", Location: "Home", Universe: "U1", Timeline: "T1", Mathematics: "M1" },
  ]);
  const byKind = Object.fromEntries(shells.map((s) => [s.kind, s.label]));
  assert.equal(byKind.math, "M1");
  assert.equal(byKind.universe, "U1");
  assert.equal(byKind.timeline, "T1");
});

test("realityShells: a base shell's label falls back to the default reality name", () => {
  const shells = realityShells([{ ID: "a", Location: "Home" }]);
  const byKind = Object.fromEntries(shells.map((s) => [s.kind, s.label]));
  assert.equal(byKind.math, DEFAULTS.Mathematics);
  assert.equal(byKind.universe, DEFAULTS.Universe);
  assert.equal(byKind.timeline, DEFAULTS.Timeline);
});

test("shellStyle: known kinds resolve, unknown falls back to math", () => {
  assert.equal(shellStyle("universe"), SHELL_STYLE.universe);
  assert.equal(shellStyle("timeline"), SHELL_STYLE.timeline);
  assert.equal(shellStyle("nope"), SHELL_STYLE.math);
});

test("shell colours match their reality transition's edge colour", () => {
  // A shell, the transition edges that cross it, and the legend all read the same
  // hue: SHELL_STYLE mirrors MODE_STYLE for math, universe and timeline.
  assert.equal(SHELL_STYLE.math.rgb, modeStyle("math").rgb);
  assert.equal(SHELL_STYLE.universe.rgb, modeStyle("universe").rgb);
  assert.equal(SHELL_STYLE.timeline.rgb, modeStyle("timeline").rgb);
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

test("FIT_GROUP_MIN_SPAN keeps a single generated node at a steady, moderate zoom (regression)", () => {
  // Regression for auto-generation framing: a one-node cluster has a zero-size
  // box, so fitView with no floor leaves the scale at its default 1 — snapping the
  // camera far out from wherever the traveller was zoomed. Flooring the box to
  // FIT_GROUP_MIN_SPAN instead zooms in on the node at a bounded, moderate scale
  // (never all the way in to MAX_SCALE), and still centres it.
  const w = 1280, h = 800;
  const group = [{ x: 120, y: 90, z: 0 }];

  const noFloor = fitView(group, IDENTITY_VIEW, w, h, FIT_GROUP_MARGIN);
  assert.equal(noFloor.scale, 1, "an unfloored single-node box collapses to the default scale");

  const floored = fitView(group, IDENTITY_VIEW, w, h, FIT_GROUP_MARGIN, FIT_GROUP_MIN_SPAN);
  assert.ok(floored.scale > noFloor.scale, "the floor zooms in on the node rather than snapping out");
  assert.ok(floored.scale < MAX_SCALE, "but never all the way in — not too close");
  assert.ok(Number.isFinite(floored.scale) && floored.scale > 0);

  const framed = { ...IDENTITY_VIEW, scale: floored.scale, ox: floored.ox, oy: floored.oy };
  const p = project(group[0], framed, w, h);
  assert.ok(Math.abs(p.x - w / 2) < 1e-6, "the single node centres horizontally");
  assert.ok(Math.abs(p.y - h / 2) < 1e-6, "the single node centres vertically");
});

test("FIT_GROUP_MIN_SPAN stops a tight cluster from maxing the zoom, but leaves a wide reveal untouched", () => {
  const w = 1280, h = 800;

  // A tight auto-generated cluster (all within a PHYS_RADIUS of each other) maxes
  // the scale without a floor — zoomed in far too close. The floor pulls it back.
  // Kept a few world units across so the required scale sits well above MAX_SCALE
  // and still clamps to it regardless of how high the ceiling is set.
  const tight = [
    { x: 0, y: 0, z: 0 },
    { x: 4, y: 0, z: 0 },
    { x: 0, y: 4, z: 0 },
  ];
  const tightNoFloor = fitView(tight, IDENTITY_VIEW, w, h, FIT_GROUP_MARGIN);
  assert.equal(tightNoFloor.scale, MAX_SCALE, "an unfloored tight cluster pins the scale to the ceiling");
  const tightFloored = fitView(tight, IDENTITY_VIEW, w, h, FIT_GROUP_MARGIN, FIT_GROUP_MIN_SPAN);
  assert.ok(tightFloored.scale < MAX_SCALE, "the floor keeps a tight cluster from zooming in too close");

  // A genuinely wide reveal exceeds the floor, so it is a no-op — the group still
  // fits exactly as it would without a floor.
  const wide = [
    { x: -4000, y: 0, z: 0 },
    { x: 4000, y: 0, z: 0 },
  ];
  const wideNoFloor = fitView(wide, IDENTITY_VIEW, w, h, FIT_GROUP_MARGIN);
  const wideFloored = fitView(wide, IDENTITY_VIEW, w, h, FIT_GROUP_MARGIN, FIT_GROUP_MIN_SPAN);
  assert.ok(Math.abs(wideFloored.scale - wideNoFloor.scale) < 1e-9, "the floor must not shrink a wide reveal");
});
