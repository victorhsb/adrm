---
kind: spec
id: SPEC-0001
title: Corpus validation command
status: accepted
date: 2026-07-30
tags: cli, validation
supersedes: 
superseded_by: 
deprecated_by: 
---
# SPEC-0001: Corpus validation command

## Status

proposed

## Context

Agents operating on ADR/SPEC corpora need to detect integrity problems — malformed files, duplicate ids, broken supersede references — before and after mutations. Today doctor only checks directory readiness, and parsing aborts on the first malformed file, so one bad file masks the rest of the corpus. The project roadmap calls for a validate command covering ADR metadata, broken supersede references, duplicate ids, and malformed files.

## Requirements

Add a read-only validate command family (canon validate, canon validate --id, canon adr validate, canon spec validate) that reports corpus integrity findings. Checks: (1) directory existence; (2) per-file parseability, isolated so one malformed file does not mask others; (3) duplicate ids across files; (4) broken references (supersedes, superseded_by, deprecated_by pointing at nonexistent ids); (5) reference reciprocity per ADR-0004 (A.superseded_by=B requires B.supersedes to contain A); (6) status validity (proposed, accepted, rejected, superseded, deprecated); (7) status/reference consistency (status superseded without superseded_by, and vice versa; deprecated without deprecated_by); (8) date format YYYY-MM-DD; (9) kind/id-prefix/directory coherence (kind field must not contradict the id prefix; a SPEC-prefixed id must not live in the ADR directory). Severity model: errors are malformed file, duplicate ids, broken references, invalid status, kind/id/directory contradictions, reciprocity violations; warnings are missing directory, status/reference inconsistency, malformed date. Envelope status is error if any error, else warning if any warning, else ok. Exit code is 4 when any error exists, else 0; a --strict flag also exits 4 when only warnings exist. Output contains findings only (no per-file ok entries) plus a summary object with files_checked, errors, warnings, ordered deterministically by path then check name. Each finding extends the Diagnostic shape with optional path and id fields and carries a concrete suggested_fix.

## Constraints

validate never mutates the corpus; remediation is expressed only through suggested_fix. Output remains JSON-first with schema_version; text format is a rendering only. Deterministic ordering in all output. Standard library only. The doctor command keeps its existing output contract while delegating to the shared validation engine in shallow mode.

## Acceptance Criteria

canon validate on a healthy corpus exits 0 with status ok and a summary of files_checked. A corpus containing a duplicate id exits 4 with status error and a duplicate_id finding naming both paths. A corpus with a superseded_by reference to a nonexistent id exits 4 with a broken_reference finding. A corpus whose only problem is a missing docs/spec directory exits 0 with status warning, and exits 4 with --strict. canon validate --id ADR-0001 validates only that document and its references. canon adr validate reports findings scoped to the ADR directory. canon doctor output remains byte-compatible with its pre-change contract. go test ./... passes with new tests exercising Run through the JSON envelope.

## Appendix: Future extensions

Date: 2026-07-30

A --fix flag for automated remediation (repairing reciprocity links, normalizing dates) was considered and deliberately deferred. Most findings have ambiguous resolutions that need human or agent judgment, and auto-fixing reciprocity would hide hand-edits that deserve scrutiny. Revisit once the check catalog proves stable in practice.
