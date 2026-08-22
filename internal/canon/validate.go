package canon

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// This file implements the shared validation engine (ADR-0009). The engine
// has two modes: shallow, used by doctor, covers directory existence and
// parseability; full, used by the validate command family, runs the complete
// corpus check catalog. Findings are Diagnostics whose Status field
// carries the severity: "error" or "warning".

// validationSummary aggregates finding counts for the validate envelope.
type validationSummary struct {
	FilesChecked int `json:"files_checked"`
	Errors       int `json:"errors"`
	Warnings     int `json:"warnings"`
}

// validationResult is the engine's full-mode output: findings only (no
// per-file ok entries) plus a summary.
type validationResult struct {
	Findings []Diagnostic
	Summary  validationSummary
}

// storesForScope returns the stores a validate run covers. An empty kind
// covers the whole corpus; a kind scopes the run to that store.
func storesForScope(repo Repo, kind string) []Store {
	if kind == "" {
		return []Store{repo.ADR, repo.Spec, repo.Domain}
	}
	return []Store{repo.StoreForKind(kind)}
}

// scannedDoc pairs a parsed document with the kind of the directory it lives
// in, so coherence checks can compare metadata against storage location.
type scannedDoc struct {
	Doc       ADR
	StoreKind string
}

// shallowDiagnostics is the engine's shallow mode: directory existence and
// parseability only. It reproduces doctor's historical check names and
// messages exactly, so doctor's output contract is preserved. A parse
// failure aborts the scan (shallow mode does not isolate malformed files)
// and returns the failing store's kind with the error.
func shallowDiagnostics(repo Repo) (checks []Diagnostic, failedKind string, err error) {
	checks = []Diagnostic{}
	for _, store := range []Store{repo.ADR, repo.Spec, repo.Domain} {
		label := store.Kind
		if !store.Exists() {
			checks = append(checks, Diagnostic{Name: label + "_directory", Status: "warning", Message: fmt.Sprintf("%s does not exist", store.Dir), SuggestedFix: fmt.Sprintf("Run `canon %s init --dry-run`, then `canon %s init`.", label, label)})
			continue
		}
		checks = append(checks, Diagnostic{Name: label + "_directory", Status: "ok", Message: fmt.Sprintf("%s exists", store.Dir)})
		adrs, listErr := store.List()
		if listErr != nil {
			return checks, label, listErr
		}
		checks = append(checks, Diagnostic{Name: label + "_parse", Status: "ok", Message: fmt.Sprintf("%d %s files parsed", len(adrs), label)})
	}
	return checks, "", nil
}

// scanStores walks every markdown file in the scoped stores and parses each
// one in isolation, so one malformed file does not mask the rest of the
// corpus. It returns the parsed documents, the scan-level findings (missing
// or unreadable directories, malformed files), and the number of files
// examined.
func scanStores(repo Repo, kind string) ([]scannedDoc, []Diagnostic, int) {
	findings := []Diagnostic{}
	var docs []scannedDoc
	filesChecked := 0

	for _, store := range storesForScope(repo, kind) {
		if !store.Exists() {
			findings = append(findings, Diagnostic{
				Name:         "missing_directory",
				Status:       "warning",
				Message:      fmt.Sprintf("%s does not exist", store.Dir),
				SuggestedFix: fmt.Sprintf("Run `canon %s init --dry-run`, then `canon %s init`.", store.Kind, store.Kind),
				Path:         store.Dir,
			})
			continue
		}
		entries, err := os.ReadDir(store.Dir)
		if err != nil {
			findings = append(findings, Diagnostic{
				Name:         "unreadable_directory",
				Status:       "error",
				Message:      fmt.Sprintf("failed to read %s: %s", store.Dir, err),
				SuggestedFix: "Check directory permissions.",
				Path:         store.Dir,
			})
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			path := filepath.Join(store.Dir, entry.Name())
			filesChecked++
			body, err := os.ReadFile(path)
			if err != nil {
				findings = append(findings, malformedFileFinding(path, err))
				continue
			}
			doc, err := parseADR(string(body))
			if err != nil {
				findings = append(findings, malformedFileFinding(path, err))
				continue
			}
			doc.Path = path
			docs = append(docs, scannedDoc{Doc: doc, StoreKind: store.Kind})
		}
	}
	return docs, findings, filesChecked
}

// validateCorpus runs the full corpus check catalog over the stores in
// scope.
func validateCorpus(repo Repo, kind string) validationResult {
	docs, findings, filesChecked := scanStores(repo, kind)

	byID := map[string]ADR{}
	for _, scanned := range docs {
		byID[scanned.Doc.ID] = scanned.Doc
	}

	findings = append(findings, duplicateIDFindings(docs)...)
	for _, scanned := range docs {
		findings = append(findings, validateDocument(scanned, byID)...)
	}

	return finalizeValidation(findings, validationSummary{FilesChecked: filesChecked})
}

// validateSingle runs the per-document checks for one document, resolving its
// references against the whole corpus. Only findings about the target
// document are reported. The boolean result is false when no parseable file
// claims the id.
func validateSingle(repo Repo, id string) (validationResult, bool) {
	_, normalized, err := normalizeID(id)
	if err != nil {
		return validationResult{}, false
	}
	docs, _, _ := scanStores(repo, "")

	byID := map[string]ADR{}
	var target scannedDoc
	found := false
	for _, scanned := range docs {
		byID[scanned.Doc.ID] = scanned.Doc
		if scanned.Doc.ID == normalized {
			target = scanned
			found = true
		}
	}
	if !found {
		return validationResult{}, false
	}

	findings := validateDocument(target, byID)
	return finalizeValidation(findings, validationSummary{FilesChecked: 1}), true
}

func malformedFileFinding(path string, err error) Diagnostic {
	return Diagnostic{
		Name:         "malformed_file",
		Status:       "error",
		Message:      fmt.Sprintf("%s: %s", path, err),
		SuggestedFix: "Fix the front matter so the file matches the documented format, or remove the file.",
		Path:         path,
	}
}

// duplicateIDFindings reports ids claimed by more than one parsed file.
func duplicateIDFindings(docs []scannedDoc) []Diagnostic {
	pathsByID := map[string][]string{}
	for _, scanned := range docs {
		pathsByID[scanned.Doc.ID] = append(pathsByID[scanned.Doc.ID], scanned.Doc.Path)
	}
	ids := make([]string, 0, len(pathsByID))
	for id := range pathsByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	findings := []Diagnostic{}
	for _, id := range ids {
		paths := pathsByID[id]
		if len(paths) < 2 {
			continue
		}
		sort.Strings(paths)
		findings = append(findings, Diagnostic{
			Name:         "duplicate_id",
			Status:       "error",
			Message:      fmt.Sprintf("id %s is claimed by multiple files: %s", id, strings.Join(paths, ", ")),
			SuggestedFix: "Edit all but one file so each id is unique, then run `canon validate` again.",
			Path:         paths[0],
			ID:           id,
		})
	}
	return findings
}

// validateDocument runs the per-document checks: status validity, date
// format, reference integrity and reciprocity, status/reference consistency,
// and kind/id/directory coherence.
func validateDocument(scanned scannedDoc, byID map[string]ADR) []Diagnostic {
	doc := scanned.Doc
	findings := []Diagnostic{}

	if !validStatus(normalizeStatus(doc.Status)) {
		findings = append(findings, Diagnostic{
			Name:         "invalid_status",
			Status:       "error",
			Message:      fmt.Sprintf("%s has invalid status %q", doc.ID, doc.Status),
			SuggestedFix: "Set status to proposed, accepted, rejected, superseded, or deprecated.",
			Path:         doc.Path,
			ID:           doc.ID,
		})
	}

	if _, err := time.Parse("2006-01-02", doc.Date); err != nil {
		findings = append(findings, Diagnostic{
			Name:         "malformed_date",
			Status:       "warning",
			Message:      fmt.Sprintf("%s has date %q, expected YYYY-MM-DD", doc.ID, doc.Date),
			SuggestedFix: "Set the date field to YYYY-MM-DD format.",
			Path:         doc.Path,
			ID:           doc.ID,
		})
	}

	// Reference integrity. Values that do not parse as ids (for example the
	// literal "manual" written by `canon deprecate`) are not references and
	// are skipped.
	refs := append([]string{}, doc.Supersedes...)
	if doc.SupersededBy != "" {
		refs = append(refs, doc.SupersededBy)
	}
	if doc.DeprecatedBy != "" {
		refs = append(refs, doc.DeprecatedBy)
	}
	for _, ref := range refs {
		_, normalized, err := normalizeID(ref)
		if err != nil {
			continue
		}
		if _, ok := byID[normalized]; !ok {
			findings = append(findings, Diagnostic{
				Name:         "broken_reference",
				Status:       "error",
				Message:      fmt.Sprintf("%s references %s, which does not exist", doc.ID, normalized),
				SuggestedFix: fmt.Sprintf("Remove the reference from %s or create %s.", doc.ID, normalized),
				Path:         doc.Path,
				ID:           doc.ID,
			})
		}
	}

	// Reciprocity per ADR-0004: A.superseded_by=B requires B.supersedes to
	// contain A.
	if doc.SupersededBy != "" {
		if _, normalized, err := normalizeID(doc.SupersededBy); err == nil {
			if other, ok := byID[normalized]; ok && !contains(other.Supersedes, doc.ID) {
				findings = append(findings, Diagnostic{
					Name:         "reciprocity_violation",
					Status:       "error",
					Message:      fmt.Sprintf("%s is superseded by %s, but %s does not list %s in supersedes", doc.ID, normalized, normalized, doc.ID),
					SuggestedFix: fmt.Sprintf("Add %s to the supersedes list of %s, or use `canon supersede --id %s --by %s --dry-run`.", doc.ID, normalized, doc.ID, normalized),
					Path:         doc.Path,
					ID:           doc.ID,
				})
			}
		}
	}

	// Status/reference consistency (warnings).
	if doc.Status == "superseded" && doc.SupersededBy == "" {
		findings = append(findings, statusReferenceFinding(doc, "status is superseded but superseded_by is empty", fmt.Sprintf("Run `canon supersede --id %s --by <replacement> --dry-run`, or change the status.", doc.ID)))
	}
	if doc.Status != "superseded" && doc.SupersededBy != "" {
		findings = append(findings, statusReferenceFinding(doc, fmt.Sprintf("superseded_by is %s but status is %s", doc.SupersededBy, doc.Status), fmt.Sprintf("Set status to superseded, or clear superseded_by in %s.", doc.ID)))
	}
	if doc.Status == "deprecated" && doc.DeprecatedBy == "" {
		findings = append(findings, statusReferenceFinding(doc, "status is deprecated but deprecated_by is empty", fmt.Sprintf("Run `canon deprecate --id %s --dry-run`, or change the status.", doc.ID)))
	}

	// Kind/id-prefix/directory coherence.
	if idKind := kindFromID(doc.ID); idKind != "" && doc.Kind != idKind {
		findings = append(findings, Diagnostic{
			Name:         "kind_mismatch",
			Status:       "error",
			Message:      fmt.Sprintf("%s has kind %q, which contradicts its id prefix", doc.ID, doc.Kind),
			SuggestedFix: "Make the kind field match the id prefix.",
			Path:         doc.Path,
			ID:           doc.ID,
		})
	}
	if scanned.StoreKind != "" && doc.Kind != scanned.StoreKind {
		findings = append(findings, Diagnostic{
			Name:         "directory_mismatch",
			Status:       "error",
			Message:      fmt.Sprintf("%s is a %s document stored in the %s directory", doc.ID, doc.Kind, scanned.StoreKind),
			SuggestedFix: fmt.Sprintf("Move the file to the %s directory, or fix its id and kind.", doc.Kind),
			Path:         doc.Path,
			ID:           doc.ID,
		})
	}

	return findings
}

func statusReferenceFinding(doc ADR, message, fix string) Diagnostic {
	return Diagnostic{
		Name:         "status_reference_inconsistency",
		Status:       "warning",
		Message:      fmt.Sprintf("%s: %s", doc.ID, message),
		SuggestedFix: fix,
		Path:         doc.Path,
		ID:           doc.ID,
	}
}

// finalizeValidation sorts findings deterministically by path then check
// name and fills in the summary counts.
func finalizeValidation(findings []Diagnostic, summary validationSummary) validationResult {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Name != findings[j].Name {
			return findings[i].Name < findings[j].Name
		}
		return findings[i].Message < findings[j].Message
	})
	for _, finding := range findings {
		switch finding.Status {
		case "error":
			summary.Errors++
		case "warning":
			summary.Warnings++
		}
	}
	return validationResult{Findings: findings, Summary: summary}
}
