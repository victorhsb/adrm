---
id: ADR-0003
title: Use a four-test model for ADR-worthiness in the agent skill
status: accepted
date: 2026-07-01
tags: 
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0003: Use a four-test model for ADR-worthiness in the agent skill

## Status

accepted

## Context

Agents operating adrm need a shared gate for deciding when to create or change an ADR. Without a model, agents record intentions as decisions, bundle unrelated choices, or write product strategy into ADRs. The project already lists surfaces that typically require ADRs (CLI contract, file format, query behavior, lifecycle semantics, output schema, storage layout, agent operating model), but that list names where commitments live, not how to recognize a commitment worth recording.

## Decision

Encode a four-test model in the bundled agent skill. An ADR is recorded only when a decision is (1) a commitment, not an intention; (2) architectural, shaping structure, contract, data model, or cross-cutting policy; (3) non-obvious, with reasoning worth preserving; and (4) narrow, one decision per ADR. Product decisions belong in Context as forces, and only their resulting architectural commitments belong in Decision. The skill also lists anti-patterns: roadmap-as-ADR, task-as-ADR, changelog-as-ADR, bundled decisions, product strategy without architectural consequence, vague commitments, and obvious decisions with no real alternatives.

## Consequences

Agents will create fewer, higher-quality ADRs. ADRs will be independently supersede-able because each is narrow. The skill itself becomes the source of truth for the model, so any change to the model requires a new ADR and a skill update.

## History: Accepted

Date: 2026-07-01

Model is needed before agents create more ADRs.
