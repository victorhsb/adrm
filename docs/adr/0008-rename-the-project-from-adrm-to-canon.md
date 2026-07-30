---
kind: adr
id: ADR-0008
title: Rename the project from adrm to canon
status: accepted
date: 2026-07-30
tags: cli, rename
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0008: Rename the project from adrm to canon

## Status

accepted

## Context

The tool manages both Architecture Decision Records and software specifications (spec-driven development). The name adrm reflects only the ADR half. canon (as in canonical) reflects the broader scope: the tool manages every canonical record of the codebase. The rename affects the Go module path, binary name, CLI examples and next_actions strings, the JSON schema_version value (adrm.v1 -> canon.v1), the agent skill name/markers/install path, build scripts, and documentation.

## Decision

Rename the project, binary, Go module, packages, skill, and schema_version from adrm to canon. Keep ADR- and SPEC- id prefixes, --kind values, and the docs/adr and docs/spec directories unchanged: those are domain terms, so existing document data stays valid. Historical ADRs keep the old name in their content because ADRs are immutable records. The schema_version change to canon.v1 is accepted as a clean break; existing .agents/skills/adrm installs are not auto-migrated and are documented as a manual step.

## Consequences

Every JSON response now reports schema_version canon.v1; any automation parsing adrm.v1 must be updated. CLI examples and next_actions use the canon binary name. Previously installed adrm skill files at .agents/skills/adrm become unmanaged and must be replaced by canon skill install. The GitHub repository and local remotes must be renamed separately. ADR-0005 remains as historical context but the tool name it references is superseded by this decision.

## History: Accepted

Date: 2026-07-30

Rename approved by the maintainer; executed in the same change.
