# Agent Workflow Guide

`canon` is meant to be operated by agents without brittle text scraping or
interactive prompts.

## Baseline loop

1. Discover capabilities.

   ```sh
   canon commands
   ```

2. Check repository state.

   ```sh
   canon doctor
   ```

   For deep integrity checks (malformed files, duplicate ids, broken
   references, reciprocity, metadata validity), run `canon validate`. `doctor`
   answers "can I work here?"; `validate` answers "is my corpus healthy?"

3. Query before mutating.

   ```sh
   canon list
   canon search --query "storage"
   canon show --id ADR-0001
   ```

4. Preview every mutation.

   ```sh
   canon append --id ADR-0001 --title "Review" --body "Still valid." --dry-run
   ```

5. Apply only after the dry-run plan is acceptable.

   ```sh
   canon append --id ADR-0001 --title "Review" --body "Still valid."
   ```

6. Verify the result.

   ```sh
   canon show --id ADR-0001
   ```

7. After mutations, confirm corpus health.

   ```sh
   canon validate
   ```

   Exit code 4 means error-severity findings exist; each finding carries a
   `suggested_fix`. Use `canon validate --id ADR-0001` to check only the
   document just changed and its references.

## Querying strategy

Use broad-to-narrow discovery:

```sh
canon list
canon list --status accepted
canon search --query "database"
canon search --tag storage
canon show --id ADR-0002
```

Use ids from command output instead of reconstructing paths. Paths can change;
ADR ids are the composable selectors.

## Planning changes

The repository-local `project-planning` skill applies this discovery workflow
whenever an agent is asked for a plan, design, implementation approach, scope,
or impact analysis. It keeps planning read-only and starts with current document
status:

```sh
go run ./cmd/canon doctor
go run ./cmd/canon --format context list --status accepted
go run ./cmd/canon --format context list --status proposed
```

Accepted ADRs and SPECs constrain the plan. Proposed documents are useful input
but are not binding. Search by task terminology and use `show` to read only the
records that actually affect the work. The skill lives at
`.agents/skills/project-planning/SKILL.md`.

Before and during planning, grilling, or any other terminology-heavy
exploration, also consult the domain model so the plan uses canonical language:

```sh
canon --format context domain list --status accepted
canon domain search --query "cancellation"
```

The domain model is the single source of truth for what things mean. When a
session resolves a new term or sharpens an old one, update the domain model in
the same session rather than batching it for later.

## Creating a decision

```sh
canon adr new --title "Use SQLite for local query index" --status proposed --dry-run
canon adr new --title "Use SQLite for local query index" --status proposed --tags "storage,query" --context "Agents need fast local lookup." --decision "Use SQLite-backed indexes." --consequences "The index can be rebuilt from ADR markdown."
canon show --id ADR-0001
```

## Creating a spec

SPEC files capture functional requirements with their own numbering and
directory. They use the same lifecycle commands as ADRs.

```sh
canon spec new --title "Local query index" --status proposed --dry-run
canon spec new --title "Local query index" --tags "storage,query" --context "Agents need fast local lookup." --requirements "Return ADRs by tag and status." --constraints "No external dependencies." --acceptance "list --tag storage returns ADR-0001."
canon show --id SPEC-0001
```

## Defining a concept

Domain entries define one canonical concept each. Always search the domain
model first; sharpen an existing entry instead of creating a parallel one.

```sh
canon domain search --query "decision"
canon domain new --title "ADR" --status proposed --dry-run
canon domain new --title "ADR" --tags "glossary" --definition "A dated, narrowly-scoped architecture commitment." --avoid "design doc: too broad; ticket: tracks work, not decisions" --relationships "See [SPEC](0002-spec.md)."
canon show --id DM-0001
```

Lifecycle semantics for entries: `accept` canonizes the term; `supersede` is
for redefinitions (a new entry replaces the old meaning); renames retitle the
same entry in place plus a `canon append` history note; `deprecate` retires a
concept as a tombstone. See `docs/domain-format.md` for the full format.

## Accepting a decision

Use `accept` to record that a proposed ADR, SPEC, or domain entry has been
approved.

```sh
canon show --id ADR-0001
canon accept --id ADR-0001 --reason "Approved by the team." --dry-run
canon accept --id ADR-0001 --reason "Approved by the team."
```

## Rejecting a decision

Use `reject` to record that an ADR, SPEC, or domain entry was turned down,
without removing it.

```sh
canon show --id ADR-0001
canon reject --id ADR-0001 --reason "Chose a different approach." --dry-run
canon reject --id ADR-0001 --reason "Chose a different approach."
```

## Superseding a decision

Create or identify the replacement document first. `supersede` requires the
replacement to exist and to be the same kind as the superseded document.

```sh
canon search --query "old query index"
canon show --id ADR-0001
canon show --id ADR-0002
canon supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current indexing strategy." --dry-run
canon supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current indexing strategy."
```

## Deprecating a decision

Use deprecation when a decision is no longer relevant and there is no direct
replacement.

```sh
canon deprecate --id ADR-0003 --reason "The component was removed." --dry-run
canon deprecate --id ADR-0003 --reason "The component was removed."
```

## Installing the agent skill

Install the CANON skill into the repository so agents can discover the local ADR
workflow without copying command output by hand.

```sh
canon skill install --dry-run
canon skill install
```

The default target is `.agents/skills/canon/SKILL.md`. Use `--skill-dir` when a
repository uses a different skill location:

```sh
canon skill install --skill-dir .agents/skills/canon --dry-run
canon skill install --skill-dir .agents/skills/canon
```

Update the installed skill after upgrading `canon`:

```sh
canon skill update --dry-run
canon skill update
```

If `skill update` reports `local_skill_modified`, inspect the file first. Only
use `--force` after deciding that overwriting local edits is acceptable.

## Error recovery

When a command fails, read:

- `error.code`
- `error.category`
- `error.message`
- `error.suggested_fix`

Prefer the suggested fix over guessing. For state-related failures, run:

```sh
canon doctor
```

## Output hygiene

For automation, use the default JSON output. Human text output is available:

```sh
canon --format text doctor
```

Do not parse text output in automation.

For bounded prompt or system-context injection, list commands support a concise
Markdown projection. Keep selection explicit so only effective decisions enter
the prompt:

```sh
canon --format context adr list --status accepted
```

Context output contains only a heading, stable ids, and titles. It omits the
JSON envelope and `next_actions`; use JSON for all programmatic processing.
