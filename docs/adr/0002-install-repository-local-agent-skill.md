---
id: ADR-0002
title: Install repository-local agent skill
status: accepted
date: 2026-06-27
tags: agents, cli, skill
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0002: Install repository-local agent skill

## Status

accepted

## Context

Agents need a repository-local skill file for ADRM workflows, and the skill source should be separable so it can move to its own project later.

## Decision

Provide skill install and update commands that write the ADRM skill to a repo-local skill directory. Keep the skill content behind a separate Go package boundary so it can be extracted later without depending on CLI internals.

## Consequences

Skill installation becomes part of the CLI contract. Mutating skill commands must support dry-run, structured errors, deterministic output, and documented update behavior.
