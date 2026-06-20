# mem9 — recall guide for AI agents

`mem9` is the user's long-term memory across past AI conversations. Use it to
recall facts, decisions, configs, and context the user established in earlier
sessions, instead of asking them to repeat themselves.

The `mem9` command is on the PATH. It talks to the user's remote memory server
automatically (configured in `~/.mem9.conf`); you do not manage connections or
auth — just run the commands.

## When to use it

Before asking the user a question, or when a task references something that
likely happened before (a server, project, credential location, past decision,
preference), **search your memory first**:

- The user mentions a host/project/tool by name → `mem9 search` it.
- You need a path, port, config location, or prior decision → recall it.
- The user says "like last time" / "the usual" / "where we left off" → recall it.

If recall returns nothing relevant, then ask the user.

## Core commands

### Search (use this most)

```
mem9 search "<natural language query>"
mem9 search "<query>" --scope=memory      # distilled long-term facts only
mem9 search "<query>" --scope=scene       # per-session conversation context only
mem9 search "<query>" --k=<n>             # control result count (default: 5)
```

`search` blends semantic (vector) and keyword (FTS) matching, fused with
reciprocal rank fusion. By default it covers both `memory` and `scene` tiers.

- Use `--scope=memory` when you want tight, specific fact recall — e.g. a config
  value, a person's role, a past decision.
- Use `--scope=scene` when you want conversational context — e.g. "what were we
  working on in that session".
- Omit `--scope` for most queries; the combined result is usually best.

Output is a ranked list:

```
 1. [memories] mem9://entities/tokyo-shadowsocks-config
      Shadowsocks config on tokyo-endpoint is at /etc/shadowsocks-rust/config.json ...
 2. [scenes]   mem9://scenes/<uuid>
      ...one-line abstract...
```

Each line is `<rank>. [tier] <uri>` followed by a snippet. The `uri` is the
handle for drilling down.

If a result has a human correction attached, it is shown right under the
snippet, flagged `⚑ CORRECTION:` (a human override of the fact). These come
from the user and **outrank** the machine-distilled fact — see "Corrections"
below.

### Drill down

```
mem9 cat <uri>      # full body/content of a memory, scene, or turn
mem9 meta <uri>     # metadata as JSON (category, slug, version, source links, timestamps)
mem9 tree <uri>     # list child URIs under a prefix
```

Typical flow: `search` to find a relevant `uri`, then `cat <uri>` to read the
full detail, and optionally `meta <uri>` to see provenance.

## The memory model (how to read results)

Memory is a pyramid; results carry a `tier` and a `uri`:

| Tier | URI shape | What it is |
|------|-----------|------------|
| memories | `mem9://profile`, `mem9://preferences/<slug>`, `mem9://entities/<slug>`, `mem9://events/<slug>` | Long-term, cross-session distilled facts. Most useful. |
| scenes | `mem9://scenes/<uuid>` | Per-conversation summaries ("what we were doing"). |
| sessions/turns | `mem9://sessions/<id>`, `.../turns/<n>` | Raw conversation evidence (ground truth). |

Memory categories:

- **profile** — stable facts about the user (singleton).
- **preferences** — recurring "prefers X", including how the user wants the AI to behave.
- **entities** — people, projects, companies, hosts, tools.
- **events** — dated decisions and milestones (immutable).

To trace a fact to its source: `meta <memory-uri>` shows `source_scene_uris`;
`cat` those scenes; scenes link back to atoms and raw turns.

## Examples

```
# "What's the config for the tokyo endpoint?"
mem9 search "tokyo endpoint shadowsocks config" --scope=memory
mem9 cat mem9://entities/tokyo-shadowsocks-config

# "Where does Jenkins store its data again?"
mem9 search "jenkins home directory disk" --scope=memory

# "What did we decide about storage?"
mem9 search "storage decision postgres"

# "What were we working on last session?"
mem9 search "recent work" --scope=scene

# Browse what categories exist
mem9 tree mem9://
mem9 tree mem9://entities/
```

## Corrections

Machine-distilled memory can be wrong or over-merged. The user can attach
**corrections** — durable, human-authored patches — to any memory `uri`. They
are not edits to the memory; they overlay it and always win over the machine fact.

- `mem9 cat <uri>`, `meta <uri>`, and `search` results automatically show any
  active corrections on that target. You do not fetch them separately.
- Each one (`⚑ CORRECTION:`) is the user's authoritative statement about the
  thing; treat it as the truth, over the distilled snippet. It may be positive
  ("she works at a bank") or negative ("she does NOT work at Huawei").
- If multiple corrections are attached, they are **additive** — apply them all;
  the **newest wins** only on a direct conflict.

### When to write a correction — strict rules

**Only write a correction when ALL of these are true:**

1. The user **explicitly asks** you to correct or update a memory (e.g. "add this
   to memory", "correct that", "remember that X is wrong").
   - The user simply stating a fact ("Ma Xin is a colleague") is **not** a
     request to write a correction. Acknowledge it and move on.

2. A **real URI already exists** for the target. Corrections patch existing
   memories — they cannot create new ones.
   - If `search` returns no URI for the subject, there is nothing to attach to.
     Do not invent a URI. New entities enter memory automatically via background
     workers after the conversation is processed.

```
mem9 correction add <mem9://uri> [<uri>...] "the corrected fact"
mem9 correction ls [<target-uri>]                    # list active corrections
mem9 correction rm <mem9://corrections/...>        # retract one you added
```

Always pass real `uri`s returned by recall — never invent them. (There is no
"forget" — a wrong fact is a negative correction, and unused memories fade on
their own.)

## Rules

- Recall **before** asking the user about anything that may be in past context.
- Treat `memories` as the user's established truth; if a memory conflicts with a
  fresh statement from the user, prefer the user and note the discrepancy.
- A human **correction** on a memory beats the memory; corrections are
  additive and the newest wins on a direct conflict. Honor them.
- Quote the `uri` when you rely on a memory, so the user can verify it.
- Do not fabricate URIs; only use ones returned by `search`/`tree`.
- Recall (`search`/`cat`/`meta`/`tree`) is read-only — memory is written
  automatically by background workers after each conversation. You never store
  new facts yourself.
- The only writes you make are `mem9 correction` commands, and only when the
  user **explicitly asks** you to correct a memory **and** a real URI already
  exists for the target. A user stating a new fact is not a write request.
