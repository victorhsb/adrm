---
kind: domain
id: DM-0002
title: Domain Model
status: accepted
date: 2026-08-01
tags: glossary
supersedes: 
superseded_by: 
deprecated_by: 
---
# DM-0002: Domain Model

## Status

accepted

## Definition

The full set of accepted domain entries; the project's single source of truth for what things mean. Each entry defines one canonical concept: its meaning, the terms to avoid and why, and its relationships to other concepts.

## Avoid

- **glossary** — a glossary is a flat word list, the Domain Model is lifecycle-managed canonical definitions
- **dictionary** — descriptive of all usage, the Domain Model is prescriptive about canonical usage
- **CONTEXT.md** — that file is a derived snapshot artifact, not the concept

## Relationships

The Domain Model defines the concepts that [ADR](0001-adr.md)s and [SPEC](0003-spec.md)s talk about. Agents search it before and during planning and update it when terms crystallize.
