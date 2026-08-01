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

Runs the corpus integrity check catalog (SPEC-0001) through the shared
validation engine (ADR-0009); `doctor` is the engine's shallow mode.
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
  stored in the wrong kind's directory).
- Warnings: `missing_directory`, `status_reference_inconsistency` (status
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
`--strict` also exits 4 when only warnings exist.

Safety: read-only.

Errors:

- `document_not_found`: no parseable document claims the `--id` value.
- `id_with_kind_scope`: `--id` passed to a kind-prefixed `validate`.

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

Safety: mutating. Supports `--dry-run`.

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

Safety: mutating. Supports `--dry-run`.

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

Safety: mutating. Supports `--dry-run`.

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

Safety: mutating. Supports `--dry-run`.

## `append`

Appends a dated appendix section to an ADR, SPEC, or domain entry.

```sh
canon append --id ADR-0002 --title "Implementation note" --body "The rollout used the local index." --dry-run
canon append --id ADR-0002 --title "Implementation note" --body "The rollout used the local index."
canon append --id SPEC-0001 --title "Review" --body "Requirements still apply." --dry-run
```

Safety: mutating. Supports `--dry-run`.

## `skill`

Prints the bundled agent skill for operating `canon`.

```sh
canon skill
```

The returned `data.content` is the generated `SKILL.md` content. The response
also includes `data.skill.name`, `data.skill.version`, `data.skill.hash`, and
`data.skill.default_install_dir`.

Safety: read-only.

## `skill install`

Installs the CANON agent skill into a repository-local skill directory.

```sh
canon skill install --dry-run
canon skill install
canon skill install --skill-dir .agents/skills/canon --dry-run
```

Flags:

- `--skill-dir`: installation directory. Default: `.agents/skills/canon`.
- `--dry-run`: preview without writing.

Effects:

- Creates the skill directory if missing.
- Writes `SKILL.md` with managed version and content-hash metadata.

The default target path is `.agents/skills/canon/SKILL.md`.

Safety: mutating. Supports `--dry-run`.

Errors:

- `skill_already_installed`: target `SKILL.md` already exists. Use
  `canon skill update --dry-run`.
- `skill_directory_create_failed`: the skill directory could not be created.
- `skill_write_failed`: `SKILL.md` could not be written.

## `skill update`

Updates an installed CANON agent skill.

```sh
canon skill update --dry-run
canon skill update
canon skill update --force --dry-run
```

Flags:

- `--skill-dir`: installation directory. Default: `.agents/skills/canon`.
- `--force`: overwrite a locally modified or unmanaged `SKILL.md`.
- `--dry-run`: preview without writing.

Behavior:

- Returns a no-op plan when the installed skill is already current.
- Updates unmodified CANON-managed skill files.
- Refuses to overwrite local edits unless `--force` is passed.

Safety: mutating. Supports `--dry-run`.

Errors:

- `skill_not_installed`: target `SKILL.md` does not exist. Use
  `canon skill install --dry-run`.
- `local_skill_modified`: target file is not an unmodified CANON-managed skill.
- `skill_read_failed`: target file could not be read.
- `skill_write_failed`: target file could not be written.
