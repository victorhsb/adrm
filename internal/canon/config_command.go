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
type configReport struct {
	Source         string            `json:"source"`
	Path           string            `json:"path,omitempty"`
	DiscoveryPaths map[string]string `json:"discovery_paths"`
	Effective      effectivePayload  `json:"effective"`
	RecognizedKeys []string          `json:"recognized_keys"`
	UnknownKeys    []string          `json:"unknown_keys"`
}

type effectivePayload struct {
	SchemaVersion string             `json:"schema_version"`
	Conventions   conventionsPayload `json:"conventions"`
}

type conventionsPayload struct {
	Append        bool                        `json:"append"`
	RequiredKinds []string                    `json:"required_kinds"`
	Validation    validationPayload           `json:"validation"`
	Lifecycle     lifecyclePayload            `json:"lifecycle"`
	Tags          map[string]tagPolicyPayload `json:"tags"`
}

type validationPayload struct {
	Strict bool `json:"strict"`
}

type lifecyclePayload struct {
	RequireReason              bool `json:"require_reason"`
	NewDocumentsMustBeProposed bool `json:"new_documents_must_be_proposed"`
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
		Source:         effective.Source,
		Path:           effective.Path,
		DiscoveryPaths: resolution.Discovery,
		Effective: effectivePayload{
			SchemaVersion: effective.SchemaVersion,
			Conventions: conventionsPayload{
				Append:        effective.AppendEnabled(),
				RequiredKinds: effective.RequiredKinds(),
				Validation:    validationPayload{Strict: effective.StrictValidation()},
				Lifecycle: lifecyclePayload{
					RequireReason:              effective.ReasonRequired(),
					NewDocumentsMustBeProposed: effective.ProposedCreationRequired(),
				},
				Tags: tags,
			},
		},
		RecognizedKeys: recognized,
		UnknownKeys:    unknown,
	}
}

// reportPayload converts the report struct into a map so text rendering and
// JSON encoding share one representation with deterministic key order.
func reportPayload(report configReport) map[string]any {
	var payload map[string]any
	if !jsonCopy(report, &payload) {
		return map[string]any{}
	}
	return payload
}

// runConfig dispatches the configuration inspection subcommands. The command
// family is read-only and never mutates configuration or documents.
func runConfig(stdout, stderr io.Writer, opts GlobalOptions, repo Repo, args []string) int {
	if len(args) == 0 {
		writeEnvelope(stdout, errorEnvelope("config", "missing_config_subcommand", "usage", `"config" requires a subcommand`, "Use `canon config show` or `canon config validate`."), opts.Format)
		return exitUsage
	}
	switch args[0] {
	case "show":
		return runConfigShow(stdout, stderr, opts, repo, args[1:])
	case "validate":
		return runConfigValidate(stdout, stderr, opts, repo, args[1:])
	default:
		writeEnvelope(stdout, errorEnvelope("config", "unknown_command", "usage", fmt.Sprintf("unknown config subcommand %q", args[0]), "Use `canon config show` or `canon config validate`."), opts.Format)
		return exitUsage
	}
}

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
		Data:    reportPayload(newConfigReport(resolution)),
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

	data := map[string]any{
		"findings": findings,
		"summary":  summary,
	}
	if resolution != nil {
		data["config"] = reportPayload(newConfigReport(*resolution))
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
