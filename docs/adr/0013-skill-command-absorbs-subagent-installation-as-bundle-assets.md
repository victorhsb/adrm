---
kind: adr
id: ADR-0013
title: Skill command absorbs subagent installation as bundle assets
status: accepted
date: 2026-08-02
tags: agents, cli, skill
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0013: Skill command absorbs subagent installation as bundle assets

## Status

accepted

## Context

ADR-0002 established skill install/update for a single embedded skill. Since then the canon skill bundle grew: the canon-record-gate skill (a multi-file payload with supporting references) and the canon-critic subagent exist only as hand-maintained repo-local files and cannot be installed into target projects. ADR-0012 bounds first-party agent environments to OpenCode, Claude, and Codex, each with its own subagent discovery and frontmatter conventions. The rejected alternative was a separate agent command group treating subagents as independent assets.

## Decision

A subagent is modeled as a component of the skill asset bundle: the agent is a wrapper around the skill, so the agent belongs to the skill and not the other way around. The skill command family absorbs subagent installation: canon skill install and canon skill update deploy all bundled assets - multiple skills, multi-file skill payloads, and their subagents rendered per target environment - as one managed bundle. No separate agent command group is introduced.

## Consequences

The CLI contract established by ADR-0002 extends to multi-asset, multi-file, and per-target installation; the same dry-run, structured-error, deterministic-output, and update-discipline requirements apply to every file in the bundle. Target-selection mechanics, directory mappings, and frontmatter rendering are observable behavior specified in a SPEC, not architecture. Bundled asset content must remain project-agnostic because it ships to other projects. Adding or removing a bundled asset changes the embedded payload, not the command contract; adding a first-party environment remains governed by ADR-0012.
