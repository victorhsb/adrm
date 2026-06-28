# Command Reference

`adrm` emits JSON by default. Global flags must appear before the command.

```sh
adrm --adr-dir docs/adr --format json list
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
- `--format`: output format. Values: `json`, `text`. Default: `json`.

## `commands`

Returns the machine-readable command registry.

```sh
adrm commands
```

Use this before automation. It declares purpose, side effects, selectors, examples,
dry-run support, and suggested next commands.

Safety: read-only.

## `doctor`

Checks whether ADR storage exists and whether ADR files can be parsed.

```sh
adrm doctor
```

Safety: read-only.

Common next action when missing storage:

```sh
adrm init --dry-run
adrm init
```

## `init`

Creates the ADR directory if it does not exist.

```sh
adrm init --dry-run
adrm init
```

Safety: mutating. Supports `--dry-run`.

## `new`

Creates a new ADR markdown file.

```sh
adrm new --title "Use SQLite for local index" --status proposed --dry-run
adrm new --title "Use SQLite for local index" --status proposed --tags "storage,query" --context "Agents need local lookup." --decision "Use SQLite."
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

## `list`

Lists ADR summaries in stable numeric order.

```sh
adrm list
adrm list --status accepted
adrm list --tag storage
```

Safety: read-only.

## `show`

Returns one ADR with metadata and markdown content.

```sh
adrm show --id ADR-0001
adrm show --id 1
```

Safety: read-only.

## `search`

Searches ADR id, title, status, tags, and markdown content.

```sh
adrm search --query "database"
adrm search database
adrm search --status deprecated
adrm search --tag storage --query local
```

Safety: read-only.

## `supersede`

Marks one ADR as superseded by another existing ADR.

```sh
adrm supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current design." --dry-run
adrm supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current design."
```

Effects:

- Sets `status: superseded`.
- Sets `superseded_by` to the replacement ADR id.
- Updates the Status section in the body.
- Appends a `History: Superseded` section.

Safety: mutating. Supports `--dry-run`.

## `deprecate`

Marks an ADR as deprecated without naming a replacement ADR.

```sh
adrm deprecate --id ADR-0003 --reason "The component was removed." --dry-run
adrm deprecate --id ADR-0003 --reason "The component was removed."
```

Effects:

- Sets `status: deprecated`.
- Sets `deprecated_by: manual`.
- Updates the Status section in the body.
- Appends a `History: Deprecated` section.

Safety: mutating. Supports `--dry-run`.

## `append`

Appends a dated appendix section to an ADR.

```sh
adrm append --id ADR-0002 --title "Implementation note" --body "The rollout used the local index." --dry-run
adrm append --id ADR-0002 --title "Implementation note" --body "The rollout used the local index."
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
