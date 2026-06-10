const API = "/api/v1/browse";
const PAGE_SIZE = 25;

const tabs = [
  { id: "guide", label: "Guide", path: null },
  { id: "overview", label: "Overview", path: "/overview" },
  { id: "sessions", label: "Sessions", path: "/sessions" },
  { id: "atoms", label: "Atoms", path: "/atoms" },
  { id: "scenes", label: "Scenes", path: "/scenes" },
  { id: "memories", label: "Memories", path: "/memories" },
  { id: "pipeline", label: "Pipeline", path: "/pipeline-state" },
  { id: "tasks", label: "Tasks", path: "/tasks" },
];

const state = {
  tab: "overview",
  overview: null,
  selectedKey: null,
  atomsFilter: { q: "", category: "all" },
  pages: {},
};

const $ = (sel) => document.querySelector(sel);
const panel = () => $("#panel");
const statusEl = () => $("#status");

function showStatus(msg) {
  const el = statusEl();
  if (!msg) {
    el.hidden = true;
    el.textContent = "";
    return;
  }
  el.hidden = false;
  el.textContent = msg;
}

async function api(path) {
  const res = await fetch(API + path);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(body.error || res.statusText);
  }
  return body;
}

function esc(s) {
  if (s == null) return "";
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function fmtTime(s) {
  if (!s) return "—";
  try {
    const d = new Date(s);
    return d.toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "numeric",
      minute: "2-digit",
    });
  } catch {
    return s;
  }
}

function fmtTimeShort(s) {
  if (!s) return "—";
  try {
    const d = new Date(s);
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  } catch {
    return s;
  }
}

function field(row, ...keys) {
  for (const k of keys) {
    if (row[k] != null && row[k] !== "") return row[k];
  }
  return null;
}

function sessionKeyFromURI(uri) {
  const m = /mypast:\/\/sessions\/([^/]+)/i.exec(uri || "");
  if (!m) return null;
  const key = m[1];
  return key.length > 10 ? key.slice(0, 8) + "…" : key;
}

function truncate(s, max) {
  if (!s) return "";
  const t = String(s).trim();
  if (t.length <= max) return t;
  return t.slice(0, max - 1) + "…";
}

function categoryBadge(cat) {
  const c = (cat || "default").toLowerCase();
  const cls = ["profile", "preferences", "entities", "events"].includes(c)
    ? `badge-${c}`
    : "badge-default";
  return `<span class="badge ${cls}">${esc(c)}</span>`;
}

function statusBadge(status) {
  const s = (status || "unknown").toLowerCase();
  const destructive = s === "failed";
  const cls = destructive ? "badge-destructive" : "badge-outline";
  return `<span class="badge ${cls} status-dot ${esc(s)}">${esc(s)}</span>`;
}

function outlineBadge(text) {
  return `<span class="badge badge-outline">${esc(text)}</span>`;
}

function pageHeader(title, description) {
  return `
    <div class="page-header">
      <h2>${esc(title)}</h2>
      ${description ? `<p>${description}</p>` : ""}
    </div>`;
}

function parseJSONL(raw) {
  if (!raw) return [];
  return raw
    .trim()
    .split("\n")
    .filter(Boolean)
    .map((line) => {
      try {
        return JSON.parse(line);
      } catch {
        return { role: "?", content: line, _parse_error: true };
      }
    });
}

function renderMessages(jsonl) {
  const msgs = parseJSONL(jsonl);
  if (!msgs.length) return '<p class="empty">No messages</p>';
  return msgs
    .map(
      (m) => `
    <div class="msg ${esc(m.role || "")}">
      <span class="role">${esc(m.role)}</span>
      ${esc(m.content || "")}
    </div>`
    )
    .join("");
}

/* —————————————————— drawer (slide-in detail) —————————————————— */

function openDrawer(title, bodyHTML) {
  const root = $("#drawer-root");
  $("#drawer-title").innerHTML = title || "";
  const body = $("#drawer-body");
  body.innerHTML = bodyHTML || "";
  body.scrollTop = 0;
  root.hidden = false;
  requestAnimationFrame(() => root.classList.add("open"));
}

function closeDrawer() {
  const root = $("#drawer-root");
  if (root.hidden) return;
  root.classList.remove("open");
  document
    .querySelectorAll(".data-table tbody tr.selected")
    .forEach((r) => r.classList.remove("selected"));
  state.selectedKey = null;
  setTimeout(() => {
    if (!root.classList.contains("open")) root.hidden = true;
  }, 250);
}

function wireDrawer() {
  $("#drawer-root")
    .querySelectorAll("[data-drawer-close]")
    .forEach((el) => el.addEventListener("click", closeDrawer));
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") closeDrawer();
  });
}

/* —————————————————— pagination —————————————————— */

function getPage(key) {
  return state.pages[key] || 1;
}

function setPage(key, n) {
  state.pages[key] = n;
}

function pageSlice(items, key) {
  const total = items.length;
  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const page = Math.min(Math.max(1, getPage(key)), pages);
  setPage(key, page);
  const start = (page - 1) * PAGE_SIZE;
  return { slice: items.slice(start, start + PAGE_SIZE), page, pages, total, start };
}

function renderPager(el, info, onGo) {
  const { page, pages, total, start, shown } = info;
  if (!total) {
    el.hidden = true;
    el.innerHTML = "";
    return;
  }
  el.hidden = false;
  const from = start + 1;
  const to = start + shown;
  const controls =
    pages > 1
      ? `
      <div class="pager-controls">
        <button type="button" class="btn btn-outline btn-sm" data-go="prev" ${page <= 1 ? "disabled" : ""}>Prev</button>
        <span class="pager-page">Page ${page} of ${pages}</span>
        <button type="button" class="btn btn-outline btn-sm" data-go="next" ${page >= pages ? "disabled" : ""}>Next</button>
      </div>`
      : "";
  el.innerHTML = `<span class="pager-info">${from}–${to} of ${total}</span>${controls}`;
  el.querySelector('[data-go="prev"]')?.addEventListener("click", () => onGo(page - 1));
  el.querySelector('[data-go="next"]')?.addEventListener("click", () => onGo(page + 1));
}

/* —————————————————— generic list panel —————————————————— */

// opts: title, description, thead, colCount, pageKey, getItems(), rowHTML(row, i),
//       emptyText, drawerFor(row) -> {title, body} | null, onRowClick(row),
//       toolbarHTML, wireToolbar(root, repaint)
function renderListPanel(p, opts) {
  const toolbar = opts.toolbarHTML
    ? `<div class="card-content" style="padding:1rem 1.5rem 0">${opts.toolbarHTML}</div>`
    : "";

  p.innerHTML = `
    ${pageHeader(opts.title, opts.description)}
    <div class="card data-table-card">
      ${toolbar}
      <div class="table-wrap">
        <table class="data-table">
          <thead>${opts.thead}</thead>
          <tbody id="list-tbody"></tbody>
        </table>
      </div>
      <div id="list-pager" class="pager" hidden></div>
    </div>`;

  const tbody = p.querySelector("#list-tbody");
  const pagerEl = p.querySelector("#list-pager");

  function paint() {
    const items = opts.getItems();
    const { slice, page, pages, total, start } = pageSlice(items, opts.pageKey);

    tbody.innerHTML = slice.length
      ? slice.map((row, i) => opts.rowHTML(row, i)).join("")
      : `<tr><td colspan="${opts.colCount}" class="empty" style="padding:2rem;text-align:center">${esc(
          opts.emptyText || "Nothing here yet"
        )}</td></tr>`;

    renderPager(pagerEl, { page, pages, total, start, shown: slice.length }, (next) => {
      setPage(opts.pageKey, next);
      paint();
    });

    tbody.querySelectorAll("tr[data-idx]").forEach((tr) => {
      tr.addEventListener("click", () => {
        const idx = Number(tr.dataset.idx);
        const row = slice[idx];
        tbody.querySelectorAll("tr.selected").forEach((r) => r.classList.remove("selected"));
        tr.classList.add("selected");
        if (opts.onRowClick) {
          opts.onRowClick(row);
          return;
        }
        const d = opts.drawerFor && opts.drawerFor(row);
        if (d) openDrawer(d.title, d.body);
      });
    });
  }

  paint();

  if (opts.wireToolbar) {
    opts.wireToolbar(p, () => {
      setPage(opts.pageKey, 1);
      paint();
    });
  }

  return paint;
}

/* —————————————————— nav —————————————————— */

function renderNav() {
  const counts = state.overview?.counts || {};
  $("#nav").innerHTML = `
    <div class="nav-group-label">Browse</div>
    ${tabs
      .map((t) => {
        const key =
          t.id === "pipeline"
            ? "pipeline_states"
            : t.id === "overview" || t.id === "guide"
              ? null
              : t.id;
        const n = key ? counts[key] ?? "—" : "";
        return `<button type="button" data-tab="${t.id}" class="${state.tab === t.id ? "active" : ""}">
        <span>${t.label}</span>
        ${n !== "" ? `<span class="count">${n}</span>` : ""}
      </button>`;
      })
      .join("")}`;

  $("#nav").querySelectorAll("button").forEach((btn) => {
    btn.addEventListener("click", () => switchTab(btn.dataset.tab));
  });
}

async function switchTab(id) {
  state.tab = id;
  state.selectedKey = null;
  renderNav();
  showStatus("");
  try {
    await renderPanel();
  } catch (err) {
    showStatus(err.message);
  }
}

async function goToSession(sessionKey) {
  state.tab = "sessions";
  state.selectedKey = sessionKey;
  renderNav();
  showStatus("");
  try {
    await renderPanel();
  } catch (err) {
    showStatus(err.message);
  }
}

async function loadOverviewCounts() {
  state.overview = await api("/overview");
  renderNav();
}

/* —————————————————— guide —————————————————— */

function renderGuideHTML() {
  return `
    ${pageHeader("Guide", "How capture and distillation layers connect.")}
    <div class="guide-flow" aria-label="Distillation pyramid">
      <div class="guide-step"><span class="guide-tier">session</span><strong>sessions</strong><span class="guide-sub">one agent conversation</span></div>
      <div class="guide-arrow">↓ hook upload</div>
      <div class="guide-step"><span class="guide-tier">T0</span><strong>session_turns</strong><span class="guide-sub">user + assistant pair</span></div>
      <div class="guide-arrow">↓ T1 extract</div>
      <div class="guide-step"><span class="guide-tier">T1</span><strong>atoms</strong><span class="guide-sub">typed facts</span></div>
      <div class="guide-arrow">↓ T2</div>
      <div class="guide-step"><span class="guide-tier">T2</span><strong>scenes</strong><span class="guide-sub">what we were doing</span></div>
      <div class="guide-arrow">↓ T3</div>
      <div class="guide-step"><span class="guide-tier">T3</span><strong>memories</strong><span class="guide-sub">profile · preferences · entities · events</span></div>
    </div>
    <div class="section">
      <h3>Memory categories</h3>
      <ul class="guide-list">
        <li><code>profile</code> — singleton identity</li>
        <li><code>preferences</code> — habits and AI behavior rules</li>
        <li><code>entities</code> — people, companies, projects</li>
        <li><code>events</code> — dated milestones (append-only)</li>
      </ul>
    </div>`;
}

/* —————————————————— atoms —————————————————— */

function filterAtoms(items) {
  const { q, category } = state.atomsFilter;
  const needle = q.trim().toLowerCase();
  return items.filter((row) => {
    const cat = (field(row, "category", "Category") || "").toLowerCase();
    if (category !== "all" && cat !== category) return false;
    if (!needle) return true;
    const content = (field(row, "content", "Content") || "").toLowerCase();
    const slug = (field(row, "slug", "Slug") || "").toLowerCase();
    const scene = (field(row, "scene_name", "SceneName") || "").toLowerCase();
    const uri = (field(row, "uri", "URI") || "").toLowerCase();
    return (
      content.includes(needle) ||
      slug.includes(needle) ||
      scene.includes(needle) ||
      uri.includes(needle)
    );
  });
}

function atomRow(row, i) {
  const content = field(row, "content", "Content") || "—";
  const category = field(row, "category", "Category");
  const scene = field(row, "scene_name", "SceneName");
  const slug = field(row, "slug", "Slug");
  const priority = field(row, "priority", "Priority") ?? "—";
  const uri = field(row, "uri", "URI");
  const created = field(row, "created_at", "CreatedAt");
  const sessionShort = sessionKeyFromURI(uri) || "—";
  const topic = [scene, slug].filter(Boolean).join(" · ") || "—";
  const priClass = Number(priority) >= 70 ? " high" : "";

  return `
    <tr data-idx="${i}">
      <td><div class="cell-primary cell-clamp-2">${esc(content)}</div></td>
      <td>${categoryBadge(category)}</td>
      <td><div class="cell-secondary">${esc(topic)}</div></td>
      <td><span class="cell-mono" title="${esc(uri)}">${esc(sessionShort)}</span></td>
      <td>
        <span class="priority-pill${priClass}">${esc(priority)}</span>
        <div class="cell-meta">${fmtTimeShort(created)}</div>
      </td>
    </tr>`;
}

function atomDrawer(row) {
  const content = field(row, "content", "Content");
  const uri = field(row, "uri", "URI");
  const category = field(row, "category", "Category");
  const scene = field(row, "scene_name", "SceneName");
  const slug = field(row, "slug", "Slug");
  const priority = field(row, "priority", "Priority");
  const created = field(row, "created_at", "CreatedAt");

  return {
    title: esc(scene || slug || "Atom"),
    body: `
      <div class="detail-badges">
        ${categoryBadge(category)}
        ${scene ? outlineBadge(scene) : ""}
        ${slug ? outlineBadge(slug) : ""}
        ${outlineBadge("P" + (priority ?? "—"))}
      </div>
      <p class="detail-body">${esc(content)}</p>
      <p class="detail-uri">${esc(uri)}</p>
      <p class="cell-meta" style="margin-top:0.5rem">${fmtTime(created)}</p>`,
  };
}

async function renderAtomsPanel(p) {
  const { items: all } = await api("/atoms");
  renderListPanel(p, {
    title: "Atoms",
    description: "Facts extracted from chat turns (T1).",
    colCount: 5,
    pageKey: "atoms",
    thead: `<tr>
      <th style="min-width:14rem">Fact</th>
      <th>Category</th>
      <th>Topic</th>
      <th>Session</th>
      <th>Priority · When</th>
    </tr>`,
    getItems: () => filterAtoms(all),
    rowHTML: atomRow,
    drawerFor: atomDrawer,
    emptyText: "No atoms match your filters",
    toolbarHTML: `
      <div class="toolbar">
        <input type="search" class="input" id="atoms-search" placeholder="Search facts…" value="${esc(state.atomsFilter.q)}" />
        <select class="select" id="atoms-category" aria-label="Category filter">
          <option value="all"${state.atomsFilter.category === "all" ? " selected" : ""}>All categories</option>
          <option value="profile"${state.atomsFilter.category === "profile" ? " selected" : ""}>profile</option>
          <option value="preferences"${state.atomsFilter.category === "preferences" ? " selected" : ""}>preferences</option>
          <option value="entities"${state.atomsFilter.category === "entities" ? " selected" : ""}>entities</option>
          <option value="events"${state.atomsFilter.category === "events" ? " selected" : ""}>events</option>
        </select>
      </div>`,
    wireToolbar: (root, repaint) => {
      const searchEl = root.querySelector("#atoms-search");
      const catEl = root.querySelector("#atoms-category");
      const apply = () => {
        state.atomsFilter.q = searchEl.value;
        state.atomsFilter.category = catEl.value;
        repaint();
      };
      searchEl.addEventListener("input", apply);
      catEl.addEventListener("change", apply);
    },
  });
}

/* —————————————————— memories —————————————————— */

function memoryRow(row, i) {
  const abstract = field(row, "abstract", "Abstract");
  const body = field(row, "body", "Body");
  const primary = abstract || truncate(body, 120) || "—";
  const secondary = abstract && body ? truncate(body, 80) : "";
  return `
    <tr data-idx="${i}">
      <td>
        <div class="cell-primary cell-clamp-2">${esc(primary)}</div>
        ${secondary ? `<div class="cell-secondary">${esc(secondary)}</div>` : ""}
      </td>
      <td>${categoryBadge(field(row, "category", "Category"))}</td>
      <td class="cell-mono">${esc(field(row, "slug", "Slug") || "—")}</td>
      <td class="cell-meta">${esc(field(row, "version", "Version") ?? "—")}</td>
      <td class="cell-meta">${fmtTimeShort(field(row, "updated_at", "UpdatedAt"))}</td>
    </tr>`;
}

function memoryDrawer(row) {
  const abstract = field(row, "abstract", "Abstract");
  const body = field(row, "body", "Body");
  const slug = field(row, "slug", "Slug");
  const category = field(row, "category", "Category");
  const version = field(row, "version", "Version");
  const updated = field(row, "updated_at", "UpdatedAt");
  return {
    title: esc(slug || category || "Memory"),
    body: `
      <div class="detail-badges">
        ${categoryBadge(category)}
        ${slug ? outlineBadge(slug) : ""}
        ${version != null ? outlineBadge("v" + version) : ""}
      </div>
      ${abstract ? `<p class="detail-body" style="font-weight:500">${esc(abstract)}</p>` : ""}
      ${body ? `<p class="detail-body">${esc(body)}</p>` : ""}
      <p class="cell-meta" style="margin-top:0.5rem">${fmtTime(updated)}</p>`,
  };
}

/* —————————————————— scenes —————————————————— */

function sceneRow(row, i) {
  const abstract = field(row, "abstract", "Abstract");
  const body = field(row, "body", "Body");
  const uri = field(row, "uri", "URI");
  return `
    <tr data-idx="${i}">
      <td>
        <div class="cell-primary cell-clamp-2">${esc(abstract || truncate(body, 100) || "—")}</div>
        <div class="cell-secondary cell-mono">${esc(truncate(uri, 48))}</div>
      </td>
      <td class="cell-mono">${esc(sessionKeyFromURI(uri) || "—")}</td>
      <td class="cell-meta">${fmtTimeShort(field(row, "updated_at", "UpdatedAt"))}</td>
    </tr>`;
}

function sceneDrawer(row) {
  const abstract = field(row, "abstract", "Abstract");
  const body = field(row, "body", "Body");
  const uri = field(row, "uri", "URI");
  const updated = field(row, "updated_at", "UpdatedAt");
  return {
    title: esc(truncate(abstract, 60) || "Scene"),
    body: `
      ${abstract ? `<p class="detail-body" style="font-weight:500">${esc(abstract)}</p>` : ""}
      ${body ? `<p class="detail-body">${esc(body)}</p>` : ""}
      <p class="detail-uri">${esc(uri)}</p>
      <p class="cell-meta" style="margin-top:0.5rem">${fmtTime(updated)}</p>`,
  };
}

/* —————————————————— tasks —————————————————— */

function taskRow(row, i) {
  const kind = field(row, "kind", "Kind");
  const status = field(row, "status", "Status");
  const err = field(row, "error", "Error");
  const sid = field(row, "session_id", "SessionID");
  const sessionShort = sid ? String(sid).slice(0, 8) + "…" : "—";
  return `
    <tr data-idx="${i}">
      <td>
        <div class="cell-primary">${esc(kind)}</div>
        ${err ? `<div class="cell-secondary cell-clamp-2">${esc(err)}</div>` : ""}
      </td>
      <td>${statusBadge(status)}</td>
      <td class="cell-mono">${esc(sessionShort)}</td>
      <td class="cell-meta">${esc(field(row, "progress", "Progress") ?? 0)}%</td>
      <td class="cell-meta">${fmtTimeShort(field(row, "created_at", "CreatedAt"))}</td>
    </tr>`;
}

function taskDrawer(row) {
  const kind = field(row, "kind", "Kind");
  const status = field(row, "status", "Status");
  const err = field(row, "error", "Error");
  const sid = field(row, "session_id", "SessionID");
  const progress = field(row, "progress", "Progress") ?? 0;
  const created = field(row, "created_at", "CreatedAt");
  return {
    title: esc(kind || "Task"),
    body: `
      <div class="detail-badges">
        ${statusBadge(status)}
        ${outlineBadge(progress + "%")}
      </div>
      ${sid ? `<p class="detail-uri">session ${esc(sid)}</p>` : ""}
      ${err ? `<p class="detail-body">${esc(err)}</p>` : `<p class="empty">No error reported.</p>`}
      <p class="cell-meta" style="margin-top:0.5rem">${fmtTime(created)}</p>`,
  };
}

/* —————————————————— pipeline —————————————————— */

function pipelineRow(row, i) {
  const sessionKey = field(row, "session_key", "SessionKey");
  const label = sessionKey ? sessionKey.slice(0, 8) + "…" : "—";
  return `
    <tr data-idx="${i}">
      <td class="cell-mono">${esc(label)}</td>
      <td>${statusBadge(field(row, "t1_status", "T1Status"))}</td>
      <td>${statusBadge(field(row, "t2_status", "T2Status"))}</td>
      <td>${statusBadge(field(row, "t3_status", "T3Status"))}</td>
      <td class="cell-meta">${esc(field(row, "warmup_threshold", "WarmupThreshold") ?? "—")}</td>
    </tr>`;
}

/* —————————————————— sessions —————————————————— */

function sessionRow(s, i) {
  return `
    <tr data-idx="${i}" data-key="${esc(s.session_key)}">
      <td>
        <div class="cell-primary">${esc(s.title || "Untitled session")}</div>
        <div class="cell-secondary cell-mono">${esc(s.session_key.slice(0, 8))}…</div>
      </td>
      <td class="cell-meta">${s.turn_count}</td>
      <td>${statusBadge(s.status)}</td>
      <td class="cell-meta">${fmtTimeShort(s.updated_at)}</td>
    </tr>`;
}

async function openSession(sessionKey) {
  state.selectedKey = sessionKey;
  showStatus("");
  const data = await api("/sessions/" + encodeURIComponent(sessionKey));
  const s = data.session;

  document.querySelectorAll("#list-tbody tr[data-key]").forEach((tr) => {
    tr.classList.toggle("selected", tr.dataset.key === sessionKey);
  });

  const ps = data.pipeline_state;
  const pipelineHTML = ps
    ? `<div class="pipeline-grid">
        <div><span>T1</span>${statusBadge(ps.t1_status)}</div>
        <div><span>T2</span>${statusBadge(ps.t2_status)}</div>
        <div><span>T3</span>${statusBadge(ps.t3_status)}</div>
      </div>
      <p class="cell-meta" style="margin-top:0.5rem">warmup_threshold: ${esc(ps.warmup_threshold)}</p>`
    : '<p class="empty">No pipeline state</p>';

  const atomsTable =
    data.atoms.length > 0
      ? `<div class="table-wrap" style="margin-top:0.75rem">
          <table class="data-table">
            <thead><tr><th>Fact</th><th>Category</th><th>Topic</th></tr></thead>
            <tbody>${data.atoms
              .map((a) => {
                const topic = [field(a, "scene_name", "SceneName"), field(a, "slug", "Slug")]
                  .filter(Boolean)
                  .join(" · ");
                return `<tr>
                  <td><div class="cell-primary cell-clamp-2">${esc(field(a, "content", "Content"))}</div></td>
                  <td>${categoryBadge(field(a, "category", "Category"))}</td>
                  <td class="cell-secondary">${esc(topic || "—")}</td>
                </tr>`;
              })
              .join("")}</tbody>
          </table>
        </div>`
      : '<p class="empty">No atoms yet</p>';

  const scenesTable =
    data.scenes.length > 0
      ? `<div class="table-wrap" style="margin-top:0.75rem">
          <table class="data-table">
            <thead><tr><th>Scene</th><th>Summary</th></tr></thead>
            <tbody>${data.scenes
              .map((sc) => {
                const name = field(sc, "display_name", "DisplayName") || "Scene";
                const abstract = field(sc, "abstract", "Abstract");
                const body = field(sc, "body", "Body");
                return `<tr>
                  <td>
                    <div class="cell-primary">${esc(name)}</div>
                    <div class="cell-secondary cell-mono">${esc(truncate(field(sc, "uri", "URI"), 48))}</div>
                  </td>
                  <td><div class="cell-primary cell-clamp-2">${esc(abstract || truncate(body, 160) || "—")}</div></td>
                </tr>`;
              })
              .join("")}</tbody>
          </table>
        </div>`
      : '<p class="empty">No scenes yet</p>';

  const turnsHTML = data.turns
    .map(
      (t) => `
      <details class="turn">
        <summary>#${t.turn_index} · ${esc(t.turn_status)} · ${fmtTime(t.created_at)}</summary>
        <div class="messages">${renderMessages(t.messages_jsonl)}</div>
      </details>`
    )
    .join("");

  openDrawer(
    esc(s.title || s.session_key),
    `
      <p class="detail-uri">${esc(s.uri)}</p>
      <h3 class="drawer-section">Pipeline</h3>
      ${pipelineHTML}
      <h3 class="drawer-section">Turns (${data.turns.length})</h3>
      ${turnsHTML || '<p class="empty">No turns yet</p>'}
      <h3 class="drawer-section">Atoms (${data.atoms.length})</h3>
      ${atomsTable}
      <h3 class="drawer-section">Scenes (${data.scenes.length})</h3>
      ${scenesTable}`
  );
}

/* —————————————————— panel router —————————————————— */

async function renderPanel() {
  const p = panel();
  const tab = tabs.find((t) => t.id === state.tab);

  if (state.tab === "guide") {
    p.innerHTML = renderGuideHTML();
    return;
  }

  if (state.tab === "overview") {
    if (!state.overview) await loadOverviewCounts();
    const c = state.overview.counts;
    p.innerHTML = `
      ${pageHeader("Overview", "Row counts across the pipeline.")}
      <div class="stats-grid">
        ${Object.entries(c)
          .map(
            ([k, v]) => `
          <div class="stat-card">
            <div class="label">${esc(k.replace(/_/g, " "))}</div>
            <div class="value">${v}</div>
          </div>`
          )
          .join("")}
      </div>
      <p class="empty">
        <button type="button" class="link-btn" data-goto-guide>Read the Guide</button>
        or open <strong>Atoms</strong> to browse extracted facts.
      </p>`;
    p.querySelector("[data-goto-guide]")?.addEventListener("click", () => switchTab("guide"));
    return;
  }

  if (state.tab === "atoms") {
    await renderAtomsPanel(p);
    return;
  }

  if (state.tab === "sessions") {
    const { items } = await api("/sessions");
    renderListPanel(p, {
      title: "Sessions",
      description: "Agent conversations and captured turns.",
      colCount: 4,
      pageKey: "sessions",
      thead: `<tr><th>Session</th><th>Turns</th><th>Status</th><th>Updated</th></tr>`,
      getItems: () => items,
      rowHTML: sessionRow,
      onRowClick: (s) => openSession(s.session_key),
      emptyText: "No sessions yet",
    });
    if (state.selectedKey) await openSession(state.selectedKey);
    return;
  }

  if (!tab?.path) return;

  const { items } = await api(tab.path);

  if (state.tab === "memories") {
    renderListPanel(p, {
      title: "Memories",
      description: "Long-term knowledge (T3).",
      colCount: 5,
      pageKey: "memories",
      thead: `<tr><th>Memory</th><th>Category</th><th>Slug</th><th>Version</th><th>Updated</th></tr>`,
      getItems: () => items,
      rowHTML: memoryRow,
      drawerFor: memoryDrawer,
      emptyText: "No memories yet",
    });
    return;
  }

  if (state.tab === "scenes") {
    renderListPanel(p, {
      title: "Scenes",
      description: "Session-level summaries (T2).",
      colCount: 3,
      pageKey: "scenes",
      thead: `<tr><th>Scene</th><th>Session</th><th>Updated</th></tr>`,
      getItems: () => items,
      rowHTML: sceneRow,
      drawerFor: sceneDrawer,
      emptyText: "No scenes yet",
    });
    return;
  }

  if (state.tab === "tasks") {
    renderListPanel(p, {
      title: "Tasks",
      description: "Async worker runs.",
      colCount: 5,
      pageKey: "tasks",
      thead: `<tr><th>Job</th><th>Status</th><th>Session</th><th>Progress</th><th>Created</th></tr>`,
      getItems: () => items,
      rowHTML: taskRow,
      drawerFor: taskDrawer,
      emptyText: "No tasks yet",
    });
    return;
  }

  if (state.tab === "pipeline") {
    renderListPanel(p, {
      title: "Pipeline",
      description: "Per-session worker state. Click a row to open the session.",
      colCount: 5,
      pageKey: "pipeline",
      thead: `<tr><th>Session</th><th>T1</th><th>T2</th><th>T3</th><th>Warmup</th></tr>`,
      getItems: () => items,
      rowHTML: pipelineRow,
      onRowClick: (row) => goToSession(field(row, "session_key", "SessionKey")),
      emptyText: "No pipeline state",
    });
    return;
  }
}

async function refresh() {
  showStatus("");
  try {
    await loadOverviewCounts();
    await renderPanel();
  } catch (err) {
    showStatus(err.message);
  }
}

$("#refresh-btn").addEventListener("click", refresh);
wireDrawer();

if (!sessionStorage.getItem("mypast-ui-seen")) {
  state.tab = "guide";
  sessionStorage.setItem("mypast-ui-seen", "1");
}

refresh();
