# rmb documentation site

Static docs built with [VitePress](https://vitepress.dev/).

**Package manager:** [pnpm](https://pnpm.io/). **Registry:** [npmmirror](https://npmmirror.com/) (`registry=https://registry.npmmirror.com` in `.npmrc`).

**Languages:** English (default) · [简体中文](/zh/)

Source drafts for long-form design docs live in `drafts/` at the repo root.

## Develop

```bash
cd docs
pnpm install
pnpm dev
```

Opens at `http://localhost:5173` by default. From repo root: `make docs-dev`.

## Build

```bash
cd docs
pnpm build
# output: docs/.vitepress/dist
pnpm preview
```

From repo root: `make docs-build`.

## Content layout

| Path | Purpose |
|------|---------|
| `concept/` | Readable explanation of the idea (pyramid, URI scheme, pipeline) |
| `guide/` | Setup, hooks, CLI for agents, corrections |
| `design/` | Full engineering design (EN + 中文) |
| `reference/` | Entity model, plan, deploy |

## Deploy options

- **GitHub Pages** — build `docs/`, publish `.vitepress/dist`
- **Caddy subpath** — e.g. `docs.rmb.colinleefish.com`
- **Embedded in rmb server** — copy dist into `internal/http/static/docs/` (not wired yet)
