---
name: project-planning
description: Plan changes to Canon using current ADRs, SPECs, the Domain Model, repository conventions, and verification practices. Use whenever the user asks for a plan, design, implementation approach, sequencing, scope, impact analysis, or says to think through a change before coding, even if they do not mention ADRs or SPECs.
---

# Canon Project Planning

Produce implementation-ready plans that respect the project's current decisions and requirements. Keep planning read-only: inspect the repository, but do not edit files or mutate ADR/SPEC state.

## Gather Context

1. Check repository health and load the current binding document index:

   ```sh
   go run ./cmd/canon doctor
   go run ./cmd/canon --format context list --status accepted
   ```

2. Discover work in progress separately so it is not mistaken for an accepted commitment:

   ```sh
   go run ./cmd/canon --format context list --status proposed
   ```

3. Search by the task's domain terms, then open only relevant documents:

   ```sh
   go run ./cmd/canon search --query "<topic>"
   go run ./cmd/canon domain search --query "<term>"
   go run ./cmd/canon show --id <ADR-or-SPEC-id>
   ```

   Accepted domain entries pin what the task's terms mean. Use their canonical
   language in the plan, and challenge terminology that conflicts with them.

4. Inspect the implementation, tests, and nearby documentation needed to understand current behavior. Prefer repository evidence over assumptions.

The context lists are discovery aids, not substitutes for reading relevant records. Cite only documents that actually constrain or inform the task.

## Interpret Documents

- Treat accepted ADRs as binding architectural constraints.
- Treat accepted SPECs as approved requirements.
- Treat accepted domain entries as the canonical meaning of terms; the Domain Model is the single source of truth for what things mean.
- Treat proposed documents as non-binding design input. Label assumptions that depend on their eventual acceptance.
- Consult rejected, superseded, or deprecated documents only when history explains the current design.
- If the requested change would alter the system architecture, CLI contract, document format, query behavior, lifecycle semantics, output schema, or storage layout, identify an ADR decision gate instead of burying the choice inside implementation steps.
- Treat project processes, agent workflows, and skill behavior as instruction changes, not architecture decisions. Update `AGENTS.md` or the relevant `SKILL.md` directly.
- Do not propose a new ADR for routine implementation details that follow existing decisions.

## Build The Plan

Make each implementation step independently understandable and executable. Name likely files or packages when repository evidence supports them, and explain the behavior changed rather than listing vague activities.

Cover:

- Current behavior and the gap being addressed.
- Governing ADRs and SPECs, with status and relevance.
- Ordered implementation steps and dependencies.
- Tests, CLI smoke checks, and documentation updates.
- Compatibility, migration, or rollout concerns when applicable.
- Open questions that block a correct implementation.

Ask a focused question before finalizing only when an unresolved choice materially changes the plan. Otherwise state the assumption and continue.

## Plan Format

```markdown
# Plan: <short title>

## Goal
<Outcome and boundaries.>

## Governing Decisions
- <ADR/SPEC id, status, and effect on this plan. Use "None found" when appropriate.>

## Current State
<Relevant implementation and documentation evidence.>

## Implementation
1. <Concrete step, likely files, and intended behavior.>
2. <Next step and dependency.>

## Verification
- <Focused tests and executable commands.>

## Risks And Open Questions
- <Risk, decision gate, assumption, or "None".>
```

Keep the plan concise enough to execute, but include the reasoning needed to avoid violating accepted decisions.
