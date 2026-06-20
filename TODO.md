# Mem9 TODO

## Now

- [ ] Implement Slice 1 storage core in Go library:
  `store`, `read`, `list`, `delete`.
- [ ] Create schema migration for `memories` and indexes.
- [ ] Build CLI commands:
  `mem9 store`, `mem9 read`, `mem9 list`, `mem9 delete`.
- [ ] Write migration script from OpenViking URIs into `mem9://` URIs.
- [ ] Add acceptance test:
  replace OpenViking store/read/list flow with Mem9 equivalents.

## Next

- [ ] Implement Slice 2 embed worker:
  single-process loop for rows where `embedding IS NULL`.
- [ ] Implement hybrid recall:
  dense cosine plus FTS weighted score.
- [ ] Define five sanity recall queries and compare with OpenViking behavior.
- [ ] Add `mem9 reembed` command for embedding dimension migration.

## Agent model idea

- [ ] Define first-class agent profile model:
  `identity.md`, `soul.md`, `skills`, and `credentials` references.
- [ ] Keep parent agent as canonical long-term memory owner.
- [ ] Spawn child agents with task-scoped context snapshots only.
- [ ] Enforce least-privilege credentials per child agent.
- [ ] Make child write-back explicit:
  children propose events or memory candidates, parent commits final memory.
- [ ] Add provenance fields for memory writes:
  `author_agent`, `source_task`, `parent_agent`, and `confidence`.
- [ ] Decide whether agent memory is URI-scoped by agent ID or shared with tags.

## Later

- [ ] Implement Slice 3 abstracts worker and `mem9 abstract`.
- [ ] Add optional `mp_tree` browsing support.
- [ ] Evaluate Chinese tokenizer upgrade only if recall quality is weak.
- [ ] Add backup automation (`pg_dump` cron).
- [ ] Add MCP wrapper (Slice 5) after CLI and library surface is stable.

## Open decisions

- [ ] Confirm URI migration strategy:
  rewrite `viking://` to `mem9://` at migration time.
- [ ] Confirm default embedding dimension:
  `1024` vs `2048`.
