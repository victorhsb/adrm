---
kind: adr
id: ADR-0017
title: Use an optional rebuildable JSONL cache for search
status: accepted
date: 2026-08-26
tags: cli
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0017: Use an optional rebuildable JSONL cache for search

## Status

accepted

## Context

Search reparses the authoritative Markdown corpus on every invocation. This preserves correctness, but file I/O and parsing scale linearly with corpus size. A cache could avoid repeated parsing, but it could also become a second source of truth, change query semantics, expose corpus content outside the repository, or add a dependency. DM-0007 defines the Corpus as the Markdown files in each present managed directory. ADR-0016 limits `.canon.jsonc` to repository conventions rather than query defaults.

## Decision

We decided to use an optional, versioned JSONL cache for search. Each corpus has a distinct cache in the user's cache area. The cache is a derived, disposable artifact outside the Corpus defined by DM-0007. Markdown remains the sole authority, so deleting the cache affects search performance rather than corpus correctness.

Search reads Markdown by default, and callers explicitly opt into the cache. If Canon cannot trust the cache, search falls back to Markdown instead of failing or hiding results. Search never rebuilds the cache implicitly. Only search may read the cache. Corpus inspection, validation, and lifecycle operations continue to read Markdown.

JSONL keeps the cache inspectable, deterministic, rebuildable, and implementable with the Go standard library. Indexed search preserves the existing matching and ordering semantics. Index enablement remains a runtime choice outside `.canon.jsonc`. SQLite, ranking, and semantic search require separate decisions because they would change dependencies or query semantics.

## Consequences

Cache failures reduce indexed-search performance instead of corpus correctness. JSONL favors inspectability and dependency-free rebuilding over richer query capabilities. The cache duplicates full document content outside the repository, so its location must remain inspectable and its contents private. Deleting the user cache is always safe.

## History: Accepted

Date: 2026-08-26

Phase B gate: decides the optional rebuildable JSONL search index contract; SQLite and semantic search remain separate future decisions.
