# Command Reference

`canon` emits JSON by default. Text and context output are explicit opt-ins.
Global flags must appear before the command.

```sh
canon --adr-dir docs/adr --spec-dir docs/spec --domain-dir docs/domain --format json list
```

Commands that create or scope documents by kind are prefixed with the kind:
`canon adr new`, `canon spec list`, `canon domain new`, and so on. Commands
that operate on one document take `--id` and route by the id prefix (`ADR-`,
`SPEC-`, or `DM-`), so they need no kind prefix.

## Output envelope

Every JSON response uses this shape:

```json
{
  "schema_version": "canon.v1",
  "command": "list",
  "status": "ok",
  "data": {},
  "warnings": [],
  "next_actions": [],
  "error": null
}
```

Fields:

- `schema_version`: response contract version.
- `command`: command that produced the response.
- `status`: `ok`, `warning`, `planned`, or `error`.
- `data`: command-specific payload.
- `warnings`: non-fatal guidance. Dry-run responses include `No changes were made.`
- `next_actions`: suggested follow-up commands with safety labels.
- `error`: structured error payload when `status` is `error`.

## Global flags

- `--adr-dir`: ADR storage directory. Default: `docs/adr`.
- `--spec-dir`: SPEC storage directory. Default: `docs/spec`.
- `--domain-dir`: domain entry storage directory. Default: `docs/domain`.
- `--format`: output format. Values: `json`, `text`, `context`. Default: `json`.
- `-t`: shorthand for `--format text`.

The `text` format is a human-readable projection. Every successful command
renders its payload data after the status line, for unprefixed and
kind-prefixed forms alike. Text is not an automation contract: parse JSON
instead.

The `context` format is supported only by `list`, `adr list`, `spec list`,
and `domain list`. It emits a bounded Markdown projection for prompt
injection. Other commands reject it with `unsupported_context_format`.

## `commands`

Returns the machine-readable command registry.

```sh
canon commands
```

Use this before automation. It declares purpose, side effects, selectors, examples,
dry-run support, and suggested next commands.

Safety: read-only.

## `version`

Prints the canon build version. `scripts/install.sh` injects the nearest git
tag plus commits since and the short hash (for example
`v0.2.0-1-g0ebd91f`). Plain `go build` reports the pseudo-version stamped in
Go build info (for example `v0.2.1-0.20260808014319-0ebd91f91670+dirty`),
and `go run` or other unstamped builds report `dev`.

```sh
canon version
```

Safety: read-only.

## `doctor`

Checks whether ADR, SPEC, and domain storage exists and whether files can be
parsed. It also runs content-level integrity checks on the domain model:

- `domain_duplicate_title`: two accepted domain entries share a title.
  Deprecate or supersede all but one so each concept has a single truth.
- `domain_dead_reference`: a live document references (by `DM-` id or markdown
  link) a domain entry that is superseded or deprecated.

```sh
canon doctor
```

Safety: read-only. Reports a warning when any directory is missing or any
integrity check fails.

Required kinds: repository configuration declares which stores must exist
(`conventions.required_kinds`, see `docs/config.md`). A missing required
store is a warning with an `init` next action; a missing non-required store
is reported as an `ok` diagnostic and never suggests initialization. Without
configuration all three stores are required, matching historical behavior.
Doctor next actions are built from the actual missing required stores.

Errors:

- `invalid_config` / `config_scope_mismatch`: the repository configuration
  could not be resolved; run `canon config validate` for findings.

Common next action when missing storage:

```sh
canon adr init --dry-run
canon adr init
canon spec init --dry-run
canon spec init
canon domain init --dry-run
canon domain init
```

## `validate` / `adr validate` / `spec validate` / `domain validate`

Runs the corpus integrity check catalog through the shared validation
engine (ADR-0009); `doctor` is the engine's shallow mode.
Plain `canon validate` covers all three directories; the prefixed forms scope
the run to one directory. `validate` never mutates the corpus.

```sh
canon validate
canon validate --strict
canon validate --id ADR-0001
canon adr validate
canon spec validate
canon domain validate
```

Flags:

- `--id`: validate only that document and its references. Plain
  `canon validate` only; the id prefix already selects the kind.
- `--strict`: exit 4 when only warnings exist.

Checks, as findings with severity in the `status` field:

- Errors: `malformed_file` (isolated per file, so one bad file does not mask
  the rest), `duplicate_id` (names both paths), `broken_reference`
  (`supersedes`, `superseded_by`, or `deprecated_by` pointing at a
  nonexistent id), `reciprocity_violation` (ADR-0004: `A.superseded_by=B`
  requires `B.supersedes` to contain `A`), `invalid_status`, `kind_mismatch`
  (kind field contradicts the id prefix), `directory_mismatch` (document
  stored in the wrong kind's directory), `disallowed_tag` (a document's tags
  fall outside the vocabulary configured in `conventions.tags` for its kind;
  one finding per document names the sorted offending tags, the allowed
  values, and the controlling config path).
- Warnings: `missing_directory` (required stores only; a missing
  non-required store is healthy and produces no finding),
  `status_reference_inconsistency` (status
  `superseded` without `superseded_by` and vice versa; status `deprecated`
  without `deprecated_by`), `malformed_date` (date is not `YYYY-MM-DD`).

Reference values that are not ids (for example the literal `manual` written
by `canon deprecate`) are not treated as references.

Output contains `findings` only (no per-file ok entries) plus a `summary`
object with `files_checked`, `errors`, and `warnings`. Findings extend the
diagnostic shape with optional `path` and `id` fields, carry a concrete
`suggested_fix`, and are ordered deterministically by path then check name.

Envelope status is `error` if any error finding exists, else `warning` if any
warning exists, else `ok`. Exit code is 4 when any error exists, otherwise 0;
`--strict` also exits 4 when only warnings exist. Repository configuration
can make strictness permanent (`conventions.validation.strict`); the flag and
the setting combine by logical OR and neither changes finding severities.

Safety: read-only.

Errors:

- `document_not_found`: no parseable document claims the `--id` value.
- `id_with_kind_scope`: `--id` passed to a kind-prefixed `validate`.
- `invalid_config` / `config_scope_mismatch`: the repository configuration
  could not be resolved; run `canon config validate` for findings.

## `config show`

Shows the effective repository configuration: source (`file` or `defaults`),
discovered path, per-kind discovery paths, every resolved convention value,
sorted recognized keys, and sorted unknown key paths. See `docs/config.md`
for the schema, defaults, and discovery rules.

```sh
canon config show
canon --format text config show
```

Safety: read-only. Rejects `--format context`.

Errors:

- `invalid_config`: the file is malformed, has an unsupported schema version,
  or declares invalid values (exit 4).
- `config_scope_mismatch`: the three stores do not resolve to one
  configuration scope (exit 4).

## `config validate`

Validates `.canon.jsonc` against the configuration schema and reports
deterministic findings with a summary. When the configuration is valid, the
response includes the same effective report as `config show`.

```sh
canon config validate
```

Findings: `malformed_config`, `unsupported_config_schema`,
`invalid_config_value`, and `config_scope_mismatch` are errors;
`unknown_config_key` is a warning so older binaries can expose keys from
future versions without failing. Exit code is 4 when any error finding
exists and 0 otherwise; `conventions.validation.strict` does not apply to
configuration validation itself.

Safety: read-only. Rejects `--format context`.

## `adr init` / `spec init` / `domain init`

Creates the ADR, SPEC, or domain directory if it does not exist.

```sh
canon adr init --dry-run
canon adr init
canon spec init --dry-run
canon spec init
canon domain init --dry-run
canon domain init
```

Flags:

- `--dry-run`: preview without writing.

Safety: mutating. Supports `--dry-run`.

## `adr new`

Creates a new ADR markdown file. ADR files capture architecture decisions.

```sh
canon adr new --title "Use SQLite for local index" --status proposed --dry-run
canon adr new --title "Use SQLite for local index" --status proposed --tags "storage,query" --context "Agents need local lookup." --decision "Use SQLite." --consequences "The index can be rebuilt."
```

Flags:

- `--title`: required.
- `--status`: optional. Default: `proposed`.
- `--tags`: comma-separated list.
- `--context`: markdown text for the Context section.
- `--decision`: markdown text for the Decision section.
- `--consequences`: markdown text for the Consequences section.
- `--dry-run`: preview without writing.

Valid statuses: `proposed`, `accepted`, `rejected`, `superseded`, `deprecated`.

Safety: mutating. Supports `--dry-run`.

Errors:

- `initial_status_restricted`: repository configuration requires new
  documents to start as `proposed` and `--status` was something else
  (exit 4, dry-run and apply alike).
- `disallowed_tag`: repository configuration restricts the kind's tag
  vocabulary (`conventions.tags`) and a requested tag is outside it
  (exit 4); the message lists every offending tag and the allowed values.
- `invalid_config`: the repository configuration could not be resolved
  (exit 4). Policy gates run before any filesystem access.

## `spec new`

Creates a new SPEC markdown file. SPEC files capture functional requirements.

```sh
canon spec new --title "Local query index" --status proposed --dry-run
canon spec new --title "Local query index" --tags "storage,query" --context "Agents need local lookup." --requirements "Return ADRs by tag and status." --constraints "No external dependencies." --acceptance "list --tag storage returns ADR-0001."
```

Flags:

- `--title`: required.
- `--status`: optional. Default: `proposed`.
- `--tags`: comma-separated list.
- `--context`: markdown text for the Context section.
- `--requirements`: markdown text for the Requirements section.
- `--constraints`: markdown text for the Constraints section.
- `--acceptance`: markdown text for the Acceptance Criteria section.
- `--dry-run`: preview without writing.

Safety: mutating. Supports `--dry-run`. Repository policy gates
(`initial_status_restricted`, `disallowed_tag`, `invalid_config`) apply as
for `adr new`; a SPEC store that the configuration does not require still
accepts new documents.

## `domain new`

Creates a new domain entry markdown file. Domain entries define one canonical
concept each: what it means, which terms to avoid (and why), and how it
relates to other concepts.

```sh
canon domain new --title "ADR" --status proposed --dry-run
canon domain new --title "ADR" --tags "glossary" --definition "A dated, narrowly-scoped architecture commitment." --avoid "design doc: too broad; ticket: tracks work, not decisions" --relationships "See [SPEC](0002-spec.md)."
```

Flags:

- `--title`: required. The canonical term being defined.
- `--status`: optional. Default: `proposed`.
- `--tags`: comma-separated list.
- `--definition`: markdown text for the Definition section. Definitions carry
  no implementation details.
- `--avoid`: avoided terms as `term: reason` pairs separated by semicolons,
  rendered as a bullet list. Reasons explain why each term is not canonical.
- `--relationships`: markdown text for the Relationships section. Reference
  other entries with relative markdown links, e.g. `[SPEC](0002-spec.md)`.
- `--dry-run`: preview without writing.

Safety: mutating. Supports `--dry-run`. Repository policy gates
(`initial_status_restricted`, `disallowed_tag`, `invalid_config`) apply as
for `adr new`.

## `list` / `adr list` / `spec list` / `domain list`

Lists ADR, SPEC, and domain entry summaries in stable order. Plain
`canon list` covers all kinds; the prefixed forms scope the listing to one
kind.

```sh
canon list
canon list --status accepted
canon adr list --status accepted
canon spec list --tag storage
canon domain list --status accepted
canon --format context adr list --status accepted
```

Flags:

- `--status`: filter by status.
- `--tag`: filter by tag.

Context output keeps filtering explicit and contains only a heading plus each
matching document's stable id and title:

```markdown
## Architecture Decision Records

- `ADR-0002`: Install repository-local agent skill
- `ADR-0003`: Use a four-test model for ADR-worthiness in the agent skill
```

It omits the response envelope, count, metadata, warnings, and `next_actions`.
An empty result contains the heading followed by `_No matching documents._`.

Safety: read-only.

## `show`

Returns one ADR, SPEC, or domain entry with metadata and markdown content.
The id prefix (`ADR-`, `SPEC-`, or `DM-`) selects the document; bare numbers
resolve to ADR.

```sh
canon show --id ADR-0001
canon show --id SPEC-0001
canon show --id DM-0001
canon show --id 1
```

Safety: read-only.

Errors:

- `invalid_config`: the repository configuration could not be resolved, so
  `show` cannot know which follow-up mutations are legal (exit 4). Fix the
  configuration; `show` never falls back to defaults here.

## `search` / `adr search` / `spec search` / `domain search`

Searches id, title, status, tags, kind, and markdown content across ADRs,
SPECs, and domain entries. Plain `canon search` covers all kinds; the
prefixed forms scope the search to one kind.

```sh
canon search --query "database"
canon search database
canon spec search --query requirements
canon domain search --query "cancellation"
canon search --status deprecated
canon adr search --tag storage --query local
```

Flags:

- `--query`: search query.
- `--status`: filter by status.
- `--tag`: filter by tag.

Safety: read-only.

## `accept`

Marks an ADR, SPEC, or domain entry as accepted.

```sh
canon accept --id ADR-0001 --reason "Approved by the team." --dry-run
canon accept --id ADR-0001 --reason "Approved by the team."
canon accept --id SPEC-0001 --reason "Requirements approved." --dry-run
canon accept --id DM-0001 --reason "Term canonized." --dry-run
```

Flags:

- `--id`: required. ADR, SPEC, or domain entry id to accept.
- `--reason`: optional. Reason recorded in the history section.
- `--dry-run`: preview without writing.

Effects:

- Sets `status: accepted`.
- Updates the Status section in the body.
- Appends a `History: Accepted` section.

Safety: mutating. Supports `--dry-run`.

Errors:

- `reason_required_by_config`: repository configuration requires a reason for
  lifecycle transitions (`conventions.lifecycle.require_reason`) and
  `--reason` was blank (exit 4, dry-run and apply alike).
- `invalid_config`: the repository configuration could not be resolved
  (exit 4).

## `reject`

Marks an ADR, SPEC, or domain entry as rejected. For domain entries, a
proposed entry that does not survive review is usually deleted instead of
rejected; `reject` remains available when a refusal should stay on record.

```sh
canon reject --id ADR-0001 --reason "Chose a different approach." --dry-run
canon reject --id ADR-0001 --reason "Chose a different approach."
```

Flags:

- `--id`: required. ADR, SPEC, or domain entry id to reject.
- `--reason`: optional. Reason recorded in the history section.
- `--dry-run`: preview without writing.

Effects:

- Sets `status: rejected`.
- Updates the Status section in the body.
- Appends a `History: Rejected` section.

Safety: mutating. Supports `--dry-run`. `reason_required_by_config` and
`invalid_config` apply as for `accept`.

## `supersede`

Marks one document as superseded by another existing document of the same
kind. Cross-kind supersede (an ADR by a SPEC, a domain entry by an ADR, and
so on) is rejected. For domain entries, supersede means *redefinition*;
renaming a concept retitles the same entry in place instead (title is
content, not lifecycle metadata) with a `canon append` history note.

```sh
canon supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current design." --dry-run
canon supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current design."
canon supersede --id SPEC-0001 --by SPEC-0002 --reason "Requirements split." --dry-run
canon supersede --id DM-0001 --by DM-0002 --reason "Definition sharpened." --dry-run
```

Effects:

- Sets `status: superseded` on the `--id` document.
- Sets `superseded_by` to the replacement id.
- Adds the superseded id to the replacement document's `supersedes` list.
- Updates the Status section in the body of the superseded document.
- Appends a `History: Superseded` section to the superseded document.

Safety: mutating. Supports `--dry-run`.

Errors:

- `cross_kind_supersede`: the `--id` and `--by` documents have different kinds.
- `superseding_adr_not_found`: the replacement document was not found.
- `reason_required_by_config`: repository configuration requires a reason and
  `--reason` was blank (exit 4).
- `invalid_config`: the repository configuration could not be resolved
  (exit 4).

## `deprecate`

Marks an ADR, SPEC, or domain entry as deprecated without naming a
replacement. A deprecated domain entry is a tombstone: the concept is leaving
the domain, but the entry stays so historical references still resolve and
future implementations know to intentionally ignore it.

```sh
canon deprecate --id ADR-0003 --reason "The component was removed." --dry-run
canon deprecate --id ADR-0003 --reason "The component was removed."
canon deprecate --id SPEC-0002 --reason "Requirements moved." --dry-run
```

Effects:

- Sets `status: deprecated`.
- Sets `deprecated_by: manual`.
- Updates the Status section in the body.
- Appends a `History: Deprecated` section.

Safety: mutating. Supports `--dry-run`. `reason_required_by_config` and
`invalid_config` apply as for `accept`.

## `append`

Appends a dated appendix section to an ADR, SPEC, or domain entry.

```sh
canon append --id ADR-0002 --title "Implementation note" --body "The rollout used the local index."
canon append --id SPEC-0001 --title "Review" --body "Requirements still apply."
```

A repository can disable this command by setting `conventions.append` to
`false` in `.canon.jsonc` at the repo root (see `docs/config.md`). When
disabled, `append` fails with `append_disabled` (exit 4) and `show` stops
suggesting it. The intent is that the corpus edits documents directly and
lets git track the history.

Safety: mutating. Supports `--dry-run`.

## `skill`

Returns the bundled skill asset catalog.

```sh
canon skill
```

`data.assets` contains the public bundle assets in deterministic order. Each
entry includes:

- `name`
- `kind` (`skill`)
- `version`
- `hash` (a deterministic aggregate content hash)
- `target_paths`

The catalog contains `canon` and `canon-record-gate`. The `canon-critic`
subagent is a component of `canon-record-gate`, not a separate catalog asset.
`data.default_skill_dir` is `.agents/skills`.

Safety: read-only.

## `skill install`

Installs the bundled skills, supporting payload files, and selected subagent
components into a repository.

```sh
canon skill install --dry-run
canon skill install
canon skill install --only canon --dry-run
canon skill install --agent claude --dry-run
canon skill install --agent codex --dry-run
canon skill install --skill-dir .agents/skills --agent opencode --agent claude --dry-run
```

Flags:

- `--skill-dir`: skill bundle root. Default: `.agents/skills`. Asset names are
  appended below this root, so the default canon path is
  `.agents/skills/canon/SKILL.md`.
- `--only <name>`: restrict installation to a bundled skill asset. Repeatable;
  valid names are `canon` and `canon-record-gate`.
- `--agent <target>`: select an agent target. Repeatable; valid targets are
  `opencode`, `claude`, and `codex`.
- `--dry-run`: preview without writing.

`--skill-dir` is a bundle root, not one asset's directory. Earlier single-skill
versions treated a custom value as the `canon` skill directory itself; when
migrating such a command, pass its parent directory instead. Skill payloads
honor `--skill-dir`, while agent target inference and `.opencode`/`.claude`/
`.codex` output paths are resolved from the current working directory.

Effects:

- Writes one `SKILL.md` per selected skill below the skill root.
- Writes supporting payload files such as
  `.agents/skills/canon-record-gate/references/boundary-examples.md`.
- When `canon-record-gate` is selected, renders the `canon-critic` component
  for each selected target:
  - OpenCode: `.opencode/agents/canon-critic.md`
  - Claude: `.claude/agents/canon-critic.md`
  - Codex: `.codex/agents/canon-critic.toml`
- Adds managed version and content-hash markers to every generated file.
  Markdown uses HTML comments; Codex TOML uses `#` comments. The Codex agent is
  read-only, carries the shared critic instructions in
  `developer_instructions`, and inherits model and reasoning selection.

When `--agent` is absent, targets are inferred from existing `.opencode`,
`.claude`, and `.codex` directories in the current project. If none exist,
installation falls back to OpenCode. Install checks every selected target file
before writing; if any file already exists, it writes nothing and reports
`skill_already_installed` with a suggestion to use `skill update`.

Safety: mutating. Supports `--dry-run`. Dry-run output contains the complete
per-file plan and the standard `No changes were made.` warning.

Errors:

- `invalid_usage`: an unknown `--only` asset, unsupported `--agent` target, or
  unexpected positional argument was supplied.
- `skill_already_installed`: one or more selected target files already exist.
- `skill_stat_failed`: a target file could not be inspected.
- `agent_target_infer_failed`: target discovery directories could not be
  inspected.
- `skill_directory_create_failed`: a target directory could not be created.
- `skill_write_failed`: a target file could not be written.

## `skill update`

Refreshes every installed file managed by the selected bundle.

```sh
canon skill update --dry-run
canon skill update
canon skill update --only canon-record-gate --dry-run
canon skill update --force --dry-run
```

Flags:

- `--skill-dir`: skill bundle root. Default: `.agents/skills`.
- `--only <name>`: restrict the update to a bundled skill asset. Repeatable.
- `--agent <target>`: select an agent target. Repeatable; valid targets are
  `opencode`, `claude`, and `codex`.
- `--force`: overwrite locally modified or unmanaged target files.
- `--dry-run`: preview without writing.

As with `skill install`, `--skill-dir` selects the skill bundle root, while
agent target inference and output paths are resolved from the current working
directory.

Behavior:

- Emits `noop` operations for current files.
- Updates older files whose managed hash markers prove they were not locally
  modified.
- Writes missing bundle files when at least one selected file is already
  installed, allowing an older single-skill installation to acquire the rest of
  the bundle.
- Refuses the whole update when any selected file is locally modified or lacks
  valid managed markers, unless `--force` is passed.
- Returns `skill_not_installed` when none of the selected files exists.
- Infers targets the same way as `skill install` when `--agent` is absent.

Safety: mutating. Supports `--dry-run`. Every dry-run, including an all-noop
plan, has `plan.dry_run: true` and the standard `No changes were made.` warning.

Errors:

- `invalid_usage`: an unknown `--only` asset, unsupported `--agent` target, or
  unexpected positional argument was supplied.
- `skill_not_installed`: no selected managed bundle file exists. Use
  `canon skill install --dry-run`.
- `local_skill_modified`: one or more selected files are locally modified or
  unmanaged. Review them before retrying with `--force`.
- `skill_read_failed`: a target file could not be read.
- `agent_target_infer_failed`: target discovery directories could not be
  inspected.
- `skill_directory_create_failed`: a target directory could not be created.
- `skill_write_failed`: a target file could not be written.
