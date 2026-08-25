---
kind: domain
id: DM-0007
title: Corpus
status: accepted
date: 2026-08-01
tags: glossary
supersedes: 
superseded_by: 
deprecated_by: 
---
# DM-0007: Corpus

## Status

accepted

## Definition

The full set of ADR, SPEC, and domain entry files across every present managed directory, treated as one subject of integrity checks. The corpus includes every present supported kind; required kinds determined by repository configuration decide readiness, not corpus membership.

## Avoid

- **repo** — a repository holds all project files, the corpus is only the managed documents
- **collection** — a collection is an arbitrary grouping, the corpus is exactly the managed directories
- **database** — the corpus is plain files, not a structured store

## Relationships

The corpus spans the three [Kind](0006-kind.md)s. Validating the corpus produces [Finding](0008-finding.md)s, at [Shallow](0009-shallow-validation.md) or [Deep](0010-deep-validation.md) depth.
