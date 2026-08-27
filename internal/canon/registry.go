package canon

import "io"

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

// commandContext carries the per-invocation state a command handler needs:
// output writers, validated global options, the document stores, the
// arguments remaining after command matching, and the matched command name.
type commandContext struct {
	stdout io.Writer
	stderr io.Writer
	opts   GlobalOptions
	repo   Repo
	args   []string
	name   string
}

// commandHandler executes one registered command against a context.
type commandHandler func(commandContext) int

// commandEntry couples the public discovery metadata for a command with its
// executable handler, so metadata and dispatch share one source of truth.
type commandEntry struct {
	info    CommandInfo
	handler commandHandler
}

// kindAdapter adapts a handler with a trailing kind parameter to the
// commandHandler signature. Kind values are frozen at declaration time from
// the same constants runKindCommand used, so table entries cannot select an
// empty or incorrect kind at dispatch.
func kindAdapter(run func(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string, kind string) int, kind string) commandHandler {
	return func(ctx commandContext) int {
		return run(ctx.stdout, ctx.stderr, ctx.opts, ctx.repo, ctx.args, kind)
	}
}

// lifecycleTransition names what a status-changing lifecycle command applies:
// the target status and the history heading recorded on the document. The
// command name comes from the matched table entry, not from the transition.
type lifecycleTransition struct {
	status       string
	historyTitle string
}

func lifecycleHandler(transition lifecycleTransition) commandHandler {
	return func(ctx commandContext) int {
		return runLifecycle(ctx.stdout, ctx.stderr, ctx.opts, ctx.repo, ctx.args, ctx.name, transition)
	}
}

// commandTable is the single ordered source of truth for the CLI surface.
// Order is the public discovery order exposed by `canon commands`. Each
// entry couples complete CommandInfo metadata with a non-nil handler; the
// table as a whole must never be JSON-encoded because handlers cannot be
// encoded. Kind, config, index, and skill prefixes are command families, not
// entries: dispatch resolves them into the matching child entries.
//
// It is a function rather than a package variable because entries reference
// handler functions that themselves read the table, which would form an
// initialization cycle. Each call returns a fresh slice, so callers can
// mutate their copy without affecting discovery or dispatch.
func commandTable() []commandEntry {
	return []commandEntry{
		{
			info: CommandInfo{
				Name:      "commands",
				Purpose:   "Return the machine-readable command registry with safety and composition metadata.",
				Safety:    "read-only",
				Examples:  []string{"canon commands"},
				Mutating:  false,
				HasDryRun: false,
			},
			handler: func(ctx commandContext) int { return runCommands(ctx.stdout, ctx.opts) },
		},
		{
			info: CommandInfo{
				Name:      "version",
				Purpose:   "Print the canon build version.",
				Safety:    "read-only",
				Examples:  []string{"canon version"},
				Mutating:  false,
				HasDryRun: false,
			},
			handler: func(ctx commandContext) int { return runVersion(ctx.stdout, ctx.opts) },
		},
		{
			info: CommandInfo{
				Name:      "doctor",
				Purpose:   "Check whether required ADR, SPEC, and domain storage is present and parseable, and flag domain-model integrity problems (duplicate accepted titles, references to superseded or deprecated entries). Required kinds come from repository configuration; absent optional stores are healthy.",
				Safety:    "read-only",
				Examples:  []string{"canon doctor"},
				Mutating:  false,
				HasDryRun: false,
			},
			handler: func(ctx commandContext) int { return runDoctor(ctx.stdout, ctx.opts, ctx.repo) },
		},
		{
			info: CommandInfo{
				Name:        "validate",
				Purpose:     "Run the full corpus integrity check catalog: malformed files, duplicate ids, broken references, reciprocity, status and date validity, kind/id/directory coherence, and configured tag vocabularies. Never mutates; exit 4 on errors, or on warnings with --strict or configured strictness.",
				Safety:      "read-only",
				Selectors:   []string{"--id", "--strict"},
				Examples:    []string{"canon validate", "canon validate --strict", "canon validate --id ADR-0001"},
				NextCommand: []string{"canon doctor", "canon show --id ADR-0001"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runValidate, ""),
		},
		{
			info: CommandInfo{
				Name:        "adr validate",
				Purpose:     "Run the corpus integrity checks scoped to the ADR directory.",
				Safety:      "read-only",
				Selectors:   []string{"--strict"},
				Examples:    []string{"canon adr validate"},
				NextCommand: []string{"canon validate"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runValidate, KindADR),
		},
		{
			info: CommandInfo{
				Name:        "spec validate",
				Purpose:     "Run the corpus integrity checks scoped to the SPEC directory.",
				Safety:      "read-only",
				Selectors:   []string{"--strict"},
				Examples:    []string{"canon spec validate"},
				NextCommand: []string{"canon validate"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runValidate, KindSPEC),
		},
		{
			info: CommandInfo{
				Name:        "domain validate",
				Purpose:     "Run the corpus integrity checks scoped to the domain entry directory.",
				Safety:      "read-only",
				Selectors:   []string{"--strict"},
				Examples:    []string{"canon domain validate"},
				NextCommand: []string{"canon validate"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runValidate, KindDomain),
		},
		{
			info: CommandInfo{
				Name:        "config show",
				Purpose:     "Show the effective repository configuration resolved from .canon.jsonc (or the defaults), including discovery paths, recognized keys, and ignored unknown keys.",
				Safety:      "read-only",
				Examples:    []string{"canon config show", "canon --format text config show"},
				NextCommand: []string{"canon config validate", "canon doctor"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: func(ctx commandContext) int {
				return runConfigShow(ctx.stdout, ctx.stderr, ctx.opts, ctx.repo, ctx.args)
			},
		},
		{
			info: CommandInfo{
				Name:        "config validate",
				Purpose:     "Validate .canon.jsonc against the configuration schema: malformed files, unsupported schema versions, invalid values, and scope mismatches are errors; unknown keys are warnings that never fail the command. Exit 4 on errors.",
				Safety:      "read-only",
				Examples:    []string{"canon config validate"},
				NextCommand: []string{"canon config show", "canon validate"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: func(ctx commandContext) int {
				return runConfigValidate(ctx.stdout, ctx.stderr, ctx.opts, ctx.repo, ctx.args)
			},
		},
		{
			info: CommandInfo{
				Name:      "adr init",
				Purpose:   "Create the ADR directory if it does not exist.",
				Safety:    "write: creates directory only",
				Examples:  []string{"canon adr init --dry-run", "canon adr init"},
				Mutating:  true,
				HasDryRun: true,
			},
			handler: kindAdapter(runInit, KindADR),
		},
		{
			info: CommandInfo{
				Name:      "spec init",
				Purpose:   "Create the SPEC directory if it does not exist.",
				Safety:    "write: creates directory only",
				Examples:  []string{"canon spec init --dry-run", "canon spec init"},
				Mutating:  true,
				HasDryRun: true,
			},
			handler: kindAdapter(runInit, KindSPEC),
		},
		{
			info: CommandInfo{
				Name:      "domain init",
				Purpose:   "Create the domain entry directory if it does not exist.",
				Safety:    "write: creates directory only",
				Examples:  []string{"canon domain init --dry-run", "canon domain init"},
				Mutating:  true,
				HasDryRun: true,
			},
			handler: kindAdapter(runInit, KindDomain),
		},
		{
			info: CommandInfo{
				Name:        "adr new",
				Purpose:     "Create a new ADR markdown file with parseable metadata. ADRs capture architecture decisions.",
				Safety:      "write: creates one markdown file",
				Selectors:   []string{"--title", "--status", "--tags"},
				Examples:    []string{`canon adr new --title "Use SQLite for local index" --status proposed --dry-run`, `canon adr new --title "Use SQLite for local index" --context "Need local querying" --decision "Use SQLite"`},
				NextCommand: []string{"canon adr list", "canon show --id ADR-0001"},
				Mutating:    true,
				HasDryRun:   true,
			},
			handler: kindAdapter(runNew, KindADR),
		},
		{
			info: CommandInfo{
				Name:        "spec new",
				Purpose:     "Create a new SPEC markdown file with parseable metadata. SPECs capture functional requirements.",
				Safety:      "write: creates one markdown file",
				Selectors:   []string{"--title", "--status", "--tags"},
				Examples:    []string{`canon spec new --title "Local query index" --requirements "Return ADRs by tag." --acceptance "list --tag storage works." --dry-run`},
				NextCommand: []string{"canon spec list", "canon show --id SPEC-0001"},
				Mutating:    true,
				HasDryRun:   true,
			},
			handler: kindAdapter(runNew, KindSPEC),
		},
		{
			info: CommandInfo{
				Name:        "domain new",
				Purpose:     "Create a new domain entry markdown file with parseable metadata. Domain entries define one canonical concept each: its definition, avoided terms with reasons, and relationships.",
				Safety:      "write: creates one markdown file",
				Selectors:   []string{"--title", "--status", "--tags", "--definition", "--avoid", "--relationships"},
				Examples:    []string{`canon domain new --title "ADR" --definition "A dated, narrowly-scoped architecture commitment." --avoid "design doc: too broad; ticket: tracks work, not decisions" --dry-run`},
				NextCommand: []string{"canon domain list", "canon show --id DM-0001"},
				Mutating:    true,
				HasDryRun:   true,
			},
			handler: kindAdapter(runNew, KindDomain),
		},
		{
			info: CommandInfo{
				Name:        "list",
				Purpose:     "List ADR, SPEC, and domain entry summaries together in deterministic order.",
				Safety:      "read-only",
				Selectors:   []string{"--status", "--tag"},
				Examples:    []string{"canon list", "canon list --status accepted", "canon --format context list --status accepted"},
				NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001", "canon show --id DM-0001", "canon search --query text"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runList, ""),
		},
		{
			info: CommandInfo{
				Name:        "adr list",
				Purpose:     "List ADR summaries in deterministic order.",
				Safety:      "read-only",
				Selectors:   []string{"--status", "--tag"},
				Examples:    []string{"canon adr list", "canon adr list --status accepted", "canon --format context adr list --status accepted"},
				NextCommand: []string{"canon show --id ADR-0001", "canon adr search --query text"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runList, KindADR),
		},
		{
			info: CommandInfo{
				Name:        "spec list",
				Purpose:     "List SPEC summaries in deterministic order.",
				Safety:      "read-only",
				Selectors:   []string{"--status", "--tag"},
				Examples:    []string{"canon spec list", "canon spec list --tag storage", "canon --format context spec list --status accepted"},
				NextCommand: []string{"canon show --id SPEC-0001", "canon spec search --query text"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runList, KindSPEC),
		},
		{
			info: CommandInfo{
				Name:        "domain list",
				Purpose:     "List domain entry summaries in deterministic order.",
				Safety:      "read-only",
				Selectors:   []string{"--status", "--tag"},
				Examples:    []string{"canon domain list", "canon domain list --status accepted", "canon --format context domain list --status accepted"},
				NextCommand: []string{"canon show --id DM-0001", "canon domain search --query text"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runList, KindDomain),
		},
		{
			info: CommandInfo{
				Name:        "show",
				Purpose:     "Return one ADR, SPEC, or domain entry with metadata and content.",
				Safety:      "read-only",
				Selectors:   []string{"--id"},
				Examples:    []string{"canon show --id ADR-0001", "canon show --id SPEC-0001", "canon show --id DM-0001"},
				NextCommand: []string{"canon append --id ADR-0001 --title Note --body ...", "canon append --id SPEC-0001 --title Note --body ..."},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: func(ctx commandContext) int { return runShow(ctx.stdout, ctx.stderr, ctx.opts, ctx.repo, ctx.args) },
		},
		{
			info: CommandInfo{
				Name:        "search",
				Purpose:     "Search title, tags, status, kind, and markdown body across ADRs, SPECs, and domain entries. Reads Markdown by default; --use-index searches the cached index when it is fresh and otherwise falls back to Markdown with a warning.",
				Safety:      "read-only",
				Selectors:   []string{"--query", "--status", "--tag", "--use-index"},
				Examples:    []string{`canon search --query "database"`, "canon search --status deprecated", `canon search --query "database" --use-index`},
				NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001", "canon index status"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runSearch, ""),
		},
		{
			info: CommandInfo{
				Name:        "adr search",
				Purpose:     "Search title, tags, status, and markdown body across ADRs only. --use-index searches the cached index with Markdown fallback.",
				Safety:      "read-only",
				Selectors:   []string{"--query", "--status", "--tag", "--use-index"},
				Examples:    []string{`canon adr search --query "database"`},
				NextCommand: []string{"canon show --id ADR-0001"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runSearch, KindADR),
		},
		{
			info: CommandInfo{
				Name:        "spec search",
				Purpose:     "Search title, tags, status, and markdown body across SPECs only. --use-index searches the cached index with Markdown fallback.",
				Safety:      "read-only",
				Selectors:   []string{"--query", "--status", "--tag", "--use-index"},
				Examples:    []string{"canon spec search --query requirements"},
				NextCommand: []string{"canon show --id SPEC-0001"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runSearch, KindSPEC),
		},
		{
			info: CommandInfo{
				Name:        "domain search",
				Purpose:     "Search title, tags, status, and markdown body across domain entries only. --use-index searches the cached index with Markdown fallback.",
				Safety:      "read-only",
				Selectors:   []string{"--query", "--status", "--tag", "--use-index"},
				Examples:    []string{`canon domain search --query "cancellation"`},
				NextCommand: []string{"canon show --id DM-0001"},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: kindAdapter(runSearch, KindDomain),
		},
		{
			info: CommandInfo{
				Name:        "index status",
				Purpose:     "Report the derived search index state for the configured corpus: absent, fresh, stale, corrupt, or unsupported, plus the cache path and schema version. The index lives under the user cache directory and never replaces Markdown authority.",
				Safety:      "read-only",
				Examples:    []string{"canon index status"},
				NextCommand: []string{"canon index rebuild --dry-run", `canon search --query "..." --use-index`},
				Mutating:    false,
				HasDryRun:   false,
			},
			handler: func(ctx commandContext) int { return runIndexStatus(ctx.stdout, ctx.opts, ctx.repo) },
		},
		{
			info: CommandInfo{
				Name:        "index rebuild",
				Purpose:     "Rebuild the search index from the authoritative Markdown corpus and replace the cached file atomically. A dry run creates no directory or file.",
				Safety:      "write: atomically replaces the cached search index",
				Selectors:   []string{"--dry-run"},
				Examples:    []string{"canon index rebuild --dry-run", "canon index rebuild"},
				NextCommand: []string{"canon index status", `canon search --query "..." --use-index`},
				Mutating:    true,
				HasDryRun:   true,
			},
			handler: func(ctx commandContext) int {
				return runIndexRebuild(ctx.stdout, ctx.stderr, ctx.opts, ctx.repo, ctx.args)
			},
		},
		{
			info: CommandInfo{
				Name:        "accept",
				Purpose:     "Mark an ADR, SPEC, or domain entry as accepted.",
				Safety:      "write: updates one document metadata block and appends history",
				Selectors:   []string{"--id", "--reason"},
				Examples:    []string{"canon accept --id ADR-0001 --reason \"Approved by the team.\" --dry-run", "canon accept --id SPEC-0001 --reason \"Requirements approved.\" --dry-run"},
				NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
				Mutating:    true,
				HasDryRun:   true,
			},
			handler: lifecycleHandler(lifecycleTransition{status: "accepted", historyTitle: "Accepted"}),
		},
		{
			info: CommandInfo{
				Name:        "reject",
				Purpose:     "Mark an ADR, SPEC, or domain entry as rejected.",
				Safety:      "write: updates one document metadata block and appends history",
				Selectors:   []string{"--id", "--reason"},
				Examples:    []string{"canon reject --id ADR-0001 --reason \"Chose a different approach.\" --dry-run", "canon reject --id SPEC-0001 --reason \"Requirements changed.\" --dry-run"},
				NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
				Mutating:    true,
				HasDryRun:   true,
			},
			handler: lifecycleHandler(lifecycleTransition{status: "rejected", historyTitle: "Rejected"}),
		},
		{
			info: CommandInfo{
				Name:        "supersede",
				Purpose:     "Mark an ADR, SPEC, or domain entry as superseded by another document of the same kind.",
				Safety:      "write: updates two document metadata blocks and appends history",
				Selectors:   []string{"--id", "--by", "--reason"},
				Examples:    []string{"canon supersede --id ADR-0001 --by ADR-0002 --reason \"Replaced by current architecture\" --dry-run", "canon supersede --id SPEC-0001 --by SPEC-0002 --reason \"Requirements split.\" --dry-run"},
				NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
				Mutating:    true,
				HasDryRun:   true,
			},
			handler: func(ctx commandContext) int {
				return runSupersede(ctx.stdout, ctx.stderr, ctx.opts, ctx.repo, ctx.args)
			},
		},
		{
			info: CommandInfo{
				Name:        "deprecate",
				Purpose:     "Mark an ADR, SPEC, or domain entry as deprecated without a direct replacement.",
				Safety:      "write: updates one document metadata block and appends history",
				Selectors:   []string{"--id", "--reason"},
				Examples:    []string{"canon deprecate --id ADR-0001 --reason \"No longer used\" --dry-run", "canon deprecate --id SPEC-0001 --reason \"Requirements moved.\" --dry-run"},
				NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
				Mutating:    true,
				HasDryRun:   true,
			},
			handler: func(ctx commandContext) int {
				return runDeprecate(ctx.stdout, ctx.stderr, ctx.opts, ctx.repo, ctx.args)
			},
		},
		{
			info: CommandInfo{
				Name:        "append",
				Purpose:     "Append a dated appendix section to an ADR, SPEC, or domain entry. Disabled when .canon.jsonc sets conventions.append to false.",
				Safety:      "write: appends markdown to one document",
				Selectors:   []string{"--id", "--title", "--body"},
				Examples:    []string{"canon append --id ADR-0001 --title \"2026 review\" --body \"Still valid.\"", "canon append --id SPEC-0001 --title Review --body \"Requirements still apply.\""},
				NextCommand: []string{"canon show --id ADR-0001", "canon show --id SPEC-0001"},
				Mutating:    true,
				HasDryRun:   true,
			},
			handler: func(ctx commandContext) int { return runAppend(ctx.stdout, ctx.stderr, ctx.opts, ctx.repo, ctx.args) },
		},
		{
			info: CommandInfo{
				Name:      "skill",
				Purpose:   "Return the bundled skill asset catalog with names, kinds, versions, hashes, and target paths.",
				Safety:    "read-only",
				Examples:  []string{"canon skill"},
				Mutating:  false,
				HasDryRun: false,
			},
			handler: func(ctx commandContext) int { return runSkillCatalog(ctx.stdout, ctx.opts) },
		},
		{
			info: CommandInfo{
				Name:        "skill install",
				Purpose:     "Install all bundled skill assets and selected subagent components into a repository.",
				Safety:      "write: creates skill payload files and target-specific agent files",
				Selectors:   []string{"--skill-dir", "--only", "--agent"},
				Examples:    []string{"canon skill install --dry-run", "canon skill install", "canon skill install --only canon --dry-run", "canon skill install --agent claude --dry-run"},
				NextCommand: []string{"canon skill update --dry-run", "canon commands"},
				Mutating:    true,
				HasDryRun:   true,
			},
			handler: func(ctx commandContext) int { return runSkillInstall(ctx.stdout, ctx.stderr, ctx.opts, ctx.args) },
		},
		{
			info: CommandInfo{
				Name:        "skill update",
				Purpose:     "Refresh every installed managed bundle file, refusing local modifications unless forced.",
				Safety:      "write: updates managed skill payload files and target-specific agent files",
				Selectors:   []string{"--skill-dir", "--only", "--agent", "--force"},
				Examples:    []string{"canon skill update --dry-run", "canon skill update", "canon skill update --only canon-record-gate --dry-run", "canon skill update --force --dry-run"},
				NextCommand: []string{"canon commands"},
				Mutating:    true,
				HasDryRun:   true,
			},
			handler: func(ctx commandContext) int { return runSkillUpdate(ctx.stdout, ctx.stderr, ctx.opts, ctx.args) },
		},
	}
}

// lookupCommand returns the table entry registered under name, or nil when no
// command matches. Names match exactly; dispatch composes two-token names for
// family children before consulting the table.
func lookupCommand(name string) *commandEntry {
	table := commandTable()
	for i := range table {
		if table[i].info.Name == name {
			return &table[i]
		}
	}
	return nil
}

// commandInfos projects the public metadata out of the table in declaration
// order. The slice is fresh and carries no handlers, so `canon commands` can
// JSON-encode it directly.
func commandInfos() []CommandInfo {
	table := commandTable()
	infos := make([]CommandInfo, 0, len(table))
	for _, entry := range table {
		infos = append(infos, entry.info)
	}
	return infos
}
