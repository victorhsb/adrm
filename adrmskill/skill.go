package adrmskill

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	Name              = "adrm"
	Version           = "3"
	FileName          = "SKILL.md"
	DefaultInstallDir = ".agents/skills/adrm"
)

type Inspection struct {
	Version      string
	DeclaredHash string
	ActualHash   string
	Managed      bool
	Current      bool
	Modified     bool
}

func TargetPath(dir string) string {
	if strings.TrimSpace(dir) == "" {
		dir = DefaultInstallDir
	}
	return filepath.Join(dir, FileName)
}

func Content() string {
	payload := managedPayload()
	return strings.Replace(payload, versionComment()+"\n", versionComment()+"\n"+hashComment(Hash())+"\n", 1)
}

func Hash() string {
	sum := sha256.Sum256([]byte(managedPayload()))
	return "sha256:" + fmt.Sprintf("%x", sum[:])
}

func Inspect(content string) Inspection {
	inspection := Inspection{ActualHash: hashWithoutHashComment(content)}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "<!-- adrm-skill-version:") && strings.HasSuffix(line, "-->"):
			inspection.Version = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "<!-- adrm-skill-version:"), "-->"))
		case strings.HasPrefix(line, "<!-- adrm-skill-hash:") && strings.HasSuffix(line, "-->"):
			inspection.DeclaredHash = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "<!-- adrm-skill-hash:"), "-->"))
		}
	}
	inspection.Managed = inspection.Version != "" && inspection.DeclaredHash != "" && inspection.DeclaredHash == inspection.ActualHash
	inspection.Current = content == Content()
	inspection.Modified = inspection.DeclaredHash != "" && inspection.DeclaredHash != inspection.ActualHash
	return inspection
}

func hashWithoutHashComment(content string) string {
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!-- adrm-skill-hash:") && strings.HasSuffix(trimmed, "-->") {
			continue
		}
		kept = append(kept, line)
	}
	sum := sha256.Sum256([]byte(strings.Join(kept, "\n")))
	return "sha256:" + fmt.Sprintf("%x", sum[:])
}

func managedPayload() string {
	return strings.TrimSpace(`---
name: adrm
description: Manage Architecture Decision Records (ADRs) and SPECs with the adrm CLI. Use whenever creating, recording, or revisiting an architectural decision; transitioning an ADR or SPEC through its lifecycle (accept, reject, supersede, deprecate, append); querying decision history; or initializing ADR storage - even if the user does not mention adrm by name.
---
`+versionComment()+`

# ADRM Agent Skill

Use adrm to manage Architecture Decision Records without guessing repository state.

## Operating rules

1. Start with `+"`adrm commands`"+` to inspect command metadata, side effects, selectors, examples, and dry-run availability.
2. Run `+"`adrm doctor`"+` before mutating ADRs. If it reports a missing ADR directory, preview initialization with `+"`adrm init --dry-run`"+` before applying `+"`adrm init`"+`.
3. Use JSON output unless a human explicitly asks for text. Every JSON response has `+"`schema_version`"+`, `+"`status`"+`, `+"`data`"+`, and optional `+"`error`"+` / `+"`next_actions`"+`.
4. For every mutating command, run the same command with `+"`--dry-run`"+` first and verify the returned plan. The plan response is how you confirm selectors and side effects before anything touches disk; a correct dry-run carries `+"`No changes were made.`"+` in warnings.
5. Use `+"`adrm list`"+`, `+"`adrm search --query ...`"+`, and `+"`adrm show --id ...`"+` to gather context before changing an ADR.
6. Prefer selectors from CLI output. ADR ids are stable strings like `+"`ADR-0001`"+` and can be passed to `+"`--id`"+` or `+"`--by`"+`.

## Common commands

Preview each of these with `+"`--dry-run`"+` first (rule 4), then apply:

`+"```sh"+`
adrm new --title "Use SQLite for local query index" --status proposed --context "Agents need fast local lookup." --decision "Use SQLite-backed indexes."
adrm accept --id ADR-0001 --reason "Approved by the team."
adrm reject --id ADR-0001 --reason "Chose a different approach."
adrm supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current storage approach."
adrm deprecate --id ADR-0003 --reason "The system no longer uses this component."
adrm append --id ADR-0002 --title "Implementation note" --body "The initial rollout used the default local index."
adrm skill install
adrm skill update
`+"```"+`

## When to create or change an ADR

`+"`adrm`"+` stores *Architecture Decision Records*, not plans, tickets, or changelogs. Use an ADR when these four tests are all true:

1. **It is a commitment, not an intention.** Past-tense: "We decided X". Not "We will add X".
2. **It is architectural.** It shapes the system's structure, contract, data model, or cross-cutting policy, and reversal would ripple.
3. **It is non-obvious.** Reasonable people might choose differently, so the reasoning is worth preserving.
4. **It is narrow.** One ADR per decision. Bundles hide the real tradeoff.

### Technical vs product decisions

A pure product decision (market, prioritization, pricing) is **not** architecture and does not belong in an ADR. It belongs in an ADR only when it **forces an architectural commitment**. In that case, the product driver goes in **Context** as a force, and the **Decision** is the architectural commitment it produced.

### `+"`adrm`"+` trigger list

Decisions that affect the CLI contract, ADR file format, query behavior, lifecycle semantics, output schema, storage layout, or agent operating model are almost always architectural. Not every change to those surfaces is a commitment, but a change that fixes a contract downstream consumers will depend on is.

### Anti-patterns

Do not create an ADR for:

- a roadmap ("we will add A, B, C")
- a ticket ("add command X")
- a changelog entry ("implemented back-references")
- a bundle of unrelated decisions
- a product strategy with no architectural consequence
- vague commitments ("be flexible")
- obvious decisions with no real alternatives

## Recovery

If a command fails, read `+"`error.code`"+`, `+"`error.category`"+`, and `+"`error.suggested_fix`"+`. Prefer the suggested next diagnostic command over guessing. For missing or unreadable ADR state, run `+"`adrm doctor`"+`.
`) + "\n"
}

func versionComment() string {
	return "<!-- adrm-skill-version: " + Version + " -->"
}

func hashComment(hash string) string {
	return "<!-- adrm-skill-hash: " + hash + " -->"
}
