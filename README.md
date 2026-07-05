# adrm

`adrm` is a CLI for managing Architecture Decision Records in agent-led workflows.

The initial command surface is intentionally agent-centric:

- JSON output by default, with a versioned envelope.
- `adrm commands` for machine-readable discovery of commands, side effects, selectors, and examples.
- `--dry-run` on every mutating command.
- Structured errors with codes, categories, and suggested fixes.
- Stable ADR ids like `ADR-0001` for command composition.
- `adrm skill` plus `adrm skill install/update` to publish repository-local
  agent instructions for operating the CLI.

## Build

```sh
go build ./cmd/adrm
```

## Install

Use the install script to build and place the binary on your PATH:

```sh
scripts/install.sh              # installs to ~/.local/bin or /usr/local/bin
scripts/install.sh --dry-run      # preview the install plan
scripts/install.sh --prefix /opt  # install to /opt/bin
```

To remove the binary later:

```sh
scripts/install.sh --uninstall
```

## Quick start

```sh
adrm commands
adrm doctor
adrm init --dry-run
adrm init
adrm new --title "Use ADRM for decisions" --status proposed --dry-run
adrm new --title "Use ADRM for decisions" --status proposed
adrm list
adrm show --id ADR-0001
adrm skill install --dry-run
adrm skill install
```

ADRs are stored as markdown files in `docs/adr` by default. Use `--adr-dir` before
the command to select another directory:

```sh
adrm --adr-dir architecture/decisions list
```

## Documentation

- [Command reference](docs/commands.md)
- [ADR file format](docs/adr-format.md)
- [Agent workflow guide](docs/agent-workflows.md)
- [Project roadmap](docs/roadmap.md)

## Design principles

`adrm` is designed for agents first:

- Discovery is structured through `adrm commands`.
- Mutations are previewable with `--dry-run`.
- Output is parseable JSON unless `--format text` is explicitly requested.
- Failures include machine-readable error codes and suggested fixes.
- Command outputs include stable ids and next actions for workflow composition.
- The agent skill can be installed into `.agents/skills/adrm/SKILL.md` and later
  updated through previewable CLI commands.
