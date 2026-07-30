# AGENTS.md

This repository contains `canon`, a Go CLI for managing Architecture Decision
Records (ADRs) and Specs (SPECs) in agent-led workflows.

## Start Here

Run these commands before making non-trivial changes:

```sh
go test ./...
go run ./cmd/canon commands
go run ./cmd/canon doctor
```

Use `go run ./cmd/canon ...` during development. The installed binary may not
exist or may be stale.

Global flags (`--adr-dir`, `--spec-dir`, `--format`) must come **before** the
subcommand: `canon --adr-dir x list`, not `canon list --adr-dir x`.

## Project Shape

- `cmd/canon`: CLI entrypoint.
- `internal/canon`: command handling (`cli.go`), storage (`store.go`), output
  envelopes, command registry, and tests.
- `skill`: bundled agent skill content embedded in Go source
  (`skill.go`), with version/hash metadata and install/update helpers.
- `scripts`: build and install helpers (e.g. `scripts/install.sh`).
- `docs/adr`: project ADRs managed by `canon`.
- `docs/commands.md`, `docs/adr-format.md`, `docs/spec-format.md`: format and
  command references.
- `docs/agent-workflows.md`: expected agent workflow.
- `docs/roadmap.md`: planned direction.

## Two Document Kinds

The CLI manages two kinds with the same parseable shape but separate
directories and independent numbering:

- ADR: stored in `docs/adr` (`--adr-dir`), ids like `ADR-0001`.
- SPEC: stored in `docs/spec` (`--spec-dir`), ids like `SPEC-0001`; captures
  functional requirements (`--requirements`, `--acceptance`).

Commands that create or scope documents by kind are kind-prefixed:
`canon adr new|list|search|init` and `canon spec new|list|search|init`
(ADR-0009). Plain `canon list` and `canon search` cover both kinds. Commands
that take `--id` route by id prefix, so they need no kind prefix. `doctor`
checks both directories; a missing `docs/spec` is a warning, not an error.

## ADR/SPEC Rules

Use the CLI to create or change documents. Do not hand-edit lifecycle metadata
unless the CLI cannot express the change.

Before any mutation, check storage and gather context:

```sh
go run ./cmd/canon doctor
go run ./cmd/canon list
go run ./cmd/canon search --query "relevant topic"
go run ./cmd/canon show --id ADR-0001
```

Always preview mutations first; every mutating command supports `--dry-run`
and the dry-run response includes the warning `No changes were made.`:

```sh
go run ./cmd/canon accept --id ADR-0001 --reason "Approved." --dry-run
```

Apply only after the dry-run plan is correct, then verify with `show`.

`supersede` updates both documents to keep the relationship reciprocal
(ADR-0004). `accept`/`reject`/`deprecate`/`append` work the same for both
kinds via `--id`.

Create a new ADR for decisions that affect the CLI contract, ADR/SPEC file
formats, query behavior, lifecycle semantics, output schema, storage layout,
or agent operating model.

## CLI Design Constraints

Keep the CLI agent-friendly:

- JSON output remains the default. Use `--format text` (or `-t`) for
  human-readable output only; do not parse text in automation.
- Every JSON response includes `schema_version`.
- Read commands are deterministic and parseable.
- Mutating commands support `--dry-run`.
- Errors include `code`, `category`, `message`, and `suggested_fix`.
- Command outputs include stable ids and useful `next_actions`.
- Human guidance must not contaminate structured stdout.

When adding a command, update:

- `internal/canon/registry.go`
- `docs/commands.md`
- `docs/agent-workflows.md` if workflows change
- tests in `internal/canon`

## Agent Skill

The bundled skill content is embedded in `skill/skill.go`
(`managedPayload`), not in a separate markdown file. When changing the skill,
edit that function and bump the `Version` const; the content hash is computed
automatically. Install/update it via:

```sh
go run ./cmd/canon skill install --dry-run
go run ./cmd/canon skill install
go run ./cmd/canon skill update --dry-run
```

## Testing

Run:

```sh
go test ./...
```

Tests exercise `Run` directly and parse the JSON envelope
(`internal/canon/cli_test.go` has `runForTest`).

For CLI smoke checks, prefer dry-run commands against a temp directory so
nothing is left behind:

```sh
go run ./cmd/canon --adr-dir /private/tmp/canon-smoke adr new --title "Smoke test" --dry-run
```

For a more complete install check:

```sh
scripts/install.sh --dry-run
```

If you build a binary directly for verification, remove the artifact before
finishing:

```sh
go build ./cmd/canon
rm canon
```

## Documentation

Keep docs close to behavior. If a command flag, output shape, document field,
or workflow changes, update the matching document (`docs/commands.md`,
`docs/adr-format.md`, `docs/spec-format.md`, `docs/agent-workflows.md`) in the
same change.

Use examples that agents can run non-interactively. Prefer `--dry-run`
examples for mutating commands.

## Coding Guidelines

- Keep the project dependency-free (stdlib only) unless a dependency has
  clear value.
- Preserve deterministic ordering in command output.
- Keep command parsing non-interactive.
- Prefer focused tests that exercise `Run` and parse JSON responses.
- Do not introduce unrelated refactors while changing CLI behavior.
