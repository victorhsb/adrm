package canon

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
			Examples:  []string{"canon commands"},
			Mutating:  false,
			HasDryRun: false,
		},
		{
			Name:      "doctor",
			Purpose:   "Check whether ADR and SPEC storage is present and parseable.",
			Safety:    "read-only",
			Examples:  []string{"canon doctor"},
			Mutating:  false,
			HasDryRun: false,
		},
		{
			Name:      "init",
			Purpose:   "Create the ADR or SPEC directory if it does not exist.",
			Safety:    "write: creates directory only",
			Selectors: []string{"--kind"},
			Examples:  []string{"canon init --kind adr --dry-run", "canon init --kind adr", "canon init --kind spec --dry-run", "canon init --kind spec"},
			Mutating:  true,
			HasDryRun: true,
		},
		{
			Name:        "new",
			Purpose:     "Create a new ADR or SPEC markdown file with parseable metadata. SPEC files capture functional requirements.",
			Safety:      "write: creates one markdown file",
			Selectors:   []string{"--kind", "--title", "--status", "--tags"},
			Examples:    []string{`canon new --kind adr --title "Use SQLite for local index" --status proposed --dry-run`, `canon new --kind adr --title "Use SQLite for local index" --context "Need local querying" --decision "Use SQLite"`, `canon new --kind spec --title "Local query index" --requirements "Return ADRs by tag." --acceptance "list --tag storage works." --dry-run`},
			NextCommand: []string{"canon list", "canon show --id ADR-0001", "canon show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "list",
			Purpose:     "List ADR and SPEC summaries in deterministic order.",
			Safety:      "read-only",
			Selectors:   []string{"--kind", "--status", "--tag"},
			Examples:    []string{"canon list", "canon list --kind adr --status accepted", "canon list --kind spec --tag storage", "canon list --kind spec"},
			NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001", "canon search query"},
			Mutating:    false,
			HasDryRun:   false,
		},
		{
			Name:        "show",
			Purpose:     "Return one ADR or SPEC with metadata and content.",
			Safety:      "read-only",
			Selectors:   []string{"--id"},
			Examples:    []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
			NextCommand: []string{"canon append --id ADR-0001 --title Note --body ... --dry-run", "canon append --id SPEC-0001 --title Note --body ... --dry-run"},
			Mutating:    false,
			HasDryRun:   false,
		},
		{
			Name:        "search",
			Purpose:     "Search title, tags, status, kind, and markdown body across ADRs and SPECs.",
			Safety:      "read-only",
			Selectors:   []string{"--query", "--kind", "--status", "--tag"},
			Examples:    []string{`canon search --query "database"`, "canon search --kind spec --query requirements", "canon search --status deprecated"},
			NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
			Mutating:    false,
			HasDryRun:   false,
		},
		{
			Name:        "accept",
			Purpose:     "Mark an ADR or SPEC as accepted.",
			Safety:      "write: updates one document metadata block and appends history",
			Selectors:   []string{"--id", "--reason"},
			Examples:    []string{"canon accept --id ADR-0001 --reason \"Approved by the team.\" --dry-run", "canon accept --id SPEC-0001 --reason \"Requirements approved.\" --dry-run"},
			NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "reject",
			Purpose:     "Mark an ADR or SPEC as rejected.",
			Safety:      "write: updates one document metadata block and appends history",
			Selectors:   []string{"--id", "--reason"},
			Examples:    []string{"canon reject --id ADR-0001 --reason \"Chose a different approach.\" --dry-run", "canon reject --id SPEC-0001 --reason \"Requirements changed.\" --dry-run"},
			NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "supersede",
			Purpose:     "Mark an ADR or SPEC as superseded by another document of the same kind.",
			Safety:      "write: updates two document metadata blocks and appends history",
			Selectors:   []string{"--id", "--by", "--reason"},
			Examples:    []string{"canon supersede --id ADR-0001 --by ADR-0002 --reason \"Replaced by current architecture\" --dry-run", "canon supersede --id SPEC-0001 --by SPEC-0002 --reason \"Requirements split.\" --dry-run"},
			NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "deprecate",
			Purpose:     "Mark an ADR or SPEC as deprecated without a direct replacement.",
			Safety:      "write: updates one document metadata block and appends history",
			Selectors:   []string{"--id", "--reason"},
			Examples:    []string{"canon deprecate --id ADR-0001 --reason \"No longer used\" --dry-run", "canon deprecate --id SPEC-0001 --reason \"Requirements moved.\" --dry-run"},
			NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "append",
			Purpose:     "Append a dated appendix section to an ADR or SPEC.",
			Safety:      "write: appends markdown to one document",
			Selectors:   []string{"--id", "--title", "--body"},
			Examples:    []string{"canon append --id ADR-0001 --title \"2026 review\" --body \"Still valid.\" --dry-run", "canon append --id SPEC-0001 --title Review --body \"Requirements still apply.\" --dry-run"},
			NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:      "skill",
			Purpose:   "Print bundled agent instructions for using this CLI safely and effectively.",
			Safety:    "read-only",
			Examples:  []string{"canon skill"},
			Mutating:  false,
			HasDryRun: false,
		},
		{
			Name:        "skill install",
			Purpose:     "Install the CANON agent skill into a repository-local skill directory.",
			Safety:      "write: creates a skill directory and SKILL.md",
			Selectors:   []string{"--skill-dir"},
			Examples:    []string{"canon skill install --dry-run", "canon skill install", "canon skill install --skill-dir .agents/skills/canon --dry-run"},
			NextCommand: []string{"canon skill update --dry-run", "canon commands"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "skill update",
			Purpose:     "Update an installed CANON agent skill when it is unchanged or explicitly forced.",
			Safety:      "write: updates one SKILL.md",
			Selectors:   []string{"--skill-dir", "--force"},
			Examples:    []string{"canon skill update --dry-run", "canon skill update", "canon skill update --force --dry-run"},
			NextCommand: []string{"canon commands"},
			Mutating:    true,
			HasDryRun:   true,
		},
	}
}
