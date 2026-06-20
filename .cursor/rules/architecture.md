# Architecture Rules

## CLI / Server Boundary

**The CLI is a pure API client. It never touches the database.**

- All database operations belong on the server (`internal/http/handler/`, workers, services).
- Every CLI subcommand in `internal/cli/command.go` must call the HTTP API via `client.Resolve()`.
- If `MEM9_URL` is not configured, commands that require the server must return a clear error — there is no local-DB fallback for CLI operations.

This is a hard constraint, not a soft preference. Do not add local-DB paths to CLI commands.

## Dependency Direction

```
CLI (internal/cli)
  └── HTTP client (internal/client)
        └── HTTP API (internal/http/handler)
              └── Services (internal/service/*)
                    └── Database (internal/db)
```

The CLI layer must not import `internal/db`, `internal/service/*`, or `gorm.io/gorm`.
