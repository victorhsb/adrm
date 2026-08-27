package canon

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// This file implements the read-only configuration inspection commands,
// `canon config show` and `canon config validate`, kept out of cli.go so the
// command surface stays focused (ADR-0005, ADR-0016).

// configReport is the JSON payload shared by config show and config validate.
// It exposes the effective policy, its provenance, and the file's key paths
// so agents can inspect configuration without reading the file themselves.
// Fields are declared in alphabetical JSON name order at every level because
// the payload was historically map-encoded; the exact key order is pinned by
// the config show and config validate goldens.
type configReport struct {
	DiscoveryPaths map[string]string `json:"discovery_paths"`
	Effective      effectivePayload  `json:"effective"`
	Path           string            `json:"path,omitempty"`
	RecognizedKeys []string          `json:"recognized_keys"`
	Source         string            `json:"source"`
	UnknownKeys    []string          `json:"unknown_keys"`
}

type effectivePayload struct {
	Conventions   conventionsPayload `json:"conventions"`
	SchemaVersion string             `json:"schema_version"`
}

type conventionsPayload struct {
	Append        bool                        `json:"append"`
	Lifecycle     lifecyclePayload            `json:"lifecycle"`
	RequiredKinds []string                    `json:"required_kinds"`
	Tags          map[string]tagPolicyPayload `json:"tags"`
	Validation    validationPayload           `json:"validation"`
}

type validationPayload struct {
	Strict bool `json:"strict"`
}

type lifecyclePayload struct {
	NewDocumentsMustBeProposed bool `json:"new_documents_must_be_proposed"`
	RequireReason              bool `json:"require_reason"`
}

type tagPolicyPayload struct {
	Allowed []string `json:"allowed"`
}

// newConfigReport projects a resolution into its stable output shape.
func newConfigReport(resolution ConfigResolution) configReport {
	effective := resolution.Effective
	tags := map[string]tagPolicyPayload{}
	for _, kind := range supportedKinds {
		if allowed, restricted := effective.AllowedTags(kind); restricted {
			tags[kind] = tagPolicyPayload{Allowed: allowed}
		}
	}
	recognized := effective.RecognizedKeys
	if recognized == nil {
		recognized = []string{}
	}
	unknown := effective.UnknownKeys
	if unknown == nil {
		unknown = []string{}
	}
	return configReport{
		DiscoveryPaths: resolution.Discovery,
		Effective: effectivePayload{
			Conventions: conventionsPayload{
				Append: effective.AppendEnabled(),
				Lifecycle: lifecyclePayload{
					NewDocumentsMustBeProposed: effective.ProposedCreationRequired(),
					RequireReason:              effective.ReasonRequired(),
				},
				RequiredKinds: effective.RequiredKinds(),
				Tags:          tags,
				Validation:    validationPayload{Strict: effective.StrictValidation()},
			},
			SchemaVersion: effective.SchemaVersion,
		},
		Path:           effective.Path,
		RecognizedKeys: recognized,
		Source:         effective.Source,
		UnknownKeys:    unknown,
	}
}

// renderText renders the report for --format text: provenance first, then
// the effective conventions, then the recognized and unknown key lists.
func (r configReport) renderText(out io.Writer) {
	fmt.Fprintf(out, "source: %s\n", r.Source)
	if r.Path != "" {
		fmt.Fprintf(out, "path: %s\n", r.Path)
	}
	fmt.Fprintln(out, "discovery paths:")
	for _, kind := range supportedKinds {
		fmt.Fprintf(out, "  %s: %s\n", kind, r.DiscoveryPaths[kind])
	}
	conv := r.Effective.Conventions
	fmt.Fprintln(out, "effective configuration:")
	fmt.Fprintf(out, "  schema_version: %s\n", r.Effective.SchemaVersion)
	fmt.Fprintf(out, "  append: %t\n", conv.Append)
	fmt.Fprintf(out, "  required_kinds: %s\n", strings.Join(conv.RequiredKinds, ", "))
	fmt.Fprintf(out, "  validation.strict: %t\n", conv.Validation.Strict)
	fmt.Fprintf(out, "  lifecycle.require_reason: %t\n", conv.Lifecycle.RequireReason)
	fmt.Fprintf(out, "  lifecycle.new_documents_must_be_proposed: %t\n", conv.Lifecycle.NewDocumentsMustBeProposed)
	if len(conv.Tags) == 0 {
		fmt.Fprintln(out, "  tags: unrestricted for every kind")
	} else {
		fmt.Fprintln(out, "  tags:")
		for _, kind := range supportedKinds {
			policy, ok := conv.Tags[kind]
			if !ok {
				fmt.Fprintf(out, "    %s: unrestricted\n", kind)
				continue
			}
			fmt.Fprintf(out, "    %s allowed: %s\n", kind, allowedTagsDisplay(policy.Allowed))
		}
	}
	fmt.Fprintf(out, "recognized keys: %s\n", joinOrDash(r.RecognizedKeys))
	fmt.Fprintf(out, "unknown keys: %s\n", joinOrDash(r.UnknownKeys))
}

// joinOrDash renders a string list for single-line text output.
func joinOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

// runConfigShow backs the `config show` command-table entry. The
// configuration inspection family is read-only and never mutates
// configuration or documents.
func runConfigShow(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	fs := newCommandFlagSet(stderr, "config show")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("config show", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	if fs.NArg() != 0 {
		writeEnvelope(stdout, usageError("config show", fmt.Sprintf("unexpected arguments: %s", strings.Join(fs.Args(), " "))), opts.Format)
		return exitUsage
	}
	resolution, cfgErr := resolveRepoPolicy(repo)
	if cfgErr != nil {
		writeEnvelope(stdout, configErrorForCommand("config show", cfgErr), opts.Format)
		return exitState
	}
	writeEnvelope(stdout, Envelope{
		Command: "config show",
		Data:    newConfigReport(resolution),
		NextActions: []NextAction{
			{Command: "canon config validate", Description: "Validate the configuration file against the schema.", Safety: "read-only"},
			{Command: "canon doctor", Description: "Check corpus readiness under the effective configuration.", Safety: "read-only"},
		},
	}, opts.Format)
	return exitOK
}

// repoConfigDiscovery is the raw outcome of discovering configuration from
// every store directory, before reconciliation rules apply.
type repoConfigDiscovery struct {
	// Discovery maps each kind to its configured store directory.
	Discovery map[string]string
	// Paths holds the unique discovered config files (cleaned absolute, sorted).
	Paths []string
	// FoundBy counts how many stores discovered each path.
	FoundBy map[string]int
	// StoreCount is the number of stores discovery started from.
	StoreCount int
}

// discoverRepoConfig finds configuration file candidates from every store.
func discoverRepoConfig(repo Repo) (repoConfigDiscovery, error) {
	discovery := repoConfigDiscovery{
		Discovery:  map[string]string{},
		FoundBy:    map[string]int{},
		StoreCount: len(supportedKinds),
	}
	for _, store := range repoStoreDirs(repo) {
		discovery.Discovery[store.Kind] = store.Dir
		absPath, found, err := findConfigFile(store.Dir)
		if err != nil {
			return discovery, err
		}
		if found {
			discovery.FoundBy[absPath]++
		}
	}
	paths := make([]string, 0, len(discovery.FoundBy))
	for path := range discovery.FoundBy {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	discovery.Paths = paths
	return discovery, nil
}

// scopeMismatchMessage describes a discovery outcome no single policy covers.
func scopeMismatchMessage(discovery repoConfigDiscovery) string {
	displayed := make([]string, 0, len(discovery.Paths))
	for _, path := range discovery.Paths {
		displayed = append(displayed, displayPath(path))
	}
	if len(discovery.Paths) <= 1 {
		return fmt.Sprintf("only some stores discover a configuration file (%s)", strings.Join(displayed, ", "))
	}
	return fmt.Sprintf("stores resolve to different configuration files: %s", strings.Join(displayed, ", "))
}

const scopeMismatchFix = "Point --adr-dir, --spec-dir, and --domain-dir at one corpus, or place .canon.jsonc at the shared root so all stores discover the same file."

// runConfigValidate validates the effective repository configuration against
// the schema. Error findings exit 4; unknown keys are warnings and never
// fail the command, so older binaries can expose future keys without turning
// them into hard failures.
func runConfigValidate(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	fs := newCommandFlagSet(stderr, "config validate")
	if help, err := parseFlags(fs, args); err != nil {
		writeEnvelope(stdout, usageError("config validate", err.Error()), opts.Format)
		return exitUsage
	} else if help {
		return exitOK
	}
	if fs.NArg() != 0 {
		writeEnvelope(stdout, usageError("config validate", fmt.Sprintf("unexpected arguments: %s", strings.Join(fs.Args(), " "))), opts.Format)
		return exitUsage
	}

	discovery, err := discoverRepoConfig(repo)
	if err != nil {
		writeEnvelope(stdout, configErrorForCommand("config validate", &ConfigError{
			Code:         "invalid_config",
			Message:      err.Error(),
			SuggestedFix: "Check file permissions along the configuration search path.",
		}), opts.Format)
		return exitState
	}

	findings := []Diagnostic{}
	filesChecked := 0
	var resolution *ConfigResolution

	switch {
	case len(discovery.Paths) == 0:
		resolution = &ConfigResolution{Effective: defaultEffectiveConfig(), Discovery: discovery.Discovery}
	case len(discovery.Paths) == 1 && discovery.FoundBy[discovery.Paths[0]] == discovery.StoreCount:
		absPath := discovery.Paths[0]
		effective, fileFindings, cfgErr := loadEffectiveFromFile(absPath)
		findings = append(findings, fileFindings...)
		filesChecked = 1
		if cfgErr == nil {
			resolution = &ConfigResolution{Effective: effective, Discovery: discovery.Discovery}
		}
	default:
		findings = append(findings, Diagnostic{
			Name:         "config_scope_mismatch",
			Status:       "error",
			Message:      scopeMismatchMessage(discovery),
			SuggestedFix: scopeMismatchFix,
		})
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Name != findings[j].Name {
			return findings[i].Name < findings[j].Name
		}
		return findings[i].Message < findings[j].Message
	})

	summary := validationSummary{FilesChecked: filesChecked}
	for _, finding := range findings {
		switch finding.Status {
		case "error":
			summary.Errors++
		case "warning":
			summary.Warnings++
		}
	}

	data := configValidatePayload{
		Findings: findings,
		Summary:  summary,
	}
	if resolution != nil {
		report := newConfigReport(*resolution)
		data.Config = &report
	}

	status := "ok"
	if summary.Errors > 0 {
		status = "error"
	} else if summary.Warnings > 0 {
		status = "warning"
	}
	writeEnvelope(stdout, Envelope{
		Command: "config validate",
		Status:  status,
		Data:    data,
		NextActions: []NextAction{
			{Command: "canon config show", Description: "Inspect the effective configuration.", Safety: "read-only"},
		},
	}, opts.Format)
	if summary.Errors > 0 {
		return exitState
	}
	return exitOK
}
