---
id: ADR-0001
title: Use an agent-first CLI for ADR management
status: accepted
date: 2026-06-27
tags: adr, agents, cli
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0001: Use an agent-first CLI for ADR management

## Status

accepted

## Context

Agents need a reliable way to manage Architecture Decision Records without relying on fragile prose parsing or interactive workflows.

## Decision

Build adrm as a JSON-first CLI with command discovery, dry-run support for mutations, structured errors, stable ADR ids, query commands, lifecycle commands, appendix support, and a bundled skill command.

## Consequences

Future CLI changes must preserve machine-readable output, previewable mutations, structured recovery guidance, and composable selectors.
