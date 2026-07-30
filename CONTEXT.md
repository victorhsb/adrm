# Canon

Canon manages Architecture Decision Records and Specs for agent-led workflows. This glossary pins the canonical language for documents and corpus health.

## Language

### Documents

**ADR**:
An architecture decision record: a dated, narrowly-scoped commitment with context, decision, and consequences.
_Avoid_: design doc, ticket, changelog entry

**SPEC**:
A functional requirements record: context, requirements, constraints, and acceptance criteria for a capability.
_Avoid_: feature request, story

**Kind**:
The document type, either `adr` or `spec`. Kinds share a parseable shape but live in separate directories with independent numbering.
_Avoid_: category, type flag

### Corpus health

**Corpus**:
The full set of ADR and SPEC files across both directories, treated as one subject of integrity checks.
_Avoid_: repo, collection, database

**Finding**:
A single validation result about the corpus: a check name, a severity, a location, and a suggested fix.
_Avoid_: error, issue, violation

**Shallow validation**:
Readiness checks answering "can I work here?": directories exist and files parse.
_Avoid_: doctor checks (that is a command name, not the concept)

**Deep validation**:
Integrity checks answering "is my corpus healthy?": metadata validity, duplicate ids, reference integrity, and reciprocity.
_Avoid_: linting, full scan
