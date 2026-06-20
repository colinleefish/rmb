# Dual hooks — local + production

Mirror every agent turn to **both** rmb instances:

- **Local:** `http://127.0.0.1:8080` (`make run` or `docker compose up`)
- **Production:** `https://rmb.colinleefish.com`

Use **two hook entries**. Cursor / Claude Code runs each command separately and passes the **full payload on stdin** to every entry — you are not sharing one pipe between processes.

`hook-submit` stays **single-target**: one invocation, one `RMB_URL`.

## Cursor `~/.cursor/hooks.json`

```json
{
  "hooks": {
    "stop": [
      {
        "command": "/bin/bash -lc 'mkdir -p \"$HOME/.cursor/hook-debug/local\"; tee \"$HOME/.cursor/hook-debug/local/last.json\" | env RMB_URL=http://127.0.0.1:8080 /Users/admin/Virginia/colinleefish/rmb/bin/rmb hook-submit --source=cursor'",
        "timeout": 5
      },
      {
        "command": "/bin/bash -lc 'mkdir -p \"$HOME/.cursor/hook-debug/prod\"; tee \"$HOME/.cursor/hook-debug/prod/last.json\" | env RMB_URL=https://rmb.colinleefish.com RMB_USERNAME=rmb RMB_PASSWORD=YOUR_PROD_PASSWORD /Users/admin/Virginia/colinleefish/rmb/bin/rmb hook-submit --source=cursor'",
        "timeout": 15
      }
    ]
  },
  "version": 1
}
```

## Claude Code `~/.claude/settings.json`

Add a second hook under `Stop` with the same pattern (`RMB_URL` + prod auth on the prod entry only).

## Why two entries (not `RMB_URLS` in code)

| Two hook entries | Fan-out inside `hook-submit` |
|------------------|------------------------------|
| Visible in agent config | Hidden in `~/.rmb.conf` |
| Independent timeouts | One timeout for both HTTP calls |
| Local failure does not block prod | Coupled sequential uploads |
| Enable/disable one target easily | Requires conf edit |

## Prerequisites

| Target | Requirement |
|--------|-------------|
| Local | `make run` on `:8080` |
| Prod | https://rmb.colinleefish.com + basic auth |

## Verify

- Local: http://127.0.0.1:8080/ui/
- Prod: https://rmb.colinleefish.com/ui/

Same agent `session_id` appears on both if both hooks succeed.

## Optional `~/.rmb.conf`

Use this for the **prod** hook only if you omit inline `RMB_USERNAME` / `RMB_PASSWORD`:

```bash
RMB_URL=https://rmb.colinleefish.com
RMB_USERNAME=rmb
RMB_PASSWORD=...
```

Do **not** put local URL here when using dual hooks — set `RMB_URL` per hook command instead.
