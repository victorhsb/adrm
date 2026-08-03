---
kind: spec
id: SPEC-0002
title: Bundled skill asset install and update
status: accepted
date: 2026-08-02
tags: agents, cli, skill
supersedes: 
superseded_by: 
deprecated_by: 
---
# SPEC-0002: Bundled skill asset install and update

## Status

accepted

## Context

Today canon skill install/update deploys exactly one embedded skill. Per ADR-0013, the skill command family absorbs subagent installation: the bundled payload becomes a managed asset bundle containing multiple skills (canon, canon-record-gate), multi-file skill payloads (canon-record-gate ships a references directory), and the canon-critic subagent rendered per target environment. Per ADR-0012 the first-party targets are OpenCode, Claude, and Codex, each with distinct subagent discovery directories and frontmatter conventions. Target projects vary in which environments they use, so installation must select targets explicitly or infer them from project layout.

## Requirements

Extend canon skill to print the full bundled asset catalog: for each asset its name, kind (skill or agent), version, content hash, and target paths. Extend canon skill install to deploy all bundled assets by default: one SKILL.md per skill under the skills directory (default .agents/skills/<name>/), all supporting payload files (for example references/boundary-examples.md), and the bundled subagent rendered into each selected target's agents directory (.opencode/agents/ for opencode, .claude/agents/ for claude) with target-appropriate frontmatter (opencode: mode subagent plus permission block; claude: tools/model-style frontmatter). A repeatable --only <name> flag restricts the operation to the named assets. A repeatable --agent <target> flag (opencode, claude, codex) selects subagent targets; when absent, targets are inferred from existing .opencode, .claude, and .codex directories in the target project, falling back to opencode when none exist. Extend canon skill update to refresh every installed managed file, reusing per-file version and hash markers: unmodified managed files are rewritten, locally modified files are refused unless --force, and current files report a noop. Install refuses to overwrite any existing target file and suggests skill update instead. Every mutating operation supports --dry-run, returns the full per-file plan, and carries the standard no-changes warning when dry.

## Constraints

No agent command group is introduced; skill remains the root command (ADR-0013). Output remains JSON-first with schema_version; text format is a rendering only. Plans and catalogs are deterministically ordered. Standard library only. Embedded asset content remains project-agnostic because it ships to other projects. Only OpenCode, Claude, and Codex are valid --agent targets (ADR-0012).

## Acceptance Criteria

canon skill lists canon and canon-record-gate as skills (no agent on the list), each with version and hash. 
canon skill install --dry-run in a fresh directory plans one write per bundled skill file, including canon-record-gate's references file, plus one subagent file per inferred or selected target, and writes nothing.
canon skill install --only canon installs only the canon skill.
canon skill install --agent claude writes .claude/agents/canon-critic.md with claude frontmatter and no .opencode directory.
canon skill install in a project containing only a .claude directory infers the claude target without flags;
in a project with no target directories it falls back to opencode.
canon skill update rewrites unmodified managed files, skips current files as noop, and refuses a locally modified file without --force. 
go test ./... passes with tests exercising Run through the JSON envelope.

## Appendix: Future extensions

Date: 2026-08-02

Codex subagent installation was deliberately deferred. Codex has no stable subagent discovery convention comparable to .opencode/agents or .claude/agents - it primarily reads AGENTS.md - so this SPEC covers skills installation for the codex target but no agent rendering. Revisit when Codex publishes a subagent discovery contract; acceptance criteria for it should be added here rather than assumed.

## Appendix: Codex skill and custom-agent compatibility

Date: 2026-08-03

This amendment supersedes the 2026-08-02 "Future extensions" deferral for
Codex. The current OpenAI documentation now defines both repository-local skill
discovery and project-scoped custom-agent discovery.

### Official Codex contract

- A repository-local skill is a directory below `.agents/skills` containing a
  `SKILL.md` with `name` and `description` frontmatter. Supporting
  `references/`, `assets/`, and `scripts/` files may live beside it. Codex scans
  `.agents/skills` from the working directory to the repository root and
  detects skill changes automatically; restart is only a recovery step when a
  change does not appear.
- A project-scoped custom Codex agent is a standalone TOML file below
  `.codex/agents/`. It must define `name`, `description`, and
  `developer_instructions`. A skill may instruct Codex to delegate work to that
  agent. Settings omitted from the agent file, including skill configuration,
  inherit from the parent session.
- Direct `.agents/skills` installation is the supported repository-local
  authoring and discovery mechanism used by this command. Plugins are the
  installable distribution mechanism for sharing reusable bundles through a
  marketplace. `canon skill install` does not claim to install or register a
  Codex plugin.

Sources: [Build skills](https://learn.chatgpt.com/docs/build-skills),
[Subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents), and
[Package your plugin](https://developers.openai.com/plugins/build/plugins).

### Amended requirements

The shared skill payload remains rooted at `.agents/skills/<name>/` by default
for every target. Target selection controls only target-specific agent
renderings; it must not change whether the selected skills are installed.

When `canon-record-gate` and the `codex` target are selected, `canon skill
install` and `canon skill update` must manage
`.codex/agents/canon-critic.toml`. The rendered TOML must:

- set `name` to `canon-critic`;
- provide a concise `description` that identifies record review as its trigger;
- set `sandbox_mode` to `read-only` because the critic must never mutate the
  corpus;
- place the shared critic instructions in `developer_instructions`, including
  the requirement to load and follow the installed `canon-record-gate` skill;
- omit a fixed `model` and reasoning effort so the agent inherits current
  parent/default selection; and
- encode the instruction body safely as valid TOML rather than interpolating
  unescaped source text.

Codex agent files must use TOML comments for managed metadata, for example
`# canon-skill-version: <version>` and
`# canon-skill-hash: sha256:<digest>`. Markdown skill and agent renderings keep
their existing HTML comment markers. Inspection and update logic must recognize
the marker syntax appropriate to each file type, exclude only the hash marker
from actual-content hashing, and preserve the same current, managed-old,
locally-modified, and unmanaged distinctions across Markdown and TOML.

The public asset catalog must include
`.codex/agents/canon-critic.toml` in `canon-record-gate.target_paths`. Explicit
`--agent codex` selection must plan only the Codex agent rendering in addition
to the shared skill files. Inference from an existing `.codex` directory remains
valid; when multiple supported target directories exist, all inferred targets
remain selected deterministically.

Optional `agents/openai.yaml` skill metadata is not required because these
skills need no OpenAI-specific UI policy or MCP dependency. Adding it later is
a managed payload change, not a prerequisite for Codex discovery.

### Amended acceptance criteria

- `canon skill` lists `.codex/agents/canon-critic.toml` among the deterministic
  target paths for `canon-record-gate`.
- `canon skill install --agent codex --dry-run` in a fresh project plans the
  selected shared skill files plus `.codex/agents/canon-critic.toml`, writes
  nothing, and plans no OpenCode or Claude agent file.
- Applying that command writes a valid Codex custom-agent TOML file with the
  required `name`, `description`, `developer_instructions`, and read-only
  sandbox setting; it does not pin a model or reasoning effort.
- The Codex critic instructions require `canon-record-gate` and preserve the
  same verdict contract and read-only behavior as the OpenCode and Claude
  renderings.
- `canon skill update --agent codex` reports `noop` for the current TOML file,
  updates an unmodified older managed TOML file, and refuses an edited or
  unmanaged TOML file without `--force`.
- Tests cover Codex path selection, valid TOML rendering and escaping,
  target-specific marker parsing and hashing, dry-run output, conflict refusal,
  update behavior, and deterministic ordering through the JSON envelope.
- `go test ./...` and `canon validate --strict` pass after implementation and
  documentation are updated.
