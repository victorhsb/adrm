# Agent Workflow Guide

`adrm` is meant to be operated by agents without brittle text scraping or
interactive prompts.

## Baseline loop

1. Discover capabilities.

   ```sh
   adrm commands
   ```

2. Check repository state.

   ```sh
   adrm doctor
   ```

3. Query before mutating.

   ```sh
   adrm list
   adrm search --query "storage"
   adrm show --id ADR-0001
   ```

4. Preview every mutation.

   ```sh
   adrm append --id ADR-0001 --title "Review" --body "Still valid." --dry-run
   ```

5. Apply only after the dry-run plan is acceptable.

   ```sh
   adrm append --id ADR-0001 --title "Review" --body "Still valid."
   ```

6. Verify the result.

   ```sh
   adrm show --id ADR-0001
   ```

## Querying strategy

Use broad-to-narrow discovery:

```sh
adrm list
adrm list --status accepted
adrm search --query "database"
adrm search --tag storage
adrm show --id ADR-0002
```

Use ids from command output instead of reconstructing paths. Paths can change;
ADR ids are the composable selectors.

## Creating a decision

```sh
adrm new --kind adr --title "Use SQLite for local query index" --status proposed --dry-run
adrm new --kind adr --title "Use SQLite for local query index" --status proposed --tags "storage,query" --context "Agents need fast local lookup." --decision "Use SQLite-backed indexes." --consequences "The index can be rebuilt from ADR markdown."
adrm show --id ADR-0001
```

## Creating a spec

SPEC files capture functional requirements with their own numbering and
directory. They use the same lifecycle commands as ADRs.

```sh
adrm new --kind spec --title "Local query index" --status proposed --dry-run
adrm new --kind spec --title "Local query index" --tags "storage,query" --context "Agents need fast local lookup." --requirements "Return ADRs by tag and status." --constraints "No external dependencies." --acceptance "list --tag storage returns ADR-0001."
adrm show --id SPEC-0001
```

## Accepting a decision

Use `accept` to record that a proposed ADR or SPEC has been approved.

```sh
adrm show --id ADR-0001
adrm accept --id ADR-0001 --reason "Approved by the team." --dry-run
adrm accept --id ADR-0001 --reason "Approved by the team."
```

## Rejecting a decision

Use `reject` to record that an ADR or SPEC was turned down, without removing it.

```sh
adrm show --id ADR-0001
adrm reject --id ADR-0001 --reason "Chose a different approach." --dry-run
adrm reject --id ADR-0001 --reason "Chose a different approach."
```

## Superseding a decision

Create or identify the replacement document first. `supersede` requires the
replacement to exist and to be the same kind as the superseded document.

```sh
adrm search --query "old query index"
adrm show --id ADR-0001
adrm show --id ADR-0002
adrm supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current indexing strategy." --dry-run
adrm supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current indexing strategy."
```

## Deprecating a decision

Use deprecation when a decision is no longer relevant and there is no direct
replacement.

```sh
adrm deprecate --id ADR-0003 --reason "The component was removed." --dry-run
adrm deprecate --id ADR-0003 --reason "The component was removed."
```

## Installing the agent skill

Install the ADRM skill into the repository so agents can discover the local ADR
workflow without copying command output by hand.

```sh
adrm skill install --dry-run
adrm skill install
```

The default target is `.agents/skills/adrm/SKILL.md`. Use `--skill-dir` when a
repository uses a different skill location:

```sh
adrm skill install --skill-dir .agents/skills/adrm --dry-run
adrm skill install --skill-dir .agents/skills/adrm
```

Update the installed skill after upgrading `adrm`:

```sh
adrm skill update --dry-run
adrm skill update
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
adrm doctor
```

## Output hygiene

For automation, use the default JSON output. Human text output is available:

```sh
adrm --format text doctor
```

Do not parse text output in automation.
