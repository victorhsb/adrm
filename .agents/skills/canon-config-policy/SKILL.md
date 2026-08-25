---
name: canon-config-policy
description: Add, change, or remove a repository policy key (.canon.jsonc convention) in Canon's configuration layer. Use whenever editing internal/canon/config.go or EffectiveConfig, gating a CLI feature behind a configured policy, or modifying docs/config.md, even if .canon.jsonc is not mentioned by name.
---

# Canon Config Policy

Additions to the configuration layer are mechanical but span every layer of the pipeline. A partial addition compiles and misbehaves: run every step below, in order. The schema contract is ADR-0016 and `docs/config.md`; read them before inventing a key shape.

## Scope Gate

ADR-0016 limits `.canon.jsonc` to conventions. Keep out output format, query defaults, storage directories, id and status vocabulary, document structure, and skill install options; those stay CLI or document-contract concerns. Policy is an enforcement floor: no flag weakens a configured setting, and flags combine with policy by logical OR, the way `--strict` combines with `conventions.validation.strict`.

## Add a Key

Each layer has one job. Raw model decodes, validation judges, effective model resolves, payload exposes, consumer enforces.

1. Add a pointer-backed field to the raw model in `internal/canon/config.go` (`rawConventions` or a nested raw struct). Pointer types keep an omitted key distinguishable from an explicit `false` or empty array.
2. Register the key path in `recognizedConfigChildren`. Skip this and every run reports the new key as an `unknown_config_key` warning.
3. Add semantic checks to `validateRawConfig` only when the value can be invalid beyond JSON type: emptiness, duplicates, unsupported members. Wrong types already fail as `invalid_config_value` through `json.UnmarshalTypeError`.
4. Add an unexported field to `EffectiveConfig` plus an exported accessor.
5. Set the default in `defaultEffectiveConfig` so repositories without the file keep their current behavior.
6. Resolve the raw value in `buildEffective`, preserving deterministic ordering for lists.
7. Project the value in `internal/canon/config_command.go`: extend the payload structs and `newConfigReport` so `config show` exposes it.
8. Enforce the policy in the consuming command (`cli.go` or `validate.go`). Consult it before any read or mutation, and surface violations as config-category errors with exit 4, in dry-run and apply modes alike.
9. Add focused tests in `internal/canon` that exercise `Run` against a temp config file, following the `config_test.go` patterns.
10. Document the key, its default, and its error codes in `docs/config.md`.

Adopting the key in this corpus is a separate step: update `.canon.jsonc` and the Repository Policy section of `AGENTS.md` in the same change.

## Change or Remove a Key

Renaming or removing a known key turns files that still declare it into `unknown_config_key` warnings, which never fail commands. Older binaries tolerate newer keys by the same rule. Breaking schema changes require a `schema_version` bump, not silent key edits.

## Verify

```sh
go test ./...
go run ./cmd/canon config validate
go run ./cmd/canon config show
go run ./cmd/canon doctor
```

Done means: `config show` reports the new key's effective value from both the file and the defaults, `config validate` reports no `unknown_config_key` finding for it, tests pass, and `docs/config.md` matches the behavior.
