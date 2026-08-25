package canon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// configFileName is the repo-root configuration file. It is discovered by
// walking up from a document store directory (for example docs/adr), so the
// configuration always belongs to the corpus it sits above. There is no
// global configuration; a repository without the file gets all defaults.
const configFileName = ".canon.jsonc"

// supportedKinds lists every document kind in canonical order. Configuration
// defaults and validation use this order so output stays deterministic.
var supportedKinds = []string{KindADR, KindSPEC, KindDomain}

// ConfigError is a structured configuration failure raised while resolving
// repository policy. Policy-aware commands surface it as a config-category
// envelope error with exit code 4.
type ConfigError struct {
	Code         string
	Message      string
	SuggestedFix string
}

func (e *ConfigError) Error() string { return e.Message }

// rawConfig mirrors the .canon.jsonc shape with pointer-backed fields so an
// omitted key is distinguishable from an explicit false or empty array.
// Defaults live in code through buildEffective, never in the file.
type rawConfig struct {
	SchemaVersion *string         `json:"schema_version"`
	Conventions   *rawConventions `json:"conventions"`
}

type rawConventions struct {
	Append        *bool                    `json:"append"`
	RequiredKinds *[]string                `json:"required_kinds"`
	Validation    *rawValidation           `json:"validation"`
	Lifecycle     *rawLifecycle            `json:"lifecycle"`
	Tags          *map[string]rawTagPolicy `json:"tags"`
}

type rawValidation struct {
	Strict *bool `json:"strict"`
}

type rawLifecycle struct {
	RequireReason              *bool `json:"require_reason"`
	NewDocumentsMustBeProposed *bool `json:"new_documents_must_be_proposed"`
}

type rawTagPolicy struct {
	Allowed *[]string `json:"allowed"`
}

// EffectiveConfig is the fully resolved repository policy: every convention
// has a concrete value, provenance records where the values came from, and
// key inspection reports which file keys were recognized or ignored.
type EffectiveConfig struct {
	SchemaVersion string

	append           bool
	requiredKinds    []string
	strictValidation bool
	requireReason    bool
	proposedCreation bool
	tagPolicies      map[string][]string // only kinds with a declared vocabulary

	// Source is "file" or "defaults".
	Source string
	// Path is the display path of the config file (repository-relative when
	// below the working directory, absolute otherwise); empty for defaults.
	Path string
	// absPath is the cleaned absolute identity path used for comparisons.
	absPath string
	// RecognizedKeys lists the sorted known JSON key paths present in the file.
	RecognizedKeys []string
	// UnknownKeys lists the sorted JSON key paths the schema does not know.
	UnknownKeys []string
}

// AppendEnabled reports whether the append command is allowed.
func (c EffectiveConfig) AppendEnabled() bool { return c.append }

// KindRequired reports whether the kind's store must exist for readiness.
func (c EffectiveConfig) KindRequired(kind string) bool {
	return contains(c.requiredKinds, kind)
}

// RequiredKinds returns the required kinds in canonical order.
func (c EffectiveConfig) RequiredKinds() []string {
	out := make([]string, len(c.requiredKinds))
	copy(out, c.requiredKinds)
	return out
}

// StrictValidation reports whether warning findings make validate exit 4.
func (c EffectiveConfig) StrictValidation() bool { return c.strictValidation }

// ReasonRequired reports whether lifecycle transitions need a --reason.
func (c EffectiveConfig) ReasonRequired() bool { return c.requireReason }

// ProposedCreationRequired reports whether new documents must start proposed.
func (c EffectiveConfig) ProposedCreationRequired() bool { return c.proposedCreation }

// AllowedTags returns the allowed tag vocabulary for a kind. The second
// result is false when the kind has no declared vocabulary (unrestricted).
func (c EffectiveConfig) AllowedTags(kind string) ([]string, bool) {
	allowed, ok := c.tagPolicies[kind]
	if !ok {
		return nil, false
	}
	out := make([]string, len(allowed))
	copy(out, allowed)
	return out, true
}

// defaultEffectiveConfig resolves every convention to its documented default,
// preserving the behavior of repositories without .canon.jsonc.
func defaultEffectiveConfig() EffectiveConfig {
	return EffectiveConfig{
		SchemaVersion: SchemaVersion,
		append:        true,
		requiredKinds: append([]string{}, supportedKinds...),
		tagPolicies:   map[string][]string{},
		Source:        "defaults",
	}
}

// configInspection is the result of decoding and validating one file: the raw
// model plus deterministic findings and key-path reports.
type configInspection struct {
	raw        rawConfig
	findings   []Diagnostic
	recognized []string
	unknown    []string
}

// recognizedConfigChildren maps each known object path to its known child
// keys. Keys outside these sets are unknown and reported as warnings. Tag
// kind names are validated semantically instead: any child of
// conventions.tags is handled there, so it never counts as unknown.
var recognizedConfigChildren = map[string]map[string]bool{
	"":                        {"schema_version": true, "conventions": true},
	"conventions":             {"append": true, "required_kinds": true, "validation": true, "lifecycle": true, "tags": true},
	"conventions.validation":  {"strict": true},
	"conventions.lifecycle":   {"require_reason": true, "new_documents_must_be_proposed": true},
	"conventions.tags":        {KindADR: true, KindSPEC: true, KindDomain: true},
	"conventions.tags.adr":    {"allowed": true},
	"conventions.tags.spec":   {"allowed": true},
	"conventions.tags.domain": {"allowed": true},
}

// collectKeyPaths walks a decoded JSON object tree and reports every present
// recognized key path and every outermost unknown key path, both sorted. It
// stops descending into unknown subtrees so one unknown object produces one
// reported path.
func collectKeyPaths(tree map[string]any) (recognized, unknown []string) {
	var walk func(prefix string, node map[string]any)
	walk = func(prefix string, node map[string]any) {
		children, known := recognizedConfigChildren[prefix]
		keys := make([]string, 0, len(node))
		for key := range node {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			if !known || !children[key] {
				unknown = append(unknown, path)
				continue
			}
			recognized = append(recognized, path)
			if child, ok := node[key].(map[string]any); ok {
				walk(path, child)
			}
		}
	}
	walk("", tree)
	return recognized, unknown
}

// inspectConfig decodes JSONC bytes into the raw model and validates every
// known value. Findings use the config check names: malformed_config,
// unsupported_config_schema, and invalid_config_value are errors;
// unknown_config_key is a warning. Unknown keys never block policy consumers.
func inspectConfig(path string, data []byte) configInspection {
	insp := configInspection{findings: []Diagnostic{}, recognized: []string{}, unknown: []string{}}
	stripped := stripJSONComments(data)

	var tree map[string]any
	if err := json.Unmarshal(stripped, &tree); err != nil {
		insp.findings = append(insp.findings, Diagnostic{
			Name:         "malformed_config",
			Status:       "error",
			Message:      fmt.Sprintf("%s is not valid JSONC: %s", path, err),
			SuggestedFix: fmt.Sprintf("Fix the JSON syntax in %s, or remove the file to use defaults.", path),
			Path:         path,
		})
		return insp
	}
	insp.recognized, insp.unknown = collectKeyPaths(tree)
	for _, key := range insp.unknown {
		insp.findings = append(insp.findings, Diagnostic{
			Name:         "unknown_config_key",
			Status:       "warning",
			Message:      fmt.Sprintf("%s declares unknown key %q, which this canon version ignores", path, key),
			SuggestedFix: "Remove the key, or upgrade canon if a newer version defines it.",
			Path:         path,
		})
	}

	if err := json.Unmarshal(stripped, &insp.raw); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			field := typeErr.Field
			if field == "" {
				field = typeErr.Struct
			}
			insp.findings = append(insp.findings, Diagnostic{
				Name:         "invalid_config_value",
				Status:       "error",
				Message:      fmt.Sprintf("%s has wrong type for %q: expected %s, got %s", path, field, typeErr.Type, typeErr.Value),
				SuggestedFix: "Fix the value so it matches the documented type, or remove the key.",
				Path:         path,
			})
			return insp
		}
		insp.findings = append(insp.findings, Diagnostic{
			Name:         "malformed_config",
			Status:       "error",
			Message:      fmt.Sprintf("%s is not valid JSONC: %s", path, err),
			SuggestedFix: fmt.Sprintf("Fix the JSON syntax in %s, or remove the file to use defaults.", path),
			Path:         path,
		})
		return insp
	}

	insp.findings = append(insp.findings, validateRawConfig(path, insp.raw)...)
	return insp
}

// validateRawConfig checks every known value semantically and returns error
// findings. It never mutates the raw model.
func validateRawConfig(path string, raw rawConfig) []Diagnostic {
	findings := []Diagnostic{}
	invalid := func(message, fix string) Diagnostic {
		return Diagnostic{Name: "invalid_config_value", Status: "error", Message: fmt.Sprintf("%s: %s", path, message), SuggestedFix: fix, Path: path}
	}

	if raw.SchemaVersion != nil && *raw.SchemaVersion != SchemaVersion {
		findings = append(findings, Diagnostic{
			Name:         "unsupported_config_schema",
			Status:       "error",
			Message:      fmt.Sprintf("%s declares unsupported schema_version %q; supported: %s", path, *raw.SchemaVersion, SchemaVersion),
			SuggestedFix: fmt.Sprintf("Set schema_version to %q, or remove the key.", SchemaVersion),
			Path:         path,
		})
	}

	if raw.Conventions == nil {
		return findings
	}
	conv := raw.Conventions

	if conv.RequiredKinds != nil {
		kinds := *conv.RequiredKinds
		if len(kinds) == 0 {
			findings = append(findings, invalid("conventions.required_kinds must not be empty", "List at least one of adr, spec, or domain, or remove the key to require all three."))
		}
		seen := map[string]bool{}
		for _, kind := range kinds {
			trimmed := strings.TrimSpace(kind)
			if !isKind(trimmed) {
				findings = append(findings, invalid(fmt.Sprintf("conventions.required_kinds contains unsupported kind %q", kind), fmt.Sprintf("Use only %s.", strings.Join(supportedKinds, ", "))))
				continue
			}
			if seen[trimmed] {
				findings = append(findings, invalid(fmt.Sprintf("conventions.required_kinds repeats kind %q", trimmed), "Remove the duplicate entry."))
			}
			seen[trimmed] = true
		}
	}

	if conv.Tags != nil {
		kinds := make([]string, 0, len(*conv.Tags))
		for kind := range *conv.Tags {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)
		for _, kind := range kinds {
			policy := (*conv.Tags)[kind]
			if !isKind(kind) {
				findings = append(findings, invalid(fmt.Sprintf("conventions.tags declares unsupported kind %q", kind), fmt.Sprintf("Use only %s as tag-policy keys.", strings.Join(supportedKinds, ", "))))
				continue
			}
			if policy.Allowed == nil {
				findings = append(findings, invalid(fmt.Sprintf("conventions.tags.%s requires an allowed array", kind), fmt.Sprintf("Add an allowed tag list, or remove conventions.tags.%s to leave the kind unrestricted.", kind)))
				continue
			}
			seen := map[string]bool{}
			for _, tag := range *policy.Allowed {
				trimmed := strings.TrimSpace(tag)
				if trimmed == "" {
					findings = append(findings, invalid(fmt.Sprintf("conventions.tags.%s.allowed contains a blank tag", kind), "Remove blank entries."))
					continue
				}
				if seen[trimmed] {
					findings = append(findings, invalid(fmt.Sprintf("conventions.tags.%s.allowed repeats tag %q", kind, trimmed), "Remove the duplicate tag."))
				}
				seen[trimmed] = true
			}
		}
	}

	return findings
}

// buildEffective resolves the validated raw model into concrete values.
// Callers must pass an inspection without error findings.
func buildEffective(insp configInspection) EffectiveConfig {
	effective := defaultEffectiveConfig()
	effective.Source = "file"
	effective.RecognizedKeys = insp.recognized
	effective.UnknownKeys = insp.unknown

	raw := insp.raw
	if raw.SchemaVersion != nil {
		effective.SchemaVersion = *raw.SchemaVersion
	}
	if raw.Conventions == nil {
		return effective
	}
	conv := raw.Conventions
	if conv.Append != nil {
		effective.append = *conv.Append
	}
	if conv.RequiredKinds != nil {
		kinds := map[string]bool{}
		for _, kind := range *conv.RequiredKinds {
			kinds[strings.TrimSpace(kind)] = true
		}
		effective.requiredKinds = []string{}
		for _, kind := range supportedKinds {
			if kinds[kind] {
				effective.requiredKinds = append(effective.requiredKinds, kind)
			}
		}
	}
	if conv.Validation != nil && conv.Validation.Strict != nil {
		effective.strictValidation = *conv.Validation.Strict
	}
	if conv.Lifecycle != nil {
		if conv.Lifecycle.RequireReason != nil {
			effective.requireReason = *conv.Lifecycle.RequireReason
		}
		if conv.Lifecycle.NewDocumentsMustBeProposed != nil {
			effective.proposedCreation = *conv.Lifecycle.NewDocumentsMustBeProposed
		}
	}
	if conv.Tags != nil {
		for _, kind := range supportedKinds {
			policy, ok := (*conv.Tags)[kind]
			if !ok || policy.Allowed == nil {
				continue
			}
			allowed := make([]string, 0, len(*policy.Allowed))
			for _, tag := range *policy.Allowed {
				allowed = append(allowed, strings.TrimSpace(tag))
			}
			sort.Strings(allowed)
			effective.tagPolicies[kind] = allowed
		}
	}
	return effective
}

// hasErrorFindings reports whether any finding has error severity.
func hasErrorFindings(findings []Diagnostic) bool {
	for _, finding := range findings {
		if finding.Status == "error" {
			return true
		}
	}
	return false
}

// findConfigFile walks up from startDir toward the filesystem root and
// returns the cleaned absolute path of the first .canon.jsonc found.
func findConfigFile(startDir string) (string, bool, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false, fmt.Errorf("resolve %s: %w", startDir, err)
	}
	for {
		path := filepath.Join(dir, configFileName)
		_, err := os.Stat(path)
		switch {
		case err == nil:
			return filepath.Clean(path), true, nil
		case errors.Is(err, os.ErrNotExist):
			parent := filepath.Dir(dir)
			if parent == dir {
				return "", false, nil
			}
			dir = parent
		default:
			return "", false, fmt.Errorf("read %s: %w", path, err)
		}
	}
}

// displayPath renders abs repository-relative when it sits below the current
// working directory and absolute otherwise. Internal identity comparisons use
// the cleaned absolute path, never this string.
func displayPath(abs string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return abs
	}
	rel, err := filepath.Rel(cwd, abs)
	if err != nil {
		return abs
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return abs
	}
	return rel
}

// loadEffectiveFromFile parses and validates one configuration file. It
// returns the effective config (valid only when the error is nil) and the
// file's findings, which include unknown-key warnings.
func loadEffectiveFromFile(absPath string) (EffectiveConfig, []Diagnostic, *ConfigError) {
	display := displayPath(absPath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return EffectiveConfig{}, nil, &ConfigError{
			Code:         "invalid_config",
			Message:      fmt.Sprintf("failed to read configuration %s: %s", display, err),
			SuggestedFix: "Check file permissions, or remove the file to use defaults.",
		}
	}
	insp := inspectConfig(display, data)
	if hasErrorFindings(insp.findings) {
		var displayErr Diagnostic
		for _, finding := range insp.findings {
			if finding.Status == "error" {
				displayErr = finding
				break
			}
		}
		return EffectiveConfig{}, insp.findings, &ConfigError{
			Code:         "invalid_config",
			Message:      displayErr.Message,
			SuggestedFix: displayErr.SuggestedFix + " Run `canon config validate` for every finding.",
		}
	}
	effective := buildEffective(insp)
	effective.Path = display
	effective.absPath = absPath
	return effective, insp.findings, nil
}

// ConfigResolution pairs an effective configuration with the store discovery
// paths that established it, keyed by kind.
type ConfigResolution struct {
	Effective EffectiveConfig
	Discovery map[string]string
}

// resolveStorePolicy discovers and resolves the configuration that governs
// one store directory. A missing file yields the all-defaults resolution.
func resolveStorePolicy(kind, storeDir string) (ConfigResolution, *ConfigError) {
	resolution := ConfigResolution{Discovery: map[string]string{kind: storeDir}}
	absPath, found, err := findConfigFile(storeDir)
	if err != nil {
		return resolution, &ConfigError{
			Code:         "invalid_config",
			Message:      err.Error(),
			SuggestedFix: "Check file permissions along the configuration search path.",
		}
	}
	if !found {
		resolution.Effective = defaultEffectiveConfig()
		return resolution, nil
	}
	effective, _, cfgErr := loadEffectiveFromFile(absPath)
	if cfgErr != nil {
		return resolution, cfgErr
	}
	resolution.Effective = effective
	return resolution, nil
}

// kindDir pairs a document kind with its configured store directory.
type kindDir struct {
	Kind string
	Dir  string
}

// repoStoreDirs returns every store's kind and configured directory in
// canonical order.
func repoStoreDirs(repo Repo) []kindDir {
	return []kindDir{
		{Kind: KindADR, Dir: repo.ADR.Dir},
		{Kind: KindSPEC, Dir: repo.Spec.Dir},
		{Kind: KindDomain, Dir: repo.Domain.Dir},
	}
}

// resolveRepoPolicy reconciles configuration discovery across all three
// store directories for corpus-wide commands. All stores must discover the
// same file, or all must discover none; mixed or conflicting sources are a
// config_scope_mismatch because no single repository policy applies.
func resolveRepoPolicy(repo Repo) (ConfigResolution, *ConfigError) {
	discovery, err := discoverRepoConfig(repo)
	resolution := ConfigResolution{Discovery: discovery.Discovery}
	if err != nil {
		return resolution, &ConfigError{
			Code:         "invalid_config",
			Message:      err.Error(),
			SuggestedFix: "Check file permissions along the configuration search path.",
		}
	}
	switch {
	case len(discovery.Paths) == 0:
		resolution.Effective = defaultEffectiveConfig()
		return resolution, nil
	case len(discovery.Paths) == 1 && discovery.FoundBy[discovery.Paths[0]] == discovery.StoreCount:
		effective, _, cfgErr := loadEffectiveFromFile(discovery.Paths[0])
		if cfgErr != nil {
			return resolution, cfgErr
		}
		resolution.Effective = effective
		return resolution, nil
	default:
		return resolution, &ConfigError{
			Code:         "config_scope_mismatch",
			Message:      scopeMismatchMessage(discovery),
			SuggestedFix: scopeMismatchFix,
		}
	}
}

// configErrorForCommand converts a ConfigError into an envelope for the
// current command. Every config-category failure exits 4.
func configErrorForCommand(command string, cfgErr *ConfigError) Envelope {
	return errorEnvelope(command, cfgErr.Code, "config", cfgErr.Message, cfgErr.SuggestedFix)
}

// configSourceLabel names where a policy value came from in user-facing
// messages: the display path of the file, or the defaults marker.
func configSourceLabel(cfg EffectiveConfig) string {
	if cfg.Source == "file" && cfg.Path != "" {
		return cfg.Path
	}
	return "configuration defaults"
}

// stripJSONComments removes // line comments and /* */ block comments that
// appear outside string literals. Newlines inside block comments are kept so
// parse error positions still line up with the source file.
func stripJSONComments(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	escaped := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		if c == '/' && i+1 < len(data) {
			switch data[i+1] {
			case '/':
				for i+1 < len(data) && data[i+1] != '\n' {
					i++
				}
				continue
			case '*':
				i += 2
				for i < len(data) && (data[i] != '*' || i+1 >= len(data) || data[i+1] != '/') {
					if data[i] == '\n' {
						out = append(out, '\n')
					}
					i++
				}
				i++
				continue
			}
		}
		out = append(out, c)
	}
	return out
}
