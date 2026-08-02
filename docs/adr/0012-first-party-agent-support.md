---
kind: adr
id: ADR-0012
title: first-party-agent-support
status: accepted
date: 2026-08-02
tags: agents, integrations, support
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0012: first-party-agent-support

## Status

accepted

## Context

Agent environments use different skill discovery, configuration, and invocation conventions. Treating every environment as first-party would expand Canon's documentation, installation, compatibility, and test obligations without a bounded support contract.

## Decision

Canon provides and maintains first-party agent integrations only for OpenCode, Claude, and Codex. Other agents may use Canon's public CLI and skill contracts, but Canon does not provide dedicated integration, compatibility guarantees, or first-party support for them.

## Consequences

Integration documentation, installers, compatibility work, and tests can focus on three named environments. Users of other agents can still compose against Canon's public contracts, but environment-specific support is community-maintained or unsupported. Adding or removing a first-party environment requires revisiting this decision.

## History: Accepted

Date: 2026-08-02

Accepted: bounds first-party integration scope to OpenCode, Claude, and Codex.
