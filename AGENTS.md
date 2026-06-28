# AGENTS.md

This repository contains `adrm`, a Go CLI for managing Architecture Decision
Records in agent-led workflows.

## Start Here

Run these commands before making non-trivial changes:

```sh
go test ./...
go run ./cmd/adrm commands
go run ./cmd/adrm doctor
```

Use `go run ./cmd/adrm ...` during development. The installed binary may not
exist or may be stale.

## Project Shape

- `cmd/adrm`: CLI entrypoint.
- `internal/adrm`: command handling, storage, output envelopes, registry, and tests.
- `docs/adr`: project ADRs managed by `adrm`.
- `docs/commands.md`: command reference.
- `docs/adr-format.md`: ADR markdown format.
- `docs/agent-workflows.md`: expected agent workflow.
- `docs/roadmap.md`: planned direction.

## ADR Rules

Use the CLI to create or change ADRs. Do not hand-edit ADR lifecycle metadata
unless the CLI cannot express the change.

Before any ADR mutation:

```sh
go run ./cmd/adrm list
go run ./cmd/adrm search --query "relevant topic"
go run ./cmd/adrm show --id ADR-0001
```

Always preview mutations first:

```sh
go run ./cmd/adrm append --id ADR-0001 --title "Note" --body "Text." --dry-run
```

Apply only after the dry-run plan is correct, then verify:

```sh
go run ./cmd/adrm show --id ADR-0001
```

Create a new ADR for decisions that affect the CLI contract, ADR file format,
query behavior, lifecycle semantics, output schema, storage layout, or agent
operating model.

## CLI Design Constraints

Keep the CLI agent-friendly:

- JSON output remains the default.
- Every JSON response includes `schema_version`.
- Read commands are deterministic and parseable.
- Mutating commands support `--dry-run`.
- Dry-run responses include the warning `No changes were made.`
- Errors include `code`, `category`, `message`, and `suggested_fix`.
- Command outputs include stable ids and useful `next_actions`.
- Human guidance must not contaminate structured stdout.

When adding a command, update:

- `internal/adrm/registry.go`
- `docs/commands.md`
- `docs/agent-workflows.md` if workflows change
- tests in `internal/adrm`

## Testing

Run:

```sh
go test ./...
```

For CLI smoke checks, prefer dry-run commands that do not leave files behind:

```sh
go run ./cmd/adrm --adr-dir /private/tmp/adrm-smoke new --title "Smoke test" --dry-run
```

If you build a binary for verification, remove the generated artifact before
finishing:

```sh
go build ./cmd/adrm
rm adrm
```

## Documentation

Keep docs close to behavior. If a command flag, output shape, ADR field, or
workflow changes, update the matching document in the same change.

Use examples that agents can run non-interactively. Prefer `--dry-run` examples
for mutating commands.

## Coding Guidelines

- Keep the project dependency-free unless a dependency has clear value.
- Preserve deterministic ordering in command output.
- Keep command parsing non-interactive.
- Prefer focused tests that exercise `Run` and parse JSON responses.
- Do not introduce unrelated refactors while changing CLI behavior.
