---
kind: domain
id: DM-0012
title: When to Use Domain Entry
status: accepted
date: 2026-08-01
tags: glossary, process
supersedes: 
superseded_by: 
deprecated_by: 
---
# DM-0012: When to Use Domain Entry

## Status

accepted

## Definition

The canonical gate for recording a domain entry. An entry is recorded only when the concept is 
1. relevant - the term appears in the project's documents or decisions and bears on what an agent or human decides, or another entry's definition relies on it; 
2. load-bearing - a wrong interpretation would steer the reader, LLM or human, in the wrong direction; 
3. non-obvious - the concept is used differently from common sense, or carries project-specific customizations worth an explicit redefinition; 
4. narrow - one concept per entry; and 
5. sustains its own weight - it answers a distinct reader question standalone: not so heavy it must split into two entries, not so shallow it is absorbed by another definition or deleted entirely.

## Avoid

- **glossary entry** — a glossary entry is a flat word definition, a domain entry is lifecycle-managed with rejected wording and relationships
- **dictionary term** — a dictionary term is descriptive of all usage, a domain entry is prescriptive about canonical usage
- **ephemeral term** — wording still in flux, entries record usage that has settled in the corpus
- **implementation detail** — internals no reader needs in order to interpret documents or decisions
- **project process** — document agent workflows and skill behavior in AGENTS.md or the relevant SKILL.md

## Relationships

This gate decides when a [Domain Entry](0005-domain-entry.md) should exist at all, and so guards membership in the [Domain Model](0002-domain-model.md). It follows the pattern set by [When to Use ADR](0004-when-to-use-adr.md) and, as a domain entry itself, must pass its own gate.
