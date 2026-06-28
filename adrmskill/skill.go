package adrmskill

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	Name              = "adrm"
	Version           = "1"
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
description: Manage Architecture Decision Records with the adrm CLI in agent-led workflows.
---
`+versionComment()+`

# ADRM Agent Skill

Use adrm to manage Architecture Decision Records without guessing repository state.

## Operating rules

1. Start with `+"`adrm commands`"+` to inspect command metadata, side effects, selectors, examples, and dry-run availability.
2. Run `+"`adrm doctor`"+` before mutating ADRs. If it reports a missing ADR directory, preview initialization with `+"`adrm init --dry-run`"+` before applying `+"`adrm init`"+`.
3. Use JSON output unless a human explicitly asks for text. Every JSON response has `+"`schema_version`"+`, `+"`status`"+`, `+"`data`"+`, and optional `+"`error`"+` / `+"`next_actions`"+`.
4. For every mutating command, run the same command with `+"`--dry-run`"+` first and verify the returned plan. The dry-run response includes `+"`No changes were made.`"+` in warnings.
5. Use `+"`adrm list`"+`, `+"`adrm search --query ...`"+`, and `+"`adrm show --id ...`"+` to gather context before changing an ADR.
6. Prefer selectors from CLI output. ADR ids are stable strings like `+"`ADR-0001`"+` and can be passed to `+"`--id`"+` or `+"`--by`"+`.

## Common workflows

Create an ADR:

`+"```sh"+`
adrm new --title "Use SQLite for local query index" --status proposed --dry-run
adrm new --title "Use SQLite for local query index" --status proposed --context "Agents need fast local lookup." --decision "Use SQLite-backed indexes."
`+"```"+`

Supersede an ADR:

`+"```sh"+`
adrm search --query "old storage decision"
adrm supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current storage approach." --dry-run
adrm supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current storage approach."
`+"```"+`

Deprecate an ADR:

`+"```sh"+`
adrm deprecate --id ADR-0003 --reason "The system no longer uses this component." --dry-run
adrm deprecate --id ADR-0003 --reason "The system no longer uses this component."
`+"```"+`

Append new context:

`+"```sh"+`
adrm append --id ADR-0002 --title "Implementation note" --body "The initial rollout used the default local index." --dry-run
adrm append --id ADR-0002 --title "Implementation note" --body "The initial rollout used the default local index."
`+"```"+`

Install or update this skill in a repository:

`+"```sh"+`
adrm skill install --dry-run
adrm skill install
adrm skill update --dry-run
adrm skill update
`+"```"+`

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
