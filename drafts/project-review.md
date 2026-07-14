# Project review: rmb vs TDAI and OpenViking

> Status: review note (2026-06). An end-to-end assessment of rmb and a
> comparison with the two reference systems it drew from: TencentDB Agent Memory
> (TDAI) and OpenViking.
>
> **Caveat:** the reference projects are not checked into this repo (`tmp/` is
> local-only), so the TDAI/OpenViking columns draw on rmb's own design-doc
> characterization (`design-l0-l4.md` §3, §13) plus general knowledge, not their
> source. Anything inferred is marked as such. The rmb assessment is grounded
> in the current code.

## What rmb is

A personal, cross-session memory layer for AI-agent conversations. Tool-agnostic
capture (Cursor / Claude Code hooks) feeds a four-tier distillation pyramid in
Postgres, exposed through hybrid recall as a CLI and HTTP API.

```
T0 turns ──► T1 atoms ──► T2 scenes ──► T3 memories ──► embeddings ──► find/search
(raw)        (facts)      (per-session) (cross-session)  (pgvector)    (vector+FTS+RRF)
```

## Current state

| Layer | Status |
|-------|--------|
| Capture (dual hooks) | Live |
| T1 / T2 / T3 workers | Live, throttled, transient-safe |
| Embeddings | 100% coverage |
| Recall (`find` / `search` / `cat` / `tree` / `meta`) | Live, dual-mode (local DB / remote API) |
| Eval | 8/8, hybrid-stack |
| Ops | Agent-driven `make ci` / `deploy`, prod healthy |

Recently fixed: T2 unknown-URI tolerance, T3 poison-bucket isolation,
version-churn provenance gate + deterministic distill, T1 routing prompt
rewrite. Still open: M1 auth-env footgun, M2 unused FTS index, M5 slug-less atom
drop, historical profile pollution (needs re-extraction), and the Ebbinghaus
usage-decay (planned, not built — see `ebbinghaus-recall.md`).

## Architecture comparison

| Dimension | rmb | TDAI (Tencent) | OpenViking |
|-----------|--------|----------------|------------|
| Tiers | T0→T3 (4) | L0→L3 layering (origin of the pyramid) | per-node L0/L1/L2 facets + dir tree |
| Capture | Tool-agnostic hooks, no SDK in agent | In-runtime / SDK-coupled (inferred) | Session API / SDK |
| Storage | Single Go binary + Postgres (pgvector) | TS service (inferred) | TS service + tree store |
| Extraction | 4-category atoms, in-call scene segmentation (from TDAI) | L1 atom extraction (the model rmb copied) | 8-category user model |
| Facets | 2 (abstract / body) — dropped the middle "overview" | — | 3 (abstract / overview / detail) |
| Addressing | `rmb://` URI scheme (from OpenViking) | — | `viking://` URIs + directory tree |
| Retrieval | vector + FTS + RRF fusion (hierarchical score-prop deferred) | persona / timer driven | hierarchical descent with score propagation (rmb's design goal) |
| Consolidation | append-first, versioned, drift-aware (arXiv:2605.12978) | continuous persona regen | upsert-style |
| Scheduling | per-session debounce + warmup (from TDAI) | three-timer pipeline | two-phase commit (from OpenViking) |

## Where rmb is genuinely stronger

1. **Tool-agnostic, zero-SDK capture.** Anything that can fire a hook and POST
   JSON works. The references are SDK / runtime-coupled (per the design doc's
   framing). This is rmb's clearest differentiator.
2. **Operational simplicity.** One Go binary + one Postgres. No tree filesystem
   to babysit, no separate vector DB. The dual-mode CLI (same binary = server,
   local tool, and remote client) is a nice touch neither reference emphasizes.
3. **Consolidation discipline.** rmb is explicitly built around the
   "continuous LLM consolidation is unreliable" paper: T0 append-only, events
   insert-only, T3 versioned (never in-place), drift-detecting eval. This is a
   more principled stance than "regenerate the persona each time."
4. **Inspectability.** Every artifact is a URI, dumpable via `cat` / `tree` /
   `meta`. Nothing hides in an opaque format.

## Where rmb is behind / weaker

1. **Retrieval is the least-finished tier.** rmb borrowed OpenViking's idea
   of hierarchical descent with score propagation but deferred it — current
   `search` is RRF over a flat candidate set. OpenViking's tree-structured
   descent is more sophisticated for deep corpora. Biggest capability gap.
2. **No middle facet.** Dropping OpenViking's "overview" was deliberate, but it
   means rerank chews the full body; fine now, a cost risk at scale.
3. **Categorization quality.** The 4-category taxonomy is cleaner than
   OpenViking's 8, but the T3 `profile` over-absorbs noise and `preferences` is
   empty — a real quality gap. (Fix shipped for new data; existing data still
   needs a re-extraction backfill.)
4. **No usage / forgetting signal yet.** TDAI's persona model implicitly
   re-weights; rmb keeps everything forever (the Ebbinghaus plan addresses
   this but isn't built).
5. **Single-namespace, single-tenant.** No scope-keying (work / personal /
   project), no multi-tenant isolation — deliberately parked, but the references
   are more general.

## Overall assessment

rmb is the most operationally disciplined and capture-flexible of the three,
and the only one explicitly engineered against consolidation drift. It
deliberately traded retrieval sophistication and generality for simplicity and
correctness — a sound bet for a personal memory store, which is its actual scope.

Its maturity profile is inverted from a research system: the plumbing (capture,
distillation, versioning, ops, recall surface) is production-solid, while the
intelligence layers (retrieval ranking, categorization quality, forgetting) are
the least mature. That is the right order to have built in, and it is where the
remaining roadmap points.

## Suggested priorities (next), measured against the peers

1. **Re-extraction backfill** — make existing memory match the new T1 routing
   (closes the categorization gap).
2. **Hierarchical / score-propagated retrieval** — the deferred OpenViking idea;
   biggest capability lift.
3. **Ebbinghaus usage-decay** — the differentiator none of them cleanly have.
4. **M1 / M2 / M5 hardening** — cheap correctness wins.
