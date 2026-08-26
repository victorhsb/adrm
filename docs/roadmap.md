# Roadmap

This project is intentionally starting with a small, dependency-free CLI. The
next steps should preserve agent-friendly behavior as the command surface grows.

## Near term

- Add `accept` and `reject` commands for explicit lifecycle transitions.
- Add richer search output with matched fields and line references.
- Add JSON schema files for command envelopes and ADR payloads.

## Querying improvements

- ~~Add a local index for larger ADR sets.~~ Implemented as an optional,
  rebuildable JSONL cache (`canon index status`, `canon index rebuild`,
  `search --use-index`; ADR-0017). Semantic search, ranking, and SQLite
  remain future work and require their own decisions.
- Support filters for date ranges, superseded relationships, and multiple tags.
- Add `graph` output for supersede/deprecation relationships.
- Add `related --id` to return linked or textually similar ADRs.

## Write safety

- Add `--expect-status` to status-changing commands to prevent stale-state writes.
- Add `--expect-hash` or equivalent optimistic concurrency guards.
- Add `--plan-file` so agents can save and apply an exact mutation plan.

## Documentation

- Add examples using real ADR corpora.
- Add a contributor guide once the internal package layout stabilizes.

## Packaging

- Add release builds for common platforms.
- Add install instructions for Homebrew and `go install`.
- Add shell completions after the command surface settles.
