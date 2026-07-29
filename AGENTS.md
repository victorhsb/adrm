# AGENTS.md

This repository contains `adrm`, a Go CLI for managing Architecture Decision
Records (ADRs) and Specs (SPECs) in agent-led workflows.

## Start Here

Run these commands before making non-trivial changes:

```sh
go test ./...
go run ./cmd/adrm commands
go run ./cmd/adrm doctor
```

Use `go run ./cmd/adrm ...` during development. The installed binary may not
exist or may be stale.

Global flags (`--adr-dir`, `--spec-dir`, `--format`) must come **before** the
subcommand: `adrm --adr-dir x list`, not `adrm list --adr-dir x`.

## Project Shape

- `cmd/adrm`: CLI entrypoint.
- `internal/adrm`: command handling (`cli.go`), storage (`store.go`), output
  envelopes, command registry, and tests.
- `adrmskill`: bundled agent skill content embedded in Go source
  (`skill.go`), with version/hash metadata and install/update helpers.
- `scripts`: build and install helpers (e.g. `scripts/install.sh`).
- `docs/adr`: project ADRs managed by `adrm`.
- `docs/commands.md`, `docs/adr-format.md`, `docs/spec-format.md`: format and
  command references.
- `docs/agent-workflows.md`: expected agent workflow.
- `docs/roadmap.md`: planned direction.

## Two Document Kinds

The CLI manages two kinds with the same parseable shape but separate
directories and independent numbering:

- ADR: default kind, stored in `docs/adr` (`--adr-dir`), ids like `ADR-0001`.
- SPEC: stored in `docs/spec` (`--spec-dir`), ids like `SPEC-0001`; captures
  functional requirements (`--requirements`, `--acceptance`).

Commands that create or list documents take `--kind adr|spec` (default
`adr`). Commands that take `--id` route by id prefix, so no `--kind` is
needed. `doctor` and `init --kind ...` handle both directories; a missing
`docs/spec` is a warning, not an error.

## ADR/SPEC Rules

Use the CLI to create or change documents. Do not hand-edit lifecycle metadata
unless the CLI cannot express the change.

Before any mutation, check storage and gather context:

```sh
go run ./cmd/adrm doctor
go run ./cmd/adrm list
go run ./cmd/adrm search --query "relevant topic"
go run ./cmd/adrm show --id ADR-0001
```

Always preview mutations first; every mutating command supports `--dry-run`
and the dry-run response includes the warning `No changes were made.`:

```sh
go run ./cmd/adrm accept --id ADR-0001 --reason "Approved." --dry-run
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

- `internal/adrm/registry.go`
- `docs/commands.md`
- `docs/agent-workflows.md` if workflows change
- tests in `internal/adrm`

## Agent Skill

The bundled skill content is embedded in `adrmskill/skill.go`
(`managedPayload`), not in a separate markdown file. When changing the skill,
edit that function and bump the `Version` const; the content hash is computed
automatically. Install/update it via:

```sh
go run ./cmd/adrm skill install --dry-run
go run ./cmd/adrm skill install
go run ./cmd/adrm skill update --dry-run
```

## Testing

Run:

```sh
go test ./...
```

Tests exercise `Run` directly and parse the JSON envelope
(`internal/adrm/cli_test.go` has `runForTest`).

For CLI smoke checks, prefer dry-run commands against a temp directory so
nothing is left behind:

```sh
go run ./cmd/adrm --adr-dir /private/tmp/adrm-smoke new --title "Smoke test" --dry-run
```

For a more complete install check:

```sh
scripts/install.sh --dry-run
```

If you build a binary directly for verification, remove the artifact before
finishing:

```sh
go build ./cmd/adrm
rm adrm
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
