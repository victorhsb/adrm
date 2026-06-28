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
			Purpose:   "Check whether ADR storage is present and parseable.",
			Safety:    "read-only",
			Examples:  []string{"adrm doctor"},
			Mutating:  false,
			HasDryRun: false,
		},
		{
			Name:      "init",
			Purpose:   "Create the ADR directory if it does not exist.",
			Safety:    "write: creates directory only",
			Examples:  []string{"adrm init --dry-run", "adrm init"},
			Mutating:  true,
			HasDryRun: true,
		},
		{
			Name:        "new",
			Purpose:     "Create a new ADR markdown file with parseable metadata.",
			Safety:      "write: creates one markdown file",
			Selectors:   []string{"--title", "--status", "--tags"},
			Examples:    []string{`adrm new --title "Use SQLite for local index" --status proposed --dry-run`, `adrm new --title "Use SQLite for local index" --context "Need local querying" --decision "Use SQLite"`},
			NextCommand: []string{"adrm list", "adrm show --id ADR-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "list",
			Purpose:     "List ADR summaries in deterministic numeric order.",
			Safety:      "read-only",
			Selectors:   []string{"--status", "--tag"},
			Examples:    []string{"adrm list", "adrm list --status accepted", "adrm list --tag storage"},
			NextCommand: []string{"adrm show --id ADR-0001", "adrm search query"},
			Mutating:    false,
			HasDryRun:   false,
		},
		{
			Name:        "show",
			Purpose:     "Return one ADR with metadata and content.",
			Safety:      "read-only",
			Selectors:   []string{"--id"},
			Examples:    []string{"adrm show --id ADR-0001"},
			NextCommand: []string{"adrm append --id ADR-0001 --title Note --body ... --dry-run"},
			Mutating:    false,
			HasDryRun:   false,
		},
		{
			Name:        "search",
			Purpose:     "Search title, tags, status, and markdown body.",
			Safety:      "read-only",
			Selectors:   []string{"--query", "--status", "--tag"},
			Examples:    []string{`adrm search --query "database"`, "adrm search --status deprecated"},
			NextCommand: []string{"adrm show --id ADR-0001"},
			Mutating:    false,
			HasDryRun:   false,
		},
		{
			Name:        "supersede",
			Purpose:     "Mark an ADR as superseded by another ADR.",
			Safety:      "write: updates one ADR metadata block and appends history",
			Selectors:   []string{"--id", "--by", "--reason"},
			Examples:    []string{"adrm supersede --id ADR-0001 --by ADR-0002 --reason \"Replaced by current architecture\" --dry-run"},
			NextCommand: []string{"adrm show --id ADR-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "deprecate",
			Purpose:     "Mark an ADR as deprecated without a direct superseding ADR.",
			Safety:      "write: updates one ADR metadata block and appends history",
			Selectors:   []string{"--id", "--reason"},
			Examples:    []string{"adrm deprecate --id ADR-0001 --reason \"No longer used\" --dry-run"},
			NextCommand: []string{"adrm show --id ADR-0001"},
			Mutating:    true,
			HasDryRun:   true,
		},
		{
			Name:        "append",
			Purpose:     "Append a dated appendix section to an ADR.",
			Safety:      "write: appends markdown to one ADR",
			Selectors:   []string{"--id", "--title", "--body"},
			Examples:    []string{"adrm append --id ADR-0001 --title \"2026 review\" --body \"Still valid.\" --dry-run"},
			NextCommand: []string{"adrm show --id ADR-0001"},
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
