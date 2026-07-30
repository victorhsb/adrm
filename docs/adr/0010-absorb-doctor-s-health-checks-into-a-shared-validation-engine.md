---
kind: adr
id: ADR-0010
title: Absorb doctor's health checks into a shared validation engine
status: proposed
date: 2026-07-30
tags: cli, validation
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0010: Absorb doctor's health checks into a shared validation engine

## Status

proposed

## Context

canon doctor checks repository readiness (directory existence, parseability). The roadmap calls for a validate command with deep corpus integrity checks (SPEC-0001). Three topologies were considered: validate as a fully separate command leaving doctor untouched; validate absorbing doctor as a pure alias; or doctor upgraded in place with no validate command. Duplicated check logic across two commands would drift, and a pure alias would sacrifice doctor's cheap pre-mutation readiness check.

## Decision

Implement corpus validation as one shared engine in internal/canon. canon validate runs the full check catalog from SPEC-0001; canon doctor becomes the engine's shallow mode (directory existence and parseability only) and keeps its existing output contract byte-compatible. This ADR records only the command-topology commitment; the validate requirements, severities, and acceptance criteria live in SPEC-0001.

## Consequences

Shallow checks exist in exactly one place, so doctor and validate cannot disagree about readiness. Doctor stays fast enough to run before every mutation. New health checks land in one engine and surface in both commands according to their mode. Agents gain a clear mental model: doctor answers 'can I work here?', validate answers 'is my corpus healthy?'
