---
kind: adr
id: ADR-0009
title: Use kind-prefixed commands instead of --kind flag
status: accepted
date: 2026-07-30
tags: cli, kinds
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0009: Use kind-prefixed commands instead of --kind flag

## Status

accepted

## Context

Commands that create or list documents select the document kind with a --kind flag (new, init, list, search). Kind-first invocation (canon adr new) reads more naturally and scopes kind-specific flags like --requirements to the kind that uses them.

## Decision

Replace --kind with kind-prefixed subcommands: canon adr new|list|search|init and canon spec new|list|search|init. The --kind flag is removed entirely. Top-level list and search remain for cross-kind views; top-level new and init are removed. Commands taking --id are unchanged since the id prefix already routes the kind.

## Consequences

Breaking CLI contract change: existing agent automations and installed skill versions that emit --kind must migrate. Registry, docs, skill payload, and tests are updated in the same change. Only one way to select a kind remains, keeping examples deterministic for agents.

## History: Accepted

Date: 2026-07-30

Approved by maintainer; implemented in the same change.
