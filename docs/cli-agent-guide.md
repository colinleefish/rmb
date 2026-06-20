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

- The user mentions a host/project/tool by name → `rmb find` or `search` it.
- You need a path, port, config location, or prior decision → recall it.
- The user says "like last time" / "the usual" / "where we left off" → recall it.

If recall returns nothing relevant, then ask the user.

## Core commands

### Search (use this most)

```
rmb search "<natural language query>"      # hybrid: meanings + keywords, across memories and scenes
rmb find "<natural language query>"         # vector-only: closest long-term memories
```

- Prefer `search` for most questions — it blends semantic + keyword matching and
  covers both distilled memories and per-session scenes.
- Use `find` when you want only the tightest long-term facts.
- Add `--k=<n>` to control result count (default: find 5, search 8).

Output is a ranked list:

```
 1. [memories] rmb://entities/tokyo-shadowsocks-config
      Shadowsocks config on tokyo-endpoint is at /etc/shadowsocks-rust/config.json ...
 2. [scenes]   rmb://scenes/<uuid>
      ...one-line abstract...
```

Each line is `<rank>. [tier] <uri>` followed by a snippet. The `uri` is the
handle for drilling down.

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
| scenes | `rmb://scenes/<uuid>` | Per-conversation summaries ("what we were doing"). |
| sessions/turns | `rmb://sessions/<id>`, `.../turns/<n>` | Raw conversation evidence (ground truth). |

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
rmb search "tokyo endpoint shadowsocks config"
rmb cat rmb://entities/tokyo-shadowsocks-config

# "Where does Jenkins store its data again?"
rmb search "jenkins home directory disk"

# "What did we decide about storage?"
rmb search "storage decision postgres"

# Browse what categories exist
rmb tree rmb://
rmb tree rmb://entities/
```

## Rules

- Recall **before** asking the user about anything that may be in past context.
- **Human corrections override memory.** Recall results and `cat`/`meta` may show
  `⚑ CORRECTION` / `⚑ RETIRED` lines attached to a memory — these are
  human-authored and authoritative. Prefer them over the memory body; if several
  conflict, the newest wins. A `RETIRED` flag means treat that memory as wrong.
- Treat `memories` as the user's established truth; if a memory conflicts with a
  fresh statement from the user, prefer the user and note the discrepancy.
- Quote the `uri` when you rely on a memory, so the user can verify it.
- Do not fabricate URIs; only use ones returned by `search`/`find`/`tree`.
- These commands are read-only recall. Memory is written automatically from
  conversations by background workers — you do not need to store anything.
