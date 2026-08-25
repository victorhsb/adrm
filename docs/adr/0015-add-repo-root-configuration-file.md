---
kind: adr
id: ADR-0015
title: Add repo-root configuration file
status: superseded
date: 2026-08-20
tags: 
supersedes: 
superseded_by: ADR-0016
deprecated_by: 
---
# ADR-0015: Add repo-root configuration file

## Status

superseded

## Context

Repository conventions like disabling append were prose in AGENTS.md that agents could miss; a CLI-enforced setting is a fact, not a suggestion.

## Decision

canon reads an optional .canon.jsonc at the repo root (JSONC, stdlib-parsed), discovered by walking up from the document store directory, holding conventions only; the first and only key is conventions.append, which disables the append command when false.

## Consequences

canon gains a configuration axis that may grow over time; appendices from before the change remain in documents.

## History: Accepted

Date: 2026-08-20

Approved during design discussion.

## History: Superseded

Date: 2026-08-24

Superseded by ADR-0016. ADR-0016 replaces the single-key configuration file with the repository policy layer contract.
