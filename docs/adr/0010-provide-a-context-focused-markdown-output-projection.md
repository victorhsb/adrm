---
kind: adr
id: ADR-0010
title: Provide a context-focused Markdown output projection
status: accepted
date: 2026-07-30
tags: agents, cli, context, output
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0010: Provide a context-focused Markdown output projection

## Status

accepted

## Context

Agents can inject Canon document inventories into system prompts, but the default JSON envelope and human-oriented text output include metadata and guidance that consume context without improving discovery. Keeping prompt rendering in each integration would duplicate filtering and formatting logic.

## Decision

Add context as an explicit opt-in output format while keeping JSON as the default. Initially, canon list, canon adr list, and canon spec list render deterministic Markdown containing only a heading and each matching document's stable id and title; they omit envelopes, counts, metadata, warnings, and next_actions. Existing selectors remain authoritative, so callers use --status accepted explicitly rather than the format changing query semantics. Commands without a defined context projection reject the format. Future commands may support context only after defining their own bounded, deterministic Markdown projection.

## Consequences

OpenCode hooks and other prompt-building integrations can inject concise, stable context without parsing or exposing Canon's full response envelope. Canon owns the projection while callers retain control over filtering. The context format is intentionally not machine-parseable JSON and does not replace the default agent contract. Adding context support to another command becomes an explicit output-contract change rather than an automatic generic rendering.
