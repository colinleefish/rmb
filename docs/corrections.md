# Corrections — a human authority layer over distilled memory

> Status: design (not yet implemented). Durable, hand-authored statements that
> outrank LLM-distilled memory, survive re-distillation, and are guaranteed to
> reach the agent at recall time — so the agent self-corrects every time.
>
> This is the "sparse explicit human merge" escape hatch that `design-l0-l4.md`
> §6.1 reserves: workers never edit memory in place; humans can.

## Problem

T1–T3 are LLM-derived and occasionally wrong. Two failure shapes seen in
practice:

- **Bad fact** — the distiller stamped `10.9.114.160` onto the Aliyun RDS
  `rmb://entities/starlink-dev-all-in-one`, but that IP is the UCloud source
  VM `…-uc`, a different machine.
- **Over-merge / over-split** — two similar-named things collapse into one slug
  (or one thing fragments into many).

We want to state the correction **once** and never see the mistake again, without
hand-editing derived rows (the next T3 rollup would overwrite the edit — the exact
churn the append-first design avoids).

## Principle: a correction is an append-only fact, not an edit

A correction is its own immutable, authored record that the system **applies** —
it never mutates the derived memory. The LLM-derived memory stays derived; the
human statement stays authoritative; neither clobbers the other. This keeps the
discipline the rest of the pyramid already follows (append-first, versioned,
URI-addressable) and makes corrections immune to re-distillation.

## Data model — `corrections`

A sibling table to `memories`, same discipline (append-only,
URI-addressable as `rmb://corrections/<uuid>`) — but **not** keyed/versioned by
target the way memories are keyed by `(category, slug)`; see Multiplicity &
ordering.

| Column | Notes |
|--------|-------|
| `id` | uuid, append-only; never `UPDATE` |
| `author` | `human` (vs implicit `worker` for derived rows) |
| `target_uris` | `text[]` — the memories/entities/scenes this patches (1..N, any tier); **required, never empty** — a correction must correct something |
| `statement` | human text, e.g. "10.9.114.160 is the -uc source VM, not the RDS." |
| `superseded_at` | per-correction identity, not per-target: set only when *this specific* correction is explicitly retracted/replaced by its URI. Writing another correction on the same memory does **not** supersede it (see Multiplicity & ordering). |
| `created_at` | |

A correction is a pure content patch: one kind, one statement, one or more
targets. (The earlier `kind`/`payload` columns and the `split`/`alias` kinds are
gone — entity resolution, including `alias`, is its own dedicated mechanism
outside the correction umbrella. See "Entity resolution lives elsewhere".)

No embedding column: every correction targets something, so it is reached purely
via the `target_uris` join (no vectors, no FTS over correction text). Corrections
are never free-floating — there is no target-less/global correction.

## Targeting — one correction, many targets

A correction is **not** limited to a single memory. Targeting is necessarily
one-to-many:

- **Cross-cutting facts:** "10.9.114.160 is the `-uc` VM, not the RDS" touches
  both `rmb://entities/starlink-dev-all-in-one` and
  `rmb://entities/starlink-dev-all-in-one-uc`.
- **Any tier ("or assets"):** every artifact is URI-addressable, so a correction
  can target a memory, a scene, or a session. The overlay matches the returned
  object's URI regardless of tier.

Resolution — a memory `M` receives correction `C` if:

```
M.uri = ANY(C.target_uris)        -- explicit pin (GIN-indexed)
```

**Rule of thumb: correct the thing, not the row.** Prefer the *logical* URI
(`rmb://entities/<slug>`), which is stable across re-distillation, so the
correction applies to every future version automatically. Scene URIs (UUIDv5)
are stable too; raw atom URIs churn on re-extraction, so pinning a correction to
a specific atom is fragile — avoid it.

## Multiplicity & ordering

Corrections are **additive**, not last-write-wins. A memory can carry many active
corrections at once, and they are not keyed by their target — so writing a second
correction on the same memory **adds** to it; it never silently replaces the first.

- **All active corrections surface together.** Fetching a memory returns every
  active correction that targets it, as a list — not just the latest.
- **Ordered newest-first.** The list is sorted by `created_at` descending, so the
  most recent correction is shown (and weighted) first.
- **Newest wins on conflict.** If two corrections disagree about the same fact,
  the most recent active one takes precedence (consistent with the pyramid's
  "resolve contradictions in favour of the most recent"). Non-conflicting
  corrections simply all apply.
- **Removal is explicit.** A correction goes away only when *that specific
  correction* (by its URI) is retracted/replaced — which sets its `superseded_at`.
  There is no automatic per-target supersession.

Worked example — memory `rmb://entities/starlink-dev-all-in-one`:

```
day 1  fix … "10.9.114.160 is the -uc source VM, not this RDS."   -> A
day 2  fix … "This RDS is 100GB; expand to >=200GB."              -> B

fetch rmb://entities/starlink-dev-all-in-one
  ⚑ CORRECTION (human, day 2): RDS is 100GB; expand to >=200GB     # B, newest first
  ⚑ CORRECTION (human, day 1): 10.9.114.160 is the -uc source VM   # A, still active
```

B does **not** cancel A — both are live. If on day 3 you find A itself is wrong,
you explicitly retract A (by its correction URI) and write A2; only then is A
superseded.

## Two enforcement points (layered)

**1. Read-time overlay — the guarantee.** When recall returns a memory, attach
any active correction that targets its URI (see Targeting above) via the
`target_uris` join — no vectors involved. Result:

```
rmb://entities/starlink-dev-all-in-one
  body: …Internal IP: 10.9.114.160…
  ⚑ CORRECTION (human, 2026-06-10): 10.9.114.160 is the -uc source VM, not the RDS.
```

Unconditional: works even if distillation never runs again, and an LLM cannot
lose it. This is what makes the fix "always" honored.

**2. Write-time injection — the cleanup.** When T3 re-distills a bucket that has
corrections, feed them into the distill prompt as **authoritative overrides**
("these human corrections take precedence over any extracted fact"). The
regenerated body comes out clean, improving snippets/embeddings over time.

Overlay is the safety net; injection is the polish. Overlay is required;
injection is a quality bonus and can come later.

## One kind: a content overlay

A correction overrides/annotates a specific memory; it may be positive ("she
works at a bank") or negative ("she does NOT work at Huawei"). That is the whole
feature — there is no `kind` column.

There is no `forget`. Deliberate forgetting is not a human action rmb
supports: a wrong fact is a negative correction, and disuse is handled by passive
usage-based decay (`docs/ebbinghaus-recall.md`). See `docs/forget-rationale.md`
for the full reasoning.

A correction always targets concrete memories — there is no target-less "assert a
global fact". A correction with nothing to correct is a contradiction; durable
facts that have no memory to attach to belong in the pipeline (or a future
human-authored memory), not here.

### Entity resolution lives elsewhere

The old `split`/`alias` kinds are **not** part of corrections. Entity resolution
(slug drift, "these two slugs are the same", "this slug is really two things")
is its own dedicated mechanism — `alias` will be its own thing, outside the
correction umbrella. Keeping corrections to a single content-patch shape is the
point of this unification.

## Precedence & the agent contract

Rule (to add to `cli-agent-guide.md`), two levels of precedence:

1. **Human over machine:** an active human correction outranks LLM-derived memory.
2. **Newest human over older human:** when corrections conflict, the most recent
   active one wins (all non-conflicting corrections still apply — see Multiplicity
   & ordering).

On conflict the agent states the corrected fact and may note "(corrected from
prior memory)". The agent already recalls before answering, and recall now
carries the overlay, so correctness is enforced at the answer boundary.

## Lifecycle

- Corrections are append-only. They accumulate (see Multiplicity & ordering); a
  new correction does **not** supersede earlier ones just for sharing a target.
- Supersession is **per correction identity**: to remove or replace a specific
  correction, retract it by its own URI (sets `superseded_at`), preserving the
  human-correction audit trail. Deleting is never required.
- Corrections live in their own table, so the delete-and-rebuild T3 flow never
  touches them; after any rebuild the overlay re-applies and the next distill
  re-injects them. They are immune to derived-data churn because they are not
  derived.

## CLI surface

```
rmb correction add <uri> [<uri>...] "statement"
rmb correction rm  <correction-uri>     # retire a specific correction
rmb correction ls  [<target-uri>]       # list active corrections
rmb meta <uri>                          # also lists corrections attached to a memory
```

Naming: the concept is **correction** everywhere — the table (`corrections`),
the model (`Correction`), the URI (`rmb://corrections/<uuid>`), the route
(`/api/v1/corrections`), the `rmb correction` command, and the overlay label
`CORRECTION`. ("Assertion" was the old internal genus name; it is gone.)

The CLI is a pure API client; writing a correction is a privileged op, so the
HTTP path requires auth.

## Scope (delivered)

- `corrections` table + migration.
- Single content-overlay shape (no `kind`).
- **Read-time overlay** wired into `search` / `cat` / `meta`.
- Precedence rule in `cli-agent-guide.md`.
- **Write-time distill injection**: active corrections are fed into T3
  as authoritative input, so the regenerated memory body reflects them and
  becomes searchable. Provenance is tracked in `memories.source_correction_uris`;
  the gate re-distills a bucket when its active correction set changes; creating
  or retracting a correction wakes T3 for the targeted memory's sessions. Events
  stay immutable (overlay only — no body injection).

Delivers "I say it once, the agent honors it forever," and the body itself is
the merged truth (orthogonal corrections are combined by the distiller).

## Deferred

- Entity resolution: `alias` (and catalog-aware slugging / slug-drift fixes) as
  its own dedicated mechanism, separate from corrections — see the drift
  discussion in the project review.
- Scope-keying (work / personal / project) for corrections.

## Document map

| Doc | Use when |
|-----|----------|
| This file | "How do human corrections override memory?" |
| [`design-l0-l4.md`](./design-l0-l4.md) §6.1 | Why append-first / explicit-merge-only |
| [`project-review.md`](./project-review.md) | Entity-resolution / slug-drift context |
| [`cli-agent-guide.md`](./cli-agent-guide.md) | The agent's recall + precedence rules |
