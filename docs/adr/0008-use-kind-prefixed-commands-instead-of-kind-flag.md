---
kind: adr
id: ADR-0008
title: Use kind-prefixed commands instead of --kind flag
status: accepted
date: 2026-07-30
tags: cli, kinds
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0008: Use kind-prefixed commands instead of --kind flag

## Status

accepted

## Context

Commands that create or list documents must define what is the type of document being dealt with. Initially done with the --kind flag, but replaced with kind-prefixed subcommands for the sake of simplifying the whole flow. When a prefix is not set for reading commands we default to reading both. For write commands it needs to return an error.

## Decision

Replace `--kind` with kind-prefixed subcommands: `canon adr new|list|search|init` and `canon spec new|list|search|init`. The `--kind` flag is removed entirely. Top-level list and search remain for cross-kind views; top-level new and init are removed. Commands taking --id are unchanged since the id prefix already routes the kind.

## Consequences

Breaking CLI contract change: existing agent automations and installed skill versions that emit --kind must migrate. Registry, docs, skill payload, and tests are updated in the same change. Only one way to select a kind remains, keeping examples deterministic for agents.
That's an expected consequence and shouldn't bring any trouble since the project is very young and only really used on branchless-pr right now.

## History: Accepted

Date: 2026-07-30

Approved by maintainer; implemented in the same change.
