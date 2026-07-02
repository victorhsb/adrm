---
id: ADR-0004
title: Maintain reciprocal supersede relationships
status: accepted
date: 2026-07-02
tags: 
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0004: Maintain reciprocal supersede relationships

## Status

accepted

## Context

The adrm data model includes both a superseded_by field on the old ADR and a supersedes list on the replacement ADR. Currently, the supersede command only updates the old ADR, leaving supersedes empty on the replacement. This makes the relationship one-directional in storage, complicates validation, and will block future graph and related queries.

## Decision

The supersede command updates both ADRs: it sets status=superseded and superseded_by on the old ADR, and adds the old ADR id to the replacement ADR's supersedes list. Both files are included in the mutation plan and saved sequentially.

## Consequences

Supersede relationships are queryable from either direction. The validate command can enforce reciprocal consistency. Future graph and related commands can rely on the supersedes field. The save is still non-atomic; full transactional safety remains a future write-safety concern.

## History: Accepted

Date: 2026-07-02

Needed before superseding ADR-0001 cleanly.
