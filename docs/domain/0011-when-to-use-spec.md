---
kind: domain
id: DM-0011
title: When to Use SPEC
status: accepted
date: 2026-08-01
tags: glossary, process
supersedes: 
superseded_by: 
deprecated_by: 
---
# DM-0011: When to Use SPEC

## Status

accepted

## Definition

The canonical gate for recording a SPEC. A SPEC is recorded only when the work is 
1. a commitment - agreed requirements, not expressed desires; 
2. behavioral - it defines the expected observable behavior of a capability, such as what running a command produces, with acceptance criteria; and 
3. narrow - one capability per record. The reasoning behind a choice belongs in an ADR; the SPEC records the agreed behavior that follows from it.

## Avoid

- **feature request** — a request expresses desire, a SPEC records agreed requirements
- **story** — stories are planning units, SPECs are acceptance-bearing records
- **bugfix** — restores behavior a SPEC already defines, so fix the code rather than the record
- **refactor** — no observable behavior change means there is nothing to specify
- **docs/process change** — document agent workflows and skill behavior in AGENTS.md or the relevant SKILL.md

## Relationships

This gate decides when a [SPEC](0003-spec.md) should exist at all; it absorbs and expands the occasion-based Avoid content of that entry, following the pattern set by [When to Use ADR](0004-when-to-use-adr.md). This repository does not use SPECs; the gate stays canonical for projects that do.
