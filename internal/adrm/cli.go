package adrm

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/victorhsb/adrm/adrmskill"
)

func Run(args []string, stdout, stderr io.Writer) int {
	global := flag.NewFlagSet("adrm", flag.ContinueOnError)
	global.SetOutput(stderr)
	opts := GlobalOptions{}
	global.StringVar(&opts.ADRDir, "adr-dir", defaultADRDir, "ADR directory")
	global.StringVar(&opts.Format, "format", "json", "output format: json or text")
	if err := global.Parse(args); err != nil {
		writeEnvelope(stdout, usageError("adrm", err.Error()), "json")
		return exitUsage
	}
	if opts.Format != "json" && opts.Format != "text" {
		writeEnvelope(stdout, errorEnvelope("adrm", "invalid_format", "usage", "format must be json or text", "Use --format json or --format text."), "json")
		return exitUsage
	}
	remaining := global.Args()
	if len(remaining) == 0 {
		writeEnvelope(stdout, Envelope{
			Command: "adrm",
			Status:  "ok",
			Data: map[string]any{
				"purpose":  "Manage Architecture Decision Records for agent workflows.",
				"commands": commandNames(),
			},
			NextActions: []NextAction{
				{Command: "adrm commands", Description: "Inspect all available commands and safety rules.", Safety: "read-only"},
				{Command: "adrm doctor", Description: "Check ADR repository readiness.", Safety: "read-only"},
			},
		}, opts.Format)
		return exitOK
	}

	command := remaining[0]
	commandArgs := remaining[1:]
	store := NewStore(opts.ADRDir)

	switch command {
	case "commands":
		return runCommands(stdout, opts)
	case "doctor":
		return runDoctor(stdout, opts, store)
	case "init":
		return runInit(stdout, stderr, opts, store, commandArgs)
	case "new":
		return runNew(stdout, stderr, opts, store, commandArgs)
	case "list":
		return runList(stdout, stderr, opts, store, commandArgs)
	case "show":
		return runShow(stdout, stderr, opts, store, commandArgs)
	case "search":
		return runSearch(stdout, stderr, opts, store, commandArgs)
	case "supersede":
		return runSupersede(stdout, stderr, opts, store, commandArgs)
	case "deprecate":
		return runDeprecate(stdout, stderr, opts, store, commandArgs)
	case "append":
		return runAppend(stdout, stderr, opts, store, commandArgs)
	case "skill":
		return runSkill(stdout, stderr, opts, commandArgs)
	default:
		writeEnvelope(stdout, errorEnvelope(command, "unknown_command", "usage", fmt.Sprintf("unknown command %q", command), "Run `adrm commands` to inspect valid commands."), opts.Format)
		return exitUsage
	}
}

func runCommands(stdout io.Writer, opts GlobalOptions) int {
	writeEnvelope(stdout, Envelope{
		Command: "commands",
		Data: map[string]any{
			"commands": commandRegistry(),
			"global_flags": []map[string]string{
				{"name": "--adr-dir", "default": defaultADRDir, "purpose": "Select ADR storage directory."},
				{"name": "--format", "default": "json", "purpose": "Choose json or text output."},
			},
		},
		NextActions: []NextAction{{Command: "adrm doctor", Description: "Check if the ADR directory is ready.", Safety: "read-only"}},
	}, opts.Format)
	return exitOK
}

func runDoctor(stdout io.Writer, opts GlobalOptions, store Store) int {
	checks := []Diagnostic{}
	if !store.Exists() {
		checks = append(checks, Diagnostic{Name: "adr_directory", Status: "warning", Message: fmt.Sprintf("%s does not exist", store.Dir), SuggestedFix: "Run `adrm init --dry-run`, then `adrm init`."})
		writeEnvelope(stdout, Envelope{
			Command: "doctor",
			Status:  "warning",
			Data:    map[string]any{"diagnostics": checks},
			NextActions: []NextAction{
				{Command: "adrm init --dry-run", Description: "Preview creating the ADR directory.", Safety: "preview"},
				{Command: "adrm init", Description: "Create the ADR directory.", Safety: "write"},
			},
		}, opts.Format)
		return exitOK
	}
	checks = append(checks, Diagnostic{Name: "adr_directory", Status: "ok", Message: fmt.Sprintf("%s exists", store.Dir)})
	adrs, err := store.List()
	if err != nil {
		env := errorEnvelope("doctor", "adr_read_failed", "io", "failed to read ADR directory", "Check file permissions and ADR front matter.")
		env.Error.Diagnostics = checks
		writeEnvelope(stdout, env, opts.Format)
		return exitIO
	}
	checks = append(checks, Diagnostic{Name: "adr_parse", Status: "ok", Message: fmt.Sprintf("%d ADR files parsed", len(adrs))})
	writeEnvelope(stdout, Envelope{
		Command: "doctor",
		Data:    map[string]any{"diagnostics": checks},
		NextActions: []NextAction{
			{Command: "adrm list", Description: "Inspect ADR inventory.", Safety: "read-only"},
			{Command: `adrm new --title "..." --dry-run`, Description: "Preview creating a new ADR.", Safety: "preview"},
		},
	}, opts.Format)
	return exitOK
}

func runInit(stdout, stderr io.Writer, opts GlobalOptions, store Store, args []string) int {
	fs := newCommandFlagSet(stderr, "init")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if err := fs.Parse(args); err != nil {
		writeEnvelope(stdout, usageError("init", err.Error()), opts.Format)
		return exitUsage
	}
	plan := Plan{DryRun: *dryRun, ChangesMade: false, Operations: []OpPlan{{Action: "mkdir", Path: store.Dir, Description: "Create ADR directory if missing."}}}
	if *dryRun {
		writeEnvelope(stdout, Envelope{
			Command: "init",
			Status:  "planned",
			Data:    plan,
			Warnings: []string{
				"No changes were made.",
			},
			NextActions: []NextAction{{Command: "adrm init", Description: "Apply this directory creation plan.", Safety: "write"}},
		}, opts.Format)
		return exitOK
	}
	if err := store.Init(); err != nil {
		writeEnvelope(stdout, errorEnvelope("init", "init_failed", "io", err.Error(), "Check directory permissions or choose another --adr-dir."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	writeEnvelope(stdout, Envelope{
		Command: "init",
		Data:    plan,
		NextActions: []NextAction{
			{Command: `adrm new --title "First decision" --dry-run`, Description: "Preview creating the first ADR.", Safety: "preview"},
		},
	}, opts.Format)
	return exitOK
}

func runNew(stdout, stderr io.Writer, opts GlobalOptions, store Store, args []string) int {
	fs := newCommandFlagSet(stderr, "new")
	title := fs.String("title", "", "ADR title")
	status := fs.String("status", "proposed", "ADR status")
	tags := fs.String("tags", "", "comma-separated tags")
	context := fs.String("context", "", "context section")
	decision := fs.String("decision", "", "decision section")
	consequences := fs.String("consequences", "", "consequences section")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if err := fs.Parse(args); err != nil {
		writeEnvelope(stdout, usageError("new", err.Error()), opts.Format)
		return exitUsage
	}
	if strings.TrimSpace(*title) == "" {
		writeEnvelope(stdout, errorEnvelope("new", "missing_title", "usage", "--title is required", `Run adrm new --title "Short decision title" --dry-run.`), opts.Format)
		return exitUsage
	}
	statusValue := normalizeStatus(*status)
	if !validStatus(statusValue) {
		writeEnvelope(stdout, errorEnvelope("new", "invalid_status", "usage", fmt.Sprintf("invalid status %q", *status), "Use proposed, accepted, rejected, superseded, or deprecated."), opts.Format)
		return exitUsage
	}
	next, err := store.NextNumber()
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("new", "next_number_failed", "io", err.Error(), "Run `adrm doctor` for diagnostics."), opts.Format)
		return exitIO
	}
	path := filepath.Join(store.Dir, fmt.Sprintf("%04d-%s.md", next, slugify(*title)))
	plan := Plan{DryRun: *dryRun, Operations: []OpPlan{{Action: "write_file", Path: path, Description: "Create new ADR markdown file."}}}
	if *dryRun {
		writeEnvelope(stdout, Envelope{
			Command: "new",
			Status:  "planned",
			Data: map[string]any{
				"plan": plan,
				"adr":  ADR{ID: formatID(next), Number: next, Title: strings.TrimSpace(*title), Status: statusValue, Date: time.Now().Format("2006-01-02"), Tags: parseList(*tags), Path: path},
			},
			Warnings: []string{"No changes were made."},
			NextActions: []NextAction{
				{Command: strings.ReplaceAll(strings.Join(append([]string{"adrm new", "--title", quoteForNextAction(*title), "--status", statusValue}, dryRunFreeArgs(*tags, *context, *decision, *consequences)...), " "), " --dry-run", ""), Description: "Apply this ADR creation plan.", Safety: "write"},
			},
		}, opts.Format)
		return exitOK
	}
	adr, err := store.WriteNew(strings.TrimSpace(*title), statusValue, parseList(*tags), *context, *decision, *consequences)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("new", "create_failed", "io", err.Error(), "Run `adrm doctor` for diagnostics."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	writeEnvelope(stdout, Envelope{
		Command: "new",
		Data:    map[string]any{"plan": plan, "adr": adrSummary(adr)},
		NextActions: []NextAction{
			{Command: fmt.Sprintf("adrm show --id %s", adr.ID), Description: "Inspect the created ADR.", Safety: "read-only"},
			{Command: "adrm list", Description: "Refresh ADR inventory.", Safety: "read-only"},
		},
	}, opts.Format)
	return exitOK
}

func runList(stdout, stderr io.Writer, opts GlobalOptions, store Store, args []string) int {
	fs := newCommandFlagSet(stderr, "list")
	status := fs.String("status", "", "filter by status")
	tag := fs.String("tag", "", "filter by tag")
	if err := fs.Parse(args); err != nil {
		writeEnvelope(stdout, usageError("list", err.Error()), opts.Format)
		return exitUsage
	}
	adrs, err := store.List()
	if err != nil {
		return handleReadError(stdout, opts, "list", err)
	}
	adrs = filterADRs(adrs, *status, *tag, "")
	writeEnvelope(stdout, Envelope{
		Command: "list",
		Data: map[string]any{
			"count": len(adrs),
			"adrs":  summaries(adrs),
		},
		NextActions: []NextAction{
			{Command: "adrm show --id ADR-0001", Description: "Inspect a selected ADR id from the result set.", Safety: "read-only"},
			{Command: "adrm search --query text", Description: "Search ADR content when the list is too broad.", Safety: "read-only"},
		},
	}, opts.Format)
	return exitOK
}

func runShow(stdout, stderr io.Writer, opts GlobalOptions, store Store, args []string) int {
	fs := newCommandFlagSet(stderr, "show")
	id := fs.String("id", "", "ADR id")
	if err := fs.Parse(args); err != nil {
		writeEnvelope(stdout, usageError("show", err.Error()), opts.Format)
		return exitUsage
	}
	if strings.TrimSpace(*id) == "" {
		writeEnvelope(stdout, errorEnvelope("show", "missing_id", "usage", "--id is required", "Use an id from `adrm list`."), opts.Format)
		return exitUsage
	}
	adr, err := store.Read(*id)
	if err != nil {
		return handleReadError(stdout, opts, "show", err)
	}
	writeEnvelope(stdout, Envelope{
		Command: "show",
		Data:    map[string]any{"adr": adr},
		NextActions: []NextAction{
			{Command: fmt.Sprintf("adrm append --id %s --title Note --body \"...\" --dry-run", adr.ID), Description: "Preview adding an appendix.", Safety: "preview"},
		},
	}, opts.Format)
	return exitOK
}

func runSearch(stdout, stderr io.Writer, opts GlobalOptions, store Store, args []string) int {
	fs := newCommandFlagSet(stderr, "search")
	query := fs.String("query", "", "search query")
	status := fs.String("status", "", "filter by status")
	tag := fs.String("tag", "", "filter by tag")
	if err := fs.Parse(args); err != nil {
		writeEnvelope(stdout, usageError("search", err.Error()), opts.Format)
		return exitUsage
	}
	if fs.NArg() > 0 && strings.TrimSpace(*query) == "" {
		*query = strings.Join(fs.Args(), " ")
	}
	adrs, err := store.List()
	if err != nil {
		return handleReadError(stdout, opts, "search", err)
	}
	results := filterADRs(adrs, *status, *tag, *query)
	writeEnvelope(stdout, Envelope{
		Command: "search",
		Data: map[string]any{
			"query":   *query,
			"count":   len(results),
			"results": searchResults(results, *query),
		},
		NextActions: []NextAction{{Command: "adrm show --id ADR-0001", Description: "Inspect a selected result id.", Safety: "read-only"}},
	}, opts.Format)
	return exitOK
}

func runSupersede(stdout, stderr io.Writer, opts GlobalOptions, store Store, args []string) int {
	fs := newCommandFlagSet(stderr, "supersede")
	id := fs.String("id", "", "ADR id to supersede")
	by := fs.String("by", "", "superseding ADR id")
	reason := fs.String("reason", "", "reason")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if err := fs.Parse(args); err != nil {
		writeEnvelope(stdout, usageError("supersede", err.Error()), opts.Format)
		return exitUsage
	}
	if strings.TrimSpace(*id) == "" || strings.TrimSpace(*by) == "" {
		writeEnvelope(stdout, errorEnvelope("supersede", "missing_selector", "usage", "--id and --by are required", "Use ids from `adrm list`, then retry with --dry-run."), opts.Format)
		return exitUsage
	}
	adr, err := store.Read(*id)
	if err != nil {
		return handleReadError(stdout, opts, "supersede", err)
	}
	byID, err := normalizeID(*by)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("supersede", "invalid_by_id", "usage", err.Error(), "Use an id like ADR-0002."), opts.Format)
		return exitUsage
	}
	if adr.ID == byID {
		writeEnvelope(stdout, errorEnvelope("supersede", "self_supersede", "state", "an ADR cannot supersede itself", "Choose a different --by ADR id."), opts.Format)
		return exitState
	}
	if _, err := store.Read(byID); err != nil {
		writeEnvelope(stdout, errorEnvelope("supersede", "superseding_adr_not_found", "state", fmt.Sprintf("superseding ADR %s was not found", byID), "Create or select the replacement ADR first, then retry with --dry-run."), opts.Format)
		return exitNotFound
	}
	plan := Plan{DryRun: *dryRun, Operations: []OpPlan{{Action: "update_file", Path: adr.Path, Description: fmt.Sprintf("Set status=superseded and superseded_by=%s.", byID)}}}
	applyCommand := fmt.Sprintf("adrm supersede --id %s --by %s", adr.ID, byID)
	if strings.TrimSpace(*reason) != "" {
		applyCommand += " --reason " + quoteForNextAction(*reason)
	}
	if *dryRun {
		writeEnvelope(stdout, dryRunEnvelope("supersede", plan, adr.ID, applyCommand), opts.Format)
		return exitOK
	}
	adr.Status = "superseded"
	adr.SupersededBy = byID
	adr.Content = setStatusSection(adr.Content, adr.Status)
	adr.Content = appendHistory(adr.Content, "Superseded", fmt.Sprintf("Superseded by %s. %s", byID, *reason))
	if err := store.Save(adr); err != nil {
		writeEnvelope(stdout, errorEnvelope("supersede", "update_failed", "io", err.Error(), "Check file permissions and retry."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	writeEnvelope(stdout, mutationEnvelope("supersede", plan, adr), opts.Format)
	return exitOK
}

func runDeprecate(stdout, stderr io.Writer, opts GlobalOptions, store Store, args []string) int {
	fs := newCommandFlagSet(stderr, "deprecate")
	id := fs.String("id", "", "ADR id to deprecate")
	reason := fs.String("reason", "", "reason")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if err := fs.Parse(args); err != nil {
		writeEnvelope(stdout, usageError("deprecate", err.Error()), opts.Format)
		return exitUsage
	}
	if strings.TrimSpace(*id) == "" {
		writeEnvelope(stdout, errorEnvelope("deprecate", "missing_id", "usage", "--id is required", "Use an id from `adrm list`, then retry with --dry-run."), opts.Format)
		return exitUsage
	}
	adr, err := store.Read(*id)
	if err != nil {
		return handleReadError(stdout, opts, "deprecate", err)
	}
	plan := Plan{DryRun: *dryRun, Operations: []OpPlan{{Action: "update_file", Path: adr.Path, Description: "Set status=deprecated and append history."}}}
	applyCommand := fmt.Sprintf("adrm deprecate --id %s", adr.ID)
	if strings.TrimSpace(*reason) != "" {
		applyCommand += " --reason " + quoteForNextAction(*reason)
	}
	if *dryRun {
		writeEnvelope(stdout, dryRunEnvelope("deprecate", plan, adr.ID, applyCommand), opts.Format)
		return exitOK
	}
	adr.Status = "deprecated"
	adr.DeprecatedBy = "manual"
	adr.Content = setStatusSection(adr.Content, adr.Status)
	adr.Content = appendHistory(adr.Content, "Deprecated", defaultText(*reason, "No reason provided."))
	if err := store.Save(adr); err != nil {
		writeEnvelope(stdout, errorEnvelope("deprecate", "update_failed", "io", err.Error(), "Check file permissions and retry."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	writeEnvelope(stdout, mutationEnvelope("deprecate", plan, adr), opts.Format)
	return exitOK
}

func runAppend(stdout, stderr io.Writer, opts GlobalOptions, store Store, args []string) int {
	fs := newCommandFlagSet(stderr, "append")
	id := fs.String("id", "", "ADR id")
	title := fs.String("title", "", "appendix title")
	body := fs.String("body", "", "appendix body")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if err := fs.Parse(args); err != nil {
		writeEnvelope(stdout, usageError("append", err.Error()), opts.Format)
		return exitUsage
	}
	if strings.TrimSpace(*id) == "" || strings.TrimSpace(*title) == "" || strings.TrimSpace(*body) == "" {
		writeEnvelope(stdout, errorEnvelope("append", "missing_appendix_input", "usage", "--id, --title, and --body are required", "Use `adrm append --id ADR-0001 --title Note --body Text --dry-run`."), opts.Format)
		return exitUsage
	}
	adr, err := store.Read(*id)
	if err != nil {
		return handleReadError(stdout, opts, "append", err)
	}
	plan := Plan{DryRun: *dryRun, Operations: []OpPlan{{Action: "append_markdown", Path: adr.Path, Description: fmt.Sprintf("Append appendix section %q.", *title)}}}
	applyCommand := fmt.Sprintf("adrm append --id %s --title %s --body %s", adr.ID, quoteForNextAction(*title), quoteForNextAction(*body))
	if *dryRun {
		writeEnvelope(stdout, dryRunEnvelope("append", plan, adr.ID, applyCommand), opts.Format)
		return exitOK
	}
	adr.Content = appendAppendix(adr.Content, *title, *body)
	if err := store.Save(adr); err != nil {
		writeEnvelope(stdout, errorEnvelope("append", "append_failed", "io", err.Error(), "Check file permissions and retry."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	writeEnvelope(stdout, mutationEnvelope("append", plan, adr), opts.Format)
	return exitOK
}

func runSkill(stdout, stderr io.Writer, opts GlobalOptions, args []string) int {
	if len(args) == 0 {
		writeEnvelope(stdout, Envelope{
			Command: "skill",
			Data: map[string]any{
				"filename": adrmskill.FileName,
				"content":  adrmskill.Content(),
				"skill": map[string]any{
					"name":                adrmskill.Name,
					"version":             adrmskill.Version,
					"hash":                adrmskill.Hash(),
					"default_install_dir": adrmskill.DefaultInstallDir,
				},
			},
			NextActions: []NextAction{
				{Command: "adrm skill install --dry-run", Description: "Preview installing the repository-local agent skill.", Safety: "preview"},
				{Command: "adrm commands", Description: "Inspect machine-readable CLI capabilities.", Safety: "read-only"},
			},
		}, opts.Format)
		return exitOK
	}
	switch args[0] {
	case "install":
		return runSkillInstall(stdout, stderr, opts, args[1:])
	case "update":
		return runSkillUpdate(stdout, stderr, opts, args[1:])
	default:
		writeEnvelope(stdout, errorEnvelope("skill", "unknown_skill_subcommand", "usage", fmt.Sprintf("unknown skill subcommand %q", args[0]), "Use `adrm skill`, `adrm skill install`, or `adrm skill update`."), opts.Format)
		return exitUsage
	}
}

func runSkillInstall(stdout, stderr io.Writer, opts GlobalOptions, args []string) int {
	fs := newCommandFlagSet(stderr, "skill install")
	skillDir := fs.String("skill-dir", adrmskill.DefaultInstallDir, "skill installation directory")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if err := fs.Parse(args); err != nil {
		writeEnvelope(stdout, usageError("skill install", err.Error()), opts.Format)
		return exitUsage
	}
	target := adrmskill.TargetPath(*skillDir)
	if _, err := os.Stat(target); err == nil {
		writeEnvelope(stdout, errorEnvelope("skill install", "skill_already_installed", "state", fmt.Sprintf("%s already exists", target), "Use `adrm skill update --dry-run` to preview updating the installed skill."), opts.Format)
		return exitState
	} else if !os.IsNotExist(err) {
		writeEnvelope(stdout, errorEnvelope("skill install", "skill_stat_failed", "io", err.Error(), "Check file permissions or choose another --skill-dir."), opts.Format)
		return exitIO
	}
	plan := skillWritePlan(*dryRun, target, "Install ADRM agent skill.")
	if *dryRun {
		writeEnvelope(stdout, skillDryRunEnvelope("skill install", plan, target, skillInstallApplyCommand(*skillDir)), opts.Format)
		return exitOK
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		writeEnvelope(stdout, errorEnvelope("skill install", "skill_directory_create_failed", "io", err.Error(), "Check directory permissions or choose another --skill-dir."), opts.Format)
		return exitIO
	}
	if err := os.WriteFile(target, []byte(adrmskill.Content()), 0o644); err != nil {
		writeEnvelope(stdout, errorEnvelope("skill install", "skill_write_failed", "io", err.Error(), "Check file permissions or choose another --skill-dir."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	writeEnvelope(stdout, Envelope{
		Command: "skill install",
		Data: map[string]any{
			"plan":  plan,
			"skill": skillMetadata(target),
		},
		NextActions: []NextAction{
			{Command: fmt.Sprintf("adrm skill update --skill-dir %s --dry-run", quoteForNextAction(*skillDir)), Description: "Preview updating the installed skill later.", Safety: "preview"},
			{Command: "adrm commands", Description: "Inspect machine-readable CLI capabilities.", Safety: "read-only"},
		},
	}, opts.Format)
	return exitOK
}

func runSkillUpdate(stdout, stderr io.Writer, opts GlobalOptions, args []string) int {
	fs := newCommandFlagSet(stderr, "skill update")
	skillDir := fs.String("skill-dir", adrmskill.DefaultInstallDir, "skill installation directory")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	force := fs.Bool("force", false, "overwrite locally modified skill file")
	if err := fs.Parse(args); err != nil {
		writeEnvelope(stdout, usageError("skill update", err.Error()), opts.Format)
		return exitUsage
	}
	target := adrmskill.TargetPath(*skillDir)
	content, err := os.ReadFile(target)
	if os.IsNotExist(err) {
		writeEnvelope(stdout, errorEnvelope("skill update", "skill_not_installed", "state", fmt.Sprintf("%s does not exist", target), "Use `adrm skill install --dry-run` to preview installing it."), opts.Format)
		return exitNotFound
	}
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("skill update", "skill_read_failed", "io", err.Error(), "Check file permissions or choose another --skill-dir."), opts.Format)
		return exitIO
	}
	inspection := adrmskill.Inspect(string(content))
	if inspection.Current {
		writeEnvelope(stdout, Envelope{
			Command: "skill update",
			Data: map[string]any{
				"plan":  skillNoopPlan(target),
				"skill": skillMetadata(target),
			},
			NextActions: []NextAction{{Command: "adrm commands", Description: "Inspect machine-readable CLI capabilities.", Safety: "read-only"}},
		}, opts.Format)
		return exitOK
	}
	if !inspection.Managed && !*force {
		writeEnvelope(stdout, errorEnvelope("skill update", "local_skill_modified", "state", fmt.Sprintf("%s is not an unmodified ADRM-managed skill file", target), "Review the file, then retry with `adrm skill update --force --dry-run` if overwriting is acceptable."), opts.Format)
		return exitState
	}
	plan := skillWritePlan(*dryRun, target, "Update ADRM agent skill.")
	if *dryRun {
		writeEnvelope(stdout, skillDryRunEnvelope("skill update", plan, target, forceAwareApplyCommand(*skillDir, *force)), opts.Format)
		return exitOK
	}
	if err := os.WriteFile(target, []byte(adrmskill.Content()), 0o644); err != nil {
		writeEnvelope(stdout, errorEnvelope("skill update", "skill_write_failed", "io", err.Error(), "Check file permissions or choose another --skill-dir."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	writeEnvelope(stdout, Envelope{
		Command: "skill update",
		Data: map[string]any{
			"plan":  plan,
			"skill": skillMetadata(target),
		},
		NextActions: []NextAction{{Command: "adrm commands", Description: "Inspect machine-readable CLI capabilities.", Safety: "read-only"}},
	}, opts.Format)
	return exitOK
}

func skillWritePlan(dryRun bool, target, description string) Plan {
	action := "write_file"
	operations := []OpPlan{
		{Action: "mkdir", Path: filepath.Dir(target), Description: "Create skill directory if missing."},
		{Action: action, Path: target, Description: description},
	}
	if strings.HasPrefix(description, "Update ") {
		action = "update_file"
		operations = []OpPlan{{Action: action, Path: target, Description: description}}
	}
	return Plan{DryRun: dryRun, ChangesMade: false, Operations: operations}
}

func skillNoopPlan(target string) Plan {
	return Plan{
		DryRun:      false,
		ChangesMade: false,
		Operations:  []OpPlan{{Action: "noop", Path: target, Description: "Installed ADRM agent skill is current."}},
	}
}

func skillDryRunEnvelope(command string, plan Plan, target, applyCommand string) Envelope {
	return Envelope{
		Command:  command,
		Status:   "planned",
		Data:     map[string]any{"plan": plan, "skill": skillMetadata(target)},
		Warnings: []string{"No changes were made."},
		NextActions: []NextAction{
			{Command: applyCommand, Description: "Apply this previewed skill mutation.", Safety: "write"},
		},
	}
}

func skillMetadata(target string) map[string]any {
	return map[string]any{
		"name":     adrmskill.Name,
		"version":  adrmskill.Version,
		"hash":     adrmskill.Hash(),
		"filename": adrmskill.FileName,
		"path":     target,
	}
}

func skillInstallApplyCommand(skillDir string) string {
	command := "adrm skill install"
	if strings.TrimSpace(skillDir) != "" && skillDir != adrmskill.DefaultInstallDir {
		command += " --skill-dir " + quoteForNextAction(skillDir)
	}
	return command
}

func forceAwareApplyCommand(skillDir string, force bool) string {
	command := "adrm skill update"
	if strings.TrimSpace(skillDir) != "" && skillDir != adrmskill.DefaultInstallDir {
		command += " --skill-dir " + quoteForNextAction(skillDir)
	}
	if force {
		command += " --force"
	}
	return command
}

func newCommandFlagSet(stderr io.Writer, name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func usageError(command, message string) Envelope {
	return errorEnvelope(command, "invalid_usage", "usage", message, "Run `adrm commands` to inspect required flags and examples.")
}

func handleReadError(stdout io.Writer, opts GlobalOptions, command string, err error) int {
	if os.IsNotExist(err) {
		writeEnvelope(stdout, errorEnvelope(command, "adr_not_found_or_uninitialized", "state", err.Error(), "Run `adrm doctor`; if the ADR directory is missing, run `adrm init`."), opts.Format)
		return exitNotFound
	}
	writeEnvelope(stdout, errorEnvelope(command, "adr_read_failed", "io", err.Error(), "Run `adrm doctor` for diagnostics."), opts.Format)
	return exitIO
}

func commandNames() []string {
	registry := commandRegistry()
	names := make([]string, 0, len(registry))
	for _, command := range registry {
		names = append(names, command.Name)
	}
	sort.Strings(names)
	return names
}

func summaries(adrs []ADR) []ADR {
	out := make([]ADR, 0, len(adrs))
	for _, adr := range adrs {
		out = append(out, adrSummary(adr))
	}
	return out
}

func adrSummary(adr ADR) ADR {
	adr.Content = ""
	return adr
}

func filterADRs(adrs []ADR, status, tag, query string) []ADR {
	status = normalizeStatus(status)
	tag = strings.TrimSpace(tag)
	query = strings.ToLower(strings.TrimSpace(query))
	var out []ADR
	for _, adr := range adrs {
		if status != "" && adr.Status != status {
			continue
		}
		if tag != "" && !contains(adr.Tags, tag) {
			continue
		}
		if query != "" && !adrMatches(adr, query) {
			continue
		}
		out = append(out, adr)
	}
	return out
}

func adrMatches(adr ADR, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		adr.ID,
		adr.Title,
		adr.Status,
		strings.Join(adr.Tags, " "),
		adr.Content,
	}, " "))
	return strings.Contains(haystack, query)
}

func searchResults(adrs []ADR, query string) []map[string]any {
	results := []map[string]any{}
	for _, adr := range adrs {
		results = append(results, map[string]any{
			"adr":     adrSummary(adr),
			"snippet": snippet(adr, query),
		})
	}
	return results
}

func snippet(adr ADR, query string) string {
	text := strings.Join(strings.Fields(adr.Content), " ")
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		if len(text) > 160 {
			return text[:160]
		}
		return text
	}
	idx := strings.Index(strings.ToLower(text), query)
	if idx < 0 {
		return ""
	}
	start := idx - 60
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + 60
	if end > len(text) {
		end = len(text)
	}
	return strings.TrimSpace(text[start:end])
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func normalizeStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func validStatus(status string) bool {
	switch status {
	case "proposed", "accepted", "rejected", "superseded", "deprecated":
		return true
	default:
		return false
	}
}

func appendHistory(content, title, body string) string {
	return strings.TrimRight(content, "\n") + fmt.Sprintf("\n\n## History: %s\n\nDate: %s\n\n%s\n", title, time.Now().Format("2006-01-02"), strings.TrimSpace(body))
}

func appendAppendix(content, title, body string) string {
	return strings.TrimRight(content, "\n") + fmt.Sprintf("\n\n## Appendix: %s\n\nDate: %s\n\n%s\n", strings.TrimSpace(title), time.Now().Format("2006-01-02"), strings.TrimSpace(body))
}

func setStatusSection(content, status string) string {
	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "## Status" {
			continue
		}
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j < len(lines) {
			lines[j] = status
			return strings.Join(lines, "\n")
		}
		return strings.Join(append(lines, "", status), "\n")
	}
	return strings.TrimRight(content, "\n") + fmt.Sprintf("\n\n## Status\n\n%s\n", status)
}

func dryRunEnvelope(command string, plan Plan, id, applyCommand string) Envelope {
	return Envelope{
		Command:  command,
		Status:   "planned",
		Data:     map[string]any{"plan": plan, "target_id": id},
		Warnings: []string{"No changes were made."},
		NextActions: []NextAction{
			{Command: applyCommand, Description: "Apply this previewed mutation.", Safety: "write"},
			{Command: fmt.Sprintf("adrm show --id %s", id), Description: "Inspect current ADR before mutating it.", Safety: "read-only"},
		},
	}
}

func mutationEnvelope(command string, plan Plan, adr ADR) Envelope {
	return Envelope{
		Command: command,
		Data:    map[string]any{"plan": plan, "adr": adrSummary(adr)},
		NextActions: []NextAction{
			{Command: fmt.Sprintf("adrm show --id %s", adr.ID), Description: "Inspect the updated ADR.", Safety: "read-only"},
		},
	}
}

func quoteForNextAction(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return fmt.Sprintf("%q", value)
}

func dryRunFreeArgs(tags, context, decision, consequences string) []string {
	var args []string
	if strings.TrimSpace(tags) != "" {
		args = append(args, "--tags", quoteForNextAction(tags))
	}
	if strings.TrimSpace(context) != "" {
		args = append(args, "--context", quoteForNextAction(context))
	}
	if strings.TrimSpace(decision) != "" {
		args = append(args, "--decision", quoteForNextAction(decision))
	}
	if strings.TrimSpace(consequences) != "" {
		args = append(args, "--consequences", quoteForNextAction(consequences))
	}
	return args
}
