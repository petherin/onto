// Onto Reality Map — a self-contained canvas front-end for the Onto facade.
// No build step, no external deps: it talks to /api/state and /api/execute
// and renders the universe graph as a force-directed constellation. The pure
// view logic (defaults, mode styling, transition detection, node colouring, and
// the 3D projection maths) lives in logic.js so it can be unit-tested in Node.

import {
  DEFAULTS,
  modeStyle,
  TRANSITION_LEGEND,
  detectTransition,
  requiredTransition,
  effectSpec,
  colorFor,
  layerZ,
  layoutTarget,
  clampScale,
  zoomOffset,
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
const lookEl = document.getElementById("look");
const logEl = document.getElementById("log");
const promptEl = document.getElementById("prompt");
const cmdInput = document.getElementById("cmd");
const confirmEl = document.getElementById("confirm");
const inspectorEl = document.getElementById("inspector");
const trailEl = document.getElementById("trail");
const saveBtn = document.getElementById("save-btn");

let state = null;
let edges = [];
const nodes = new Map(); // id -> {id, name, x, y, z, vx, vy, vz}
// view holds the camera. It's snapped to the vertical orientation at startup
// (see setVerticalView and the boot call at the end of the file) so the
// depth-layer stack reads as a ladder from the first frame; dragging, the wheel,
// and the view buttons mutate it from there.
const view = { scale: 1, ox: 0, oy: 0, rotX: 0, rotY: 0 };

// setVerticalView / setDefaultView back the view buttons and the initial framing.
// Vertical is a near-top-down angle with no yaw, so z (nesting depth) drives
// screen-y: base reality sits near the top and the deepest layer at the bottom,
// with an upward pan nudging the base layer to the top of the canvas rather than
// its centre. Default is the free three-quarter view.
function setVerticalView() {
  view.rotX = -1.42; view.rotY = 0;
  view.ox = 0; view.oy = -canvas.clientHeight * 0.32;
  view.scale = 1;
}
function setDefaultView() {
  view.rotX = 0.5; view.rotY = 0.35;
  view.ox = 0; view.oy = 0; view.scale = 1;
}
let logCount = 0;
// Mirrors stateDTO.Dirty: true when the session has unsaved mutations. Drives
// the save-button indicator and the beforeunload guard.
let dirty = false;

async function api(path, body) {
  const opts = body
    ? { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }
    : {};
  const res = await fetch(path, opts);
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

function apply(s) {
  const prev = state && state.session;
  state = s;
  const added = syncNodes(s.graph);
  renderHUD(s.session);
  renderLook(s);
  renderLog(s);
  renderInspector();
  renderTrail(s);
  renderDirty(s);
  // Bottom-right prompt shows the current node's name (a short, shell-like
  // cue); the full onto:// address lives in the loc badge in the header.
  // Hovering the prompt still reveals the address.
  const cur = currentNodeSnapshot(s);
  const name = cur ? cur.Name : (s.session ? s.session.Location : "");
  promptEl.textContent = `[${name}] > `;
  promptEl.title = (s.session && s.session.ShortOntoAddress) || name;
  renderConfirm(s);
  const mode = detectTransition(prev, s.session);
  if (mode) {
    triggerEffect(mode);
    // Halo whatever locations this transition just revealed, in the transition's
    // own colour, so the eye is drawn to what changed. Physical moves can add
    // nodes too, but only a reality transition arms the halo.
    const rgb = modeStyle(mode).rgb;
    const now = performance.now();
    for (const id of added) {
      const n = nodes.get(id);
      if (n) { n.spawn = now; n.spawnRgb = rgb; }
    }
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
  // The node's plain name is shown in the command prompt instead.
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
}

// currentNodeSnapshot finds the graph node the session is currently at, so
// callers can read its display Name and Description.
function currentNodeSnapshot(s) {
  const id = s && s.session && s.session.Location;
  return ((s.graph && s.graph.Nodes) || []).find((n) => n.ID === id) || null;
}

function renderLook(s) { lookEl.textContent = s.look || ""; }

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

// renderInspector fills the side-panel inspector with the hovered node's
// details, falling back to the current location when nothing is hovered. The
// footer answers "how do I get there?": you-are-here, click-to-travel for a
// reachable same-reality node, a command chip when a reality transition is
// required (requiredTransition), or no-route when it's otherwise unreachable.
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

// renderTrail renders the session's journey history (SessionSnapshot.History)
// as an ordered list, newest last. It re-renders only when the history changes
// (length + last entry) so it doesn't churn on every unrelated state refresh.
function renderTrail(s) {
  if (!trailEl) return;
  const hist = (s.session && s.session.History) || [];
  const sig = hist.length + "|" + (hist[hist.length - 1] || "");
  if (sig === trailEl.dataset.sig) return;
  trailEl.dataset.sig = sig;
  if (!hist.length) { trailEl.innerHTML = ""; return; }
  const items = hist.map((h) => `<li>${escapeHtml(h)}</li>`).join("");
  const label = hist.length === 1 ? "move" : "moves";
  trailEl.innerHTML =
    `<div class="trail-title">journey · ${hist.length} ${label}</div>` +
    `<ol class="trail-list">${items}</ol>`;
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

function draw() {
  const dpr = window.devicePixelRatio || 1;
  if (canvas.width !== canvas.clientWidth * dpr) {
    canvas.width = canvas.clientWidth * dpr;
    canvas.height = canvas.clientHeight * dpr;
  }
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  const cur = state && state.session ? state.session.Location : null;

  ctx.lineWidth = 1;
  for (const e of edges) {
    const a = nodes.get(e.From), b = nodes.get(e.To);
    if (!a || !b) continue;
    const pa = toScreen(a), pb = toScreen(b);
    // Each mode has its own hue + dash: solid blue for ordinary travel, a
    // distinct dashed colour for every reality transition.
    const st = modeStyle(e.Mode);
    ctx.strokeStyle = `rgba(${st.rgb},${st.dash.length ? 0.32 : 0.22})`;
    ctx.setLineDash(st.dash);
    ctx.beginPath();
    ctx.moveTo(pa.x, pa.y);
    ctx.lineTo(pb.x, pb.y);
    ctx.stroke();
  }
  ctx.setLineDash([]);

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
    ctx.fillStyle = isCur ? "#57e2a5" : colorFor(n);
    ctx.beginPath();
    ctx.arc(p.x, p.y, r, 0, Math.PI * 2);
    ctx.fill();
    ctx.fillStyle = isCur ? "#d7e0ff" : "#7a86b6";
    ctx.font = `${12 * Math.max(view.scale * p.persp, 0.7)}px ui-monospace, monospace`;
    // Labels grow long fast, so abbreviate them by default; reveal the full name
    // for the current location and whichever node the pointer is hovering.
    const showFull = isCur || n.id === hoveredId;
    ctx.fillText(showFull ? n.name : abbreviateLabel(n.name), p.x + r + 4, p.y + 4);
    // Depth badge: the nesting-depth number to the left of each nested node, so
    // depth is legible without rotating. Base (depth 0) nodes are left unbadged.
    if (n.depth > 0) {
      ctx.fillStyle = isCur ? "#d7e0ff" : "#5a6699";
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
    run("travel " + hit.id);
    return;
  }
  // A plain drag pans the map; Shift+drag orbits it in 3D. When starting an
  // orbit we capture the pivot — the world point under the cursor — so the drag
  // spins the view about that point rather than the canvas centre.
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
    if (hoveredId !== null) { hoveredId = null; renderInspector(); }
    return;
  }
  const hit = nodeAt(e.offsetX, e.offsetY);
  const id = hit ? hit.id : null;
  if (id !== hoveredId) { hoveredId = id; renderInspector(); }
  canvas.style.cursor = hit && hit.reachable ? "pointer" : "default";
});
canvas.addEventListener("mouseleave", () => {
  if (hoveredId !== null) { hoveredId = null; renderInspector(); }
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

document.querySelectorAll("button[data-cmd]").forEach((b) => {
  b.addEventListener("click", () => run(b.dataset.cmd));
});

// View-orientation buttons (client-side only, no server round-trip). "vertical"
// snaps to the depth-ladder view; "reset" restores the free three-quarter view.
// The running tick() redraws automatically.
document.getElementById("view-vertical").addEventListener("click", setVerticalView);
document.getElementById("view-reset").addEventListener("click", setDefaultView);

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
document.getElementById("time-go").addEventListener("click", () => {
  const v = document.getElementById("time-input").value;
  if (!v) return;
  const d = new Date(v);
  if (isNaN(d.getTime())) return;
  run("time " + d.toISOString().replace(/\.\d{3}Z$/, "Z"));
});

// Observer picker: send whatever perspective the user typed (free-form, e.g.
// Bat, Dog, Machine); the facade creates the branch on demand. Enter submits.
const observerInput = document.getElementById("observer-input");
function runObserve() {
  const v = observerInput.value.trim();
  if (v) run("observe " + v);
}
document.getElementById("observe-go").addEventListener("click", runObserve);
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
