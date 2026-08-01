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

Global flags (`--adr-dir`, `--spec-dir`, `--domain-dir`, `--format`) must come
**before** the subcommand: `canon --adr-dir x list`, not `canon list --adr-dir x`.

## Project Shape

- `cmd/canon`: CLI entrypoint.
- `internal/canon`: command handling (`cli.go`), storage (`store.go`), output
  envelopes, command registry, and tests.
- `skill`: bundled agent skill content embedded in Go source
  (`skill.go`), with version/hash metadata and install/update helpers.
- `.agents/skills/project-planning`: repository-local planning workflow that
  discovers current ADR and SPEC constraints before producing a plan.
- `scripts`: build and install helpers (e.g. `scripts/install.sh`).
- `docs/adr`: project ADRs managed by `canon`.
- `docs/domain`: the project Domain Model (one canonical concept per entry)
  managed by `canon`.
- `docs/commands.md`, `docs/adr-format.md`, `docs/spec-format.md`,
  `docs/domain-format.md`: format and command references.
- `docs/agent-workflows.md`: expected agent workflow.
- `docs/roadmap.md`: planned direction.

## Three Document Kinds

The CLI manages three kinds with the same parseable shape but separate
directories and independent numbering:

- ADR: stored in `docs/adr` (`--adr-dir`), ids like `ADR-0001`.
- SPEC: stored in `docs/spec` (`--spec-dir`), ids like `SPEC-0001`; captures
  functional requirements (`--requirements`, `--acceptance`).
- Domain entry: stored in `docs/domain` (`--domain-dir`), ids like `DM-0001`;
  defines one canonical concept per entry (`--definition`, `--avoid`,
  `--relationships`). The set of accepted entries is the Domain Model, the
  single source of truth for what things mean.

Commands that create or scope documents by kind are kind-prefixed:
`canon adr new|list|search|init`, `canon spec new|list|search|init`, and
`canon domain new|list|search|init` (ADR-0009). Plain `canon list` and
`canon search` cover all kinds. Commands that take `--id` route by id prefix,
so they need no kind prefix. `doctor` checks all three directories; a missing
`docs/spec` or `docs/domain` is a warning, not an error. Doctor also flags
domain-model integrity problems: duplicate accepted titles and references to
superseded or deprecated entries.

## Document Rules (ADR/SPEC/Domain)

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

Create ADRs only for project architecture. Architectural commitments may affect
the CLI contract, ADR/SPEC file formats, query behavior, lifecycle semantics,
output schema, or storage layout. Do not create ADRs for project processes,
agent workflows, or skill behavior; update `AGENTS.md` or the relevant
`SKILL.md` directly. The canonical gate is DM-0004 (`canon show --id DM-0004`,
"When to Use ADR"); note that the bundled skill intentionally restates the
rubric generically and must not reference this project's ADR or DM ids, since
it ships to other projects.

## CLI Design Constraints

Keep the CLI agent-friendly:

- JSON output remains the default. Use `--format text` (or `-t`) for
  human-readable output only. Use `--format context` only with list commands
  when producing bounded Markdown for prompt injection; do not parse either
  text format in automation.
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

The separate `.agents/skills/project-planning/SKILL.md` is project-specific,
not generated by `canon skill update`. Keep it aligned with planning workflow
changes.

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
