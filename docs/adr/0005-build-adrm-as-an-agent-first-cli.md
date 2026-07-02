---
id: ADR-0005
title: Build adrm as an agent-first CLI
status: accepted
date: 2026-07-02
tags: 
supersedes: ADR-0001
superseded_by: 
deprecated_by: 
---
# ADR-0005: Build adrm as an agent-first CLI

## Status

accepted

## Context

ADR-0001 established that adrm should be agent-first, but bundled that architectural stance with a list of concrete surface features. The core commitment is worth restating as a narrow decision, with the surface consequences separated out. Agents and downstream tooling need a deterministic, parseable, previewable interface; humans can opt into text output.

## Decision

Build adrm as a CLI whose primary consumer is an autonomous agent. The contract is JSON-first, deterministic, previewable, and composable: every command emits a versioned envelope; mutating commands support --dry-run; stable ADR ids act as selectors; command discovery is machine-readable; and failures carry structured error codes with suggested fixes. Human-readable text output is opt-in via --format text, not the default.

## Consequences

All new commands and output fields must preserve the versioned envelope, dry-run semantics, and structured errors. Agent workflows can rely on stable ids and next_actions. The bundled skill codifies safe agent usage. Future changes to the default output format or to the absence of dry-run support are breaking changes for agents.

## History: Accepted

Date: 2026-07-02

Restates the core agent-first commitment narrowly.
