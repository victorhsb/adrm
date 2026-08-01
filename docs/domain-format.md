# Domain Entry File Format

`canon` stores domain entries as markdown files that share the ADR front-matter
shape. A domain entry defines one canonical concept; the full set of accepted
entries is the project's Domain Model, the single source of truth for what
things mean. The `kind` front-matter field (`domain`) distinguishes entries
from ADRs and SPECs.

Default domain directory:

```text
docs/domain
```

Default filename pattern:

```text
0001-adr.md
```

Override the directory with the global flag:

```sh
canon --domain-dir docs/domain domain new --title "ADR"
```

## Front matter

```markdown
---
kind: domain
id: DM-0001
title: ADR
status: proposed
date: 2026-08-01
tags: glossary
supersedes:
superseded_by:
deprecated_by:
---
```

Fields mirror the ADR front matter. The differences are:

- `kind`: `domain`.
- `id`: stable domain entry id. Format: `DM-0001`. Numbering is independent of
  ADRs and SPECs.
- `title`: the canonical term being defined. One concept per entry.

## Body

New domain entries use this body shape:

```markdown
# DM-0001: ADR

## Status

proposed

## Definition

A dated, narrowly-scoped architecture commitment with context, decision, and
consequences.

## Avoid

- **design doc** — too broad; an ADR is one dated decision, not an evolving design overview
- **ticket** — tickets track work; ADRs record commitments

## Relationships

An ADR records the decision a [SPEC](0003-spec.md)'s requirements depend on.
```

Section conventions:

- **Definition**: what the concept is. Definitions carry no implementation
  details; an entry says what a thing means, not how it is built.
- **Avoid**: terms that must not be used for this concept, each with a reason.
  The `--avoid` flag takes `term: reason` pairs separated by semicolons and
  renders them as bullets.
- **Relationships**: free text referencing other entries with relative
  markdown links, e.g. `[SPEC](0003-spec.md)` between entries, or
  `[ADR](../domain/0001-adr.md)` from ADR/SPEC files. Links make the concept
  graph machine-extractable without any structured front matter.

The front matter is the source of truth for parsing. The Status section is a
human-readable mirror updated by the lifecycle commands.

## Lifecycle

Domain entries use the same lifecycle commands as ADRs and SPECs: `accept`,
`reject`, `supersede`, `deprecate`, and `append`. Supersede must stay within
the same kind. The verbs carry these meanings for entries:

- **accept**: the term is canonical; challenge fuzzy usage against it.
- **reject**: rare. A proposed entry that does not survive review is usually
  deleted instead; reject only when the refusal should stay on record.
- **supersede**: redefinition only. A changed meaning gets a new entry that
  supersedes the old one, preserving history.
- **rename**: not a lifecycle event. Retitle the same entry in place (title is
  content, not lifecycle metadata) and record the former name with
  `canon append --id DM-0001 --title "Renamed" --body "Formerly: Session."`.
  Whether a rename should also rename the file is an open question; ids, not
  filenames, are the identity.
- **deprecate**: the concept is leaving the domain. The entry stays as a
  tombstone so historical references still resolve and future implementations
  know to intentionally ignore it.

## Integrity checks

`canon doctor` runs two content-level checks on the domain model:

- `domain_duplicate_title`: two accepted entries share a title. Deprecate or
  supersede all but one; each concept has a single truth.
- `domain_dead_reference`: a live document references (by `DM-` id or markdown
  link) an entry that is superseded or deprecated. Update the reference to the
  successor shown in `superseded_by`, or remove it.

## History and Appendices

History and appendix sections look identical to the ADR equivalents. Use
appendices for clarification notes, renames, and review outcomes.
