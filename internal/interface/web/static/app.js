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
  colorFor,
  project,
  depthAlpha,
  abbreviateLabel,
} from "./logic.js";

const canvas = document.getElementById("map");
const ctx = canvas.getContext("2d");
const axesEl = document.getElementById("axes");
const costEl = document.getElementById("cost-value");
const lookEl = document.getElementById("look");
const logEl = document.getElementById("log");
const promptEl = document.getElementById("prompt");
const cmdInput = document.getElementById("cmd");
const confirmEl = document.getElementById("confirm");

let state = null;
let edges = [];
const nodes = new Map(); // id -> {id, name, x, y, z, vx, vy, vz}
const view = { scale: 1, ox: 0, oy: 0, rotX: 0, rotY: 0 };
let logCount = 0;

async function api(path, body) {
  const opts = body
    ? { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }
    : {};
  const res = await fetch(path, opts);
  return res.json();
}

async function refresh() { apply(await api("/api/state")); }
async function run(command) { apply(await api("/api/execute", { command })); }

function apply(s) {
  const prev = state && state.session;
  state = s;
  syncNodes(s.graph);
  renderHUD(s.session);
  renderLook(s);
  renderLog(s);
  promptEl.textContent = s.prompt;
  renderConfirm(s);
  const mode = detectTransition(prev, s.session);
  if (mode) triggerEffect(mode);
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
  const seed = nodes.get(state && state.session ? state.session.Location : null);
  for (const n of graph.Nodes || []) {
    if (!nodes.has(n.ID)) {
      const a = Math.random() * Math.PI * 2;
      const r = 40 + Math.random() * 120;
      nodes.set(n.ID, {
        id: n.ID,
        name: n.Name || n.ID,
        quantum: n.Quantum,
        reachable: n.Reachable,
        x: (seed ? seed.x : 0) + Math.cos(a) * r,
        y: (seed ? seed.y : 0) + Math.sin(a) * r,
        z: (seed ? seed.z : 0) + (Math.random() - 0.5) * 160,
        vx: 0, vy: 0, vz: 0,
      });
    } else {
      const node = nodes.get(n.ID);
      node.name = n.Name || n.ID;
      node.quantum = n.Quantum;
      node.reachable = n.Reachable;
    }
  }
}

function badge(label, value, active) {
  const cls = active ? "badge active" : "badge";
  return `<span class="${cls}">${label} <b>${value}</b></span>`;
}

function renderHUD(sess) {
  if (!sess) return;
  const parts = [
    badge("loc", sess.Location, false),
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

// ── Force-directed layout ──────────────────────────────────────────────────

function tick() {
  const arr = [...nodes.values()];
  // Repulsion between every pair of nodes, now in three dimensions.
  for (let i = 0; i < arr.length; i++) {
    for (let j = i + 1; j < arr.length; j++) {
      const a = arr[i], b = arr[j];
      let dx = a.x - b.x, dy = a.y - b.y, dz = a.z - b.z;
      let d2 = dx * dx + dy * dy + dz * dz || 0.01;
      const f = 2600 / d2;
      const d = Math.sqrt(d2);
      const ux = dx / d, uy = dy / d, uz = dz / d;
      a.vx += ux * f; a.vy += uy * f; a.vz += uz * f;
      b.vx -= ux * f; b.vy -= uy * f; b.vz -= uz * f;
    }
  }
  // Springs along edges pull connected nodes toward a rest length.
  for (const e of edges) {
    const a = nodes.get(e.From), b = nodes.get(e.To);
    if (!a || !b) continue;
    const dx = b.x - a.x, dy = b.y - a.y, dz = b.z - a.z;
    const d = Math.sqrt(dx * dx + dy * dy + dz * dz) || 0.01;
    const f = (d - 110) * 0.02;
    const ux = dx / d, uy = dy / d, uz = dz / d;
    a.vx += ux * f; a.vy += uy * f; a.vz += uz * f;
    b.vx -= ux * f; b.vy -= uy * f; b.vz -= uz * f;
  }
  // Gentle pull to origin plus damping and integration.
  for (const n of arr) {
    n.vx += -n.x * 0.002; n.vy += -n.y * 0.002; n.vz += -n.z * 0.002;
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
// reports a change we queue a ripple that drawEffects paints from the current
// location, tinted with that transition's colour.
const effects = [];
const EFFECT_MS = 900;

function triggerEffect(mode) {
  effects.push({ mode, start: performance.now() });
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
    const r = (isCur ? 9 : 6) * view.scale * p.persp;
    // Fade nodes by depth so a busy map stays readable; the current location
    // always stays at full opacity so you never lose track of where you are.
    ctx.globalAlpha = isCur ? 1 : depthAlpha(p.depth, minDepth, maxDepth);
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
  }
  ctx.globalAlpha = 1;

  drawEffects(cur);
}

// drawEffects paints staggered expanding rings from the current location for
// each active transition, tinted with that transition's colour, then drops
// effects that have finished. Called every frame from draw() (tick drives it).
function drawEffects(curId) {
  if (!effects.length) return;
  const now = performance.now();
  const node = curId ? nodes.get(curId) : null;
  const origin = node ? toScreen(node) : { x: canvas.clientWidth / 2, y: canvas.clientHeight / 2 };
  for (let i = effects.length - 1; i >= 0; i--) {
    const t = (now - effects[i].start) / EFFECT_MS;
    if (t >= 1) { effects.splice(i, 1); continue; }
    const st = modeStyle(effects[i].mode);
    for (let k = 0; k < 3; k++) {
      const tt = t - k * 0.12;
      if (tt <= 0) continue;
      const radius = 12 + tt * 130 * view.scale;
      const alpha = (1 - tt) * 0.9;
      ctx.beginPath();
      ctx.strokeStyle = `rgba(${st.rgb},${alpha})`;
      ctx.lineWidth = 2.5 * (1 - tt) + 0.5;
      ctx.arc(origin.x, origin.y, radius, 0, Math.PI * 2);
      ctx.stroke();
    }
  }
  ctx.lineWidth = 1;
}

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
  // Shift+drag rotates the map; a plain drag pans it.
  dragging = { x: e.offsetX, y: e.offsetY, rotate: e.shiftKey };
});
// Hint clickability: show a pointer cursor over reachable nodes (the blue
// "travel here" ones), the default cursor otherwise. Skipped while dragging so
// panning/rotating keeps its own cursor.
canvas.addEventListener("mousemove", (e) => {
  if (dragging) { hoveredId = null; return; }
  const hit = nodeAt(e.offsetX, e.offsetY);
  hoveredId = hit ? hit.id : null;
  canvas.style.cursor = hit && hit.reachable ? "pointer" : "default";
});
canvas.addEventListener("mouseleave", () => { hoveredId = null; });
window.addEventListener("mousemove", (e) => {
  if (!dragging) return;
  if (dragging.rotate) {
    view.rotY += e.movementX * 0.01;
    view.rotX += e.movementY * 0.01;
  } else {
    view.ox += e.movementX;
    view.oy += e.movementY;
  }
});
window.addEventListener("mouseup", () => { dragging = null; });
canvas.addEventListener("wheel", (e) => {
  e.preventDefault();
  const factor = e.deltaY < 0 ? 1.1 : 0.9;
  view.scale = Math.min(3, Math.max(0.3, view.scale * factor));
}, { passive: false });

document.querySelectorAll("button[data-cmd]").forEach((b) => {
  b.addEventListener("click", () => run(b.dataset.cmd));
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
    TRANSITION_LEGEND.map(([mode, label]) => {
      const st = modeStyle(mode);
      return `<span class="legend-item"><i class="line" style="background:rgb(${st.rgb})"></i>${label}</span>`;
    }).join("");
})();

refresh();
requestAnimationFrame(tick);
