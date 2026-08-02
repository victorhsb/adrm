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
