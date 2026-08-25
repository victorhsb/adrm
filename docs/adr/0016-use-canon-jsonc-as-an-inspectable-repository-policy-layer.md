---
kind: adr
id: ADR-0016
title: Use .canon.jsonc as an inspectable repository policy layer
status: accepted
date: 2026-08-24
tags: cli, config
supersedes: ADR-0015
superseded_by: 
deprecated_by: 
---
# ADR-0016: Use .canon.jsonc as an inspectable repository policy layer

## Status

accepted

## Context

ADR-0015 introduced .canon.jsonc with a single append switch. Repository conventions beyond append (required kinds, validation strictness, lifecycle gates, tag vocabularies) live in prose that agents can miss; a CLI-enforced, inspectable policy is a fact, not a suggestion.

## Decision

.canon.jsonc becomes Canon's inspectable repository policy contract. Scope stays conventions-only: output format, query defaults, storage directories, id and status vocabulary, document structure, and skill install options remain CLI or document-contract concerns. Corpus-wide commands resolve one effective configuration per corpus by reconciling discovery from all three stores and fail with config_scope_mismatch on disagreement. Policy is an enforcement floor: repository settings combine with flags by logical OR, no flag weakens them, there are no bypass flags, and dry runs pass through the same gates as real mutations. canon config show and canon config validate expose the effective configuration and validate it, reporting unknown keys as warnings without rejecting them. schema_version canon.v1 governs compatibility; explicit unsupported versions fail. Defaults preserve prior behavior when the file or a key is absent.

## Consequences

The 'first and only key' constraint of ADR-0015 is obsolete; initial keys cover required kinds, validation strictness, lifecycle reasons, proposed-only creation, and per-kind tag vocabularies. Exact key semantics live in docs/config.md and the test suite, not in this record. Corpora with mixed store roots fail closed until their directory flags point at one policy scope.

## History: Accepted

Date: 2026-08-24

Approved in the policy-layer plan: the corpus needs an inspectable, enforced configuration contract before adopting required kinds, strictness, lifecycle, and tag policies.
