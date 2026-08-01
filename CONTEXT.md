# Canon

Canon manages Architecture Decision Records, Specs, and the Domain Model for agent-led workflows. This glossary pins the canonical language for documents and corpus health.

> This file is a human-friendly snapshot. The single source of truth for what
> things mean is the Domain Model itself: `canon domain list` (see
> `docs/domain/`). Update entries with `canon domain` commands, then refresh
> this snapshot.

## Language

### Documents

**ADR**:
An architecture decision record: a dated, narrowly-scoped commitment with context, decision, and consequences.
_Avoid_: design doc, ticket, changelog entry

**SPEC**:
A functional requirements record: context, requirements, constraints, and acceptance criteria for a capability.
_Avoid_: feature request, story

**Domain Entry**:
A single canonical definition of one domain concept: its meaning, avoided terms with reasons, and relationships to other concepts. Ids look like `DM-0001`; the title is the canonical term.
_Avoid_: glossary entry, dictionary term

**Domain Model**:
The full set of accepted Domain Entries; the project's single source of truth for what things mean.
_Avoid_: glossary, dictionary, CONTEXT.md (this file is an artifact, not the concept)

**Kind**:
The document type: `adr`, `spec`, or `domain`. Kinds share a parseable shape but live in separate directories with independent numbering.
_Avoid_: category, type flag

### Corpus health

**Corpus**:
The full set of ADR, SPEC, and domain entry files across all three directories, treated as one subject of integrity checks.
_Avoid_: repo, collection, database

**Finding**:
A single validation result about the corpus: a check name, a severity, a location, and a suggested fix.
_Avoid_: error, issue, violation

**Shallow validation**:
Readiness checks answering "can I work here?": directories exist and files parse.
_Avoid_: doctor checks (that is a command name, not the concept)

**Deep validation**:
Integrity checks answering "is my corpus healthy?": metadata validity, duplicate ids or titles, reference integrity, and reciprocity.
_Avoid_: linting, full scan
