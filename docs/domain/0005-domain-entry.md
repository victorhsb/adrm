---
kind: domain
id: DM-0005
title: Domain Entry
status: accepted
date: 2026-08-01
tags: glossary
supersedes: 
superseded_by: 
deprecated_by: 
---
# DM-0005: Domain Entry

## Status

accepted

## Definition

A single canonical definition of one domain concept: its meaning, avoided terms with reasons, and relationships to other concepts. Ids look like DM-0001; the title is the canonical term.

## Avoid

- **glossary entry** — a glossary entry is a flat word definition, a domain entry is lifecycle-managed with rejected wording and relationships
- **dictionary term** — a dictionary term is descriptive of all usage, a domain entry is prescriptive about canonical usage

## Relationships

Domain entries are the members of the [Domain Model](0002-domain-model.md). [Kind](0006-kind.md) distinguishes domain entries from [ADR](0001-adr.md)s and [SPEC](0003-spec.md)s.
