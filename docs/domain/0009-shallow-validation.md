---
kind: domain
id: DM-0009
title: Shallow validation
status: accepted
date: 2026-08-01
tags: glossary
supersedes: 
superseded_by: 
deprecated_by: 
---
# DM-0009: Shallow validation

## Status

accepted

## Definition

Readiness checks answering 'can I work here?': required directories exist and every present file parses. Directories for kinds the repository configuration does not require may be absent without weakening readiness.

## Avoid

- **doctor checks** — doctor is a command name, not the concept
- **smoke test** — a smoke test exercises behavior, shallow validation only checks storage readiness

## Relationships

Shallow validation is the readiness half of validating the [Corpus](0007-corpus.md); the integrity half is [Deep validation](0010-deep-validation.md). Both produce [Finding](0008-finding.md)s.
