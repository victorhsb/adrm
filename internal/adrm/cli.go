package adrm

import (
	"errors"
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

// Repo holds the stores for every supported document kind. The CLI focuses on
// ADRs and SPECs; both share the same parseable shape but live in separate
// directories with independent numbering.
type Repo struct {
	ADR  Store
	Spec Store
}

func NewRepo(opts GlobalOptions) Repo {
	return Repo{
		ADR:  NewStore(opts.ADRDir, KindADR),
		Spec: NewStore(opts.SpecDir, KindSPEC),
	}
}

// StoreForKind returns the store for the requested kind. Unknown or empty
// kinds resolve to the ADR store, which is the default document kind.
func (r Repo) StoreForKind(kind string) Store {
	if kind == KindSPEC {
		return r.Spec
	}
	return r.ADR
}

// StoreForID selects a store by inspecting the id prefix. Bare numbers and
// ADR- prefixed ids resolve to the ADR store; SPEC- prefixed ids resolve to
// the SPEC store.
func (r Repo) StoreForID(id string) (Store, error) {
	kind, _, err := normalizeID(id)
	if err != nil {
		return Store{}, err
	}
	return r.StoreForKind(kind), nil
}

// All returns every parseable document across both stores, in stable order.
// Missing directories are treated as empty rather than errors so that a fresh
// repository with only one kind initialized still lists cleanly.
func (r Repo) All() ([]ADR, error) {
	var docs []ADR
	for _, store := range []Store{r.ADR, r.Spec} {
		adrs, err := store.List()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		docs = append(docs, adrs...)
	}
	sort.Slice(docs, func(i, j int) bool {
		if docs[i].Kind != docs[j].Kind {
			return docs[i].Kind < docs[j].Kind
		}
		return docs[i].Number < docs[j].Number
	})
	return docs, nil
}

func Run(args []string, stdout, stderr io.Writer) int {
	global := flag.NewFlagSet("adrm", flag.ContinueOnError)
	global.SetOutput(stderr)
	opts := GlobalOptions{}
	humanReadable := false
	global.StringVar(&opts.ADRDir, "adr-dir", defaultADRDir, "ADR directory")
	global.StringVar(&opts.SpecDir, "spec-dir", defaultSpecDir, "SPEC directory")
	global.StringVar(&opts.Format, "format", "json", "output format: json or text")
	global.BoolVar(&humanReadable, "t", false, "shorthand for --format text")
	if help, err := parseFlags(global, args); err != nil {
		writeEnvelope(stdout, usageError("adrm", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	if humanReadable {
		opts.Format = "text"
	}
	if opts.Format != "json" && opts.Format != "text" {
		writeEnvelope(stdout, errorEnvelope("adrm", "invalid_format", "usage", "format must be json or text", "Use --format json or --format text, or -t for text."), "json")
		return exitUsage
	}
	remaining := global.Args()
	if len(remaining) == 0 {
		writeEnvelope(stdout, Envelope{
			Command: "adrm",
			Status:  "ok",
			Data: map[string]any{
				"purpose":  "Manage Architecture Decision Records and Specs for agent workflows.",
				"commands": commandNames(),
			},
			NextActions: []NextAction{
				{Command: "adrm commands", Description: "Inspect all available commands and safety rules.", Safety: "read-only"},
				{Command: "adrm doctor", Description: "Check ADR and SPEC repository readiness.", Safety: "read-only"},
			},
		}, opts.Format)
		return exitOK
	}

	command := remaining[0]
	commandArgs := remaining[1:]
	repo := NewRepo(opts)

	switch command {
	case "commands":
		return runCommands(stdout, opts)
	case "doctor":
		return runDoctor(stdout, opts, repo)
	case "init":
		return runInit(stdout, stderr, opts, repo, commandArgs)
	case "new":
		return runNew(stdout, stderr, opts, repo, commandArgs)
	case "list":
		return runList(stdout, stderr, opts, repo, commandArgs)
	case "show":
		return runShow(stdout, stderr, opts, repo, commandArgs)
	case "search":
		return runSearch(stdout, stderr, opts, repo, commandArgs)
	case "accept":
		return runLifecycle(stdout, stderr, opts, repo, commandArgs, "accept", "accepted", "Accepted")
	case "reject":
		return runLifecycle(stdout, stderr, opts, repo, commandArgs, "reject", "rejected", "Rejected")
	case "supersede":
		return runSupersede(stdout, stderr, opts, repo, commandArgs)
	case "deprecate":
		return runDeprecate(stdout, stderr, opts, repo, commandArgs)
	case "append":
		return runAppend(stdout, stderr, opts, repo, commandArgs)
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
				{"name": "--spec-dir", "default": defaultSpecDir, "purpose": "Select SPEC storage directory."},
				{"name": "--format", "default": "json", "purpose": "Choose json or text output."},
				{"name": "-t", "default": "false", "purpose": "Shorthand for --format text."},
			},
		},
		NextActions: []NextAction{{Command: "adrm doctor", Description: "Check if the ADR and SPEC directories are ready.", Safety: "read-only"}},
	}, opts.Format)
	return exitOK
}

func runDoctor(stdout io.Writer, opts GlobalOptions, repo Repo) int {
	checks := []Diagnostic{}
	for _, store := range []Store{repo.ADR, repo.Spec} {
		label := store.Kind
		if !store.Exists() {
			checks = append(checks, Diagnostic{Name: label + "_directory", Status: "warning", Message: fmt.Sprintf("%s does not exist", store.Dir), SuggestedFix: fmt.Sprintf("Run `adrm init --kind %s --dry-run`, then `adrm init --kind %s`.", label, label)})
			continue
		}
		checks = append(checks, Diagnostic{Name: label + "_directory", Status: "ok", Message: fmt.Sprintf("%s exists", store.Dir)})
		adrs, err := store.List()
		if err != nil {
			env := errorEnvelope("doctor", label+"_read_failed", "io", fmt.Sprintf("failed to read %s directory", label), "Check file permissions and front matter.")
			env.Error.Diagnostics = checks
			writeEnvelope(stdout, env, opts.Format)
			return exitIO
		}
		checks = append(checks, Diagnostic{Name: label + "_parse", Status: "ok", Message: fmt.Sprintf("%d %s files parsed", len(adrs), label)})
	}
	anyMissing := !repo.ADR.Exists() || !repo.Spec.Exists()
	if anyMissing {
		writeEnvelope(stdout, Envelope{
			Command: "doctor",
			Status:  "warning",
			Data:    map[string]any{"diagnostics": checks},
			NextActions: []NextAction{
				{Command: "adrm init --kind adr --dry-run", Description: "Preview creating the ADR directory.", Safety: "preview"},
				{Command: "adrm init --kind spec --dry-run", Description: "Preview creating the SPEC directory.", Safety: "preview"},
			},
		}, opts.Format)
		return exitOK
	}
	writeEnvelope(stdout, Envelope{
		Command: "doctor",
		Data:    map[string]any{"diagnostics": checks},
		NextActions: []NextAction{
			{Command: "adrm list", Description: "Inspect ADR and SPEC inventory.", Safety: "read-only"},
			{Command: `adrm new --kind adr --title "..." --dry-run`, Description: "Preview creating a new ADR.", Safety: "preview"},
			{Command: `adrm new --kind spec --title "..." --dry-run`, Description: "Preview creating a new SPEC.", Safety: "preview"},
		},
	}, opts.Format)
	return exitOK
}

func runInit(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	fs := newCommandFlagSet(stderr, "init")
	kind := fs.String("kind", KindADR, "document kind: adr or spec")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("init", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	kindValue := normalizeKind(*kind)
	if kindValue == "" {
		writeEnvelope(stdout, errorEnvelope("init", "invalid_kind", "usage", fmt.Sprintf("invalid kind %q", *kind), "Use --kind adr or --kind spec."), opts.Format)
		return exitUsage
	}
	store := repo.StoreForKind(kindValue)
	plan := Plan{DryRun: *dryRun, ChangesMade: false, Operations: []OpPlan{{Action: "mkdir", Path: store.Dir, Description: fmt.Sprintf("Create %s directory if missing.", kindValue)}}}
	if *dryRun {
		writeEnvelope(stdout, Envelope{
			Command: "init",
			Status:  "planned",
			Data:    plan,
			Warnings: []string{
				"No changes were made.",
			},
			NextActions: []NextAction{{Command: fmt.Sprintf("adrm init --kind %s", kindValue), Description: "Apply this directory creation plan.", Safety: "write"}},
		}, opts.Format)
		return exitOK
	}
	if err := store.Init(); err != nil {
		writeEnvelope(stdout, errorEnvelope("init", "init_failed", "io", err.Error(), "Check directory permissions or choose another directory flag."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	writeEnvelope(stdout, Envelope{
		Command: "init",
		Data:    plan,
		NextActions: []NextAction{
			{Command: fmt.Sprintf(`adrm new --kind %s --title "First %s" --dry-run`, kindValue, kindValue), Description: "Preview creating the first document.", Safety: "preview"},
		},
	}, opts.Format)
	return exitOK
}

func runNew(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	fs := newCommandFlagSet(stderr, "new")
	kind := fs.String("kind", KindADR, "document kind: adr or spec")
	title := fs.String("title", "", "document title")
	status := fs.String("status", "proposed", "document status")
	tags := fs.String("tags", "", "comma-separated tags")
	context := fs.String("context", "", "context section")
	decision := fs.String("decision", "", "decision section (adr)")
	consequences := fs.String("consequences", "", "consequences section (adr)")
	requirements := fs.String("requirements", "", "requirements section (spec)")
	constraints := fs.String("constraints", "", "constraints section (spec)")
	acceptance := fs.String("acceptance", "", "acceptance criteria section (spec)")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("new", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	kindValue := normalizeKind(*kind)
	if kindValue == "" {
		writeEnvelope(stdout, errorEnvelope("new", "invalid_kind", "usage", fmt.Sprintf("invalid kind %q", *kind), "Use --kind adr or --kind spec."), opts.Format)
		return exitUsage
	}
	if strings.TrimSpace(*title) == "" {
		writeEnvelope(stdout, errorEnvelope("new", "missing_title", "usage", "--title is required", `Run adrm new --kind `+kindValue+` --title "Short title" --dry-run.`), opts.Format)
		return exitUsage
	}
	statusValue := normalizeStatus(*status)
	if !validStatus(statusValue) {
		writeEnvelope(stdout, errorEnvelope("new", "invalid_status", "usage", fmt.Sprintf("invalid status %q", *status), "Use proposed, accepted, rejected, superseded, or deprecated."), opts.Format)
		return exitUsage
	}
	store := repo.StoreForKind(kindValue)
	next, err := store.NextNumber()
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("new", "next_number_failed", "io", err.Error(), "Run `adrm doctor` for diagnostics."), opts.Format)
		return exitIO
	}
	path := filepath.Join(store.Dir, fmt.Sprintf("%04d-%s.md", next, slugify(*title)))
	plan := Plan{DryRun: *dryRun, Operations: []OpPlan{{Action: "write_file", Path: path, Description: fmt.Sprintf("Create new %s markdown file.", kindValue)}}}
	sections := newSections(kindValue, *context, *decision, *consequences, *requirements, *constraints, *acceptance)
	if *dryRun {
		writeEnvelope(stdout, Envelope{
			Command: "new",
			Status:  "planned",
			Data: map[string]any{
				"plan": plan,
				"adr":  ADR{Kind: kindValue, ID: formatID(kindValue, next), Number: next, Title: strings.TrimSpace(*title), Status: statusValue, Date: time.Now().Format("2006-01-02"), Tags: parseList(*tags), Path: path},
			},
			Warnings: []string{"No changes were made."},
			NextActions: []NextAction{
				{Command: strings.Join(append([]string{"adrm new", "--kind", kindValue, "--title", quoteForNextAction(*title), "--status", statusValue}, newDryRunFreeArgs(kindValue, *tags, *context, *decision, *consequences, *requirements, *constraints, *acceptance)...), " "), Description: "Apply this document creation plan.", Safety: "write"},
			},
		}, opts.Format)
		return exitOK
	}
	adr, err := store.WriteNew(strings.TrimSpace(*title), statusValue, parseList(*tags), sections)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("new", "create_failed", "io", err.Error(), "Run `adrm doctor` for diagnostics."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	writeEnvelope(stdout, Envelope{
		Command: "new",
		Data:    map[string]any{"plan": plan, "adr": adrSummary(adr)},
		NextActions: []NextAction{
			{Command: fmt.Sprintf("adrm show --id %s", adr.ID), Description: "Inspect the created document.", Safety: "read-only"},
			{Command: fmt.Sprintf("adrm list --kind %s", kindValue), Description: "Refresh document inventory.", Safety: "read-only"},
		},
	}, opts.Format)
	return exitOK
}

func newSections(kind, context, decision, consequences, requirements, constraints, acceptance string) map[string]string {
	sections := map[string]string{"context": context}
	switch kind {
	case KindSPEC:
		sections["requirements"] = requirements
		sections["constraints"] = constraints
		sections["acceptance"] = acceptance
	default:
		sections["decision"] = decision
		sections["consequences"] = consequences
	}
	return sections
}

func newDryRunFreeArgs(kind, tags, context, decision, consequences, requirements, constraints, acceptance string) []string {
	var args []string
	if strings.TrimSpace(tags) != "" {
		args = append(args, "--tags", quoteForNextAction(tags))
	}
	if strings.TrimSpace(context) != "" {
		args = append(args, "--context", quoteForNextAction(context))
	}
	switch kind {
	case KindSPEC:
		if strings.TrimSpace(requirements) != "" {
			args = append(args, "--requirements", quoteForNextAction(requirements))
		}
		if strings.TrimSpace(constraints) != "" {
			args = append(args, "--constraints", quoteForNextAction(constraints))
		}
		if strings.TrimSpace(acceptance) != "" {
			args = append(args, "--acceptance", quoteForNextAction(acceptance))
		}
	default:
		if strings.TrimSpace(decision) != "" {
			args = append(args, "--decision", quoteForNextAction(decision))
		}
		if strings.TrimSpace(consequences) != "" {
			args = append(args, "--consequences", quoteForNextAction(consequences))
		}
	}
	return args
}

func runList(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	fs := newCommandFlagSet(stderr, "list")
	kind := fs.String("kind", "", "filter by kind: adr, spec, or empty for all")
	status := fs.String("status", "", "filter by status")
	tag := fs.String("tag", "", "filter by tag")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("list", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	kindValue := normalizeKind(*kind)
	if *kind != "" && kindValue == "" {
		writeEnvelope(stdout, errorEnvelope("list", "invalid_kind", "usage", fmt.Sprintf("invalid kind %q", *kind), "Use --kind adr or --kind spec, or omit it to list all."), opts.Format)
		return exitUsage
	}
	docs, err := docsForKind(repo, kindValue)
	if err != nil {
		return handleReadError(stdout, opts, "list", err)
	}
	docs = filterADRs(docs, *status, *tag, "")
	writeEnvelope(stdout, Envelope{
		Command: "list",
		Data: map[string]any{
			"count": len(docs),
			"adrs":  summaries(docs),
		},
		NextActions: []NextAction{
			{Command: "adrm show --id ADR-0001", Description: "Inspect a selected id from the result set.", Safety: "read-only"},
			{Command: "adrm search --query text", Description: "Search ADR and SPEC content when the list is too broad.", Safety: "read-only"},
		},
	}, opts.Format)
	return exitOK
}

func runShow(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	fs := newCommandFlagSet(stderr, "show")
	id := fs.String("id", "", "document id")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("show", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	if strings.TrimSpace(*id) == "" {
		writeEnvelope(stdout, errorEnvelope("show", "missing_id", "usage", "--id is required", "Use an id from `adrm list`."), opts.Format)
		return exitUsage
	}
	store, err := repo.StoreForID(*id)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("show", "invalid_id", "usage", err.Error(), "Use an id like ADR-0001 or SPEC-0001."), opts.Format)
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

func runSearch(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	fs := newCommandFlagSet(stderr, "search")
	query := fs.String("query", "", "search query")
	kind := fs.String("kind", "", "filter by kind: adr, spec, or empty for all")
	status := fs.String("status", "", "filter by status")
	tag := fs.String("tag", "", "filter by tag")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("search", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	if fs.NArg() > 0 && strings.TrimSpace(*query) == "" {
		*query = strings.Join(fs.Args(), " ")
	}
	kindValue := normalizeKind(*kind)
	if *kind != "" && kindValue == "" {
		writeEnvelope(stdout, errorEnvelope("search", "invalid_kind", "usage", fmt.Sprintf("invalid kind %q", *kind), "Use --kind adr or --kind spec, or omit it to search all."), opts.Format)
		return exitUsage
	}
	docs, err := docsForKind(repo, kindValue)
	if err != nil {
		return handleReadError(stdout, opts, "search", err)
	}
	results := filterADRs(docs, *status, *tag, *query)
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

func docsForKind(repo Repo, kind string) ([]ADR, error) {
	if kind == "" {
		return repo.All()
	}
	store := repo.StoreForKind(kind)
	if !store.Exists() {
		return []ADR{}, nil
	}
	return store.List()
}

func runLifecycle(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string, command, status, historyTitle string) int {
	fs := newCommandFlagSet(stderr, command)
	id := fs.String("id", "", fmt.Sprintf("document id to %s", command))
	reason := fs.String("reason", "", "reason")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError(command, err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	if strings.TrimSpace(*id) == "" {
		writeEnvelope(stdout, errorEnvelope(command, "missing_id", "usage", "--id is required", "Use an id from `adrm list`, then retry with --dry-run."), opts.Format)
		return exitUsage
	}
	store, err := repo.StoreForID(*id)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope(command, "invalid_id", "usage", err.Error(), "Use an id like ADR-0001 or SPEC-0001."), opts.Format)
		return exitUsage
	}
	adr, err := store.Read(*id)
	if err != nil {
		return handleReadError(stdout, opts, command, err)
	}
	plan := Plan{DryRun: *dryRun, Operations: []OpPlan{{Action: "update_file", Path: adr.Path, Description: fmt.Sprintf("Set status=%s and append history.", status)}}}
	applyCommand := fmt.Sprintf("adrm %s --id %s", command, adr.ID)
	if strings.TrimSpace(*reason) != "" {
		applyCommand += " --reason " + quoteForNextAction(*reason)
	}
	if *dryRun {
		writeEnvelope(stdout, dryRunEnvelope(command, plan, adr.ID, applyCommand), opts.Format)
		return exitOK
	}
	adr.Status = status
	adr.Content = setStatusSection(adr.Content, adr.Status)
	adr.Content = appendHistory(adr.Content, historyTitle, defaultText(*reason, "No reason provided."))
	if err := store.Save(adr); err != nil {
		writeEnvelope(stdout, errorEnvelope(command, "update_failed", "io", err.Error(), "Check file permissions and retry."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	writeEnvelope(stdout, mutationEnvelope(command, plan, adr), opts.Format)
	return exitOK
}

func runSupersede(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	fs := newCommandFlagSet(stderr, "supersede")
	id := fs.String("id", "", "document id to supersede")
	by := fs.String("by", "", "superseding document id")
	reason := fs.String("reason", "", "reason")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("supersede", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	if strings.TrimSpace(*id) == "" || strings.TrimSpace(*by) == "" {
		writeEnvelope(stdout, errorEnvelope("supersede", "missing_selector", "usage", "--id and --by are required", "Use ids from `adrm list`, then retry with --dry-run."), opts.Format)
		return exitUsage
	}
	store, err := repo.StoreForID(*id)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("supersede", "invalid_id", "usage", err.Error(), "Use an id like ADR-0001 or SPEC-0001."), opts.Format)
		return exitUsage
	}
	adr, err := store.Read(*id)
	if err != nil {
		return handleReadError(stdout, opts, "supersede", err)
	}
	byKind, byID, err := normalizeID(*by)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("supersede", "invalid_by_id", "usage", err.Error(), "Use an id like ADR-0002 or SPEC-0002."), opts.Format)
		return exitUsage
	}
	if adr.ID == byID {
		writeEnvelope(stdout, errorEnvelope("supersede", "self_supersede", "state", "a document cannot supersede itself", "Choose a different --by id."), opts.Format)
		return exitState
	}
	byStore := repo.StoreForKind(byKind)
	byADR, err := byStore.Read(byID)
	if err != nil {
		if os.IsNotExist(err) {
			writeEnvelope(stdout, errorEnvelope("supersede", "superseding_adr_not_found", "state", fmt.Sprintf("superseding document %s was not found", byID), "Create or select the replacement document first, then retry with --dry-run."), opts.Format)
			return exitNotFound
		}
		return handleReadError(stdout, opts, "supersede", err)
	}
	if adr.Kind != byADR.Kind {
		writeEnvelope(stdout, errorEnvelope("supersede", "cross_kind_supersede", "state", fmt.Sprintf("%s (%s) cannot be superseded by %s (%s)", adr.ID, adr.Kind, byADR.ID, byADR.Kind), "Supersede within the same kind: replace an ADR with an ADR or a SPEC with a SPEC."), opts.Format)
		return exitState
	}
	plan := Plan{DryRun: *dryRun, Operations: []OpPlan{
		{Action: "update_file", Path: adr.Path, Description: fmt.Sprintf("Set status=superseded and superseded_by=%s.", byID)},
		{Action: "update_file", Path: byADR.Path, Description: fmt.Sprintf("Add %s to supersedes list.", adr.ID)},
	}}
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

	originalBySupersedes := byADR.Supersedes
	byADR.Supersedes = cleanList(append(byADR.Supersedes, adr.ID))

	if err := byStore.Save(byADR); err != nil {
		writeEnvelope(stdout, errorEnvelope("supersede", "update_failed", "io", err.Error(), "Check file permissions and retry."), opts.Format)
		return exitIO
	}
	if err := store.Save(adr); err != nil {
		// Best-effort rollback: try to restore the replacement document's supersedes list.
		byADR.Supersedes = originalBySupersedes
		_ = byStore.Save(byADR)
		writeEnvelope(stdout, errorEnvelope("supersede", "update_failed", "io", err.Error(), "Check file permissions and retry."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	writeEnvelope(stdout, mutationEnvelope("supersede", plan, adr), opts.Format)
	return exitOK
}

func runDeprecate(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	fs := newCommandFlagSet(stderr, "deprecate")
	id := fs.String("id", "", "document id to deprecate")
	reason := fs.String("reason", "", "reason")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("deprecate", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	if strings.TrimSpace(*id) == "" {
		writeEnvelope(stdout, errorEnvelope("deprecate", "missing_id", "usage", "--id is required", "Use an id from `adrm list`, then retry with --dry-run."), opts.Format)
		return exitUsage
	}
	store, err := repo.StoreForID(*id)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("deprecate", "invalid_id", "usage", err.Error(), "Use an id like ADR-0001 or SPEC-0001."), opts.Format)
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

func runAppend(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	fs := newCommandFlagSet(stderr, "append")
	id := fs.String("id", "", "document id")
	title := fs.String("title", "", "appendix title")
	body := fs.String("body", "", "appendix body")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("append", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	if strings.TrimSpace(*id) == "" || strings.TrimSpace(*title) == "" || strings.TrimSpace(*body) == "" {
		writeEnvelope(stdout, errorEnvelope("append", "missing_appendix_input", "usage", "--id, --title, and --body are required", "Use `adrm append --id ADR-0001 --title Note --body Text --dry-run`."), opts.Format)
		return exitUsage
	}
	store, err := repo.StoreForID(*id)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("append", "invalid_id", "usage", err.Error(), "Use an id like ADR-0001 or SPEC-0001."), opts.Format)
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
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("skill install", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
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
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("skill update", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
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

// parseFlags parses args with fs. When -h/--help is requested it returns
// help=true and the flag package has already printed usage to the flagset
// output, so the caller should return without emitting an envelope.
func parseFlags(fs *flag.FlagSet, args []string) (help bool, err error) {
	err = fs.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		return true, nil
	}
	return false, err
}

func usageError(command, message string) Envelope {
	return errorEnvelope(command, "invalid_usage", "usage", message, "Run `adrm commands` to inspect required flags and examples.")
}

func handleReadError(stdout io.Writer, opts GlobalOptions, command string, err error) int {
	if os.IsNotExist(err) {
		writeEnvelope(stdout, errorEnvelope(command, "adr_not_found_or_uninitialized", "state", err.Error(), "Run `adrm doctor`; if the directory is missing, run `adrm init`."), opts.Format)
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
		adr.Kind,
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
			{Command: fmt.Sprintf("adrm show --id %s", id), Description: "Inspect current document before mutating it.", Safety: "read-only"},
		},
	}
}

func mutationEnvelope(command string, plan Plan, adr ADR) Envelope {
	return Envelope{
		Command: command,
		Data:    map[string]any{"plan": plan, "adr": adrSummary(adr)},
		NextActions: []NextAction{
			{Command: fmt.Sprintf("adrm show --id %s", adr.ID), Description: "Inspect the updated document.", Safety: "read-only"},
		},
	}
}

func quoteForNextAction(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return fmt.Sprintf("%q", value)
}
