// Onto Reality Map — a self-contained canvas front-end for the Onto facade.
// No build step, no external deps: it talks to /api/state and /api/execute
// and renders the universe graph as a force-directed constellation. The pure
// view logic (defaults, mode styling, transition detection, node colouring, and
// the 3D projection maths) lives in logic.js so it can be unit-tested in Node.

import {
  DEFAULTS,
  modeStyle,
  edgeWeight,
  TRANSITION_LEGEND,
  TRANSITIONS,
  detectTransition,
  requiredTransition,
  physicalRoute,
  routeTotals,
  effectSpec,
  soundSpec,
  sessionMoved,
  colorFor,
  layerZ,
  layoutTarget,
  clampScale,
  zoomOffset,
  fitView,
  FIT_GROUP_MARGIN,
  unproject,
  panToScreen,
  ROTATE_SPEED,
  edgeRestLength,
  spawnHalo,
  project,
  depthAlpha,
  abbreviateLabel,
  escapeHtml,
} from "./logic.js";

const canvas = document.getElementById("map");
const ctx = canvas.getContext("2d");
const brandEl = document.getElementById("brand");
const axesEl = document.getElementById("axes");
const costEl = document.getElementById("cost-value");
const budgetEl = document.getElementById("budget");
const budgetValueEl = document.getElementById("budget-value");
const objectiveEl = document.getElementById("objective");
const logEl = document.getElementById("log");
const cmdInput = document.getElementById("cmd");
const confirmEl = document.getElementById("confirm");
const inspectorEl = document.getElementById("inspector");
const saveBtn = document.getElementById("save-btn");
const questBtn = document.getElementById("quest-btn");

let state = null;
let edges = [];
const nodes = new Map(); // id -> {id, name, x, y, z, vx, vy, vz}

// Hovered route preview: the physical path (EdgeSnapshot list) a click would
// travel from the current location to the hovered reachable node, plus its
// summed totals. Both are null when nothing routable is hovered. draw() traces
// the path and labels each hop's cost; renderInspector() shows the totals.
let previewPath = null;
let previewTotals = null;

// themeColors caches the theme-dependent canvas colours (the CSS handles the
// DOM). They are read from the CSS custom properties so the map follows the
// light/dark toggle without duplicating the palette here; refreshThemeColors()
// re-reads them whenever the theme changes so we never call getComputedStyle per
// frame. The reachable/quantum/transition hues (logic.js) read the same on both
// themes, so only the label/current-node/depth tints need swapping.
const themeColors = { ink: "#d7e0ff", dim: "#7a86b6", good: "#57e2a5", depth: "#5a6699" };
function refreshThemeColors() {
  const cs = getComputedStyle(document.body);
  const read = (name, fallback) => cs.getPropertyValue(name).trim() || fallback;
  themeColors.ink = read("--ink", themeColors.ink);
  themeColors.dim = read("--dim", themeColors.dim);
  themeColors.good = read("--good", themeColors.good);
  themeColors.depth = read("--depth", themeColors.depth);
}
// view holds the camera. It's snapped to the vertical orientation at startup
// (see setVerticalView and the boot call at the end of the file) so the
// depth-layer stack reads as a ladder from the first frame; dragging, the wheel,
// and the view buttons mutate it from there.
const view = { scale: 1, ox: 0, oy: 0, rotX: 0, rotY: 0 };
// viewAnim tweens the camera (scale + pan) toward a target over a short time. It
// backs the auto-framing that runs when a move reveals new nodes, so the camera
// glides to the new framing instead of snapping. Any manual camera change (drag,
// wheel, or a view button) clears it so the user always wins.
let viewAnim = null;

// setVerticalView / setDefaultView back the view buttons and the initial framing.
// Vertical is a near-top-down angle with no yaw, so z (nesting depth) drives
// screen-y: base reality sits near the top and the deepest layer at the bottom,
// with an upward pan nudging the base layer to the top of the canvas rather than
// its centre. Default is the free three-quarter view.
function setVerticalView() {
  viewAnim = null;
  view.rotX = -1.42; view.rotY = 0;
  view.ox = 0; view.oy = -canvas.clientHeight * 0.32;
  view.scale = 1;
}
function setDefaultView() {
  viewAnim = null;
  view.rotX = 0.5; view.rotY = 0.35;
  view.ox = 0; view.oy = 0; view.scale = 1;
}
// setFitView keeps the current rotation but re-frames the map so every node fits
// on screen at once (fitView computes the scale + pan). Use it to recover from a
// map that has sprawled or zoomed off-frame without losing the current angle.
function setFitView() {
  viewAnim = null;
  const fit = fitView(nodes.values(), view, canvas.clientWidth, canvas.clientHeight);
  view.scale = fit.scale; view.ox = fit.ox; view.oy = fit.oy;
}
// frameToFit smoothly re-frames the map to the same "everything on screen" target
// as setFitView, but animates the camera there (via viewAnim in tick) instead of
// snapping. Called when a move reveals new locations so the freshly-added nodes
// are always brought on screen alongside as much of the existing map as fits.
function frameToFit() {
  const fit = fitView(nodes.values(), view, canvas.clientWidth, canvas.clientHeight);
  viewAnim = {
    t0: performance.now(),
    dur: 600,
    from: { scale: view.scale, ox: view.ox, oy: view.oy },
    to: fit,
  };
}
// frameToGroup makes a freshly-revealed group of nodes the hero: it smoothly
// centres and zooms the camera on just those nodes (FIT_GROUP_MARGIN leaves them
// in the middle of the canvas with the surrounding old realities still visible
// around the edges), so a travelled-to location is focused and never off-screen.
// Falls back to frameToFit if the group is empty or somehow un-framable.
function frameToGroup(ids) {
  const group = ids.map((id) => nodes.get(id)).filter(Boolean);
  if (!group.length) { frameToFit(); return; }
  const fit = fitView(group, view, canvas.clientWidth, canvas.clientHeight, FIT_GROUP_MARGIN);
  viewAnim = {
    t0: performance.now(),
    dur: 600,
    from: { scale: view.scale, ox: view.ox, oy: view.oy },
    to: fit,
  };
}
// frameToNode smoothly pans the camera so a single node sits at the canvas
// centre, keeping the current zoom and rotation. Used when a move changes the
// current location without revealing new nodes — stepping back, or travelling to
// an already-seen node — so the view always follows where you are, however you
// got there, without the jarring rezoom a full re-fit would cause.
function frameToNode(id) {
  const n = nodes.get(id);
  if (!n) return;
  const w = canvas.clientWidth, h = canvas.clientHeight;
  const pan = panToScreen(n, view, w, h, w / 2, h / 2);
  viewAnim = {
    t0: performance.now(),
    dur: 600,
    from: { scale: view.scale, ox: view.ox, oy: view.oy },
    to: { scale: view.scale, ox: pan.ox, oy: pan.oy },
  };
}
let logCount = 0;
// Mirrors stateDTO.Dirty: true when the session has unsaved mutations. Drives
// the save-button indicator and the beforeunload guard.
let dirty = false;

// API_BASE lets the SPA and API live on different origins. It's empty by default
// (config.js), so calls stay relative for same-origin dev (make web); when the
// SPA is served from S3 and the API from ECS/ALB, provisioning sets
// window.ONTO_API_BASE to the API's absolute URL (e.g. http://api.onto.world).
const API_BASE = (typeof window !== "undefined" && window.ONTO_API_BASE) || "";

async function api(path, body) {
  const opts = body
    ? { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }
    : {};
  const res = await fetch(API_BASE + path, opts);
  if (!res.ok) throw new Error(`${path} responded ${res.status}`);
  return res.json();
}

// refresh/run funnel every request through apply(). A failed request keeps the
// last-good state on screen and logs it, rather than surfacing an unhandled
// promise rejection or blanking the UI.
async function refresh() {
  try { apply(await api("/api/state")); }
  catch (err) { console.error("failed to load state", err); }
}
async function run(command) {
  try { apply(await api("/api/execute", { command })); }
  catch (err) { console.error(`command failed: ${command}`, err); }
}

// runMove runs a world-changing command (a transition, travel, observe or time
// branch) and, if it turns out to have been refused — the session neither moved
// nor spent (sessionMoved) — gives blocked feedback: a short error cue and, when
// the press came from a button, a flash on that button. Used for every control
// that should advance the map, so a dead-end press is never silent.
async function runMove(command, btn) {
  const before = state && state.session;
  await run(command);
  if (!sessionMoved(before, state && state.session)) {
    playSound("blocked");
    if (btn) flashBlocked(btn);
  }
}

// flashBlocked briefly pulses a button to show its action was refused. Removing
// then re-adding the class (with a forced reflow between) restarts the CSS
// animation even on rapid repeat presses; the class is cleared when it ends.
function flashBlocked(btn) {
  btn.classList.remove("blocked-flash");
  void btn.offsetWidth;
  btn.classList.add("blocked-flash");
  btn.addEventListener("animationend", () => btn.classList.remove("blocked-flash"), { once: true });
}

function apply(s) {
  const prev = state && state.session;
  state = s;
  const hadNodes = nodes.size > 0;
  const added = syncNodes(s.graph);
  renderHUD(s.session);
  renderLog(s);
  // The location and edges may have changed, so recompute any hovered preview
  // against the new state before the inspector reads it.
  updatePreview();
  renderInspector();
  renderDirty(s);
  renderConfirm(s);
  const mode = detectTransition(prev, s.session);
  if (mode) {
    triggerEffect(mode);
    playSound(mode);
    // Halo whatever locations this transition just revealed, in the transition's
    // own colour, so the eye is drawn to what changed. Physical moves can add
    // nodes too, but only a reality transition arms the halo.
    const rgb = modeStyle(mode).rgb;
    const now = performance.now();
    for (const id of added) {
      const n = nodes.get(id);
      if (n) { n.spawn = now; n.spawnRgb = rgb; }
    }
  } else if (prev && s.session && prev.Location !== s.session.Location) {
    // A plain physical move within the same reality (the travel command, or a
    // click on a reachable node): detectTransition finds no reality change, so
    // play the soft travel cue here. Ordinary travel is then as audible as a
    // transition, just quieter and shorter. A refused travel never reaches this
    // branch — the location is unchanged, so runMove sounds the blocked cue.
    playSound("travel");
  }
  // Auto-frame the camera so it always follows where you are, however you got
  // there. A move that reveals new locations focuses on the freshly-revealed
  // group as the hero (frameToGroup), centred and never off-screen with the older
  // realities still visible around it. A move that changes location without
  // revealing anything — stepping back, or travelling to an already-seen node —
  // pans to re-centre the current location (frameToNode), keeping the current
  // zoom. Both are skipped on the first population and after a reset-map (both
  // start from an empty cache, hadNodes === false) so the deliberate boot framing
  // stands rather than being overridden.
  const locChanged = !prev || prev.Location !== s.session.Location;
  if (hadNodes) {
    if (added.length) frameToGroup(added);
    else if (locChanged) frameToNode(s.session.Location);
  }
}

// renderConfirm shows the Confirm/Cancel bar and locks the free-text input
// while the server is awaiting a 'home' confirmation, so the two-step flow is
// obvious and can't be sidestepped by typing an unrelated command.
function renderConfirm(s) {
  if (s.awaitingHomeConfirm) {
    confirmEl.classList.remove("hidden");
    cmdInput.disabled = true;
  } else {
    confirmEl.classList.add("hidden");
    cmdInput.disabled = false;
  }
}

// syncNodes adds any newly-seen locations near the current node and keeps the
// live edge list. Positions persist across refreshes so the map stays stable.
function syncNodes(graph) {
  edges = graph.Edges || [];
  const added = [];
  for (const n of graph.Nodes || []) {
    // Every node has a deterministic x/y home (layoutTarget): its reality's
    // centre plus a fixed physical offset, so the map is ordered and repeatable.
    // The tick() target-spring holds nodes there while repulsion nudges them
    // apart just enough to avoid overlap. z is owned by the depth layer.
    const target = layoutTarget(n);
    if (!nodes.has(n.ID)) {
      // Seed at the target (with a tiny jitter so overlapping seeds separate)
      // instead of a random ring, so a node animates in near its final home
      // rather than flying across the map.
      nodes.set(n.ID, {
        id: n.ID,
        name: n.Name || n.ID,
        quantum: n.Quantum,
        depth: n.Depth || 0,
        reachable: n.Reachable,
        info: nodeInfo(n),
        tx: target.x,
        ty: target.y,
        x: target.x + (Math.random() - 0.5) * 8,
        y: target.y + (Math.random() - 0.5) * 8,
        z: layerZ(n.Depth) + (Math.random() - 0.5) * 20,
        vx: 0, vy: 0, vz: 0,
      });
      added.push(n.ID);
    } else {
      const node = nodes.get(n.ID);
      node.name = n.Name || n.ID;
      node.quantum = n.Quantum;
      node.depth = n.Depth || 0;
      node.reachable = n.Reachable;
      node.info = nodeInfo(n);
      node.tx = target.x;
      node.ty = target.y;
    }
  }
  return added;
}

function badge(label, value, active, extraClass = "") {
  const cls = "badge" + (active ? " active" : "") + (extraClass ? " " + extraClass : "");
  return `<span class="${cls}">${escapeHtml(label)} <b>${escapeHtml(value)}</b></span>`;
}

function renderHUD(sess) {
  if (!sess) return;
  // The brand label carries the proper onto:// address of the current location
  // (URL form): "onto" and "://" keep their styling, the rest is the address.
  const rest = (sess.ShortOntoAddress || "").replace(/^onto:\/\//, "");
  brandEl.innerHTML = `onto<span>://</span><b class="brand-addr">${escapeHtml(rest)}</b>`;
  brandEl.title = sess.OntoAddress || sess.ShortOntoAddress || "";
  const parts = [
    badge("math", sess.Mathematics, sess.Mathematics !== DEFAULTS.Mathematics),
    badge("universe", sess.Universe, sess.Universe !== DEFAULTS.Universe),
    badge("timeline", sess.Timeline, sess.Timeline !== DEFAULTS.Timeline),
    badge("quantum", sess.Quantum, sess.Quantum !== DEFAULTS.Quantum),
    badge("sim", sess.Simulation, sess.Simulation > 0),
    badge("cons", sess.Consensus, sess.Consensus > 0),
    badge("observer", sess.Observer, sess.Observer !== DEFAULTS.Observer),
  ];
  axesEl.innerHTML = parts.join("");
  costEl.textContent = Math.round(sess.CumulativeCost || 0);
  renderGame(sess);
}

// renderGame updates the budget chip and objective badge from the session's
// game state. Both are hidden entirely when no budget/target is in force, so a
// non-game session shows only the cost chip.
function renderGame(sess) {
  // The 'new quest' button only re-rolls a quest in game mode; outside game mode
  // there is no budget or objective pool to draw from, so disable it. HasBudget
  // is the game-mode signal — game mode always installs a budget, non-game
  // sessions never do.
  if (questBtn) questBtn.disabled = !sess.HasBudget;
  if (budgetEl) {
    // A budget of zero is not shown at all: no limit is in force (unlimited
    // spending). A finite budget that has been spent down to nothing is shown as
    // "exhausted" — a limit is in force but no money is left — so the two are
    // never confused.
    if (sess.HasBudget) {
      budgetEl.style.display = "";
      const exhausted = (sess.RemainingBudget || 0) <= 0;
      budgetValueEl.textContent = exhausted ? "exhausted" : Math.round(sess.RemainingBudget || 0);
      budgetEl.classList.toggle("low", exhausted);
      budgetEl.title = exhausted ? "Budget exhausted — no spending left" : "Budget remaining";
    } else {
      budgetEl.style.display = "none";
    }
  }
  if (!objectiveEl) return;
  if (!sess.HasTarget) { objectiveEl.style.display = "none"; return; }
  objectiveEl.style.display = "";
  const target = (sess.TargetShortAddress || "").replace(/^onto:\/\//, "");
  const par = Math.round(sess.Par || 0);
  const count = sess.ObjectiveCount || 0;
  const done = sess.ObjectivesDone || 0;
  if (sess.Won) {
    objectiveEl.className = "objective won";
    objectiveEl.textContent = `✓ you win — ${stars(sess.Stars)} (${Math.round(sess.CumulativeCost || 0)} / par ${par})`;
  } else if (sess.ReachedTarget) {
    objectiveEl.className = "objective reached";
    objectiveEl.textContent = `objective ${done + 1}/${count} reached — return home to complete it (par ${par})`;
  } else {
    objectiveEl.className = "objective";
    objectiveEl.textContent = `objective ${done + 1}/${count}: reach ${target} & return home (par ${par})`;
  }
}

// stars renders a 0..3 efficiency rating as filled and empty stars, matching
// the CLI win banner.
function stars(n) {
  n = Math.max(0, Math.min(3, n | 0));
  return "★".repeat(n) + "☆".repeat(3 - n);
}

function renderLog(s) {
  const hist = (s.session && s.session.History) || [];
  // Only re-render when something changed to avoid flicker/scroll resets.
  const signature = (s.response || "") + "|" + hist.length;
  if (signature === logEl.dataset.sig) return;
  logEl.dataset.sig = signature;
  if (s.response) {
    const div = document.createElement("div");
    div.className = "entry";
    div.textContent = s.response;
    logEl.appendChild(div);
    logEl.scrollTop = logEl.scrollHeight;
    if (++logCount > 40) logEl.removeChild(logEl.firstChild);
  }
}

// nodeInfo distils a NodeSnapshot into the fields the inspector reads: display
// name, description, canonical onto:// address, reachability, and the reality
// axes requiredTransition() diffs against the session. Stored on each node in
// syncNodes so the inspector needs no second graph lookup.
function nodeInfo(n) {
  return {
    id: n.ID,
    name: n.Name || n.ID,
    description: n.Description || "",
    address: n.OntoAddress || "",
    reachable: !!n.Reachable,
    Mathematics: n.Mathematics,
    Universe: n.Universe,
    Timeline: n.Timeline,
    Quantum: n.Quantum,
    Simulation: n.Simulation,
    Consensus: n.Consensus,
    Observer: n.Observer,
  };
}

// updatePreview recomputes the hovered route preview: the physical path from the
// current location to the hovered node, but only when that node is reachable by
// ordinary travel. It is cleared for the current node, unreachable nodes, or no
// hover, so the map only ever previews a journey a click would actually make.
// Called whenever the hover or the underlying state changes.
function updatePreview() {
  const sess = state && state.session;
  const cur = sess && sess.Location;
  const node = hoveredId ? nodes.get(hoveredId) : null;
  if (cur && node && node.reachable && hoveredId !== cur) {
    previewPath = physicalRoute(cur, hoveredId, edges);
    previewTotals = previewPath ? routeTotals(previewPath) : null;
  } else {
    previewPath = null;
    previewTotals = null;
  }
}

// renderInspector fills the floating top-left inspector. It describes where you
// are — the current node — by default, and switches to the hovered node's
// details while the pointer is over one. The footer answers "how do I get
// there?": you-are-here, click-to-travel (with the previewed route's hops, cost
// and distance) for a reachable same-reality node, a command chip when a reality
// transition is required (requiredTransition), or no-route when it's otherwise
// unreachable.
function renderInspector() {
  if (!inspectorEl) return;
  const sess = state && state.session;
  const curId = sess && sess.Location;
  const targetId = hoveredId || curId;
  const node = targetId ? nodes.get(targetId) : null;
  const info = node && node.info;
  if (!info) { inspectorEl.innerHTML = ""; return; }
  let status;
  if (info.id === curId) {
    status = '<span class="insp-status here">you are here</span>';
  } else if (info.reachable) {
    status = '<span class="insp-status go">click to travel</span>';
    // Surface the previewed route's totals: hop count, summed travel cost, and
    // distance in km — the same figures the CLI `route` command reports.
    if (previewTotals) {
      const hops = `${previewTotals.steps} ${previewTotals.steps === 1 ? "hop" : "hops"}`;
      status +=
        `<span class="insp-status dim">${hops} · cost ${Math.round(previewTotals.cost)} · ${previewTotals.distance.toFixed(1)} km</span>`;
    }
  } else {
    const t = requiredTransition(sess, info);
    if (t) {
      const st = modeStyle(t.mode);
      status =
        '<span class="insp-status">needs</span>' +
        `<span class="cmd-chip" style="border-color:rgba(${st.rgb},0.5);color:rgb(${st.rgb})">${escapeHtml(t.command)}</span>` +
        `<span class="insp-status dim">to reach this ${escapeHtml(t.label)}</span>`;
    } else {
      status = '<span class="insp-status blocked">no route from here</span>';
    }
  }
  const desc = info.description ? `<div class="insp-desc">${escapeHtml(info.description)}</div>` : "";
  const addr = info.address ? `<div class="insp-addr" title="onto address">${escapeHtml(info.address)}</div>` : "";
  inspectorEl.innerHTML =
    `<div class="insp-title">${escapeHtml(info.name)}</div>` + desc + addr +
    `<div class="insp-foot">${status}</div>`;
}

// renderDirty mirrors stateDTO.Dirty onto the save button (a dot appears when
// there are unsaved mutations) and keeps the module `dirty` flag the
// beforeunload guard reads in sync.
function renderDirty(s) {
  dirty = !!(s && s.dirty);
  if (saveBtn) saveBtn.classList.toggle("dirty", dirty);
}

// ── Force-directed layout ──────────────────────────────────────────────────

function tick() {
  const arr = [...nodes.values()];
  // Repulsion between every pair of nodes, now in three dimensions. It is only
  // there to keep nodes that share a layout home from overlapping, so it is
  // gentle — the deterministic target spring below owns the overall shape.
  for (let i = 0; i < arr.length; i++) {
    for (let j = i + 1; j < arr.length; j++) {
      const a = arr[i], b = arr[j];
      let dx = a.x - b.x, dy = a.y - b.y, dz = a.z - b.z;
      let d2 = dx * dx + dy * dy + dz * dz || 0.01;
      const f = 1400 / d2;
      const d = Math.sqrt(d2);
      const ux = dx / d, uy = dy / d, uz = dz / d;
      a.vx += ux * f; a.vy += uy * f; a.vz += uz * f;
      b.vx -= ux * f; b.vy -= uy * f; b.vz -= uz * f;
    }
  }
  // Springs along edges pull connected nodes toward a rest length. They act in
  // the x/y plane only: z is owned by the depth-layer spring below, so an edge
  // never drags a node off its own nesting layer. The rest length varies by mode
  // (edgeRestLength): physical edges stay short so a reality's locations cluster,
  // while reality-transition edges rest much longer, pushing each child
  // sub-graph away from its parent. This is now a soft trim on top of the
  // deterministic target spring, so it eases connected nodes together without
  // overriding the imposed reality layout.
  for (const e of edges) {
    const a = nodes.get(e.From), b = nodes.get(e.To);
    if (!a || !b) continue;
    const dx = b.x - a.x, dy = b.y - a.y;
    const d = Math.sqrt(dx * dx + dy * dy) || 0.01;
    const f = (d - edgeRestLength(e.Mode)) * 0.008;
    const ux = dx / d, uy = dy / d;
    a.vx += ux * f; a.vy += uy * f;
    b.vx -= ux * f; b.vy -= uy * f;
  }
  // x/y: a firm spring toward the node's deterministic layout home (tx/ty), so
  // realities settle into ordered, predictable clusters instead of a random
  // organic scatter. z: a stiffer spring toward the node's depth layer, so
  // nested realities stack into shells. Then damping and integration.
  for (const n of arr) {
    n.vx += (n.tx - n.x) * 0.03; n.vy += (n.ty - n.y) * 0.03;
    n.vz += (layerZ(n.depth) - n.z) * 0.05;
    n.vx *= 0.86; n.vy *= 0.86; n.vz *= 0.86;
    n.x += n.vx; n.y += n.vy; n.z += n.vz;
  }
  // Advance the camera auto-framing tween (set by frameToFit), easing scale and
  // pan toward the fit target with an ease-in-out; cleared once it completes.
  if (viewAnim) {
    const p = Math.min(1, (performance.now() - viewAnim.t0) / viewAnim.dur);
    const e = p < 0.5 ? 2 * p * p : 1 - Math.pow(-2 * p + 2, 2) / 2;
    view.scale = viewAnim.from.scale + (viewAnim.to.scale - viewAnim.from.scale) * e;
    view.ox = viewAnim.from.ox + (viewAnim.to.ox - viewAnim.from.ox) * e;
    view.oy = viewAnim.from.oy + (viewAnim.to.oy - viewAnim.from.oy) * e;
    if (p >= 1) viewAnim = null;
  }
  draw();
  requestAnimationFrame(tick);
}

// toScreen is a thin wrapper over the pure project() maths in logic.js, binding
// it to the live view and the current canvas size. It passes the CSS-pixel size
// (clientWidth/clientHeight), not the device-pixel backing store, because draw()
// applies a devicePixelRatio transform to the context — so centring on
// clientWidth/2 lands the origin in the middle of the canvas on HiDPI screens
// too. Labels are drawn at the projected point, so they stay upright.
function toScreen(n) { return project(n, view, canvas.clientWidth, canvas.clientHeight); }

// ── Transition animation ─────────────────────────────────────────────────────
// detectTransition (logic.js) diffs a reality axis between snapshots; when it
// reports a change we queue an effect. Each transition plays its own
// character-matched animation (effectSpec picks the kind + duration): a universe
// shift fades reality to black and back, a quantum shift flickers through
// superposed ghosts, an observer shift blinks new eyes open, and so on. The
// colour always comes from that transition's modeStyle, so the effect matches
// the edges and the legend.
const effects = [];

function triggerEffect(mode) {
  const { kind, duration } = effectSpec(mode);
  effects.push({ mode, kind, duration, start: performance.now() });
}

// ── Transition sound ─────────────────────────────────────────────────────────
// Each reality transition plays its own layered, cinematic cue (soundSpec picks
// the voices in logic.js). The Web Audio context is created lazily on the first
// transition — always inside the user gesture that triggered the move — so the
// browser's autoplay policy is satisfied without a separate "click to enable"
// step. A muted preference persists in localStorage and gates playback entirely.
//
// Each cue is sound-designed in a Star Wars / Alien spirit — organic and grungy
// rather than clean, so voices carry per-voice distortion (drive), ring
// modulation, FM at inharmonic ratios, and random pitch jitter (see logic.js).
// The graph gives every cue a shared, scored space. Each voice fans into a dry
// bus and a reverb send; the reverb is a ConvolverNode fed a synthesised
// decaying-noise impulse (no asset) after a short pre-delay, so tails bloom like
// a film score. Both buses pass through a gentle WaveShaper (analog-style warmth)
// and a DynamicsCompressor (glue + peak taming) before the destination.
// MASTER_GAIN keeps the mix gentle so a move never startles.
const MASTER_GAIN = 0.12;
const REVERB_SECONDS = 3.4;   // impulse length — how long the tail rings
const REVERB_DECAY = 2.2;     // higher = faster decay (steeper tail)
const REVERB_WET = 0.38;      // reverb send level relative to the dry bus
const REVERB_PREDELAY = 0.03; // gap before the tail — pushes the space back
let audioCtx = null;
let dryBus = null;    // voices → here (dry) → saturation
let wetSend = null;   // voices → here → pre-delay → convolver → saturation
let noiseBuffer = null; // shared white-noise source buffer for noise voices
let muted = localStorage.getItem("onto.muted") === "1";

// makeImpulse builds a stereo impulse response as exponentially-decaying white
// noise — a cheap, dependency-free convolution reverb that sounds like a large,
// soft hall rather than a specific room.
function makeImpulse(ctx) {
  const rate = ctx.sampleRate;
  const len = Math.floor(rate * REVERB_SECONDS);
  const buf = ctx.createBuffer(2, len, rate);
  for (let ch = 0; ch < 2; ch++) {
    const data = buf.getChannelData(ch);
    for (let i = 0; i < len; i++) {
      data[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, REVERB_DECAY);
    }
  }
  return buf;
}

// makeNoise builds a couple of seconds of stereo white noise, looped by every
// "noise" voice to synthesise whooshes, risers, and air.
function makeNoise(ctx) {
  const len = Math.floor(ctx.sampleRate * 2);
  const buf = ctx.createBuffer(2, len, ctx.sampleRate);
  for (let ch = 0; ch < 2; ch++) {
    const data = buf.getChannelData(ch);
    for (let i = 0; i < len; i++) data[i] = Math.random() * 2 - 1;
  }
  return buf;
}

// saturationCurve is a soft tanh-style shaping curve: near-linear at low level,
// gently rounding peaks, so the mix reads as warm rather than clipped.
function saturationCurve() {
  const n = 1024;
  const curve = new Float32Array(n);
  const k = 1.6;
  for (let i = 0; i < n; i++) {
    const x = (i / (n - 1)) * 2 - 1;
    curve[i] = Math.tanh(k * x);
  }
  return curve;
}

// makeDriveCurve builds a per-voice distortion curve — the "grunge". amount 0 is
// clean; higher values fold peaks over into gritty, saturated harmonics so a
// voice reads as an abused analog source (metal, tape, blown speaker) rather than
// a clean oscillator. Uses the classic k-parameter soft-clip shape.
function makeDriveCurve(amount) {
  const n = 1024;
  const curve = new Float32Array(n);
  const k = amount * 60;
  for (let i = 0; i < n; i++) {
    const x = (i / (n - 1)) * 2 - 1;
    curve[i] = ((1 + k) * x) / (1 + k * Math.abs(x));
  }
  return curve;
}

// ensureAudio lazily builds the context and the shared master chain on the first
// transition. Returns false if the browser has no Web Audio support.
function ensureAudio() {
  if (audioCtx) return true;
  const Ctor = window.AudioContext || window.webkitAudioContext;
  if (!Ctor) return false;
  audioCtx = new Ctor();
  const master = audioCtx.createDynamicsCompressor();
  const shaper = audioCtx.createWaveShaper();
  shaper.curve = saturationCurve();
  shaper.oversample = "2x";
  shaper.connect(master);
  master.connect(audioCtx.destination);
  dryBus = audioCtx.createGain();
  dryBus.gain.value = 1;
  dryBus.connect(shaper);
  const reverb = audioCtx.createConvolver();
  reverb.buffer = makeImpulse(audioCtx);
  const preDelay = audioCtx.createDelay(1);
  preDelay.delayTime.value = REVERB_PREDELAY;
  wetSend = audioCtx.createGain();
  wetSend.gain.value = REVERB_WET;
  wetSend.connect(preDelay).connect(reverb).connect(shaper);
  noiseBuffer = makeNoise(audioCtx);
  return true;
}

// applyRing ring-modulates a voice: the signal passes through a gain node whose
// gain is swung by a sine at ring.freq. At depth 1 the gain crosses zero — true
// ring modulation, the clangorous metallic sidebands of classic sci-fi robots
// and alien voices; below 1 it stays partly open, blending toward tremolo.
// Returns the ring node so the caller keeps building the chain.
function applyRing(v, inNode, t0, t2) {
  const depth = v.ring.depth === undefined ? 1 : v.ring.depth;
  const ring = audioCtx.createGain();
  ring.gain.value = 1 - depth; // base offset; the modulator swings ±depth around it
  const mod = audioCtx.createOscillator();
  mod.frequency.value = v.ring.freq;
  const md = audioCtx.createGain();
  md.gain.value = depth;
  mod.connect(md).connect(ring.gain);
  mod.start(t0);
  mod.stop(t2 + 0.05);
  inNode.connect(ring);
  return ring;
}

// applyFM frequency-modulates an oscillator carrier: a modulator at freq×ratio
// bends the carrier's frequency by ±depth Hz. Non-integer ratios give inharmonic
// partials — struck metal, groaning hulls, organic "clang" — instead of a clean
// pitch, which is what pulls the timbre away from a synthesiser.
function applyFM(v, carrier, t0, t2) {
  const mod = audioCtx.createOscillator();
  mod.frequency.value = v.freq * v.fm.ratio;
  const md = audioCtx.createGain();
  md.gain.value = v.fm.depth;
  mod.connect(md).connect(carrier.frequency);
  mod.start(t0);
  mod.stop(t2 + 0.05);
}

// applyLFO wires a modulation oscillator onto a voice: a gain target multiplies
// the envelope (tremolo/flicker) via an inserted node; a pitch target sways the
// source's detune (vibrato/wobble); a filter target sweeps the filter cutoff
// (organic movement). Returns the (possibly new) output node so the caller can
// keep building the chain. The modulator runs for the voice's whole life.
function applyLFO(v, source, filter, envOut, t0, t2) {
  const mod = audioCtx.createOscillator();
  mod.frequency.value = v.lfo.freq;
  const depth = audioCtx.createGain();
  depth.gain.value = v.lfo.depth;
  mod.connect(depth);
  mod.start(t0);
  mod.stop(t2 + 0.05);
  if (v.lfo.target === "pitch" && source.detune) {
    depth.connect(source.detune); // cents
    return envOut;
  }
  if (v.lfo.target === "filter" && filter) {
    depth.connect(filter.frequency); // Hz
    return envOut;
  }
  // gain flicker: a node whose gain oscillates about 1 by ±depth.
  const trem = audioCtx.createGain();
  trem.gain.value = 1;
  depth.connect(trem.gain);
  envOut.connect(trem);
  return trem;
}

function playSound(mode) {
  if (muted) return;
  try {
    if (!ensureAudio()) return;
    if (audioCtx.state === "suspended") audioCtx.resume();
    const now = audioCtx.currentTime;
    for (const v of soundSpec(mode).voices) {
      const t0 = now + (v.delay || 0);
      const t1 = t0 + v.attack;
      const t2 = t1 + v.release;

      // Source: filtered white noise for "noise" voices, otherwise an oscillator
      // (optionally FM'd for inharmonic, organic timbres).
      let source;
      if (v.type === "noise") {
        source = audioCtx.createBufferSource();
        source.buffer = noiseBuffer;
        source.loop = true;
      } else {
        source = audioCtx.createOscillator();
        source.type = v.type;
        source.frequency.setValueAtTime(v.freq, t0);
        if (v.freqEnd) source.frequency.exponentialRampToValueAtTime(v.freqEnd, t2);
        // detune + a random per-play jitter so repeats never sound identical.
        const jitter = v.jitter ? (Math.random() * 2 - 1) * v.jitter : 0;
        if (v.detune || jitter) source.detune.setValueAtTime((v.detune || 0) + jitter, t0);
        if (v.fm) applyFM(v, source, t0, t2);
      }

      // Per-voice chain:
      //   source → [ring mod] → [drive] → [filter sweep] → gain(env) → [LFO] → [pan]
      let node = source;
      if (v.ring) node = applyRing(v, node, t0, t2);
      if (v.drive) {
        const shaper = audioCtx.createWaveShaper();
        shaper.curve = makeDriveCurve(v.drive);
        shaper.oversample = "4x";
        node.connect(shaper);
        node = shaper;
      }
      let filter = null;
      if (v.filter) {
        filter = audioCtx.createBiquadFilter();
        filter.type = v.filter.type || "lowpass";
        filter.Q.value = v.filter.q || 1;
        filter.frequency.setValueAtTime(v.filter.freq, t0);
        if (v.filter.freqEnd) filter.frequency.exponentialRampToValueAtTime(v.filter.freqEnd, t2);
        node.connect(filter);
        node = filter;
      }
      const gain = audioCtx.createGain();
      // Linear attack up to the voice's peak, then an exponential decay to near
      // silence (exponential ramps can't reach exactly 0).
      gain.gain.setValueAtTime(0.0001, t0);
      gain.gain.linearRampToValueAtTime(v.gain * MASTER_GAIN, t1);
      gain.gain.exponentialRampToValueAtTime(0.0001, t2);
      node.connect(gain);
      let out = gain;
      if (v.lfo) out = applyLFO(v, source, filter, out, t0, t2);
      if (v.pan !== undefined && audioCtx.createStereoPanner) {
        const panner = audioCtx.createStereoPanner();
        panner.pan.value = v.pan;
        out.connect(panner);
        out = panner;
      }
      // Send each voice to both the dry bus and the reverb, so every layer sits
      // in the same scored space.
      out.connect(dryBus);
      out.connect(wetSend);
      source.start(t0);
      source.stop(t2 + 0.05);
    }
  } catch (err) {
    console.error("sound failed", err);
  }
}

// Mute toggle: flips the preference, persists it, and reflects state in the
// button glyph/title. Muting never touches the audio graph — playSound simply
// short-circuits — so an in-flight sound is left to finish naturally.
const muteBtn = document.getElementById("mute");
function renderMute() {
  if (!muteBtn) return;
  muteBtn.textContent = muted ? "🔕" : "🔔";
  muteBtn.title = muted ? "Transition sounds off — click to unmute" : "Transition sounds on — click to mute";
  muteBtn.classList.toggle("muted", muted);
}
if (muteBtn) {
  muteBtn.addEventListener("click", () => {
    muted = !muted;
    localStorage.setItem("onto.muted", muted ? "1" : "0");
    renderMute();
  });
  renderMute();
}

// Theme toggle: flips the colour scheme, persists the choice, and re-reads the
// canvas colours so the map follows the DOM. Dark is the default; the choice is
// remembered across reloads. renderTheme() applies the current state, so calling
// it on load restores a saved light preference before the first paint.
const themeBtn = document.getElementById("theme");
let light = localStorage.getItem("onto.theme") === "light";
function renderTheme() {
  document.body.classList.toggle("light", light);
  refreshThemeColors();
  if (themeBtn) {
    themeBtn.textContent = light ? "☀️" : "🌙";
    themeBtn.title = light ? "Light scheme — click for dark" : "Dark scheme — click for light";
  }
}
if (themeBtn) {
  themeBtn.addEventListener("click", () => {
    light = !light;
    localStorage.setItem("onto.theme", light ? "light" : "dark");
    renderTheme();
  });
}
renderTheme();

// Label-mode toggle: cycles the control buttons between text + icons, text only,
// and icons only, persisting the choice. A mode class on the controls container
// drives the CSS that hides the label or the icon span; the toggle's own glyph
// reflects the current mode.
const LABEL_MODES = ["both", "text", "icons"];
const controlsEl = document.querySelector(".controls");
const labelsBtn = document.getElementById("labels");
let labelMode = localStorage.getItem("onto.labels") || "both";
if (!LABEL_MODES.includes(labelMode)) labelMode = "both";
function renderLabels() {
  if (controlsEl) {
    controlsEl.classList.toggle("labels-text", labelMode === "text");
    controlsEl.classList.toggle("labels-icons", labelMode === "icons");
  }
  if (labelsBtn) {
    labelsBtn.textContent = labelMode === "both" ? "Aa🙂" : labelMode === "text" ? "Aa" : "🙂";
    const desc = labelMode === "both" ? "text + icons" : labelMode === "text" ? "text only" : "icons only";
    labelsBtn.title = `Button labels: ${desc} — click to cycle`;
  }
}
if (labelsBtn) {
  labelsBtn.addEventListener("click", () => {
    labelMode = LABEL_MODES[(LABEL_MODES.indexOf(labelMode) + 1) % LABEL_MODES.length];
    localStorage.setItem("onto.labels", labelMode);
    renderLabels();
  });
}
renderLabels();

// Help modal: an About/Help overlay explaining reality transitions. Its axis
// table is built from the same TRANSITIONS source of truth the legend uses, so
// the colours, commands, and costs never drift from the map.
(function initHelp() {
  const modal = document.getElementById("help-modal");
  const openBtn = document.getElementById("help-btn");
  const closeBtn = document.getElementById("help-close");
  const backdrop = document.getElementById("help-backdrop");
  const axesBox = document.getElementById("help-axes");
  if (!modal || !openBtn) return;
  if (axesBox) {
    axesBox.innerHTML = TRANSITIONS.map(({ mode, label, command, cost, what, refs }) => {
      const st = modeStyle(mode);
      const links = (refs || [])
        .map((r) => `<a class="help-ref" href="${escapeHtml(r.url)}" target="_blank" rel="noopener noreferrer">${escapeHtml(r.label)} ↗</a>`)
        .join("");
      const refsRow = links ? `<span class="help-axis-refs">${links}</span>` : "";
      return (
        '<div class="help-axis">' +
        `<i class="line" style="background:rgb(${st.rgb})"></i>` +
        `<span class="help-axis-name">${escapeHtml(label)}</span>` +
        `<span class="cmd-chip" style="border-color:rgba(${st.rgb},0.5);color:rgb(${st.rgb})">${escapeHtml(command)}</span>` +
        `<span class="help-axis-cost">${escapeHtml(cost || "")}</span>` +
        `<span class="help-axis-what">${escapeHtml(what || "")}</span>` +
        refsRow +
        "</div>"
      );
    }).join("");
  }
  const open = () => modal.classList.remove("hidden");
  const close = () => modal.classList.add("hidden");
  openBtn.addEventListener("click", open);
  if (closeBtn) closeBtn.addEventListener("click", close);
  if (backdrop) backdrop.addEventListener("click", close);
  window.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && !modal.classList.contains("hidden")) {
      e.preventDefault();
      close();
    }
  });
})();

function draw() {
  const dpr = window.devicePixelRatio || 1;
  if (canvas.width !== canvas.clientWidth * dpr) {
    canvas.width = canvas.clientWidth * dpr;
    canvas.height = canvas.clientHeight * dpr;
  }
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  const cur = state && state.session ? state.session.Location : null;

  for (const e of edges) {
    const a = nodes.get(e.From), b = nodes.get(e.To);
    if (!a || !b) continue;
    const pa = toScreen(a), pb = toScreen(b);
    // Each mode has its own hue + dash: solid blue for ordinary travel, a
    // distinct dashed colour for every reality transition. Cost owns the heft:
    // edgeWeight maps EdgeSnapshot.Cost to a width and opacity (log-scaled), so
    // expensive exotic jumps read as thick, solid lines and cheap steps as thin,
    // faint ones — the same cost the hover preview labels, now legible at rest.
    const st = modeStyle(e.Mode);
    const w = edgeWeight(e.Cost);
    ctx.lineWidth = w.width;
    ctx.strokeStyle = `rgba(${st.rgb},${w.alpha})`;
    ctx.setLineDash(st.dash);
    ctx.beginPath();
    ctx.moveTo(pa.x, pa.y);
    ctx.lineTo(pb.x, pb.y);
    ctx.stroke();
  }
  ctx.setLineDash([]);

  // Route preview: while a reachable node is hovered, trace the physical path a
  // click would travel over the base edges — brighter, thicker solid segments in
  // the current-location green — and label each hop with its cost, so the map
  // answers "how far, and at what price?" before you commit. Drawn under the
  // nodes so they stay legible on top; the totals go in the inspector.
  if (previewPath && previewPath.length) {
    ctx.save();
    ctx.setLineDash([]);
    ctx.lineWidth = 2.5;
    // The same green the current-location halo uses, so the route reads as "your"
    // path rather than another mode's edge colour.
    ctx.strokeStyle = "rgba(87,226,165,0.9)";
    for (const e of previewPath) {
      const a = nodes.get(e.From), b = nodes.get(e.To);
      if (!a || !b) continue;
      const pa = toScreen(a), pb = toScreen(b);
      ctx.beginPath();
      ctx.moveTo(pa.x, pa.y);
      ctx.lineTo(pb.x, pb.y);
      ctx.stroke();
    }
    ctx.fillStyle = themeColors.good;
    ctx.font = "11px ui-monospace, monospace";
    ctx.textAlign = "center";
    for (const e of previewPath) {
      const a = nodes.get(e.From), b = nodes.get(e.To);
      if (!a || !b) continue;
      const pa = toScreen(a), pb = toScreen(b);
      ctx.fillText(String(Math.round(e.Cost || 0)), (pa.x + pb.x) / 2, (pa.y + pb.y) / 2 - 4);
    }
    ctx.textAlign = "left";
    ctx.restore();
  }

  // Project every node, then paint far-to-near so nearer nodes overlap farther
  // ones — the depth cue that sells the 3D rotation.
  const drawn = [...nodes.values()].map((n) => ({ n, p: toScreen(n) }));
  drawn.sort((a, b) => b.p.depth - a.p.depth);
  // Depth range of what's on screen drives the relative fade (see depthAlpha).
  let minDepth = Infinity, maxDepth = -Infinity;
  for (const { p } of drawn) {
    if (p.depth < minDepth) minDepth = p.depth;
    if (p.depth > maxDepth) maxDepth = p.depth;
  }
  for (const { n, p } of drawn) {
    const isCur = n.id === cur;
    // Keep a small floor on the radius so nodes stay visible (as dots) even when
    // zoomed right out, instead of shrinking to sub-pixel and vanishing.
    const r = Math.max((isCur ? 9 : 6) * view.scale * p.persp, isCur ? 2.5 : 1.5);
    // Fade nodes by depth so a busy map stays readable; the current location
    // always stays at full opacity so you never lose track of where you are.
    const nodeAlpha = isCur ? 1 : depthAlpha(p.depth, minDepth, maxDepth);
    ctx.globalAlpha = nodeAlpha;
    // Spawn halo: a faint sphere around a node a reality transition just
    // revealed, fading and swelling over a couple of seconds (spawnHalo) before
    // it clears itself. Drawn behind the node, tinted with the transition colour.
    if (n.spawn) {
      const halo = spawnHalo(performance.now() - n.spawn);
      if (halo) {
        ctx.globalAlpha = halo.alpha * nodeAlpha;
        ctx.fillStyle = `rgba(${n.spawnRgb || "255,255,255"},1)`;
        ctx.beginPath();
        ctx.arc(p.x, p.y, r + halo.grow, 0, Math.PI * 2);
        ctx.fill();
        ctx.globalAlpha = nodeAlpha;
      } else {
        n.spawn = 0;
      }
    }
    if (isCur) {
      ctx.fillStyle = "rgba(87,226,165,0.18)";
      ctx.beginPath();
      ctx.arc(p.x, p.y, r + 10, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.fillStyle = isCur ? themeColors.good : colorFor(n);
    ctx.beginPath();
    ctx.arc(p.x, p.y, r, 0, Math.PI * 2);
    ctx.fill();
    ctx.fillStyle = isCur ? themeColors.ink : themeColors.dim;
    ctx.font = `${12 * Math.max(view.scale * p.persp, 0.7)}px ui-monospace, monospace`;
    // Labels grow long fast, so abbreviate them by default; reveal the full name
    // for the current location and whichever node the pointer is hovering.
    const showFull = isCur || n.id === hoveredId;
    ctx.fillText(showFull ? n.name : abbreviateLabel(n.name), p.x + r + 4, p.y + 4);
    // Depth badge: the nesting-depth number to the left of each nested node, so
    // depth is legible without rotating. Base (depth 0) nodes are left unbadged.
    if (n.depth > 0) {
      ctx.fillStyle = isCur ? themeColors.ink : themeColors.depth;
      ctx.textAlign = "right";
      ctx.fillText(String(n.depth), p.x - r - 4, p.y + 4);
      ctx.textAlign = "left";
    }
  }
  ctx.globalAlpha = 1;

  drawEffects(cur);
}

// drawEffects advances each active transition and hands it to the renderer for
// its kind (EFFECT_RENDERERS), passing normalised progress t in [0,1), the
// transition's rgb, the screen-space origin (the current location), and the
// canvas size. Finished effects are dropped. Called every frame from draw().
function drawEffects(curId) {
  if (!effects.length) return;
  const now = performance.now();
  const node = curId ? nodes.get(curId) : null;
  const w = canvas.clientWidth, h = canvas.clientHeight;
  const origin = node ? toScreen(node) : { x: w / 2, y: h / 2 };
  for (let i = effects.length - 1; i >= 0; i--) {
    const e = effects[i];
    const t = (now - e.start) / e.duration;
    if (t >= 1) { effects.splice(i, 1); continue; }
    const render = EFFECT_RENDERERS[e.kind] || EFFECT_RENDERERS.ripple;
    ctx.save();
    render(t, modeStyle(e.mode).rgb, origin, w, h);
    ctx.restore();
  }
}

// Each renderer draws one transition for progress t in [0,1). They run inside a
// ctx.save()/restore() so they can set alpha, line styles, and clips freely.
const EFFECT_RENDERERS = {
  // Plain travel and unknown modes: staggered expanding rings from the origin.
  ripple(t, rgb, o) {
    for (let k = 0; k < 3; k++) {
      const tt = t - k * 0.12;
      if (tt <= 0) continue;
      ctx.beginPath();
      ctx.strokeStyle = `rgba(${rgb},${(1 - tt) * 0.9})`;
      ctx.lineWidth = 2.5 * (1 - tt) + 0.5;
      ctx.arc(o.x, o.y, 12 + tt * 130 * view.scale, 0, Math.PI * 2);
      ctx.stroke();
    }
  },

  // Universe shift: the whole screen dissolves to a dark tinted void at the
  // midpoint, then the new bubble universe fades back in.
  fade(t, rgb, o, w, h) {
    const a = Math.sin(t * Math.PI);            // 0 → 1 → 0
    ctx.fillStyle = `rgba(6,4,14,${Math.min(1, a * 1.6)})`;
    ctx.fillRect(0, 0, w, h);
    ctx.fillStyle = `rgba(${rgb},${a * 0.25})`; // a wash of the universe's hue
    ctx.fillRect(0, 0, w, h);
  },

  // Quantum shift: several jittered, flickering ghost rings — superposed
  // possibilities collapsing to one.
  superposition(t, rgb, o) {
    const fade = 1 - t;
    for (let k = 0; k < 6; k++) {
      const jx = (Math.random() - 0.5) * 26 * fade;
      const jy = (Math.random() - 0.5) * 26 * fade;
      const r = 10 + t * 120 * view.scale + Math.random() * 22;
      ctx.beginPath();
      ctx.strokeStyle = `rgba(${rgb},${(0.15 + Math.random() * 0.5) * fade})`;
      ctx.lineWidth = 1 + Math.random() * 2;
      ctx.arc(o.x + jx, o.y + jy, r, 0, Math.PI * 2);
      ctx.stroke();
    }
  },

  // Timeline jump: a bright bar sweeps sideways across the map with a trailing
  // wake, like scrubbing along a filmstrip.
  sweep(t, rgb, o, w, h) {
    const x = t * (w + 160) - 80;
    const g = ctx.createLinearGradient(x - 90, 0, x + 30, 0);
    g.addColorStop(0, `rgba(${rgb},0)`);
    g.addColorStop(0.85, `rgba(${rgb},${0.45 * (1 - t * 0.4)})`);
    g.addColorStop(1, `rgba(${rgb},${0.85 * (1 - t * 0.4)})`);
    ctx.fillStyle = g;
    ctx.fillRect(x - 90, 0, 120, h);
    ctx.strokeStyle = `rgba(235,240,255,${0.55 * (1 - t)})`;
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.moveTo(x, 0);
    ctx.lineTo(x, h);
    ctx.stroke();
  },

  // Simulation shift: the frame tears into horizontally-displaced scanline
  // bands, as if the world is being re-rendered.
  glitch(t, rgb, o, w, h) {
    const fade = 1 - t;
    for (let b = 0; b < 16; b++) {
      if (Math.random() > 0.55) continue;
      const by = Math.random() * h;
      const bh = 3 + Math.random() * 18;
      const off = (Math.random() - 0.5) * 70 * fade;
      ctx.fillStyle = `rgba(${rgb},${(0.08 + Math.random() * 0.3) * fade})`;
      ctx.fillRect(off, by, w, bh);
    }
  },

  // Observer shift: eyelids close from top and bottom and reopen — waking as a
  // different observer.
  blink(t, rgb, o, w, h) {
    const a = Math.sin(t * Math.PI);           // fully shut at the midpoint
    const lid = a * (h / 2 + 2);
    ctx.fillStyle = "rgba(3,4,11,0.97)";
    ctx.fillRect(0, 0, w, lid);
    ctx.fillRect(0, h - lid, w, lid);
    ctx.fillStyle = `rgba(${rgb},${0.6 * a})`; // glowing rim on each lid edge
    ctx.fillRect(0, lid - 2, w, 2);
    ctx.fillRect(0, h - lid, w, 2);
  },

  // Consensus drift: a heavy, wobbling shockwave rolls outward as agreed reality
  // settles into a new shape.
  shockwave(t, rgb, o) {
    const fade = 1 - t;
    const base = 10 + t * 200 * view.scale;
    ctx.strokeStyle = `rgba(${rgb},${0.9 * fade})`;
    ctx.lineWidth = 3 * fade + 0.5;
    ctx.beginPath();
    for (let ang = 0; ang <= Math.PI * 2 + 0.16; ang += 0.16) {
      const r = base + Math.sin(ang * 6 + t * 18) * 12 * fade;
      const px = o.x + Math.cos(ang) * r, py = o.y + Math.sin(ang) * r;
      if (ang === 0) ctx.moveTo(px, py);
      else ctx.lineTo(px, py);
    }
    ctx.closePath();
    ctx.stroke();
  },

  // Time shift: a clock hand sweeps a full turn around the origin, tracing a
  // dial as it goes.
  clock(t, rgb, o) {
    const R = 130 * view.scale;
    const fade = 1 - t;
    ctx.strokeStyle = `rgba(${rgb},${0.35 * fade})`;
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    ctx.arc(o.x, o.y, R, 0, Math.PI * 2);
    ctx.stroke();
    const ang = -Math.PI / 2 + t * Math.PI * 2;
    ctx.strokeStyle = `rgba(${rgb},${0.9 * fade})`;
    ctx.lineWidth = 3;
    ctx.beginPath();
    ctx.moveTo(o.x, o.y);
    ctx.lineTo(o.x + Math.cos(ang) * R, o.y + Math.sin(ang) * R);
    ctx.stroke();
    ctx.strokeStyle = `rgba(${rgb},${0.55 * fade})`;
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.arc(o.x, o.y, R, -Math.PI / 2, ang);
    ctx.stroke();
  },

  // Mathematical shift: the underlying grid of structure flashes into view and
  // fades, exposing the maths beneath the world.
  grid(t, rgb, o, w, h) {
    const a = Math.sin(t * Math.PI);
    ctx.strokeStyle = `rgba(${rgb},${0.35 * a})`;
    ctx.lineWidth = 1;
    ctx.beginPath();
    const gap = 40;
    for (let x = 0; x <= w; x += gap) { ctx.moveTo(x, 0); ctx.lineTo(x, h); }
    for (let y = 0; y <= h; y += gap) { ctx.moveTo(0, y); ctx.lineTo(w, y); }
    ctx.stroke();
  },
};

// ── Interaction ────────────────────────────────────────────────────────────

function nodeAt(sx, sy) {
  for (const n of nodes.values()) {
    const p = toScreen(n);
    const dx = sx - p.x, dy = sy - p.y;
    if (dx * dx + dy * dy <= (10 * view.scale + 4) ** 2) return n;
  }
  return null;
}

let dragging = null;
// The node under the pointer, if any. draw() reveals its full (un-abbreviated)
// label; cleared while dragging and when the pointer leaves the canvas.
let hoveredId = null;
canvas.addEventListener("mousedown", (e) => {
  const hit = nodeAt(e.offsetX, e.offsetY);
  if (hit && state && hit.id !== state.session.Location) {
    runMove("travel " + hit.id);
    return;
  }
  // A plain drag pans the map; Shift+drag orbits it in 3D. When starting an
  // orbit we capture the pivot — the world point under the cursor — so the drag
  // spins the view about that point rather than the canvas centre.
  // A manual pan/orbit cancels any in-flight auto-framing so the user always wins.
  viewAnim = null;
  dragging = { x: e.offsetX, y: e.offsetY, rotate: e.shiftKey };
  if (dragging.rotate) {
    dragging.pivot = unproject(e.offsetX, e.offsetY, view, canvas.clientWidth, canvas.clientHeight);
  }
});
// Hint clickability: show a pointer cursor over reachable nodes (the blue
// "travel here" ones), the default cursor otherwise. Skipped while dragging so
// panning/rotating keeps its own cursor.
canvas.addEventListener("mousemove", (e) => {
  if (dragging) {
    if (hoveredId !== null) { hoveredId = null; updatePreview(); renderInspector(); }
    return;
  }
  const hit = nodeAt(e.offsetX, e.offsetY);
  const id = hit ? hit.id : null;
  if (id !== hoveredId) { hoveredId = id; updatePreview(); renderInspector(); }
  canvas.style.cursor = hit && hit.reachable ? "pointer" : "default";
});
canvas.addEventListener("mouseleave", () => {
  if (hoveredId !== null) { hoveredId = null; updatePreview(); renderInspector(); }
});
window.addEventListener("mousemove", (e) => {
  if (!dragging) return;
  if (dragging.rotate) {
    // Orbit about the grabbed pivot: apply the free rotation deltas (horizontal
    // motion turns yaw, vertical turns pitch), then re-pan so the pivot stays
    // anchored under the point where the drag began — the map spins about the
    // cursor, not the canvas centre.
    view.rotY += e.movementX * ROTATE_SPEED;
    view.rotX += e.movementY * ROTATE_SPEED;
    const pan = panToScreen(dragging.pivot, view, canvas.clientWidth, canvas.clientHeight, dragging.x, dragging.y);
    view.ox = pan.ox;
    view.oy = pan.oy;
  } else {
    view.ox += e.movementX;
    view.oy += e.movementY;
  }
});
window.addEventListener("mouseup", () => { dragging = null; });
// Warn before leaving with unsaved mutations, so a branch-heavy session isn't
// lost to an accidental navigation. Armed only while the server reports dirty.
window.addEventListener("beforeunload", (e) => {
  if (!dirty) return;
  e.preventDefault();
  e.returnValue = "";
});
canvas.addEventListener("wheel", (e) => {
  e.preventDefault();
  viewAnim = null; // manual zoom cancels any in-flight auto-framing
  const step = e.deltaY < 0 ? 1.1 : 0.9;
  // clampScale keeps zoom sane; its floor is low so a big, sprawling map can be
  // pulled right out to fit on screen. Then re-centre the pan on the pointer so
  // the zoom homes in on whatever is under the cursor, not the canvas centre.
  // `applied` is the real scale ratio after clamping, so panning doesn't drift
  // once zoom is pinned at its min/max.
  const newScale = clampScale(view.scale * step);
  const applied = newScale / view.scale;
  view.ox = zoomOffset(view.ox, e.offsetX, canvas.clientWidth / 2, applied);
  view.oy = zoomOffset(view.oy, e.offsetY, canvas.clientHeight / 2, applied);
  view.scale = newScale;
}, { passive: false });

// Commands that legitimately don't move the session (they manage the session or
// map, not the position) are run plainly; everything else goes through runMove so
// a refused transition/travel flashes the button and sounds the blocked cue.
const NON_MOVE_CMDS = new Set(["save", "home", "quest"]);
document.querySelectorAll("button[data-cmd]").forEach((b) => {
  b.addEventListener("click", () => {
    const cmd = b.dataset.cmd;
    if (NON_MOVE_CMDS.has(cmd.split(" ")[0])) run(cmd);
    else runMove(cmd, b);
  });
});

// View-orientation buttons (client-side only, no server round-trip). "vertical"
// snaps to the depth-ladder view; "reset" restores the free three-quarter view;
// "fit" keeps the current angle but re-frames so every node is on screen. The
// running tick() redraws automatically.
document.getElementById("view-vertical").addEventListener("click", setVerticalView);
document.getElementById("view-reset").addEventListener("click", setDefaultView);
document.getElementById("view-fit").addEventListener("click", setFitView);

// Reset map: full server-side reset back to the starting realities. Distinct
// from view-reset (which only moves the camera) — this discards every branch
// reality transitions created. Clear the local node cache first so removed
// nodes vanish, and null out state so re-applying base reality isn't read as a
// reality transition (which would fire a stray effect/halo).
document.getElementById("reset-map").addEventListener("click", async () => {
  try {
    const s = await api("/api/reset", {});
    nodes.clear();
    state = null;
    apply(s);
  } catch (err) { console.error("reset failed", err); }
});

document.getElementById("cmdform").addEventListener("submit", (e) => {
  e.preventDefault();
  const v = cmdInput.value.trim();
  cmdInput.value = "";
  if (v) run(v);
});

// Time picker: the datetime-local value is wall-clock local time; convert it to
// a UTC RFC3339 timestamp (dropping milliseconds) so it matches the format the
// facade's `time <RFC3339>` command expects.
const timeGoBtn = document.getElementById("time-go");
timeGoBtn.addEventListener("click", () => {
  const v = document.getElementById("time-input").value;
  if (!v) return;
  const d = new Date(v);
  if (isNaN(d.getTime())) return;
  runMove("time " + d.toISOString().replace(/\.\d{3}Z$/, "Z"), timeGoBtn);
});

// Observer picker: send whatever perspective the user typed (free-form, e.g.
// Bat, Dog, Machine); the facade creates the branch on demand. Enter submits.
const observerInput = document.getElementById("observer-input");
const observeGoBtn = document.getElementById("observe-go");
function runObserve() {
  const v = observerInput.value.trim();
  if (v) runMove("observe " + v, observeGoBtn);
}
observeGoBtn.addEventListener("click", runObserve);
observerInput.addEventListener("keydown", (e) => {
  if (e.key === "Enter") { e.preventDefault(); runObserve(); }
});

// Home confirmation: buttons plus y / n / Esc shortcuts, active only while the
// server reports it is awaiting confirmation.
document.getElementById("confirm-yes").addEventListener("click", () => run("y"));
document.getElementById("confirm-no").addEventListener("click", () => run("n"));
window.addEventListener("keydown", (e) => {
  if (!state || !state.awaitingHomeConfirm) return;
  if (e.key === "y" || e.key === "Y") { e.preventDefault(); run("y"); }
  else if (e.key === "n" || e.key === "N" || e.key === "Escape") { e.preventDefault(); run("n"); }
});

// Right-pane resizer: drag the handle to set the sidebar width. The canvas
// re-fits itself in draw() on the next frame, so no explicit resize is needed.
(function initResizer() {
  const resizer = document.getElementById("resizer");
  const side = document.getElementById("side");
  const MIN = 240;      // keep the controls usable
  const MAP_MIN = 320;  // always leave room for the map
  let active = false;
  resizer.addEventListener("mousedown", (e) => {
    e.preventDefault();
    active = true;
    resizer.classList.add("dragging");
    document.body.style.userSelect = "none";
  });
  window.addEventListener("mousemove", (e) => {
    if (!active) return;
    const want = window.innerWidth - e.clientX;
    const max = window.innerWidth - MAP_MIN;
    side.style.width = Math.max(MIN, Math.min(max, want)) + "px";
  });
  window.addEventListener("mouseup", () => {
    if (!active) return;
    active = false;
    resizer.classList.remove("dragging");
    document.body.style.userSelect = "";
  });
})();

// Populate the transitions key from MODE_STYLE so its colours never drift from
// the edges and animations they describe.
(function buildTransitionsLegend() {
  const el = document.getElementById("legend-transitions");
  if (!el) return;
  el.innerHTML =
    '<span class="legend-title">transitions</span>' +
    TRANSITION_LEGEND.map(({ mode, label, command }) => {
      const st = modeStyle(mode);
      return `<span class="legend-item"><i class="line" style="background:rgb(${st.rgb})"></i>${escapeHtml(label)}<span class="legend-cmd">${escapeHtml(command)}</span></span>`;
    }).join("");
})();

setVerticalView(); // open in the depth-ladder orientation
refresh();
requestAnimationFrame(tick);
