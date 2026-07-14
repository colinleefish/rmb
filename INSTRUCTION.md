# rmb — recall guide for AI agents

`rmb` is the user's long-term memory across past AI conversations. Use it to
recall facts, decisions, configs, and context the user established in earlier
sessions, instead of asking them to repeat themselves.

The `rmb` command is on the PATH. It talks to the user's remote memory server
automatically (configured in `~/.rmb.conf`); you do not manage connections or
auth — just run the commands.

## When to use it

Before asking the user a question, or when a task references something that
likely happened before (a server, project, credential location, past decision,
preference), **search your memory first**:

- The user mentions a host/project/tool by name → `rmb search` it.
- You need a path, port, config location, or prior decision → recall it.
- The user says "like last time" / "the usual" / "where we left off" → recall it.
- The user asks you to **do something** (jokes, deploy, SSH, etc.) → `rmb search` it;
  if a `[skills]` hit matches, read and follow that skill before improvising.

If recall returns nothing relevant, then ask the user.

## Core commands

### Search (use this most)

```
rmb search "<natural language query>"
rmb search "<query>" --scope=memory      # distilled long-term facts only
rmb search "<query>" --scope=scene       # per-session conversation context only
rmb search "<query>" --k=<n>             # control result count (default: 5)
```

`search` blends semantic (vector) and keyword (FTS) matching, fused with
reciprocal rank fusion. By default it covers **memory**, **scene**, and **skills** tiers.

- Use `--scope=memory` when you want tight, specific fact recall — e.g. a config
  value, a person's role, a past decision.
- Use `--scope=scene` when you want conversational context — e.g. "what were we
  working on in that session".
- Use `--scope=skill` when you only want curated playbooks.
- Omit `--scope` for most queries; the combined result is usually best.

Output is a ranked list:

```
 1. [memories] rmb://entities/tokyo-shadowsocks-config
      Shadowsocks config on tokyo-endpoint is at /etc/shadowsocks-rust/config.json ...
 2. [scenes]   rmb://scenes/<uuid>
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
rmb cat <uri>      # full body/content of a memory, scene, or turn
rmb meta <uri>     # metadata as JSON (category, slug, version, source links, timestamps)
rmb tree <uri>     # list child URIs under a prefix
```

Typical flow: `search` to find a relevant `uri`, then `cat <uri>` to read the
full detail, and optionally `meta <uri>` to see provenance.

## The memory model (how to read results)

Memory is a pyramid; results carry a `tier` and a `uri`:

| Tier | URI shape | What it is |
|------|-----------|------------|
| memories | `rmb://profile`, `rmb://preferences/<slug>`, `rmb://entities/<slug>`, `rmb://events/<slug>` | Long-term, cross-session distilled facts. Most useful. |
| skills | `rmb://skills/<name>` | Curated Agent Skills bundles (separate from memory pyramid). |
| scenes | `rmb://scenes/<uuid>` | Per-conversation summaries ("what we were doing"). |
| turns | `rmb://turns/<uuid>` | Raw user+assistant exchange (`meta` → `session_id`). |
| atoms | `rmb://atoms/<uuid>` | Structured facts from a session (`meta` → `session_id`). |
| sessions | `rmb://sessions/<id>` | Session abstract — container for a conversation. |

Memory categories:

- **profile** — stable facts about the user (singleton).
- **preferences** — recurring "prefers X", including how the user wants the AI to behave.
- **entities** — people, projects, companies, hosts, tools.
- **events** — dated decisions and milestones (immutable).

## Skills (Agent Skills bundles)

Skills are **not** distilled memory — they are curated playbooks at `rmb://skills/<name>`.
**Check skills before improvising** on a user request that might match one.

| Tier | Command | Content |
|------|---------|---------|
| Catalog | `rmb tree rmb://skills/` | name + description |
| Activation | `rmb cat rmb://skills/<name>` | full SKILL.md |
| Resources | `rmb cat rmb://skills/<name>/<path>` | bundled scripts/references |

Flow: `rmb search "<task>"` → if `[skills]` hit → `rmb cat rmb://skills/<name>` → follow SKILL.md.

Local cache for script execution: `rmb skill pull <name>` → `~/.rmb/skills/<name>/`.
Push edits: `rmb skill put <name>` from that directory.
Do not use `~/.cursor/skills/` or `~/.claude/skills/`.

To trace a fact to its source: `meta <memory-uri>` shows `source_scene_uris`;
`cat` those scenes; scenes link back to atoms and raw turns.

## Examples

```
# "What's the config for the tokyo endpoint?"
rmb search "tokyo endpoint shadowsocks config" --scope=memory
rmb cat rmb://entities/tokyo-shadowsocks-config

# "Where does Jenkins store its data again?"
rmb search "jenkins home directory disk" --scope=memory

# "What did we decide about storage?"
rmb search "storage decision postgres"

# "What were we working on last session?"
rmb search "recent work" --scope=scene

# Browse what categories exist
rmb tree rmb://
rmb tree rmb://entities/
rmb tree rmb://skills/

# Agent skill playbook
rmb cat rmb://skills/my-skill
rmb skill pull my-skill   # → ~/.rmb/skills/my-skill/
```

## Corrections

Machine-distilled memory can be wrong or over-merged. The user can attach
**corrections** — durable, human-authored patches — to any memory `uri`. They
are not edits to the memory; they overlay it and always win over the machine fact.

- `rmb cat <uri>`, `meta <uri>`, and `search` results automatically show any
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
rmb correction add <rmb://uri> [<uri>...] "the corrected fact"
rmb correction ls [<target-uri>]                    # list active corrections
rmb correction rm <rmb://corrections/...>        # retract one you added
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
- The only writes you make are `rmb correction` commands, and only when the
  user **explicitly asks** you to correct a memory **and** a real URI already
  exists for the target. A user stating a new fact is not a write request.
