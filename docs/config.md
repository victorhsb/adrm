# Configuration

`canon` reads one optional configuration file, `.canon.jsonc`, from the
repository root. There is no global configuration: settings always belong to
the corpus they sit above (ADR-0016).

Configuration is a repository policy layer. It declares conventions the CLI
enforces as a floor: no flag weakens them, there are no bypass flags, and dry
runs pass through the same gates as real mutations. Configuration stays
limited to repository conventions. Output format, query defaults, storage
directories, id and status vocabulary, document structure, and skill install
options remain CLI or document-contract concerns and never move into the file.

## Discovery and scope

Discovery walks up from a store directory (for example `docs/adr`) until it
finds `.canon.jsonc` or reaches the filesystem root, so passing `--adr-dir`
at another corpus also points at that corpus's configuration. A store that
does not exist yet still discovers configuration by walking up from its
configured path.

Kind- or id-scoped commands resolve policy from the selected store. Corpus
commands (`doctor`, plain `validate`, `config show`, `config validate`)
reconcile discovery across all three stores: every store must discover the
same file, or every store must discover none. Mixed or conflicting sources
fail with `config_scope_mismatch` (exit 4) instead of guessing which policy
wins; point all three directory flags at one corpus to recover.

A repository without the file gets all defaults, which match the behavior of
versions before the policy layer existed.

## Inspecting configuration

```sh
canon config show
canon config validate
canon --format text config show
```

`config show` reports the source (`file` or `defaults`), the discovered path,
the per-kind discovery paths, every effective convention value, the sorted
recognized keys present in the file, and the sorted unknown key paths.
`config validate` reports deterministic findings and a summary, and includes
the same effective report when the configuration is valid. Both are read-only
and reject the `context` output format.

## Shape and defaults

```jsonc
{
  "schema_version": "canon.v1",
  "conventions": {
    "append": false,
    "required_kinds": ["adr", "domain"],
    "validation": {
      "strict": true
    },
    "lifecycle": {
      "require_reason": true,
      "new_documents_must_be_proposed": true
    },
    "tags": {
      "adr": {
        "allowed": ["adr", "agents", "cli", "config"]
      },
      "domain": {
        "allowed": ["glossary", "process"]
      }
    }
  }
}
```

The file is JSONC: JSON with `//` line comments and `/* */` block comments,
parsed with the standard library after comment stripping.

When a file or key is absent, these defaults resolve in code:

| Setting | Default | Effect |
|---|---|---|
| `schema_version` | `canon.v1` | Omitted means v1; an explicit unsupported version is invalid. |
| `conventions.append` | `true` | Keep the `append` command enabled. |
| `conventions.required_kinds` | `adr`, `spec`, `domain` | All three stores are required for readiness. |
| `conventions.validation.strict` | `false` | Strictness stays invocation-only via `--strict`. |
| `conventions.lifecycle.require_reason` | `false` | Lifecycle reasons stay optional. |
| `conventions.lifecycle.new_documents_must_be_proposed` | `false` | Any valid creation status stays allowed. |
| `conventions.tags.<kind>.allowed` | absent | An omitted kind has no tag restriction. |

## Keys

- `schema_version` (string, optional): must be `canon.v1` when present.
- `conventions.append` (bool, optional): when `false`, the `append` command
  fails with `append_disabled` (exit 4) and `show` stops suggesting it.
  Existing appendices in documents are untouched.
- `conventions.required_kinds` (array of strings, optional): the kinds whose
  stores must exist for corpus readiness. Must be a non-empty,
  duplicate-free subset of `adr`, `spec`, and `domain`. Requiredness declares
  readiness, not feature enablement: a missing required store keeps the
  existing warning, a missing non-required store is healthy and never
  suggests `init`, and a non-required store that exists is still scanned,
  listed, and validated. Kind-prefixed commands stay available for
  non-required kinds.
- `conventions.validation.strict` (bool, optional): when `true`, `validate`
  exits 4 whenever warnings exist. It combines with the `--strict` flag by
  logical OR; neither weakens the other. Finding severities and envelope
  status are unchanged. `config validate` ignores this key so an older binary
  can always report unknown future keys without failing on them.
- `conventions.lifecycle.require_reason` (bool, optional): when `true`,
  `accept`, `reject`, `supersede`, and `deprecate` reject a blank `--reason`
  with `reason_required_by_config` (exit 4), in dry-run and apply modes.
- `conventions.lifecycle.new_documents_must_be_proposed` (bool, optional):
  when `true`, `adr new`, `spec new`, and `domain new` reject any `--status`
  other than `proposed` with `initial_status_restricted` (exit 4).
- `conventions.tags.<kind>.allowed` (array of strings, optional): the tag
  vocabulary for one kind. Declaring it restricts the kind: `new` rejects
  tags outside the list with `disallowed_tag` (exit 4), and deep validation
  reports existing documents whose tags fall outside as error findings.
  Allowed tags must be non-blank after trimming and duplicate-free;
  comparison is exact and case-sensitive, matching `--tag` filters. An
  explicit empty array permits no tags. Kinds not listed under `tags` are
  unrestricted, and hand-edited documents are never rewritten.

## Validation and unknown keys

Hard configuration errors (each reported by `config validate` as an error
finding, and by policy-aware commands as a `config`-category error with exit
4):

- `malformed_config`: the file is not valid JSONC.
- `unsupported_config_schema`: an explicit `schema_version` other than
  `canon.v1`.
- `invalid_config_value`: wrong JSON types, empty or duplicate or unknown
  `required_kinds`, unknown `tags` kinds, and blank or duplicate allowed
  tags.
- `config_scope_mismatch`: corpus stores resolve to different files, or only
  some stores discover a file.

Unknown object keys are ignored by policy consumers so newer files stay
readable by older binaries. `config show` reports their sorted JSON paths in
`unknown_keys`, and `config validate` emits one `unknown_config_key` warning
per key while still exiting 0 when they are the only findings.

Every command that consults configuration fails with `invalid_config` (exit
4) when the file is malformed, unsupported, or semantically invalid, before
any read or mutation. `init` never loads configuration, so broken policy
cannot block storage recovery.

## Recovery

- Malformed file: run `canon config validate` to see the finding, then fix
  the syntax or remove the file to return to defaults.
- Unwritable mutation: run `canon config show` and compare the effective
  values against the conventions you intended; either change the command
  (status, tags, reason) or edit the policy.
- Scope mismatch: point `--adr-dir`, `--spec-dir`, and `--domain-dir` at one
  corpus root, or place `.canon.jsonc` at the shared root.

See ADR-0016 for the decision behind this policy layer.
