# Pi agent hook for rmb

[Pi](https://pi.dev) is a minimal terminal coding agent. Unlike Cursor or Claude
Code, Pi does not ship shell hooks — capture is done via a **Pi extension** that
listens for `agent_settled` and pipes a JSON payload to `rmb hook-submit`.

## How it works

```
Pi turn completes → agent_settled event → extension builds payload
  → rmb hook-submit --source=pi (stdin JSON)
  → POST /api/v1/sessions/:id/upload
```

Use `agent_settled` (not `agent_end`) so uploads happen only after Pi finishes
auto-retry, compaction, and queued follow-ups — same semantics as Cursor `stop`
or Claude Code `Stop`.

### Chat data sources

| Field | Source |
|-------|--------|
| `session_id` | `ctx.sessionManager.getSessionId()` |
| `session_file` | `ctx.sessionManager.getSessionFile()` — Pi JSONL at `~/.pi/agent/sessions/...` |
| `last_assistant_message` | Last assistant text block on the current branch |
| User prompt | Parsed from session JSONL by `internal/hook/pi.go` |

Pi session format: [`session-format.md`](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/session-format.md)

## Install

1. Build or install `rmb` on your PATH.

2. Link the extension into Pi's extension directory:

```bash
mkdir -p ~/.pi/agent/extensions
ln -sf /path/to/rmb/integrations/pi/rmb-hook.ts ~/.pi/agent/extensions/rmb-hook.ts
```

3. Restart Pi. The extension loads automatically from `~/.pi/agent/extensions/`.

## Configure target

Default upload target is `http://127.0.0.1:8080`. Override per shell or in
`~/.rmb.conf`:

```bash
export RMB_URL=http://127.0.0.1:8080
# optional if rmb is not on PATH:
export RMB_HOOK_BIN=/path/to/rmb/bin/rmb
```

For production dual-target capture, run two Pi extension copies is not
supported — instead wrap `submitHook` or add a second `pi.exec` call mirroring
[`drafts/hooks-dual.md`](../../drafts/hooks-dual.md).

## Payload shape

```json
{
  "agent": "pi",
  "session_id": "uuid-from-session-header",
  "session_file": "/Users/you/.pi/agent/sessions/--path--/2024-12-03_uuid.jsonl",
  "transcript_path": "/Users/you/.pi/agent/sessions/--path--/2024-12-03_uuid.jsonl",
  "cwd": "/your/project",
  "last_assistant_message": "final assistant text for this settled run",
  "hook_event_name": "agent_settled"
}
```

## Manual test

```bash
printf '%s\n' '{
  "agent": "pi",
  "session_id": "test-session",
  "session_file": "'"$HOME"'/.pi/agent/sessions/test.jsonl",
  "last_assistant_message": "hello from pi"
}' | rmb hook-submit --source=pi
```

## References

- Pi extensions: https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/extensions.md
- Pi RPC events (`agent_settled`, `turn_end`): https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/rpc.md
- rmb hook parser: `internal/hook/pi.go`
