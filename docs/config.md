# Configuration

`canon` reads one optional configuration file, `.canon.jsonc`, from the
repository root. There is no global configuration: settings always belong to
the corpus they sit above.

Discovery walks up from the document's storage directory (for example
`docs/adr`) until it finds `.canon.jsonc` or reaches the filesystem root. A
repository without the file gets all defaults. Because discovery anchors on
the storage directory, passing `--adr-dir` to point at another corpus also
points at that corpus's configuration.

The file is JSONC: JSON with `//` line comments and `/* */` block comments,
parsed with the standard library after comment stripping. Unknown keys are
ignored so newer files stay readable by older binaries. A malformed file
fails the affected command with `invalid_config` (exit 4); fix or remove the
file to recover.

## Shape

```jsonc
{
  // Edit documents directly; git tracks the history.
  "schema_version": "canon.v1",
  "conventions": {
    "append": false
  }
}
```

## Keys

- `schema_version` (string, optional): `canon.v1` when present.
- `conventions.append` (bool, optional): when `false`, the `append` command
  is disabled for this corpus. Calls fail with `append_disabled` (exit 4),
  both with and without `--dry-run`, and `show` no longer suggests `append`
  as a next action. Existing appendices in documents are left untouched; only
  the command is switched off. Default: `true`.

See ADR-0015 for the decision behind this file.
