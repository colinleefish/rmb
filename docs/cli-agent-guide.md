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

- The user mentions a host/project/tool by name → `mem9 find` or `search` it.
- You need a path, port, config location, or prior decision → recall it.
- The user says "like last time" / "the usual" / "where we left off" → recall it.

If recall returns nothing relevant, then ask the user.

## Core commands

### Search (use this most)

```
mem9 search "<natural language query>"      # hybrid: meanings + keywords, across memories and scenes
mem9 find "<natural language query>"         # vector-only: closest long-term memories
```

- Prefer `search` for most questions — it blends semantic + keyword matching and
  covers both distilled memories and per-session scenes.
- Use `find` when you want only the tightest long-term facts.
- Add `--k=<n>` to control result count (default: find 5, search 8).

Output is a ranked list:

```
 1. [memories] mem9://entities/tokyo-shadowsocks-config
      Shadowsocks config on tokyo-endpoint is at /etc/shadowsocks-rust/config.json ...
 2. [scenes]   mem9://scenes/<uuid>
      ...one-line abstract...
```

Each line is `<rank>. [tier] <uri>` followed by a snippet. The `uri` is the
handle for drilling down.

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
mem9 search "tokyo endpoint shadowsocks config"
mem9 cat mem9://entities/tokyo-shadowsocks-config

# "Where does Jenkins store its data again?"
mem9 search "jenkins home directory disk"

# "What did we decide about storage?"
mem9 search "storage decision postgres"

# Browse what categories exist
mem9 tree mem9://
mem9 tree mem9://entities/
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
