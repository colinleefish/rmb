---
layout: home

hero:
  name: rmb
  text: Memory that survives the session
  tagline: Capture every agent conversation, distill it into facts and scenes, recall it across tools — without an SDK in your agent.
  actions:
    - theme: brand
      text: Understand the idea
      link: /concept/
    - theme: alt
      text: Get started
      link: /guide/getting-started

features:
  - icon: 🪝
    title: Tool-agnostic capture
    details: Cursor and Claude Code hooks POST raw turns to a single Go server. No agent runtime changes beyond a hook entry.
  - icon: 🔺
    title: Four-tier distillation
    details: Turns → atoms → scenes → long-term memories. Each layer is addressable by URI and inspectable from the CLI.
  - icon: 🔍
    title: Hybrid recall
    details: Vector search over abstracts, full-text over bodies, fused ranking — so agents find facts and conversational context.
  - icon: 🗄️
    title: Postgres-native
    details: One binary, one database. No filesystem artifact trees. Every artifact lives in a column you can query.
---

## The problem

AI agents forget. Session context resets. You re-explain servers, preferences, and past decisions every time you open a new chat.

**rmb** is a personal memory server: it ingests conversations in the background, distills durable knowledge, and exposes it through `rmb search` so the next agent session can recall what you already established.

## How it differs

| Typical "memory" product | rmb |
|---|---|
| Opaque blob the model can't inspect | Every fact has a `rmb://` URI; `cat`, `tree`, `meta` work |
| SDK inside the agent | Hook + HTTP upload only |
| One summary per chat | Pyramid — drill from memory down to raw turns |
| Merge everything aggressively | Append-first; human corrections override machine facts |

## Status

T1–T3 workers (atoms, scenes, memories) ship today. Production runs at [rmb.colinleefish.com](https://rmb.colinleefish.com). See the [implementation plan](/reference/plan) for roadmap detail.
