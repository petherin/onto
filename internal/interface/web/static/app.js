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
  TRANSITIONS_BY_COST,
  detectTransition,
  transitionIntensity,
  dramaSpec,
  impactVoices,
  SHATTER_MIN_INTENSITY,
  requiredTransition,
  physicalRoute,
  routeTotals,
  effectSpec,
  soundSpec,
  sessionMoved,
  colorFor,
  layerZ,
  layoutTarget,
  realityShells,
  shellStyle,
  clampScale,
  zoomOffset,
  fitView,
  FIT_GROUP_MARGIN,
  FIT_GROUP_MIN_SPAN,
  unproject,
  panToScreen,
  panToScreenOrbit,
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
// The latest NodeSnapshot list, kept as-is so drawRealityShells can regroup it
// into nested reality shells each frame (realityShells reuses layoutTarget/layerZ
// to place each shell exactly where the layout settles its nodes).
let graphNodes = [];

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
// pivot is the world point a Shift+drag orbit rotates about, drawn as a small
// crosshair. Until the user drags it (pivotPinned), it tracks the current
// location ("you are here") so rotation already spins about something meaningful;
// dragging the crosshair pins it to a fixed world point of the user's choosing.
const pivot = { x: 0, y: 0, z: 0 };
let pivotPinned = false;

// The two orientation presets (Ladder / Angled) back the segmented view toggle
// and the initial framing; Fit is an independent framing action. Ladder is a
// near-top-down angle with no yaw, so z (nesting depth) drives screen-y: base
// reality sits at the top and the deepest layer at the bottom. Angled is the
// free three-quarter view. Both now fit the whole map to the canvas at their own
// angle in a single click (fitToContent), so choosing an orientation also frames
// it — no follow-up Fit press needed. Before any nodes exist (first boot) there
// is nothing to measure, so each falls back to a tuned static framing.
function setVerticalView() {
  viewAnim = null;
  view.rotX = -1.42; view.rotY = 0;
  if (!fitToContent()) { view.ox = 0; view.oy = -canvas.clientHeight * 0.32; view.scale = 1; }
  setActiveView("view-vertical");
}
function setDefaultView() {
  viewAnim = null;
  view.rotX = 0.5; view.rotY = 0.35;
  if (!fitToContent()) { view.ox = 0; view.oy = 0; view.scale = 1; }
  setActiveView("view-reset");
}
// setFitView keeps the current rotation but re-frames the map so every node fits
// on screen at once. Use it to recover from a map that has sprawled or zoomed
// off-frame without losing the current angle; it leaves the active orientation
// as it is, since Fit does not change the viewing angle.
function setFitView() {
  viewAnim = null;
  fitToContent();
}
// fitToContent snaps scale + pan so every node fits the canvas at the current
// rotation, returning false (and changing nothing) when no nodes exist yet.
function fitToContent() {
  if (!nodes.size) return false;
  const fit = fitView(nodes.values(), view, canvas.clientWidth, canvas.clientHeight);
  view.scale = fit.scale; view.ox = fit.ox; view.oy = fit.oy;
  return true;
}
// setActiveView highlights whichever orientation preset is current and keeps the
// segmented toggle's aria-pressed state in step; clearActiveView drops the
// highlight when a manual orbit (Shift+drag) leaves both presets behind.
function setActiveView(activeId) {
  for (const id of ["view-vertical", "view-reset"]) {
    const b = document.getElementById(id);
    if (!b) continue;
    const on = id === activeId;
    b.classList.toggle("is-active", on);
    b.setAttribute("aria-pressed", on ? "true" : "false");
  }
}
function clearActiveView() {
  for (const id of ["view-vertical", "view-reset"]) {
    const b = document.getElementById(id);
    if (!b) continue;
    b.classList.remove("is-active");
    b.setAttribute("aria-pressed", "false");
  }
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
// FIT_GROUP_MIN_SPAN floors the framing box so a small auto-generated cluster
// (one to three nodes) settles at a steady, moderate zoom — zoomed in on the new
// node(s), never snapped far out (a single node's zero-size box) or in too close
// (a tight cluster maxing the scale). Falls back to frameToFit if the group is
// empty or somehow un-framable.
function frameToGroup(ids) {
  const group = ids.map((id) => nodes.get(id)).filter(Boolean);
  if (!group.length) { frameToFit(); return; }
  const fit = fitView(group, view, canvas.clientWidth, canvas.clientHeight, FIT_GROUP_MARGIN, FIT_GROUP_MIN_SPAN);
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
    const intensity = transitionIntensity(mode);
    triggerEffect(mode, intensity);
    playSound(mode, intensity);
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
  graphNodes = graph.Nodes || [];
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

// REPULSION_MAX_NODES caps the all-pairs anti-overlap force. It is O(n²) per
// frame, so on an enormous, deeply-chained map (hundreds of generated nodes in
// one reality) it is what makes the map feel sluggish. Above this count we skip
// it entirely: repulsion only ever nudged co-located nodes apart, and the
// deterministic target spring below already lays every node out at its own home
// (the downward chain tree spreads siblings and depth on its own), so dropping
// it costs almost nothing visually while keeping the frame responsive.
const REPULSION_MAX_NODES = 350;

function tick() {
  const arr = [...nodes.values()];
  // Repulsion between every pair of nodes, now in three dimensions. It is only
  // there to keep nodes that share a layout home from overlapping, so it is
  // gentle — the deterministic target spring below owns the overall shape. On a
  // huge map (see REPULSION_MAX_NODES) it is skipped to keep the frame quick.
  if (arr.length <= REPULSION_MAX_NODES) {
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

// triggerEffect queues a transition's animation. Every transition plays its own
// character-matched effect (effectSpec), and a costly one — scaled by its σ cost
// via transitionIntensity/dramaSpec — also shakes the screen, harder the dearer
// the move, so cheap shifts get a near-zero shake and keep their quiet character.
// The number-storm overlay that bursts the old structure apart and reassembles
// it as the new one is the mathematical-structure jump's signature alone: it
// fires only for mode "math" (at full intensity), so other big moves (timeline,
// universe) still shake with cost but don't borrow the numbers — their own
// dramatic flourishes are still to be designed.
function triggerEffect(mode, intensity = 0) {
  const now = performance.now();
  const { kind, duration } = effectSpec(mode);
  effects.push({ mode, kind, duration, start: now });
  const drama = dramaSpec(intensity);
  triggerShake(drama.shakeAmp, drama.shakeDuration);
  if (mode === "math" && drama.shatter) {
    effects.push({ mode, kind: "shatter", duration: drama.duration, start: now, drama, particles: null });
  }
}

// ── Screen shake ─────────────────────────────────────────────────────────────
// A costly transition rattles the canvas. triggerShake arms a decaying jitter
// scaled by the move's intensity (dramaSpec); shakeOffset returns the current
// frame's offset (or null once spent) for draw() to translate the whole scene
// by. The amplitude eases out (squared decay) so the world settles rather than
// stopping dead. A near-zero amplitude (a cheap shift) arms nothing.
let shake = null;

function triggerShake(amp, duration) {
  if (amp <= 0.1) return;
  shake = { start: performance.now(), duration, amp };
}

function shakeOffset() {
  if (!shake) return null;
  const p = (performance.now() - shake.start) / shake.duration;
  if (p >= 1) { shake = null; return null; }
  const a = shake.amp * (1 - p) * (1 - p);
  return { x: (Math.random() * 2 - 1) * a, y: (Math.random() * 2 - 1) * a };
}

// ── Transition sound ─────────────────────────────────────────────────────────
// Each reality transition plays its own layered, cinematic cue (soundSpec picks
// the voices in logic.js). The Web Audio context is created lazily — primed on
// the first user gesture (see unlockAudio) so the browser's autoplay policy is
// satisfied without a separate "click to enable" step, and lazily created by
// playSound too if a cue somehow fires first. A muted preference persists in
// localStorage and gates playback entirely.
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
//
// One exception to the all-synthesised rule: the mathematical-structure jump
// also layers a single real recorded explosion (a CC0 sample, see
// MATH_EXPLOSION_URL) under its crystal chord, so the dearest move detonates.
const MASTER_GAIN = 0.12;
const REVERB_SECONDS = 3.4;   // impulse length — how long the tail rings
const REVERB_DECAY = 2.2;     // higher = faster decay (steeper tail)
const REVERB_WET = 0.38;      // reverb send level relative to the dry bus
const REVERB_PREDELAY = 0.03; // gap before the tail — pushes the space back
// The mathematical-structure jump layers a real recorded explosion under its
// synthesised crystal chord — the one audio asset in the app: a CC0 "weird
// explosion" (freesound.org/s/336009, public domain, no attribution required),
// embedded and served with the SPA. MATH_EXPLOSION_GAIN is its level relative to
// MASTER_GAIN, before the intensity lift in playSound.
const MATH_EXPLOSION_URL = "sounds/math-explosion.mp3";
const MATH_EXPLOSION_GAIN = 5.0;
let audioCtx = null;
let dryBus = null;    // voices → here (dry) → saturation
let wetSend = null;   // voices → here → pre-delay → convolver → saturation
let noiseBuffer = null; // shared white-noise source buffer for noise voices
let mathExplosionBuffer = null; // decoded math-only explosion sample (loadMathExplosion)
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
  loadMathExplosion();
  return true;
}

// loadMathExplosion fetches and decodes the one real audio asset in the app — a
// CC0 "weird explosion" layered under the mathematical-structure cue (playSound).
// It's fired once from ensureAudio and cached, so the sample is decoded before a
// transition fires; a failure just leaves the synthesised cue to play alone. The
// URL is relative to the SPA, so it resolves against the static bundle's origin
// (never ONTO_API_BASE) whether same-origin or split-deployed.
async function loadMathExplosion() {
  if (mathExplosionBuffer) return;
  try {
    const res = await fetch(MATH_EXPLOSION_URL);
    mathExplosionBuffer = await audioCtx.decodeAudioData(await res.arrayBuffer());
  } catch (err) {
    console.error("math explosion sample load failed", err);
  }
}

// playSample fires a decoded audio buffer through the same dry + reverb buses
// every synthesised voice uses, so a recorded sample sits in the same scored
// space (saturation, glue compression, and the shared tail) as the rest of the cue.
function playSample(buffer, gainValue, when) {
  const src = audioCtx.createBufferSource();
  src.buffer = buffer;
  const gain = audioCtx.createGain();
  gain.gain.value = gainValue * MASTER_GAIN;
  src.connect(gain);
  gain.connect(dryBus);
  gain.connect(wetSend);
  src.start(when);
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

function playSound(mode, intensity = 0) {
  if (muted) return;
  try {
    if (!ensureAudio()) return;
    if (audioCtx.state === "suspended") audioCtx.resume();
    const now = audioCtx.currentTime;
    // Big moves hit harder: a costly transition (transitionIntensity) both lifts
    // every voice's level and, past the shatter threshold, adds a low-end impact
    // (impactVoices) under the character cue, so the sound scales with the σ cost
    // the same way the shake and the number storm do.
    const i = Math.max(0, Math.min(1, intensity));
    const gainScale = 0.75 + i * 0.6;
    const spec = soundSpec(mode);
    const voices = i >= SHATTER_MIN_INTENSITY ? [...spec.voices, ...impactVoices(i)] : spec.voices;
    for (const v of voices) {
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
      gain.gain.linearRampToValueAtTime(v.gain * MASTER_GAIN * gainScale, t1);
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
    // The mathematical-structure jump's signature: a real recorded explosion
    // fired at the top of the cue, beneath the synthesised crystal chord, so the
    // dearest move actually detonates instead of only chiming. Math-only, scaled
    // by the same intensity lift as the voices, and skipped silently if the
    // sample hasn't finished decoding yet.
    if (mode === "math" && mathExplosionBuffer) {
      playSample(mathExplosionBuffer, MATH_EXPLOSION_GAIN * gainScale, now);
    }
  } catch (err) {
    console.error("sound failed", err);
  }
}

// First-gesture audio unlock. A browser only lets an AudioContext start (or
// resume from "suspended") inside a real user gesture. Every cue is played after
// an await (the /api/execute round-trip), which on WebKit (Safari, iOS) falls
// outside the gesture window and would leave the context suspended, so no sound
// ever reaches the speakers. Chromium and Firefox use "sticky" activation and
// forgive this; Safari does not. Priming the context on the first pointer/key/
// touch anywhere — synchronously, inside that gesture — means every later
// playSound just works. It runs regardless of mute so unmuting mid-session needs
// no special case: a muted context simply sits idle (playSound short-circuits).
// The listeners are one-shot.
function unlockAudio() {
  if (!ensureAudio()) return;
  if (audioCtx.state === "suspended") audioCtx.resume();
}
for (const ev of ["pointerdown", "keydown", "touchstart"]) {
  window.addEventListener(ev, unlockAudio, { once: true });
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
// table is built from the same cost-sorted view the legend uses, so the colours,
// commands, and costs never drift from the map and it reads cheapest-first.
(function initHelp() {
  const modal = document.getElementById("help-modal");
  const openBtn = document.getElementById("help-btn");
  const closeBtn = document.getElementById("help-close");
  const backdrop = document.getElementById("help-backdrop");
  const axesBox = document.getElementById("help-axes");
  if (!modal || !openBtn) return;
  if (axesBox) {
    axesBox.innerHTML = TRANSITIONS_BY_COST.map(({ mode, label, command, cost, what, refs }) => {
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

// CUR_MIN_RADIUS is the on-screen radius the current location never drops below,
// whatever the zoom or map size. It is deliberately well above the ordinary dot
// floor (1.5) so the "you are here" marker (plus its green glow, drawn at
// r + 10) stays easy to spot even pulled right out over an enormous graph.
const CUR_MIN_RADIUS = 8;

// drawRealityShells paints the nested reality shells (realityShells in logic.js)
// as faint circles behind the graph, so the map's Tegmark nesting reads at rest:
// the mathematical structure (Level IV) as the outermost shell, a bubble universe
// (Level II) nested inside it, and a timeline (Level I) as the innermost thin
// dashed ring. The quantum branch (Level III) has no shell — its nodes stay
// tinted.
//
// The shells are computed directly in *screen space*: every node is projected
// first (toScreen), and realityShells is handed those projected points so it
// measures each ring's centre/radius from where the nodes actually land on the
// canvas. This makes containment hold by construction at any zoom or rotation —
// a ring encloses the exact screen positions of its members, so a node can never
// project outside its shell (the failure of the old world-space ring, which was a
// flat circle at one averaged depth that perspective could tip an off-depth node
// out of). encloseChildren still runs inside realityShells, now in screen space,
// so the Tegmark nesting holds on screen too. Shells arrive outermost-first, so
// painting in order lays the big faint math shell behind the nested ones.
function drawRealityShells() {
  if (!graphNodes.length) return;
  const cur = state && state.session ? state.session.Location : null;
  // Project every node up front to its live (force-settled) screen position,
  // falling back to its deterministic home for one not yet seeded, and track the
  // largest on-screen node-dot radius (matching the dot sizing in draw()). The
  // rings are then padded by that so a node's whole disc — not just its centre —
  // stays inside its shell. Adding one constant to every shell radius preserves
  // the nesting encloseChildren guarantees, since it grows parent and child alike.
  const screen = new Map();
  let maxDot = 0;
  for (const n of graphNodes) {
    const live = nodes.get(n.ID);
    let world;
    if (live) {
      world = { x: live.x, y: live.y, z: live.z };
    } else {
      const home = layoutTarget(n);
      world = { x: home.x, y: home.y, z: layerZ(n.Depth) };
    }
    const p = toScreen(world);
    screen.set(n.ID, p);
    const isCur = n.ID === cur;
    const dot = Math.max((isCur ? 9 : 6) * view.scale * p.persp, isCur ? CUR_MIN_RADIUS : 1.5);
    if (dot > maxDot) maxDot = dot;
  }
  const screenPos = (n) => {
    const p = screen.get(n.ID);
    return p ? { x: p.x, y: p.y, z: p.depth } : { x: 0, y: 0, z: 0 };
  };
  ctx.save();
  ctx.textAlign = "center";
  const shells = realityShells(graphNodes, screenPos);
  for (const s of shells) {
    const st = shellStyle(s.kind);
    // s.cx/s.cy/s.radius are already screen-space; pad by the node-dot size so the
    // disc stays inside. No per-shell perspective scale — that is baked into the
    // projected points the shell was measured from.
    const r = s.radius + maxDot;
    if (!(r > 0)) continue;
    ctx.beginPath();
    ctx.arc(s.cx, s.cy, r, 0, Math.PI * 2);
    if (st.fill > 0) {
      ctx.fillStyle = `rgba(${st.rgb},${st.fill})`;
      ctx.fill();
    }
    ctx.lineWidth = st.width;
    ctx.strokeStyle = `rgba(${st.rgb},${st.line})`;
    ctx.setLineDash(st.dash);
    ctx.stroke();
    // Label the shell with the reality version it denotes (its universe / maths /
    // timeline name), sitting just outside the ring at its top. Nested shells have
    // strictly smaller radii, so their labels stack clear of one another there.
    if (s.label) {
      ctx.setLineDash([]);
      ctx.font = `${11 * Math.max(view.scale, 0.7)}px ui-monospace, monospace`;
      ctx.fillStyle = `rgba(${st.rgb},0.85)`;
      ctx.fillText(abbreviateLabel(s.label, 22), s.cx, s.cy - r - 4);
    }
  }
  ctx.restore();
  ctx.setLineDash([]);
}

// PIVOT_HIT_R is the screen-space radius (px) within which a mousedown grabs the
// pivot crosshair to reposition it, rather than starting a pan.
const PIVOT_HIT_R = 16;

// syncPivotToCurrent keeps an un-pinned pivot on the current location so a
// Shift+drag orbit spins about "you are here" by default. Once the user drags
// the crosshair (pivotPinned), the pivot stays wherever they put it.
function syncPivotToCurrent(curId) {
  if (pivotPinned) return;
  const node = curId ? nodes.get(curId) : null;
  if (node) { pivot.x = node.x; pivot.y = node.y; pivot.z = node.z; }
}

// pivotAt reports whether screen point (sx, sy) is close enough to the pivot
// crosshair to grab it (within PIVOT_HIT_R px of its projected centre).
function pivotAt(sx, sy) {
  const p = toScreen(pivot);
  return Math.hypot(sx - p.x, sy - p.y) <= PIVOT_HIT_R;
}

// drawPivot renders the rotation pivot as a gapped crosshair reticle with a ring.
// It sits over the current-location dot by default, so it's drawn larger than the
// node halo, with a light stroke over a dark backing to stay legible against both
// the green node and the dark background. It reads brighter once pinned (a fixed
// user-chosen point) than while it follows the current location.
function drawPivot() {
  const p = toScreen(pivot);
  const ring = 13;   // reticle ring radius — wider than the current-node halo (r+10)
  const arm = 20;    // outer reach of each crosshair arm
  const gap = 6;     // inner gap so the point/node underneath stays visible
  const draw = () => {
    ctx.beginPath();
    ctx.moveTo(p.x - arm, p.y); ctx.lineTo(p.x - gap, p.y);
    ctx.moveTo(p.x + gap, p.y); ctx.lineTo(p.x + arm, p.y);
    ctx.moveTo(p.x, p.y - arm); ctx.lineTo(p.x, p.y - gap);
    ctx.moveTo(p.x, p.y + gap); ctx.lineTo(p.x, p.y + arm);
    ctx.stroke();
    ctx.beginPath();
    ctx.arc(p.x, p.y, ring, 0, Math.PI * 2);
    ctx.stroke();
  };
  ctx.save();
  ctx.globalAlpha = pivotPinned ? 1 : 0.75;
  // Dark backing for contrast against the light nodes/shells.
  ctx.strokeStyle = "rgba(0,0,0,0.65)";
  ctx.lineWidth = 3.5;
  draw();
  // Bright foreground line.
  ctx.strokeStyle = themeColors.ink;
  ctx.lineWidth = 1.5;
  draw();
  ctx.restore();
}

function draw() {
  const dpr = window.devicePixelRatio || 1;
  if (canvas.width !== canvas.clientWidth * dpr) {
    canvas.width = canvas.clientWidth * dpr;
    canvas.height = canvas.clientHeight * dpr;
  }
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  // Screen shake: a costly transition rattles the whole scene (triggerShake).
  // Applied as a decaying translate after the clear, so every layer — shells,
  // edges, nodes, effects, and the pivot — shudders together and settles.
  const sh = shakeOffset();
  if (sh) ctx.translate(sh.x, sh.y);
  const cur = state && state.session ? state.session.Location : null;
  syncPivotToCurrent(cur);

  // Reality shells first, so the nested Tegmark-level circles sit behind the
  // edges and nodes as faint context rather than over them.
  drawRealityShells();

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
  // ones — the depth cue that sells the 3D rotation. The current location is
  // forced to the very end so it always paints on top: on an enormous map a
  // nearer node could otherwise cover it, and you must never lose track of where
  // you are.
  const drawn = [...nodes.values()].map((n) => ({ n, p: toScreen(n) }));
  drawn.sort((a, b) => {
    if (a.n.id === cur) return 1;
    if (b.n.id === cur) return -1;
    return b.p.depth - a.p.depth;
  });
  // Depth range of what's on screen drives the relative fade (see depthAlpha).
  let minDepth = Infinity, maxDepth = -Infinity;
  for (const { p } of drawn) {
    if (p.depth < minDepth) minDepth = p.depth;
    if (p.depth > maxDepth) maxDepth = p.depth;
  }
  for (const { n, p } of drawn) {
    const isCur = n.id === cur;
    // Keep a floor on the radius so nodes stay visible (as dots) even when zoomed
    // right out, instead of shrinking to sub-pixel and vanishing. The current
    // location gets a much larger floor (CUR_MIN_RADIUS) that never shrinks with
    // zoom or map size, so however far out you pull an enormous map you can still
    // find where you are; other nodes keep the small dot floor.
    const r = Math.max((isCur ? 9 : 6) * view.scale * p.persp, isCur ? CUR_MIN_RADIUS : 1.5);
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
  drawPivot();
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
    render(t, modeStyle(e.mode).rgb, origin, w, h, e);
    ctx.restore();
  }
}

// Each renderer draws one transition for progress t in [0,1). They run inside a
// ctx.save()/restore() so they can set alpha, line styles, and clips freely.
//
// buildShatter creates the one-off cloud of glyphs the mathematical-structure
// jump shatters into: each glyph gets a random outward direction and distance
// (scaled by the move's spread), a spin, a size (bigger at full intensity, which
// is where the math jump runs), a short numeric-or-symbolic token, and a target
// node on the new structure — sampled from the live graph with a small
// offset — to reassemble onto. A per-glyph launch delay staggers the burst so
// the explosion ripples out rather than leaving all at once. With no nodes yet
// it falls back to the origin, so the storm still resolves to where you are.
function buildShatter(drama) {
  const ids = [...nodes.keys()];
  const out = [];
  for (let k = 0; k < drama.glyphs; k++) {
    out.push({
      angle: Math.random() * Math.PI * 2,
      dist: drama.spread * (0.35 + Math.random() * 0.65),
      spin: (Math.random() * 2 - 1) * Math.PI * 2,
      size: 14 + Math.random() * (14 + drama.intensity * 30),
      delay: Math.random() * 0.12,
      glyph: shatterGlyph(),
      nodeId: ids.length ? ids[Math.floor(Math.random() * ids.length)] : null,
      offX: (Math.random() * 2 - 1) * 10,
      offY: (Math.random() * 2 - 1) * 10,
    });
  }
  return out;
}

// SHATTER_SYMBOLS is the palette of esoteric mathematical notation the structure
// shatters into alongside the digits — integrals and contour integrals, del and
// partials, n-ary sums/products, set and logic operators, tensor/direct sums,
// aleph and beth numbers, the Weierstrass ℘, blackboard-bold number sets, Greek,
// and a few arcane relations (bowtie, multimap, pitchfork). All are BMP code
// points (single UTF-16 units) so plain string indexing samples them cleanly,
// and the renderer uses a symbol-capable font fallback so none render as tofu.
const SHATTER_SYMBOLS =
  "∫∮∯∇∂∑∏∐√∞∅∈∋⊂⊃∪∩⋃⋂∀∃∴∵≡≅≈⊕⊗⊙⊢⊨⋀⋁⨁⨂⨆⨅⨌⟨⟩↦⇒⇔ℵℶℷ℘ℝℂℤℕℚλπφψΩΓΔΘΛΞΣΦΨ≺≻⋈⋔⊸⅋";

// shatterGlyph returns one token the structure shatters into: about half the time
// an esoteric mathematical symbol (occasionally a short cluster), otherwise a
// short run of digits, so raw numbers and notation spill out together rather than
// a uniform digit rain.
function shatterGlyph() {
  if (Math.random() < 0.5) {
    const n = Math.random() < 0.3 ? 2 : 1;
    let s = "";
    for (let i = 0; i < n; i++) s += SHATTER_SYMBOLS[Math.floor(Math.random() * SHATTER_SYMBOLS.length)];
    return s;
  }
  const len = 1 + Math.floor(Math.random() * 3);
  let s = "";
  for (let i = 0; i < len; i++) s += Math.floor(Math.random() * 10);
  return s;
}

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

  // The mathematical-structure jump's signature (armed for mode "math" only in
  // triggerEffect): the old structure detonates into a storm of numbers that
  // shoots out from where you are — fast, on an ease-out burst so it explodes
  // then coasts — before reassembling onto the new structure's node positions.
  // Particles are built once (stored on the effect) then re-targeted to the live
  // nodes every frame, so they land on the new structure even as the camera
  // reframes and the layout settles. Each glyph draws a short motion-blur trail
  // (a few dimmer copies at earlier times) so the explosion and the gather both
  // streak, and a per-glyph launch delay ripples the blast outward.
  shatter(t, rgb, o, w, h, e) {
    if (!e.particles) e.particles = buildShatter(e.drama);
    const easeOut = (x) => 1 - Math.pow(1 - x, 3);          // fast then coasting
    const smooth = (x) => x * x * (3 - 2 * x);
    // posAt places a glyph at animation time tt: an ease-out fling apart for the
    // first half, then a smooth draw-back to its target node over the second.
    const posAt = (p, tt) => {
      const lt = Math.max(0, Math.min(1, (tt - p.delay) / (1 - p.delay)));
      const burst = easeOut(Math.min(1, lt / 0.5));
      const gather = smooth(Math.max(0, (lt - 0.45) / 0.55));
      const ex = o.x + Math.cos(p.angle) * p.dist * burst;
      const ey = o.y + Math.sin(p.angle) * p.dist * burst;
      const tgt = p.nodeId && nodes.has(p.nodeId) ? toScreen(nodes.get(p.nodeId)) : o;
      return {
        x: ex + (tgt.x + p.offX - ex) * gather,
        y: ey + (tgt.y + p.offY - ey) * gather,
        gather,
      };
    };
    const fadeIn = Math.min(1, t / 0.1);
    const fadeOut = 1 - Math.max(0, (t - 0.9) / 0.1);
    const trails = [0, 0.035, 0.07];   // trailing copies at earlier times
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    for (const p of e.particles) {
      // ui-monospace keeps the digits crisp; the symbol fonts cover any esoteric
      // glyph the mono face lacks (per-glyph fallback), so notation never tofus.
      ctx.font = `${p.size}px ui-monospace, "Apple Symbols", "Segoe UI Symbol", "STIX Two Math", monospace`;
      for (let ti = trails.length - 1; ti >= 0; ti--) {
        const { x, y, gather } = posAt(p, t - trails[ti]);
        const alpha = fadeIn * fadeOut * (1 - gather * 0.8) * (1 - ti * 0.32);
        if (alpha <= 0) continue;
        ctx.save();
        ctx.translate(x, y);
        ctx.rotate(p.spin * (t - trails[ti]));
        ctx.fillStyle = `rgba(${rgb},${alpha})`;
        ctx.fillText(p.glyph, 0, 0);
        ctx.restore();
      }
    }
  },
};

// ── Interaction ────────────────────────────────────────────────────────────

// nodeAt returns the node under the screen point, matching what the user sees and
// can act on. Among every node within the hit radius it prefers the nearest
// *reachable* one (the blue "travel here" nodes), falling back to the nearest node
// of any kind so unreachable places can still be hovered and inspected. This
// matters where nodes crowd a single spot — as an auto-generated chain does, since
// every node in a reality springs to the same layout home — because the old
// first-in-insertion-order pick often returned the current location (a click on
// your own node just starts a drag, so it silently "fails to move") or an
// unreachable node stacked on top (travel refused), even though a reachable node
// sat right there. "Nearest" is the smallest projected depth — the same
// front-to-back order draw() paints in — so the pick is the one on top.
function nodeAt(sx, sy) {
  const cur = state && state.session ? state.session.Location : null;
  let top = null, topDepth = Infinity;
  let reachable = null, reachableDepth = Infinity;
  for (const n of nodes.values()) {
    const p = toScreen(n);
    const dx = sx - p.x, dy = sy - p.y;
    if (dx * dx + dy * dy > (10 * view.scale + 4) ** 2) continue;
    if (p.depth < topDepth) { topDepth = p.depth; top = n; }
    if (n.reachable && n.id !== cur && p.depth < reachableDepth) {
      reachableDepth = p.depth; reachable = n;
    }
  }
  return reachable || top;
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
  // A manual pan/orbit/pivot-move cancels any in-flight auto-framing so the user
  // always wins.
  viewAnim = null;
  // Grabbing the pivot crosshair (within PIVOT_HIT_R) starts a pivot-move drag:
  // dragging it repositions and pins the rotation centre to a new world point.
  if (pivotAt(e.offsetX, e.offsetY)) {
    dragging = { x: e.offsetX, y: e.offsetY, movePivot: true };
    return;
  }
  // A plain drag pans the map; Shift+drag orbits it in 3D about the pivot.
  dragging = { x: e.offsetX, y: e.offsetY, rotate: e.shiftKey };
  if (dragging.rotate) {
    // An orbit leaves both orientation presets behind, so drop the toggle
    // highlight — it should only be lit while the view matches a preset angle.
    clearActiveView();
    // Anchor the orbit on the pivot's current screen position so the crosshair
    // stays fixed while the scene spins about it.
    dragging.pivot = pivot;
    const sp = toScreen(pivot);
    dragging.x = sp.x;
    dragging.y = sp.y;
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
  // Over the pivot crosshair, hint that it can be grabbed and moved.
  if (pivotAt(e.offsetX, e.offsetY)) canvas.style.cursor = "move";
  else canvas.style.cursor = hit && hit.reachable ? "pointer" : "default";
});
canvas.addEventListener("mouseleave", () => {
  if (hoveredId !== null) { hoveredId = null; updatePreview(); renderInspector(); }
});
window.addEventListener("mousemove", (e) => {
  if (!dragging) return;
  if (dragging.movePivot) {
    // Reposition the pivot by unprojecting the cursor onto the view plane through
    // the origin, then pin it there so it stops following the current location.
    const rect = canvas.getBoundingClientRect();
    const p = unproject(e.clientX - rect.left, e.clientY - rect.top, view, canvas.clientWidth, canvas.clientHeight);
    pivot.x = p.x; pivot.y = p.y; pivot.z = p.z;
    pivotPinned = true;
    return;
  }
  if (dragging.rotate) {
    // Orbit about the pivot: apply the free rotation deltas (horizontal motion
    // turns yaw, vertical turns pitch), then re-pan so the pivot stays anchored at
    // its screen position captured at drag start — the map spins about the visible
    // crosshair, not the canvas centre. The re-pan uses panToScreenOrbit (an
    // orthographic anchor), not the perspective panToScreen: the latter let the
    // pivot's perspective factor blow up as the angle swung it toward the focal
    // plane and threw the map off screen ("flies away when I rotate"). The
    // orthographic pan is bounded by the on-screen extent, so the orbit stays put.
    view.rotY += e.movementX * ROTATE_SPEED;
    view.rotX += e.movementY * ROTATE_SPEED;
    const pan = panToScreenOrbit(dragging.pivot, view, canvas.clientWidth, canvas.clientHeight, dragging.x, dragging.y);
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

// View-orientation buttons (client-side only, no server round-trip). "ladder"
// snaps to the depth-ladder view and fits it; "angled" restores the free
// three-quarter view and fits it; "fit" keeps the current angle but re-frames so
// every node is on screen. The running tick() redraws automatically. (Element
// ids are kept as view-vertical/view-reset for stability; only labels changed.)
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

// Populate the reality-shell key from SHELL_STYLE so the ring colours match the
// nested circles drawRealityShells paints. Each row names a Tegmark level and its
// shell, outermost first; the quantum branch (Level III) is intentionally absent
// since it has no shell (its nodes are tinted instead — see the node-colour key).
(function buildShellsLegend() {
  const el = document.getElementById("legend-shells");
  if (!el) return;
  const rows = [
    { kind: "math", label: "mathematical", level: "IV" },
    { kind: "universe", label: "universe", level: "II" },
    { kind: "timeline", label: "timeline", level: "I" },
  ];
  el.innerHTML =
    '<span class="legend-title">reality shells</span>' +
    rows.map(({ kind, label, level }) => {
      const st = shellStyle(kind);
      return `<span class="legend-item"><i class="ring" style="border-color:rgba(${st.rgb},0.8)"></i>${escapeHtml(label)}<span class="legend-cmd">Lvl ${escapeHtml(level)}</span></span>`;
    }).join("");
})();

setVerticalView(); // open in the depth-ladder orientation
refresh();
requestAnimationFrame(tick);
