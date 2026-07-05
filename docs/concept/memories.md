# Long-term memories

**T3 memories** are durable knowledge distilled **across sessions**. They are what agents usually want when you ask "what does rmb know about me?"

## Four categories

T1 atoms and T3 memories share one taxonomy:

| Category | URI pattern | Cardinality | What it holds |
|----------|-------------|-------------|---------------|
| **profile** | `rmb://profile` | Singleton | Stable facts about the user — location, role, health, core traits |
| **preferences** | `rmb://preferences/<slug>` | Many | Standing dispositions — tech choices, workflow habits, **AI behavior rules** |
| **entities** | `rmb://entities/<slug>` | Many | Third parties — people, projects, hosts, companies |
| **events** | `rmb://events/<slug>` | Many | Dated decisions and milestones — **append-only, immutable** |

### profile

One logical document for "who is this user." No slug — always `rmb://profile`.

> "Colin lives in Beijing." · "Allergic to peanuts."

### preferences

Recurring "prefers X / wants X / always Y" — including how the user wants AI assistants to behave.

> `rmb://preferences/go-services` — "Prefers single-binary Go services."  
> `rmb://preferences/answer-length` — "Prefers short answers."

`instruction`-style rules roll into **preferences**, not a separate category.

### entities

Named persistent things that are **not** the user.

> `rmb://entities/jenkins` — home directory, timer migration, …  
> `rmb://entities/yao-qiankun` — colleague metadata.

Slugs are **canonical names** for the thing itself (`jenkins`, not `fix-jenkins-timer`).

### events

Immutable milestones. Slugs are often date-prefixed.

> `rmb://events/2026-05-17-postgres-only-decision`

Workers **insert only** — no merge or in-place rewrite for events.

## How memories are created

The **T3 worker** (global, mutex):

1. Collect sessions with pending T3 status (or scheduled rollup tick)
2. Load **changed scenes** and route atoms by category + slug
3. LLM distills into `abstract` + `body` per target memory URI
4. **INSERT** new version row; supersede previous active row (`superseded_at`)
5. Never `UPDATE body` in place — versioning keeps audit trail

```text
scenes (session-local narrative)
    → route atoms by category/slug
    → distill per memory URI
    → versioned memories row
```

## Versioning

Multiple physical rows can share one logical URI. The **active** row is `superseded_at IS NULL`.

```bash
rmb cat rmb://profile          # latest active version
rmb meta rmb://profile         # version, timestamps, source_scene_uris
```

## Human corrections

Machine distillation can be wrong. Users attach **corrections** to a memory URI — durable overlays that **always beat** the distilled body.

```bash
rmb correction add rmb://entities/jenkins "Home is /var/lib/jenkins, not /opt"
```

See [Corrections](/guide/corrections).

## Priority

Atoms carry `priority` (0–100, or `-1` for "never drop"):

| Range | Meaning |
|-------|---------|
| 80–100 | Critical — health, taboos, core traits, strict rules |
| 50–79 | Ordinary |
| &lt; 50 | Weak signal — candidate to downweight |
| -1 | Sentinel — absolute behavior rules (use sparingly) |

## Recall

```bash
rmb search "jenkins home directory"    # memories + scenes
rmb find "postgres storage decision"   # memories only (vector)
rmb cat rmb://entities/jenkins
```

Trace provenance: `meta` → `source_scene_uris` → scene bodies → atom URIs → turns.

## Further reading

- [CLI for agents](/guide/cli-for-agents) — how agents should use recall
- [Design §6 taxonomy](/design/l0-l3#_6-memory-taxonomy-t1-and-t3-share-these)
- [Consolidation review (中文)](/zh/design/consolidation) — why append-first
