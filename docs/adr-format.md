# ADR File Format

`adrm` stores ADRs as markdown files with a small front matter block. ADRs and
SPECs share the same parseable shape; the `kind` front-matter field selects the
document type. See `docs/spec-format.md` for the SPEC body.

Default ADR directory:

```text
docs/adr
```

Default filename pattern:

```text
0001-use-sqlite-for-local-index.md
```

## Front matter

```markdown
---
kind: adr
id: ADR-0001
title: Use SQLite for local index
status: proposed
date: 2026-06-27
tags: storage, query
supersedes:
superseded_by:
deprecated_by:
---
```

Fields:

- `kind`: document kind. `adr` for architecture decisions. Absent values
  default to `adr` for backward compatibility with older ADR files.
- `id`: stable ADR id. Format: `ADR-0001`.
- `title`: short decision title.
- `status`: one of `proposed`, `accepted`, `rejected`, `superseded`, `deprecated`.
- `date`: creation date in `YYYY-MM-DD`.
- `tags`: comma-separated tags.
- `supersedes`: comma-separated ids this ADR replaces. The `supersede` command populates this field on the replacement ADR.
- `superseded_by`: id of the ADR that replaces this one. The `supersede` command populates this field on the superseded ADR.
- `deprecated_by`: marker for deprecation source. The current CLI writes `manual`.

## Body

New ADRs use this body shape:

```markdown
# ADR-0001: Use SQLite for local index

## Status

proposed

## Context

Agents need fast local lookup.

## Decision

Use SQLite-backed indexes.

## Consequences

Local search has a durable index that can be rebuilt.
```

The front matter is the source of truth for parsing. The Status section is updated
as a human-readable mirror when status-changing commands run.

## Appendices

`adrm append` adds sections like:

```markdown
## Appendix: Implementation note

Date: 2026-06-27

The initial rollout used the default local index.
```

Use appendices for post-decision notes, implementation observations, review
outcomes, and follow-up context that does not replace the original decision.

## History

Status-changing commands add history sections:

```markdown
## History: Superseded

Date: 2026-06-27

Superseded by ADR-0002. ADR-0002 captures the current design.
```

History sections preserve the reason for status changes in the ADR itself.
