---
description: Markdown authoring rules to satisfy markdownlint defaults
globs: ["**/*.md", "**/*.markdown"]
alwaysApply: true
---

# Markdown rules

Follow markdownlint default rules when generating or editing any `.md` file.

## Structure

- **MD041**: First line must be a top-level `# Heading` (no prose, no frontmatter prose before it).
- **MD025**: Only one `# H1` per document.
- **MD001**: Heading levels increase by one (no `##` → `####`).
- **MD003**: Use ATX headings (`#`, `##`), never Setext (`===`, `---`) and no closing `#`.
- **MD024**: No duplicate heading text under the same parent.
- **MD026**: No trailing punctuation (`.`, `,`, `;`, `:`, `!`) in headings. `?` is fine.

## Spacing & blank lines

- **MD022**: Headings surrounded by one blank line above and below.
- **MD031**: Fenced code blocks surrounded by blank lines.
- **MD032**: Lists surrounded by blank lines.
- **MD047**: File ends with exactly one trailing newline.
- **MD012**: No more than one consecutive blank line.
- **MD009**: No trailing spaces (don't use two-space line breaks; use a blank line).
- **MD010**: No hard tabs; use spaces.

## Lists

- **MD004**: Use `-` for unordered list bullets consistently.
- **MD007**: Indent nested list items by 2 spaces.
- **MD029**: Ordered lists use `1.` style (or sequential), pick one and stay consistent.
- **MD030**: One space after the list marker (`- item`, not `-  item`).

## Code

- **MD040**: Every fenced code block has a language tag (` ```bash `, ` ```python `, ` ```text ` if none fits).
- **MD046**: Use fenced code blocks (```), not indented code blocks.
- **MD048**: Use backtick fences (```), not tildes (~~~).
- **MD014**: Don't prefix shell commands with `$` unless showing output too.

## Inline

- **MD013**: Avoid lines longer than ~120 chars where reasonable; tables, code, and URLs are exempt.
- **MD033**: Avoid raw HTML unless necessary.
- **MD034**: Wrap bare URLs in `<https://...>` or `[text](url)`.
- **MD036**: Don't use bold/italic as a substitute for a heading.
- **MD037/MD038/MD039**: No spaces inside emphasis `**x**`, code `` `x` ``, or link text `[x](url)`.
- **MD042**: No empty links `[]()`.
- **MD049/MD050**: Use `_` for italic and `**` for bold consistently.

## When in doubt

Prefer the simplest construct that renders correctly on GitHub. If a rule conflicts with the user's explicit instruction, follow the user.
