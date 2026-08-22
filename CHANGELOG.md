# Changelog

## Unreleased

- Added `canon version`, printing the build version (`dev` for source builds;
  release builds inject it via `-ldflags "-X .../internal/canon.Version=vX.Y.Z"`).

## v0.2.0 - 2026-08-04

- Renamed the project from `adrm` to `canon` (ADR-0007).
- Added the Domain Model as a third document kind (ADR-0011): entries live in
  `docs/domain` with `DM-0001`-style ids and capture one canonical concept per
  entry via `--definition`, `--avoid`, and `--relationships`. The set of
  accepted entries is the single source of truth for what things mean.
- Replaced the `--kind` flag with kind-prefixed commands (ADR-0008):
  `adr`/`spec`/`domain new|list|search|validate|init`. Plain `list`,
  `search`, and `validate` cover all kinds, and commands taking `--id` route
  by id prefix.
- Added `canon validate`, a deep corpus integrity catalog covering
  malformed files, duplicate ids, broken references, reciprocity, status and
  date validity, and kind/id/directory coherence. It runs through a shared
  validation engine (ADR-0009) that absorbed `doctor`'s checks: `doctor` is
  now the engine's shallow mode and additionally flags domain-model integrity
  problems (duplicate accepted titles, references to superseded or deprecated
  entries). A missing `docs/spec` or `docs/domain` directory is a warning, not
  an error.
- Added `--format context` for list commands (ADR-0010), a bounded Markdown
  output projection intended for prompt injection.
- Added the `canon-record-gate` skill and the `canon-critic` read-only
  subagent, which judges whether an ADR, SPEC, or domain entry earns its place
  before creation or after acceptance (ADR-0012).
- Changed the `skill` command to manage a bundle of assets (ADR-0013):
  `skill install` installs the full bundle and `skill update` previews and
  applies updates with per-asset versions and per-file hashes, refusing
  locally modified or unmanaged files unless `--force` is given. Added Codex
  custom-agent rendering alongside OpenCode and Claude targets.
- Added CI workflows for tests, lint, and corpus QA (`validate` on
  `docs/**` changes; `doctor` is the validation engine's shallow mode, so a
  passing `validate` implies a passing `doctor`), and a project website with
  Canon branding.
- Declared support for macOS and Linux only; Windows is not supported
  (ADR-0014).

## v0.1.0 - 2026-07-29

First release, as `adrm`: an agent-first CLI for managing Architecture
Decision Records (ADRs) and Specs (SPECs) (ADR-0001, ADR-0005, ADR-0006).

- Implemented `init`, `new`, `list`, `show`, `search`, `accept`, `reject`,
  `supersede`, `deprecate`, and `append` commands for ADR and SPEC kinds,
  selected via a `--kind` flag.
- Added JSON output envelopes with `schema_version` as the default for every
  command, with `--format text` (`-t`) for human-readable rendering.
- Added `--dry-run` to all mutating commands; dry-run responses include the
  warning `No changes were made.`
- Added reciprocal `supersede` handling so both documents stay in sync
  (ADR-0004).
- Added structured errors with `code`, `category`, `message`, and
  `suggested_fix`, plus `next_actions` in command outputs.
- Added the `commands` machine-readable command registry and `doctor` storage
  health checks.
- Added the bundled, repository-local agent skill (ADR-0002) with embedded
  assets, `skill install` and `skill update` helpers, and ADR-worthiness
  guidance (ADR-0003).
- Added `scripts/install.sh` and README install instructions.
