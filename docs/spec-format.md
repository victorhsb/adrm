# SPEC File Format

`canon` stores SPECs as markdown files that share the ADR front-matter shape. A
SPEC captures functional requirements; an ADR captures architecture decisions.
The `kind` front-matter field (`spec`) distinguishes the two.

Default SPEC directory:

```text
docs/spec
```

Default filename pattern:

```text
0001-local-query-index.md
```

Override the directory with the global flag:

```sh
canon --spec-dir docs/spec spec new --title "Local query index"
```

## Front matter

```markdown
---
kind: spec
id: SPEC-0001
title: Local query index
status: proposed
date: 2026-07-22
tags: storage, query
supersedes:
superseded_by:
deprecated_by:
---
```

Fields mirror the ADR front matter. The differences are:

- `kind`: `spec`.
- `id`: stable SPEC id. Format: `SPEC-0001`. Numbering is independent of ADRs.

## Body

New SPECs use this body shape:

```markdown
# SPEC-0001: Local query index

## Status

proposed

## Context

Agents need fast local lookup of decision records.

## Requirements

Return ADRs by tag and status without scanning every file.

## Constraints

No external dependencies; the index must be rebuildable from markdown.

## Acceptance Criteria

`canon list --tag storage` returns ADR-0001 in deterministic order.
```

The front matter is the source of truth for parsing. The Status section is a
human-readable mirror updated by the lifecycle commands.

## Lifecycle

SPECs use the same lifecycle commands as ADRs: `accept`, `reject`, `supersede`,
`deprecate`, and `append`. Supersede must stay within the same kind: a SPEC can
only supersede another SPEC, and an ADR can only supersede an ADR.

## History and Appendices

History and appendix sections look identical to the ADR equivalents. Use
appendices for clarification notes and review outcomes.
