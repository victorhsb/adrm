---
kind: domain
id: DM-0010
title: Deep validation
status: accepted
date: 2026-08-01
tags: glossary
supersedes: 
superseded_by: 
deprecated_by: 
---
# DM-0010: Deep validation

## Status

accepted

## Definition

Integrity checks answering 'is my corpus healthy?': metadata validity, duplicate ids or titles, reference integrity, reciprocity, and configured metadata conventions such as allowed tag vocabularies.

## Avoid

- **linting** — linting enforces style, deep validation checks integrity
- **full scan** — a full scan describes effort, not the integrity question being answered

## Relationships

Deep validation is the integrity half of validating the [Corpus](0007-corpus.md); the readiness half is [Shallow validation](0009-shallow-validation.md). Both produce [Finding](0008-finding.md)s.
