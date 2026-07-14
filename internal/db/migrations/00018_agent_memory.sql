-- +goose Up
-- Curated agent guide at rmb://agent (singleton; not T3-distilled).
-- +goose StatementBegin
ALTER TABLE memories DROP CONSTRAINT IF EXISTS memories_category_check;
ALTER TABLE memories ADD CONSTRAINT memories_category_check CHECK (
    category IN ('profile', 'preferences', 'entities', 'events', 'agent')
);

INSERT INTO memories (
    id, uri, category, slug, version, abstract, body,
    source_scene_uris, source_correction_uris, created_at, updated_at
)
SELECT
    gen_random_uuid(),
    'rmb://agent',
    'agent',
    NULL,
    1,
    'Agent recall guide',
    $body$## Memory pyramid (T0 → T3)

| Tier | URI | What |
|------|-----|------|
| sessions | rmb://sessions/<id> | conversation container |
| turns | rmb://turns/<uuid> | raw user+assistant exchange |
| atoms | rmb://atoms/<uuid> | facts extracted from one session |
| scenes | rmb://scenes/<uuid> | per-session summary |
| memories | see below | long-term distilled facts |

## Memory uris

profile | entities/<slug> | preferences/<slug> | events/<slug> | scenes/<uuid>

## Memory categories (T3)

| Category | URI | Content |
|----------|-----|--------|
| profile | rmb://profile | singleton — who the user is |
| agent | rmb://agent | singleton — how to use rmb (this doc) |
| preferences | rmb://preferences/<slug> | how the user wants AI to behave |
| entities | rmb://entities/<slug> | people, projects, hosts, tools |
| events | rmb://events/<slug> | dated decisions (immutable) |

## CLI rules

- Do not run serve — it starts the server. Recall uses RMB_URL.
- search "<query>" before asking the user, then cat / meta / tree as needed.
- search [--scope=...] — only search accepts --scope. cat/tree/meta take a single uri.
- tree <uri-prefix> — browse rmb://entities/, rmb://preferences/, rmb://profile (not rmb://memories/).
- Never invent uris. correction add only when the user explicitly asks and search returned a real uri.
- Recall is read-only. Workers distill new facts after conversations.
$body$,
    '{}',
    '{}',
    now(),
    now()
WHERE NOT EXISTS (
    SELECT 1 FROM memories WHERE uri = 'rmb://agent' AND superseded_at IS NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM memories WHERE uri = 'rmb://agent';

ALTER TABLE memories DROP CONSTRAINT IF EXISTS memories_category_check;
ALTER TABLE memories ADD CONSTRAINT memories_category_check CHECK (
    category IN ('profile', 'preferences', 'entities', 'events')
);
-- +goose StatementEnd
