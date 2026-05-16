# Hook Integration

## Design Intent

Agent tools (Cursor, Claude Code, and potentially Codex) each support hooks —
scripts that run at lifecycle events like conversation end. The goal is to wire
`mypast hook-submit` into all of them so every conversation is captured,
regardless of which tool the user is in.

The chosen trigger is the **conversation-end / stop** event:

- Cursor: `afterAgentResponse` (fires after each agent reply) or `stop`
- Claude Code: `Stop`
- Codex: TBD

## Hook Configuration

Hooks are configured in two places. Cursor respects both simultaneously
(when "Third-party skills" is enabled in Settings → Features):

**`~/.cursor/hooks.json`** — Cursor-native format:

```json
{
  "hooks": {
    "afterAgentResponse": [
      {
        "command": "/path/to/mypast/bin/mypast hook-submit --source=cursor",
        "timeout": 5
      }
    ]
  },
  "version": 1
}
```

**`~/.claude/settings.json`** — Claude Code format (also read by Cursor):

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/mypast/bin/mypast hook-submit --source=cc"
          }
        ]
      }
    ]
  }
}
```

## How Each Tool Passes Information

| Tool        | Mechanism | Payload shape                                              |
|-------------|-----------|-----------------------------------------------------------|
| Cursor      | stdin     | JSON with `conversation_id`, `text`, `transcript_path`, **`cursor_version`**, `workspace_roots` |
| Claude Code | stdin     | JSON with `session_id`, `transcript_path` — **no `cursor_version`** |
| Codex       | unknown   | TBD                                                       |

The presence of `cursor_version` in the payload is the reliable signal that
Cursor is the caller, regardless of which config file the hook came from.

## The Double-Fire Problem

When a conversation ends in Cursor, **both** hook files are evaluated and all
matching hooks execute. This means both:

- `mypast hook-submit --source=cursor` (from `~/.cursor/hooks.json`)
- `mypast hook-submit --source=cc` (from `~/.claude/settings.json`)

...fire for the same response, producing duplicate `session_turns` rows.

## Solution: Source-Aware Silent Skip

`mypast hook-submit --source=cc` must detect when it has been invoked by Cursor
(rather than Claude Code) and silently exit without uploading.

**Detection rule:** if the stdin payload contains a `cursor_version` field, the
caller is Cursor — skip and exit 0.

```
--source=cc + cursor_version present  →  silent no-op
--source=cc + no cursor_version       →  proceed with CC upload logic
--source=cursor                       →  always proceed
```

This lets you configure hooks in both files and have:

- Cursor conversations → captured by `--source=cursor` only
- Claude Code conversations → captured by `--source=cc` only
- No duplicates in either case

## Turn Shape Contract

Each uploaded `session_turn` should contain exactly two JSONL lines:

```
{"role":"user",    "content":"..."}
{"role":"assistant","content":"..."}
```

The user message is the last `user` text entry before the matched assistant
entry in the transcript. The assistant is matched by exact `text` field from
the payload (Cursor `afterAgentResponse`), or by last assistant text entry
(Claude Code `Stop` fallback).

## Known Issues / History

### `stop` payload has no `text` field

Cursor's `stop` event payload does not include a `text` field (the final
assistant reply). Without it, `buildMessagesFromTranscript` cannot locate the
correct assistant entry and falls back to uploading the user message only —
producing user-only turns. This is why `afterAgentResponse` is used for Cursor
(it does include `text`), while Claude Code `Stop` uses transcript-tail lookup.

### Legacy delta-uploader conflict

An earlier script (`mypast-cursor-session-end.ts`) uploaded transcript deltas
in batches on `sessionEnd`/`stop`. It ran concurrently with `hook-submit`,
causing turns with 10–19 lines of intermediate tool/progress messages. It has
been removed from all hook configs.

### Cursor reads `~/.claude/settings.json`

Cursor 3.5+ loads Claude Code hook configs when "Third-party skills" is
enabled. All hooks from all sources execute — they are not mutually exclusive.
Any `Stop` hook in `~/.claude/settings.json` fires in Cursor too, which is why
the source-aware skip is necessary.

Reference: https://cursor.com/docs/reference/third-party-hooks
