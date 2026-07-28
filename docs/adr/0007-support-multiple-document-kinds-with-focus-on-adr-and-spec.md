---
kind: adr
id: ADR-0007
title: Support multiple document kinds with focus on ADR and SPEC
status: accepted
date: 2026-07-22
tags: adr, cli, kinds, spec
supersedes:
superseded_by:
deprecated_by:
---
# ADR-0007: Support multiple document kinds with focus on ADR and SPEC

## Status

accepted

## Context

adrm only managed ADRs, but agent workflows also need to capture functional requirements. A single CLI should manage multiple kinds while keeping the ADR focus.

## Decision

Introduce a kind field (adr or spec) in front matter, separate storage directories with independent numbering, route commands by kind selector or id prefix, and reject cross-kind supersede.

## Consequences

ADRs and SPECs share lifecycle commands and parseable shape. Existing ADR files without a kind field default to adr. Supersede is constrained to the same kind.

## History: Accepted

Date: 2026-07-22

Captures the multi-kind design now implemented by the CLI.
