# Why `mypast` has no `forget` assertion

> Decision: **dropped.** The human authority layer is a single thing — a
> **correction** (a content overlay). Deliberate forgetting is not a human action
> mypast supports; forgetting is the job of passive, usage-based decay (see
> `docs/ebbinghaus-recall.md`). This doc records the reasoning so we don't
> re-litigate it.
>
> (Terminology note: this doc predates the rename from "assertion" to
> "correction" and the removal of the `split`/`alias` kinds. The reasoning stands;
> read "assertion kind `correct`" as today's "correction".)

## Origin

`forget` shipped as a second assertion kind alongside `correct`, rendered as a
`⚑ RETIRED` flag on recall. While designing **write-time distill injection**
(folding human assertions into the regenerated memory body — Phase 2 of the
correction layer), we tried to define what `forget` should *do* during a
distill. Working through it dismantled the kind entirely.

## The reasoning chain

### 1. `forget`-with-a-statement is just a negative `correct`

Under distill injection, every assertion is "an authoritative human instruction
to the distiller." A `correct` adds a fact; a `forget "drop the Huawei stuff"`
removes one. But removal is expressible as a **negative correction** —
`correct "she does NOT work at Huawei"` already makes the distiller drop those
facts. So a `forget` carrying a statement has *zero* extra power over `correct`.
It is the same mechanism with a different label.

### 2. The only thing `correct` can't express is "kill the whole record"

A statement-less `forget` ("this memory shouldn't exist") is the sole case
`correct` cannot cover, because there is no fact to assert. That is a
**structural / visibility** operation, not a content one — so it would need a
different mechanism (a recall-time suppression filter), not the distiller.

### 3. But suppression can't actually "forget" anything

mypast is **evidence-driven**: a memory is a *view* re-derived from append-only
atoms/scenes. As long as the evidence exists, T3 keeps re-deriving the memory at
its stable slug URI. So a suppression flag on the URI does not erase anything —
it only hides the view, while the entity keeps silently accumulating new
evidence underneath. "We forgot her, but she's still there getting new info,
just never shown." The word *forget* promises a terminality the architecture
cannot deliver.

This also exposed an ugly lifecycle: if suppression is **sticky** (survives
re-distills, per the "human beats machine" principle), a forgotten slug stays
hidden even when genuinely new, useful information arrives. If suppression
**lapses** on new evidence, then machine-generated atoms silently override an
explicit human signal — violating the very principle the assertion layer exists
to protect. Neither branch is clean.

### 4. Humans don't forget by choice

The decisive point: **forgetting is passive.** People don't decide to forget;
unused memories fade on their own. "Choosing to forget" is really *pretending*
to forget — the information is still there. A deliberate `forget` command is
exactly that pretense, and the architecture makes the pretense literal (the
evidence regenerates the memory regardless).

### 5. The project already models forgetting honestly

`docs/ebbinghaus-recall.md` already specifies the right mechanism: **usage-based
soft decay.** Useful memories (recalled often) stay prominent; unused ones fade
in ranking until they rarely surface — never deleted, always reversible because
the evidence persists. That is forgetting as it actually works: passive,
gradual, reinforcement-driven. No command required.

## Decision

Drop `forget`. The human authority layer is **one kind: `correct`** — a content
overlay that may be positive ("she works at a bank") or negative ("she does NOT
work at Huawei"), merged into the body at distill time.

Everything `forget` reached for is covered, more honestly, elsewhere:

| Intent | Honest mechanism |
|--------|------------------|
| "This fact is wrong" | negative `correct` — distiller drops it |
| "I don't use this anymore" | Ebbinghaus decay — fades by disuse (passive) |
| "Block it forever, never surface" | a separate, honestly-named `mute`/`block` — build only if a real need appears; it is suppression, not forgetting |

This mirrors the earlier removal of the `assert` kind: *if there is nothing to
correct, it is not a correction* → and now: *you cannot choose to forget, so
there is no `forget`.*

## What we did NOT drop

Entity resolution (`alias`, slug-drift fixes) is **deferred-but-intended**, not
decided against. Unlike `forget`, the "she became a different entity under the
same slug" edge case is real — it just lives in its own dedicated mechanism,
outside the correction umbrella (see `docs/corrections.md`), rather than as a
correction kind.

## Cleanup performed

- Migration `00007`: `assertions_kind_check` tightened to `('correct', 'split',
  'alias')` (no live `forget` rows existed).
- `model.AssertionKindForget` removed.
- `assertion.Create` accepts only `correct` and always requires a statement.
- CLI `parseAssertionKind` / usage drop `forget`.
- `inspect` / CLI recall rendering drop the `RETIRED` branch (only `CORRECTION`
  remains).
- `docs/corrections.md` updated: kinds = `correct` only; forgetting = decay.
