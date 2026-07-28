package adrm

type CommandInfo struct {
	Name        string   `json:"name"`
	Purpose     string   `json:"purpose"`
	Mutating    bool     `json:"mutating"`
	HasDryRun   bool     `json:"has_dry_run"`
	Safety      string   `json:"safety"`
	Selectors   []string `json:"selectors,omitempty"`
	Examples    []string `json:"examples"`
	NextCommand []string `json:"next_commands,omitempty"`
}

func commandRegistry() []CommandInfo {
	return []CommandInfo{
		{
			Name:      "commands",
			Purpose:   "Return the machine-readable command registry with safety and composition metadata.",
			Safety:    "read-only",
			Examples:  []string{"adrm commands"},
			Mutating:  false,
			HasDryRun: false,
		},
		{
			Name:      "doctor",
			Purpose:   "Check whether ADR and SPEC storage is present and parseable.",
			Safety:    "read-only",
			Examples:  []string{"adrm doctor"},
			Mutating:  false,
			HasDryRun: false,
		},
		{
			Name:      "init",
			Purpose:   "Create the ADR or SPEC directory if it does not exist.",
			Safety:    "write: creates directory only",
			Selectors: []string{"--kind"},
			Examples:  []string{"adrm init --kind adr --dry-run", "adrm init --kind adr", "adrm init --kind spec --dry-run", "adrm init --kind spec"},
			Mutating:  true,
			HasDryRun: true,
		},
		{
			Name:        "new",
			Purpose:     "Create a new ADR or SPEC markdown file with parseable metadata. SPEC files capture functional requirements.",
			Safety:      "write: creates one markdown file",
			Selectors:   []string{"--kind", "--title", "--status", "--tags"},
			Examples:    []string{`adrm new --kind adr --title "Use SQLite for local index" --status proposed --dry-run`, `adrm new --kind adr --title "Use SQLite for local index" --context "Need local querying" --decision "Use SQLite"`, `adrm new --kind spec --title "Local query index" --requirements "Return ADRs by tag." --acceptance "list --tag storage works." --dry-run`},
			NextCommand: []string{"adrm list", "adrm show --id ADR-0001", "adrm show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "list",
			Purpose:     "List ADR and SPEC summaries in deterministic order.",
			Safety:      "read-only",
			Selectors:   []string{"--kind", "--status", "--tag"},
			Examples:    []string{"adrm list", "adrm list --kind adr --status accepted", "adrm list --kind spec --tag storage", "adrm list --kind spec"},
			NextCommand: []string{"adrm show --id ADR-0001", "adrm show --id SPEC-0001", "adrm search query"},
			Mutating:    false,
			HasDryRun:   false,
		},
		{
			Name:        "show",
			Purpose:     "Return one ADR or SPEC with metadata and content.",
			Safety:      "read-only",
			Selectors:   []string{"--id"},
			Examples:    []string{"adrm show --id ADR-0001", "adrm show --id SPEC-0001"},
			NextCommand: []string{"adrm append --id ADR-0001 --title Note --body ... --dry-run", "adrm append --id SPEC-0001 --title Note --body ... --dry-run"},
			Mutating:    false,
			HasDryRun:   false,
		},
		{
			Name:        "search",
			Purpose:     "Search title, tags, status, kind, and markdown body across ADRs and SPECs.",
			Safety:      "read-only",
			Selectors:   []string{"--query", "--kind", "--status", "--tag"},
			Examples:    []string{`adrm search --query "database"`, "adrm search --kind spec --query requirements", "adrm search --status deprecated"},
			NextCommand: []string{"adrm show --id ADR-0001", "adrm show --id SPEC-0001"},
			Mutating:    false,
			HasDryRun:   false,
		},
		{
			Name:        "accept",
			Purpose:     "Mark an ADR or SPEC as accepted.",
			Safety:      "write: updates one document metadata block and appends history",
			Selectors:   []string{"--id", "--reason"},
			Examples:    []string{"adrm accept --id ADR-0001 --reason \"Approved by the team.\" --dry-run", "adrm accept --id SPEC-0001 --reason \"Requirements approved.\" --dry-run"},
			NextCommand: []string{"adrm show --id ADR-0001", "adrm show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "reject",
			Purpose:     "Mark an ADR or SPEC as rejected.",
			Safety:      "write: updates one document metadata block and appends history",
			Selectors:   []string{"--id", "--reason"},
			Examples:    []string{"adrm reject --id ADR-0001 --reason \"Chose a different approach.\" --dry-run", "adrm reject --id SPEC-0001 --reason \"Requirements changed.\" --dry-run"},
			NextCommand: []string{"adrm show --id ADR-0001", "adrm show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "supersede",
			Purpose:     "Mark an ADR or SPEC as superseded by another document of the same kind.",
			Safety:      "write: updates two document metadata blocks and appends history",
			Selectors:   []string{"--id", "--by", "--reason"},
			Examples:    []string{"adrm supersede --id ADR-0001 --by ADR-0002 --reason \"Replaced by current architecture\" --dry-run", "adrm supersede --id SPEC-0001 --by SPEC-0002 --reason \"Requirements split.\" --dry-run"},
			NextCommand: []string{"adrm show --id ADR-0001", "adrm show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "deprecate",
			Purpose:     "Mark an ADR or SPEC as deprecated without a direct replacement.",
			Safety:      "write: updates one document metadata block and appends history",
			Selectors:   []string{"--id", "--reason"},
			Examples:    []string{"adrm deprecate --id ADR-0001 --reason \"No longer used\" --dry-run", "adrm deprecate --id SPEC-0001 --reason \"Requirements moved.\" --dry-run"},
			NextCommand: []string{"adrm show --id ADR-0001", "adrm show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "append",
			Purpose:     "Append a dated appendix section to an ADR or SPEC.",
			Safety:      "write: appends markdown to one document",
			Selectors:   []string{"--id", "--title", "--body"},
			Examples:    []string{"adrm append --id ADR-0001 --title \"2026 review\" --body \"Still valid.\" --dry-run", "adrm append --id SPEC-0001 --title Review --body \"Requirements still apply.\" --dry-run"},
			NextCommand: []string{"adrm show --id ADR-0001", "adrm show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:      "skill",
			Purpose:   "Print bundled agent instructions for using this CLI safely and effectively.",
			Safety:    "read-only",
			Examples:  []string{"adrm skill"},
			Mutating:  false,
			HasDryRun: false,
		},
		{
			Name:        "skill install",
			Purpose:     "Install the ADRM agent skill into a repository-local skill directory.",
			Safety:      "write: creates a skill directory and SKILL.md",
			Selectors:   []string{"--skill-dir"},
			Examples:    []string{"adrm skill install --dry-run", "adrm skill install", "adrm skill install --skill-dir .agents/skills/adrm --dry-run"},
			NextCommand: []string{"adrm skill update --dry-run", "adrm commands"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "skill update",
			Purpose:     "Update an installed ADRM agent skill when it is unchanged or explicitly forced.",
			Safety:      "write: updates one SKILL.md",
			Selectors:   []string{"--skill-dir", "--force"},
			Examples:    []string{"adrm skill update --dry-run", "adrm skill update", "adrm skill update --force --dry-run"},
			NextCommand: []string{"adrm commands"},
			Mutating:    true,
			HasDryRun:   true,
		},
	}
}
