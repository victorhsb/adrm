package canon

import (
	"fmt"
	"io"
	"strings"
)

// runIndexCommand dispatches the index subcommands. The index is a derived
// cache (ADR-0017): status only reports it, and rebuild regenerates it
// atomically from the authoritative Markdown corpus.
func runIndexCommand(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	suggested := "Use `canon index status` or `canon index rebuild --dry-run`."
	if len(args) == 0 {
		writeEnvelope(stdout, errorEnvelope("index", "missing_index_subcommand", "usage", "\"index\" requires a subcommand", suggested), opts.Format)
		return exitUsage
	}
	switch args[0] {
	case "status":
		return runIndexStatus(stdout, opts, repo)
	case "rebuild":
		return runIndexRebuild(stdout, stderr, opts, repo, args[1:])
	default:
		writeEnvelope(stdout, errorEnvelope("index", "unknown_command", "usage", fmt.Sprintf("unknown index subcommand %q", args[0]), suggested), opts.Format)
		return exitUsage
	}
}

// runIndexStatus reports the derived index state for the configured corpus:
// absent, fresh, stale, corrupt, or unsupported, plus the inspectable cache
// path and schema version. It never creates, modifies, or deletes the index.
func runIndexStatus(stdout io.Writer, opts GlobalOptions, repo Repo) int {
	path, err := indexPathFor(repo)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("index status", "index_path_failed", "io", err.Error(), "Check the HOME or XDG_CACHE_HOME environment variables."), opts.Format)
		return exitIO
	}
	idx, state, reason, version := readSearchIndex(path)
	documents := 0
	if state == "" {
		state, reason = indexFreshness(idx.Manifest, repo)
		documents = len(idx.Records)
	}
	nextActions := []NextAction{}
	if state == indexStateFresh {
		nextActions = append(nextActions, NextAction{Command: `canon search --query "..." --use-index`, Description: "Search through the fresh index.", Safety: "read-only"})
	} else {
		nextActions = append(nextActions,
			NextAction{Command: "canon index rebuild --dry-run", Description: "Preview rebuilding the index.", Safety: "preview"},
			NextAction{Command: "canon index rebuild", Description: "Rebuild the index from the authoritative Markdown corpus.", Safety: "write"},
		)
	}
	writeEnvelope(stdout, Envelope{
		Command: "index status",
		Data: indexStatusPayload{
			State:         state,
			Path:          path,
			SchemaVersion: version,
			Documents:     documents,
			Reason:        reason,
		},
		NextActions: nextActions,
	}, opts.Format)
	return exitOK
}

// runIndexRebuild regenerates the search index from every present managed
// kind and replaces the cache file atomically. A dry run reads the corpus
// and plans the replacement without creating any directory or file.
func runIndexRebuild(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	fs := newCommandFlagSet(stderr, "index rebuild")
	dryRun := fs.Bool("dry-run", false, "preview changes")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("index rebuild", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	if fs.NArg() != 0 {
		writeEnvelope(stdout, usageError("index rebuild", fmt.Sprintf("unexpected arguments: %s", strings.Join(fs.Args(), " "))), opts.Format)
		return exitUsage
	}
	path, err := indexPathFor(repo)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("index rebuild", "index_path_failed", "io", err.Error(), "Check the HOME or XDG_CACHE_HOME environment variables."), opts.Format)
		return exitIO
	}
	idx, err := buildSearchIndex(repo)
	if err != nil {
		writeEnvelope(stdout, errorEnvelope("index rebuild", "index_build_failed", "io", err.Error(), "Run `canon validate` to find malformed files, or `canon doctor` for diagnostics."), opts.Format)
		return exitIO
	}
	plan := Plan{DryRun: *dryRun, Operations: []OpPlan{{Action: "write_file", Path: path, Description: fmt.Sprintf("Replace the search index with %d indexed documents.", len(idx.Records))}}}
	payload := indexRebuildPayload{
		Plan:          plan,
		Path:          path,
		SchemaVersion: idx.Manifest.SchemaVersion,
		Documents:     len(idx.Records),
	}
	if *dryRun {
		writeEnvelope(stdout, Envelope{
			Command:  "index rebuild",
			Status:   "planned",
			Data:     payload,
			Warnings: []string{"No changes were made."},
			NextActions: []NextAction{
				{Command: "canon index rebuild", Description: "Apply this index rebuild plan.", Safety: "write"},
			},
		}, opts.Format)
		return exitOK
	}
	if err := writeSearchIndex(path, idx); err != nil {
		writeEnvelope(stdout, errorEnvelope("index rebuild", "index_write_failed", "io", err.Error(), "Check the cache directory permissions and retry."), opts.Format)
		return exitIO
	}
	plan.ChangesMade = true
	payload.Plan = plan
	writeEnvelope(stdout, Envelope{
		Command: "index rebuild",
		Data:    payload,
		NextActions: []NextAction{
			{Command: "canon index status", Description: "Confirm the rebuilt index is fresh.", Safety: "read-only"},
			{Command: `canon search --query "..." --use-index`, Description: "Search through the rebuilt index.", Safety: "read-only"},
		},
	}, opts.Format)
	return exitOK
}
