# OpenCode plugin for rmb

[OpenCode](https://opencode.ai) uses in-process TypeScript plugins rather than
shell hooks. Capture is done via a plugin that listens for session idle events
and pipes a JSON payload to `rmb hook-submit`.

## How it works

```
OpenCode turn completes → session.status idle (or session.idle)
  → plugin fetches session messages via SDK
  → rmb hook-submit --source=opencode (stdin JSON)
  → POST /api/v1/sessions/:id/upload
```

Use `session.status` with `status.type === "idle"` only. Do not also handle the
deprecated `session.idle` event — OpenCode emits both for the same settle, which
duplicates uploads.

### Chat data sources

| Field | Source |
|-------|--------|
| `session_id` | `event.properties.sessionID` |
| `last_user_message` | Last user text parts from `client.session.messages()` |
| `last_assistant_message` | Last assistant text parts from `client.session.messages()` |
| `cwd` | Plugin `directory` context |

OpenCode stores transcripts in SQLite (`~/.local/share/opencode/opencode.db`),
not JSONL files. The plugin reads messages through the OpenCode SDK and includes
both sides of the turn in the payload.

OpenCode session IDs look like `ses_0a53a1eafffexdrKwI0l3k9ewh`. rmb's upload
API requires UUIDs, so `internal/hook/opencode.go` maps each OpenCode ID to a
deterministic UUIDv5 (`opencode:<ses_id>` under a fixed namespace). The same
OpenCode session always lands in the same rmb session row.

## Install

1. Build or install `rmb` on your PATH.

2. Ensure `@opencode-ai/plugin` is available (OpenCode installs it when you add
   plugins under `~/.config/opencode/`).

3. Copy the plugin (concrete file, not a symlink):

```bash
mkdir -p ~/.config/opencode/plugin
cp /path/to/rmb/integrations/opencode/rmb-hook.ts ~/.config/opencode/plugin/rmb-hook.ts
```

The installed plugin defaults to `/Users/liguanghui/Virginia/colinleefish/rmb/bin/rmb`
and runs `make build` in that repo if the binary is missing. Override with
`RMB_HOOK_BIN` when needed.

4. Restart OpenCode. Plugins load from `~/.config/opencode/plugin/` (global) or
   `.opencode/plugin/` (project).

## Configure target

Default upload target is `http://127.0.0.1:8080`. Override per shell or in
`~/.rmb.conf`:

```bash
export RMB_URL=http://127.0.0.1:8080
# optional override (default: $RMB_REPO/bin/rmb, built on demand):
export RMB_HOOK_BIN=/path/to/rmb/bin/rmb
```

## Payload shape

```json
{
  "agent": "opencode",
  "session_id": "ses_0a53a1eafffexdrKwI0l3k9ewh",
  "last_user_message": "add opencode hook support",
  "last_assistant_message": "Done — added opencode.go and the plugin.",
  "session_db_path": "/Users/you/.local/share/opencode/opencode.db",
  "cwd": "/your/project",
  "hook_event_name": "session.status"
}
```

## Manual test

```bash
printf '%s\n' '{
  "agent": "opencode",
  "session_id": "test-session",
  "last_user_message": "hello opencode",
  "last_assistant_message": "hello from opencode"
}' | rmb hook-submit --source=opencode
```

## References

- OpenCode plugins: https://opencode.ai/docs/plugins
- OpenCode events (`session.status`, `session.idle`): plugin SDK `event` hook
- rmb hook parser: `internal/hook/opencode.go`
