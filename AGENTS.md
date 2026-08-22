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
- `skill`: bundled agent skill and subagent source payloads under
  `skill/assets`, embedded and rendered by `skill.go` with per-asset versions,
  per-file hashes, and install/update helpers.
- `.agents/skills/project-planning`: repository-local planning workflow that
  discovers current ADR and SPEC constraints before producing a plan.
- `.opencode/agents/canon-critic.md`: managed OpenCode rendering of the bundled
  read-only subagent that judges whether an ADR, SPEC, or Domain entry is worth
  keeping or creating. Invocable via `@canon-critic`.
- `scripts`: build and install helpers (e.g. `scripts/install.sh`).
- `docs/adr`: project ADRs managed by `canon`.
- `docs/domain`: the project Domain Model (one canonical concept per entry)
  managed by `canon`.
- `.canon.jsonc`: repository configuration (conventions only, discovered from
  the corpus upward; see `docs/config.md` and ADR-0015).
- `docs/commands.md`, `docs/config.md`, `docs/adr-format.md`,
  `docs/spec-format.md`, `docs/domain-format.md`: format and command
  references.
- `docs/agent-workflows.md`: expected agent workflow.
- `docs/roadmap.md`: planned direction.

## Three Document Kinds

The CLI manages three kinds with the same parseable shape but separate
directories and independent numbering:

- ADR: stored in `docs/adr` (`--adr-dir`), ids like `ADR-0001`.
- SPEC: stored in `docs/spec` (`--spec-dir`), ids like `SPEC-0001`; captures
  functional requirements (`--requirements`, `--acceptance`). This repository
  does not use SPECs: `docs/spec` is absent, behavioral guarantees live in
  the test suite, and new SPECs should not be created here. The CLI still
  supports the kind for other projects.
- Domain entry: stored in `docs/domain` (`--domain-dir`), ids like `DM-0001`;
  defines one canonical concept per entry (`--definition`, `--avoid`,
  `--relationships`). The set of accepted entries is the Domain Model, the
  single source of truth for what things mean.

Commands that create or scope documents by kind are kind-prefixed:
`canon adr new|list|search|validate|init`, `canon spec new|list|search|validate|init`,
and `canon domain new|list|search|validate|init` (ADR-0008). Plain `canon list`,
`canon search`, and `canon validate` cover all kinds. Commands that take
`--id` route by id prefix, so they need no kind prefix. `doctor` checks all
three directories; a missing `docs/spec` or `docs/domain` is a warning, not
an error. Doctor also flags domain-model integrity problems: duplicate
accepted titles and references to superseded or deprecated entries.
`validate` runs the deep integrity catalog — malformed files,
duplicate ids, broken references, reciprocity, metadata validity, and
kind/id/directory coherence — through the shared validation engine
(ADR-0009); `doctor` is that engine's shallow mode. Use `doctor` to answer
"can I work here?" and `validate` to answer "is my corpus healthy?"

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

Single-document mutations (`accept`, `reject`, `deprecate`, `new`) are
reversible via git, so run them directly and verify with `show`:

```sh
go run ./cmd/canon accept --id ADR-0001 --reason "Approved."
go run ./cmd/canon show --id ADR-0001
```

This corpus disables `append` through `.canon.jsonc` (ADR-0015). To evolve a
document, edit its file directly and let git keep the history; do not
hand-edit lifecycle metadata (status, dates, id relationships).

Every mutating command still supports `--dry-run`, and a dry-run response
includes the warning `No changes were made.` `--dry-run` is required only
before `supersede`, which updates both documents to keep the relationship
reciprocal (ADR-0004) and is the messiest mutation to walk back.
`accept`/`reject`/`deprecate` work the same for every kind via `--id`.

Create ADRs only for project architecture. Architectural commitments may affect
the CLI contract, ADR/SPEC file formats, query behavior, lifecycle semantics,
output schema, or storage layout. Do not create ADRs for project processes,
agent workflows, or skill behavior; update `AGENTS.md` or the relevant
`SKILL.md` directly. The canonical gate is DM-0004 (`canon show --id DM-0004`,
"When to Use ADR"); note that the bundled skill intentionally restates the
rubric generically and must not reference this project's ADR or DM ids, since
it ships to other projects. The other kinds have their own canonical gates:
DM-0011 ("When to Use SPEC") and DM-0012 ("When to Use Domain Entry").

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

## Agent Skill Bundle

Bundled source payloads live under `skill/assets` and are embedded by
`skill/skill.go`. The public catalog contains the `canon` and
`canon-record-gate` skills; `canon-critic` is rendered as a component of
`canon-record-gate` for OpenCode or Claude targets. When changing an asset,
edit its source payload and bump that asset's version in `skill/skill.go`;
content hashes are computed automatically. Keep bundled content project-agnostic
because it ships to other projects.

Preview and apply bundle updates via:

```sh
go run ./cmd/canon skill update --dry-run
go run ./cmd/canon skill update
```

`skill update` refuses locally modified or unmanaged files. Review the listed
files first, then use `--force --dry-run` and `--force` only when overwriting
them is intentional.

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

CI mirrors these checks: `.github/workflows/ci.yml` runs `gofmt`, `go vet`,
golangci-lint (`.golangci.yml`), `go test ./... -race` on Linux/macOS (the
supported platforms, per ADR-0014),
CLI smoke checks, and `scripts/install.sh --dry-run`;
`.github/workflows/corpus.yml` runs `validate` when `docs/**` changes
(`doctor` is redundant there: it is the validation engine's shallow mode, so
a passing `validate` implies a passing `doctor`). Keep CI green with the same commands run locally.

## Documentation

Keep docs close to behavior. If a command flag, output shape, document field,
or workflow changes, update the matching document (`docs/commands.md`,
`docs/adr-format.md`, `docs/spec-format.md`, `docs/agent-workflows.md`) in the
same change.

Use examples that agents can run non-interactively. Show `--dry-run` only
where a preview is required, such as `supersede`, or already standard
practice, such as `skill update`.

## Coding Guidelines

- Keep the project dependency-free (stdlib only) unless a dependency has
  clear value.
- Preserve deterministic ordering in command output.
- Keep command parsing non-interactive.
- Prefer focused tests that exercise `Run` and parse JSON responses.
- Do not introduce unrelated refactors while changing CLI behavior.
