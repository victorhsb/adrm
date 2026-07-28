# Command Reference

`adrm` emits JSON by default. Global flags must appear before the command.

```sh
adrm --adr-dir docs/adr --spec-dir docs/spec --format json list
```

## Output envelope

Every JSON response uses this shape:

```json
{
  "schema_version": "adrm.v1",
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
- `--format`: output format. Values: `json`, `text`. Default: `json`.
- `-t`: shorthand for `--format text`.

## `commands`

Returns the machine-readable command registry.

```sh
adrm commands
```

Use this before automation. It declares purpose, side effects, selectors, examples,
dry-run support, and suggested next commands.

Safety: read-only.

## `doctor`

Checks whether ADR and SPEC storage exists and whether files can be parsed.

```sh
adrm doctor
```

Safety: read-only. Reports a warning when either directory is missing.

Common next action when missing storage:

```sh
adrm init --kind adr --dry-run
adrm init --kind adr
adrm init --kind spec --dry-run
adrm init --kind spec
```

## `init`

Creates the ADR or SPEC directory if it does not exist.

```sh
adrm init --kind adr --dry-run
adrm init --kind adr
adrm init --kind spec --dry-run
adrm init --kind spec
```

Flags:

- `--kind`: document kind. Values: `adr`, `spec`. Default: `adr`.
- `--dry-run`: preview without writing.

Safety: mutating. Supports `--dry-run`.

## `new`

Creates a new ADR or SPEC markdown file. SPEC files capture functional
requirements; ADR files capture architecture decisions.

Create an ADR:

```sh
adrm new --kind adr --title "Use SQLite for local index" --status proposed --dry-run
adrm new --kind adr --title "Use SQLite for local index" --status proposed --tags "storage,query" --context "Agents need local lookup." --decision "Use SQLite." --consequences "The index can be rebuilt."
```

Create a SPEC:

```sh
adrm new --kind spec --title "Local query index" --status proposed --dry-run
adrm new --kind spec --title "Local query index" --tags "storage,query" --context "Agents need local lookup." --requirements "Return ADRs by tag and status." --constraints "No external dependencies." --acceptance "list --tag storage returns ADR-0001."
```

Flags:

- `--kind`: document kind. Values: `adr`, `spec`. Default: `adr`.
- `--title`: required.
- `--status`: optional. Default: `proposed`.
- `--tags`: comma-separated list.
- `--context`: markdown text for the Context section (both kinds).
- `--decision`: markdown text for the Decision section (adr).
- `--consequences`: markdown text for the Consequences section (adr).
- `--requirements`: markdown text for the Requirements section (spec).
- `--constraints`: markdown text for the Constraints section (spec).
- `--acceptance`: markdown text for the Acceptance Criteria section (spec).
- `--dry-run`: preview without writing.

Valid statuses: `proposed`, `accepted`, `rejected`, `superseded`, `deprecated`.

Safety: mutating. Supports `--dry-run`.

## `list`

Lists ADR and SPEC summaries in stable order.

```sh
adrm list
adrm list --kind adr --status accepted
adrm list --kind spec --tag storage
adrm list --kind spec
```

Flags:

- `--kind`: filter by kind. Values: `adr`, `spec`, or empty to list both.
- `--status`: filter by status.
- `--tag`: filter by tag.

Safety: read-only.

## `show`

Returns one ADR or SPEC with metadata and markdown content. The id prefix
(`ADR-` or `SPEC-`) selects the document; bare numbers resolve to ADR.

```sh
adrm show --id ADR-0001
adrm show --id SPEC-0001
adrm show --id 1
```

Safety: read-only.

## `search`

Searches id, title, status, tags, kind, and markdown content across ADRs and
SPECs.

```sh
adrm search --query "database"
adrm search database
adrm search --kind spec --query requirements
adrm search --status deprecated
adrm search --tag storage --query local
```

Flags:

- `--query`: search query.
- `--kind`: filter by kind. Values: `adr`, `spec`, or empty to search both.
- `--status`: filter by status.
- `--tag`: filter by tag.

Safety: read-only.

## `accept`

Marks an ADR or SPEC as accepted.

```sh
adrm accept --id ADR-0001 --reason "Approved by the team." --dry-run
adrm accept --id ADR-0001 --reason "Approved by the team."
adrm accept --id SPEC-0001 --reason "Requirements approved." --dry-run
```

Flags:

- `--id`: required. ADR or SPEC id to accept.
- `--reason`: optional. Reason recorded in the history section.
- `--dry-run`: preview without writing.

Effects:

- Sets `status: accepted`.
- Updates the Status section in the body.
- Appends a `History: Accepted` section.

Safety: mutating. Supports `--dry-run`.

## `reject`

Marks an ADR or SPEC as rejected.

```sh
adrm reject --id ADR-0001 --reason "Chose a different approach." --dry-run
adrm reject --id ADR-0001 --reason "Chose a different approach."
```

Flags:

- `--id`: required. ADR or SPEC id to reject.
- `--reason`: optional. Reason recorded in the history section.
- `--dry-run`: preview without writing.

Effects:

- Sets `status: rejected`.
- Updates the Status section in the body.
- Appends a `History: Rejected` section.

Safety: mutating. Supports `--dry-run`.

## `supersede`

Marks one document as superseded by another existing document of the same
kind. Cross-kind supersede (an ADR by a SPEC, or vice versa) is rejected.

```sh
adrm supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current design." --dry-run
adrm supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current design."
adrm supersede --id SPEC-0001 --by SPEC-0002 --reason "Requirements split." --dry-run
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

Marks an ADR or SPEC as deprecated without naming a replacement.

```sh
adrm deprecate --id ADR-0003 --reason "The component was removed." --dry-run
adrm deprecate --id ADR-0003 --reason "The component was removed."
adrm deprecate --id SPEC-0002 --reason "Requirements moved." --dry-run
```

Effects:

- Sets `status: deprecated`.
- Sets `deprecated_by: manual`.
- Updates the Status section in the body.
- Appends a `History: Deprecated` section.

Safety: mutating. Supports `--dry-run`.

## `append`

Appends a dated appendix section to an ADR or SPEC.

```sh
adrm append --id ADR-0002 --title "Implementation note" --body "The rollout used the local index." --dry-run
adrm append --id ADR-0002 --title "Implementation note" --body "The rollout used the local index."
adrm append --id SPEC-0001 --title "Review" --body "Requirements still apply." --dry-run
```

Safety: mutating. Supports `--dry-run`.

## `skill`

Prints the bundled agent skill for operating `adrm`.

```sh
adrm skill
```

The returned `data.content` is the generated `SKILL.md` content. The response
also includes `data.skill.name`, `data.skill.version`, `data.skill.hash`, and
`data.skill.default_install_dir`.

Safety: read-only.

## `skill install`

Installs the ADRM agent skill into a repository-local skill directory.

```sh
adrm skill install --dry-run
adrm skill install
adrm skill install --skill-dir .agents/skills/adrm --dry-run
```

Flags:

- `--skill-dir`: installation directory. Default: `.agents/skills/adrm`.
- `--dry-run`: preview without writing.

Effects:

- Creates the skill directory if missing.
- Writes `SKILL.md` with managed version and content-hash metadata.

The default target path is `.agents/skills/adrm/SKILL.md`.

Safety: mutating. Supports `--dry-run`.

Errors:

- `skill_already_installed`: target `SKILL.md` already exists. Use
  `adrm skill update --dry-run`.
- `skill_directory_create_failed`: the skill directory could not be created.
- `skill_write_failed`: `SKILL.md` could not be written.

## `skill update`

Updates an installed ADRM agent skill.

```sh
adrm skill update --dry-run
adrm skill update
adrm skill update --force --dry-run
```

Flags:

- `--skill-dir`: installation directory. Default: `.agents/skills/adrm`.
- `--force`: overwrite a locally modified or unmanaged `SKILL.md`.
- `--dry-run`: preview without writing.

Behavior:

- Returns a no-op plan when the installed skill is already current.
- Updates unmodified ADRM-managed skill files.
- Refuses to overwrite local edits unless `--force` is passed.

Safety: mutating. Supports `--dry-run`.

Errors:

- `skill_not_installed`: target `SKILL.md` does not exist. Use
  `adrm skill install --dry-run`.
- `local_skill_modified`: target file is not an unmodified ADRM-managed skill.
- `skill_read_failed`: target file could not be read.
- `skill_write_failed`: target file could not be written.
