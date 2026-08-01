---
kind: adr
id: ADR-0012
title: Add the Domain Model as a third document kind
status: accepted
date: 2026-08-01
tags: cli, domain, kinds
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0012: Add the Domain Model as a third document kind

## Status

accepted

## Context

Canon managed the why (ADRs) and the how (SPECs) but not the what. Domain terminology lived in a hand-maintained CONTEXT.md glossary outside canon's lifecycle, search, and integrity tooling, so the corpus had no canonical, tool-maintained source of truth for what things mean.

## Decision

Add a third kind, domain (docs/domain, DM-0001 ids, --domain-dir, canon domain new|list|search|init). One Domain Entry defines one concept; the accepted entries are the Domain Model, the single source of truth for meaning. Entry bodies have Definition (implementation-free), Avoid (term: reason pairs rendered as bullets), and Relationships (free text using relative markdown links). Lifecycle verbs are inherited with pinned semantics: supersede means redefinition only; a rename retitles the same entry in place plus a canon append history note; deprecate leaves a concept tombstone. Doctor gains its first content-aware checks, scoped to the domain model: duplicate accepted titles and references from live documents to superseded or deprecated entries. JSON envelope keys stay adr/adrs, schema stays canon.v1, and bare-number ids still resolve to ADR. CONTEXT.md becomes a derived snapshot of the Domain Model.

## Consequences

Agents define, search, and challenge terminology through the same envelope, dry-run, and lifecycle machinery as decisions and requirements. Front matter is unchanged across all three kinds. Doctor is no longer purely shallow, though the new checks are scoped to the domain model. Open question, deliberately undecided: whether renaming a concept should also rename its file (no retitle command exists). Follow-up outside this change: update the user-level domain-modeling skill to prefer canon domain over CONTEXT.md when canon is initialized.

## History: Accepted

Date: 2026-08-01

Decided in a grill-with-docs design review; see ADR-0007 and ADR-0009 for the kind and command-prefix precedents this extends.
