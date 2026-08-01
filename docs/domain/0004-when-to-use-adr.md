---
kind: domain
id: DM-0004
title: When to Use ADR
status: accepted
date: 2026-08-01
tags: glossary, process
supersedes: 
superseded_by: 
deprecated_by: 
---
# DM-0004: When to Use ADR

## Status

accepted

## Definition

The canonical gate for recording an architecture decision. An ADR is recorded only when the decision is (1) a commitment, not an intention - past tense, we decided X; (2) architectural - it affects the CLI contract, document file formats, query behavior, lifecycle semantics, output schema, or storage layout; (3) non-obvious - reasonable people could choose differently, so the reasoning is worth preserving; and (4) narrow - one decision per record. Product drivers belong in the ADR's Context as forces; only the architectural commitment they force belongs in Decision.

## Avoid

- **roadmap item** — intentions and plans are not commitments
- **ticket** — tasks track work, ADRs record decisions
- **changelog entry** — changelogs describe what changed, ADRs explain why
- **product strategy** — belongs in an ADR only when it forces an architectural commitment
- **project process** — agent workflows and skill behavior belong in AGENTS.md or the relevant SKILL.md, not in ADRs

## Relationships

This gate decides when an [ADR](0001-adr.md) should exist at all. The bundled canon skill teaches the rubric generically to any project; this entry is its canonical definition for canon itself. ADR-0003 first adopted the model and is deprecated in favor of this entry.
