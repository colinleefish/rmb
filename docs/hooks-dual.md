# Dual hooks — local + production

Mirror every agent turn to **both** mem9 instances:

- **Local:** `http://127.0.0.1:8080` (`make run` or `docker compose up`)
- **Production:** `https://mem.colinleefish.com`

Use **two hook entries**. Cursor / Claude Code runs each command separately and passes the **full payload on stdin** to every entry — you are not sharing one pipe between processes.

`hook-submit` stays **single-target**: one invocation, one `MEM9_URL`.

## Cursor `~/.cursor/hooks.json`

```json
{
  "hooks": {
    "stop": [
      {
        "command": "/bin/bash -lc 'mkdir -p \"$HOME/.cursor/hook-debug/local\"; tee \"$HOME/.cursor/hook-debug/local/last.json\" | env MEM9_URL=http://127.0.0.1:8080 /Users/admin/Virginia/colinleefish/mem9/bin/mem9 hook-submit --source=cursor'",
        "timeout": 5
      },
      {
        "command": "/bin/bash -lc 'mkdir -p \"$HOME/.cursor/hook-debug/prod\"; tee \"$HOME/.cursor/hook-debug/prod/last.json\" | env MEM9_URL=https://mem.colinleefish.com MEM9_USERNAME=mem9 MEM9_PASSWORD=YOUR_PROD_PASSWORD /Users/admin/Virginia/colinleefish/mem9/bin/mem9 hook-submit --source=cursor'",
        "timeout": 15
      }
    ]
  },
  "version": 1
}
```

## Claude Code `~/.claude/settings.json`

Add a second hook under `Stop` with the same pattern (`MEM9_URL` + prod auth on the prod entry only).

## Why two entries (not `MEM9_URLS` in code)

| Two hook entries | Fan-out inside `hook-submit` |
|------------------|------------------------------|
| Visible in agent config | Hidden in `~/.mem9.conf` |
| Independent timeouts | One timeout for both HTTP calls |
| Local failure does not block prod | Coupled sequential uploads |
| Enable/disable one target easily | Requires conf edit |

## Prerequisites

| Target | Requirement |
|--------|-------------|
| Local | `make run` on `:8080` |
| Prod | https://mem.colinleefish.com + basic auth |

## Verify

- Local: http://127.0.0.1:8080/ui/
- Prod: https://mem.colinleefish.com/ui/

Same agent `session_id` appears on both if both hooks succeed.

## Optional `~/.mem9.conf`

Use this for the **prod** hook only if you omit inline `MEM9_USERNAME` / `MEM9_PASSWORD`:

```bash
MEM9_URL=https://mem.colinleefish.com
MEM9_USERNAME=mem9
MEM9_PASSWORD=...
```

Do **not** put local URL here when using dual hooks — set `MEM9_URL` per hook command instead.
