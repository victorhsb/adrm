package canon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripJSONComments(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"line comment", "{\n// note\n}\n", "{\n\n}\n"},
		{"trailing line comment", "{\"a\": 1} // done", "{\"a\": 1} "},
		{"block comment", "{/* hidden */}", "{}"},
		{"block comment keeps newlines", "{/* a\nb */}", "{\n}"},
		{"slashes inside string stay", "{\"url\": \"https://x/*y\"}", "{\"url\": \"https://x/*y\"}"},
		{"escaped quote in string", "{\"a\": \"\\\"//\"}", "{\"a\": \"\\\"//\"}"},
		{"no comments", "{\"a\": 1}", "{\"a\": 1}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := string(stripJSONComments([]byte(tc.input))); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveStorePolicyDefaultsWhenAbsent(t *testing.T) {
	t.Parallel()
	resolution, cfgErr := resolveStorePolicy(KindADR, t.TempDir())
	if cfgErr != nil {
		t.Fatalf("cfgErr = %v", cfgErr)
	}
	cfg := resolution.Effective
	if cfg.Source != "defaults" || cfg.Path != "" {
		t.Fatalf("source = %q path = %q, want defaults", cfg.Source, cfg.Path)
	}
	if !cfg.AppendEnabled() {
		t.Fatal("append should default to enabled")
	}
	for _, kind := range supportedKinds {
		if !cfg.KindRequired(kind) {
			t.Fatalf("kind %s should be required by default", kind)
		}
	}
	if cfg.StrictValidation() || cfg.ReasonRequired() || cfg.ProposedCreationRequired() {
		t.Fatalf("strict/lifecycle policies should default to disabled: %+v", cfg)
	}
	if _, restricted := cfg.AllowedTags(KindADR); restricted {
		t.Fatal("tags should default to unrestricted")
	}
}

func TestResolveStorePolicyDiscoversAncestor(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	deep := filepath.Join(root, "docs", "adr")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	data := "{\n// no appendices here\n\"conventions\": {\"append\": false}\n}\n"
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	resolution, cfgErr := resolveStorePolicy(KindADR, deep)
	if cfgErr != nil {
		t.Fatalf("cfgErr = %v", cfgErr)
	}
	if resolution.Effective.Source != "file" || filepath.Clean(resolution.Effective.Path) != filepath.Join(root, configFileName) {
		t.Fatalf("source = %q path = %q", resolution.Effective.Source, resolution.Effective.Path)
	}
	if resolution.Effective.AppendEnabled() {
		t.Fatal("append should be disabled")
	}
}

func TestResolveStorePolicyReportsMalformedFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, cfgErr := resolveStorePolicy(KindADR, root)
	if cfgErr == nil {
		t.Fatal("expected config error")
	} else if cfgErr.Code != "invalid_config" {
		t.Fatalf("code = %q, want invalid_config", cfgErr.Code)
	}
}

// TestAppendDisabledByConfig exercises the append gate end to end: a corpus
// whose repo-root config sets conventions.append=false rejects both apply and
// dry-run attempts with a config error.
func TestAppendDisabledByConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	adrDir := filepath.Join(root, "docs", "adr")
	if err := os.MkdirAll(adrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := "{\n// edit documents directly; git is the history\n\"schema_version\": \"canon.v1\",\n\"conventions\": {\"append\": false}\n}\n"
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _ := runForTest(t, "--adr-dir", adrDir, "adr", "new", "--title", "Use SQLite"); code != exitOK {
		t.Fatalf("adr new code = %d", code)
	}
	code, env := runForTest(t, "--adr-dir", adrDir, "append", "--id", "ADR-0001", "--title", "Review", "--body", "Still valid.")
	if code != exitState {
		t.Fatalf("code = %d, want %d", code, exitState)
	}
	errObj := env["error"].(map[string]any)
	if errObj["code"] != "append_disabled" || errObj["category"] != "config" {
		t.Fatalf("error = %#v", errObj)
	}
	code, env = runForTest(t, "--adr-dir", adrDir, "append", "--id", "ADR-0001", "--title", "Review", "--body", "Still valid.", "--dry-run")
	if code != exitState {
		t.Fatalf("dry-run code = %d, want %d", code, exitState)
	}
	if env["error"].(map[string]any)["code"] != "append_disabled" {
		t.Fatalf("dry-run error = %#v", env["error"])
	}
	code, env = runForTest(t, "--adr-dir", adrDir, "show", "--id", "ADR-0001")
	if code != exitOK {
		t.Fatalf("show code = %d", code)
	}
	if actions, ok := env["next_actions"].([]any); ok {
		for _, raw := range actions {
			if strings.Contains(raw.(map[string]any)["command"].(string), "append") {
				t.Fatalf("show suggests disabled append: %#v", raw)
			}
		}
	}
}

func TestAppendEnabledWithoutConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Use SQLite"); code != exitOK {
		t.Fatalf("adr new code = %d", code)
	}
	if code, env := runForTest(t, "--adr-dir", dir, "append", "--id", "ADR-0001", "--title", "Review", "--body", "Still valid."); code != exitOK {
		t.Fatalf("append code = %d, error = %#v", code, env["error"])
	}
}

// policyFlagsForTest creates a temp corpus root holding the listed kind
// directories (unlisted kinds are configured but absent) plus an optional
// config file, and returns the global flags that point every store at it.
func policyFlagsForTest(t *testing.T, config string, present ...string) (string, []string) {
	t.Helper()
	root := t.TempDir()
	for _, kind := range present {
		if err := os.MkdirAll(filepath.Join(root, "docs", kind), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if config != "" {
		if err := os.WriteFile(filepath.Join(root, configFileName), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, []string{
		"--adr-dir", filepath.Join(root, "docs", "adr"),
		"--spec-dir", filepath.Join(root, "docs", "spec"),
		"--domain-dir", filepath.Join(root, "docs", "domain"),
	}
}

func errorCode(env map[string]any) string {
	errObj, ok := env["error"].(map[string]any)
	if !ok {
		return ""
	}
	code, _ := errObj["code"].(string)
	return code
}

func TestResolvePartialAndCompleteFiles(t *testing.T) {
	t.Parallel()
	partial := `{"conventions": {"lifecycle": {"require_reason": true}}}`
	root, dirs := policyFlagsForTest(t, partial)
	_ = root
	resolution, cfgErr := resolveStorePolicy(KindADR, dirs[1])
	if cfgErr != nil {
		t.Fatalf("cfgErr = %v", cfgErr)
	}
	cfg := resolution.Effective
	if !cfg.AppendEnabled() || cfg.StrictValidation() || cfg.ProposedCreationRequired() {
		t.Fatalf("partial file must keep other defaults: %+v", cfg)
	}
	if !cfg.ReasonRequired() {
		t.Fatal("require_reason should be true")
	}
	for _, kind := range supportedKinds {
		if !cfg.KindRequired(kind) {
			t.Fatalf("partial file must require every kind: %s", kind)
		}
	}

	complete := `{
		"schema_version": "canon.v1",
		"conventions": {
			"append": false,
			"required_kinds": ["adr", "domain"],
			"validation": {"strict": true},
			"lifecycle": {"require_reason": true, "new_documents_must_be_proposed": true},
			"tags": {"adr": {"allowed": ["cli", "config"]}, "domain": {"allowed": []}}
		}
	}`
	_, dirs = policyFlagsForTest(t, complete)
	resolution, cfgErr = resolveStorePolicy(KindADR, dirs[1])
	if cfgErr != nil {
		t.Fatalf("cfgErr = %v", cfgErr)
	}
	cfg = resolution.Effective
	if cfg.AppendEnabled() || !cfg.StrictValidation() || !cfg.ReasonRequired() || !cfg.ProposedCreationRequired() {
		t.Fatalf("complete file values not applied: %+v", cfg)
	}
	if got := cfg.RequiredKinds(); len(got) != 2 || got[0] != "adr" || got[1] != "domain" {
		t.Fatalf("required kinds = %v, want canonical [adr domain]", got)
	}
	allowed, restricted := cfg.AllowedTags(KindADR)
	if !restricted || len(allowed) != 2 || allowed[0] != "cli" || allowed[1] != "config" {
		t.Fatalf("adr tags = %v restricted = %v", allowed, restricted)
	}
	if allowed, restricted := cfg.AllowedTags(KindDomain); !restricted || len(allowed) != 0 {
		t.Fatalf("domain tags = %v restricted = %v, want explicit empty", allowed, restricted)
	}
	if _, restricted := cfg.AllowedTags(KindSPEC); restricted {
		t.Fatal("spec should be unrestricted when omitted from tags")
	}
}

func TestSchemaVersionHandling(t *testing.T) {
	t.Parallel()
	_, dirs := policyFlagsForTest(t, `{"schema_version": "canon.v1"}`)
	if _, cfgErr := resolveStorePolicy(KindADR, dirs[1]); cfgErr != nil {
		t.Fatalf("explicit canon.v1 must pass: %v", cfgErr)
	}
	_, dirs = policyFlagsForTest(t, `{"schema_version": "canon.v2"}`)
	_, cfgErr := resolveStorePolicy(KindADR, dirs[1])
	if cfgErr == nil || !strings.Contains(cfgErr.Message, "unsupported schema_version") {
		t.Fatalf("unsupported version must fail: %v", cfgErr)
	}
	insp := inspectConfig(".canon.jsonc", []byte(`{"schema_version": "canon.v2"}`))
	found := false
	for _, f := range insp.findings {
		if f.Name == "unsupported_config_schema" && f.Status == "error" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unsupported_config_schema finding: %#v", insp.findings)
	}
}

func TestUnknownKeysCollectedSortedAndIgnored(t *testing.T) {
	t.Parallel()
	content := `{
		"schema_version": "canon.v1",
		"zz_top": 1,
		"conventions": {
			"append": true,
			"validation": {"strict": false, "future_mode": "x"},
			"ui": {"theme": "dark"}
		}
	}`
	insp := inspectConfig(".canon.jsonc", []byte(content))
	want := []string{"conventions.ui", "conventions.validation.future_mode", "zz_top"}
	if strings.Join(insp.unknown, ",") != strings.Join(want, ",") {
		t.Fatalf("unknown = %v, want %v", insp.unknown, want)
	}
	if hasErrorFindings(insp.findings) {
		t.Fatalf("unknown keys must not be errors: %#v", insp.findings)
	}
	effective := buildEffective(insp)
	if !effective.AppendEnabled() || effective.StrictValidation() {
		t.Fatalf("known values must still apply around unknown keys: %+v", effective)
	}
	warnings := 0
	for _, f := range insp.findings {
		if f.Name == "unknown_config_key" && f.Status == "warning" {
			warnings++
		}
	}
	if warnings != len(want) {
		t.Fatalf("expected one warning per unknown key: %#v", insp.findings)
	}
	recognized := strings.Join(insp.recognized, ",")
	for _, key := range []string{"schema_version", "conventions", "conventions.append", "conventions.validation", "conventions.validation.strict"} {
		if !strings.Contains(recognized, key) {
			t.Fatalf("recognized keys missing %q: %v", key, insp.recognized)
		}
	}
}

func TestInvalidConfigValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		content string
		finding string
	}{
		{"wrong type", `{"conventions": {"append": "no"}}`, "invalid_config_value"},
		{"empty required kinds", `{"conventions": {"required_kinds": []}}`, "invalid_config_value"},
		{"duplicate required kinds", `{"conventions": {"required_kinds": ["adr", "adr"]}}`, "invalid_config_value"},
		{"unknown required kind", `{"conventions": {"required_kinds": ["adr", "memo"]}}`, "invalid_config_value"},
		{"unknown tag-policy kind", `{"conventions": {"tags": {"memo": {"allowed": ["x"]}}}}`, "invalid_config_value"},
		{"missing allowed array", `{"conventions": {"tags": {"adr": {}}}}`, "invalid_config_value"},
		{"blank tag", `{"conventions": {"tags": {"adr": {"allowed": ["  "]}}}}`, "invalid_config_value"},
		{"duplicate tag after trim", `{"conventions": {"tags": {"adr": {"allowed": ["cli", " cli "]}}}}`, "invalid_config_value"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			insp := inspectConfig(".canon.jsonc", []byte(tc.content))
			if !hasErrorFindings(insp.findings) {
				t.Fatalf("expected error finding: %#v", insp.findings)
			}
			if insp.findings[0].Name != tc.finding && !containsFinding(insp.findings, tc.finding) {
				t.Fatalf("expected %s finding: %#v", tc.finding, insp.findings)
			}
			_, dirs := policyFlagsForTest(t, tc.content)
			if _, cfgErr := resolveStorePolicy(KindADR, dirs[1]); cfgErr == nil {
				t.Fatal("invalid config must fail policy resolution")
			}
		})
	}
}

func containsFinding(findings []Diagnostic, name string) bool {
	for _, f := range findings {
		if f.Name == name {
			return true
		}
	}
	return false
}

func TestFindConfigFileFromAbsentDirectory(t *testing.T) {
	t.Parallel()
	root, dirs := policyFlagsForTest(t, `{"conventions": {"append": false}}`, "adr")
	// The spec and domain directories are configured but absent; discovery
	// must still find the root file by walking up from the absent path.
	absPath, found, err := findConfigFile(dirs[3])
	if err != nil || !found {
		t.Fatalf("found = %v err = %v", found, err)
	}
	if absPath != filepath.Join(root, configFileName) {
		t.Fatalf("absPath = %q", absPath)
	}
}

func TestResolveRepoPolicyScenarios(t *testing.T) {
	t.Parallel()

	// All stores under one root share one file.
	_, dirs := policyFlagsForTest(t, `{"conventions": {"append": false}}`, "adr", "domain")
	repo := NewRepo(GlobalOptions{ADRDir: dirs[1], SpecDir: dirs[3], DomainDir: dirs[5], Format: "json"})
	resolution, cfgErr := resolveRepoPolicy(repo)
	if cfgErr != nil {
		t.Fatalf("shared root must resolve: %v", cfgErr)
	}
	if resolution.Effective.Source != "file" || resolution.Effective.AppendEnabled() {
		t.Fatalf("resolution = %+v", resolution.Effective)
	}

	// No file anywhere yields defaults.
	_, dirs = policyFlagsForTest(t, "", "adr")
	repo = NewRepo(GlobalOptions{ADRDir: dirs[1], SpecDir: dirs[3], DomainDir: dirs[5], Format: "json"})
	resolution, cfgErr = resolveRepoPolicy(repo)
	if cfgErr != nil {
		t.Fatalf("no file must resolve to defaults: %v", cfgErr)
	}
	if resolution.Effective.Source != "defaults" {
		t.Fatalf("source = %q", resolution.Effective.Source)
	}

	// Only some stores discovering a file is a scope mismatch.
	rootA := t.TempDir()
	rootB := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootA, configFileName), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	repo = NewRepo(GlobalOptions{
		ADRDir:    filepath.Join(rootA, "docs", "adr"),
		SpecDir:   filepath.Join(rootB, "docs", "spec"),
		DomainDir: filepath.Join(rootB, "docs", "domain"),
		Format:    "json",
	})
	if _, cfgErr := resolveRepoPolicy(repo); cfgErr == nil || cfgErr.Code != "config_scope_mismatch" {
		t.Fatalf("partial discovery must be a scope mismatch: %v", cfgErr)
	}

	// Two different discovered files are a scope mismatch.
	if err := os.WriteFile(filepath.Join(rootB, configFileName), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, cfgErr := resolveRepoPolicy(repo); cfgErr == nil || cfgErr.Code != "config_scope_mismatch" {
		t.Fatalf("conflicting files must be a scope mismatch: %v", cfgErr)
	}
}

func TestConfigShowDefaults(t *testing.T) {
	t.Parallel()
	_, dirs := policyFlagsForTest(t, "", "adr")
	args := append(append([]string{}, dirs...), "config", "show")
	code, env := runForTest(t, args...)
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	data := env["data"].(map[string]any)
	if data["source"] != "defaults" {
		t.Fatalf("source = %v", data["source"])
	}
	if _, hasPath := data["path"]; hasPath {
		t.Fatalf("defaults must omit path: %#v", data)
	}
	effective := data["effective"].(map[string]any)
	conventions := effective["conventions"].(map[string]any)
	if conventions["append"] != true || conventions["validation"].(map[string]any)["strict"] != false {
		t.Fatalf("effective = %#v", effective)
	}
}

func TestConfigValidateUnknownKeysExitZero(t *testing.T) {
	t.Parallel()
	_, dirs := policyFlagsForTest(t, `{"schema_version": "canon.v1", "conventions": {"append": false}, "future_key": true}`)
	args := append(append([]string{}, dirs...), "config", "validate")
	code, env := runForTest(t, args...)
	if code != exitOK {
		t.Fatalf("warning-only config validate code = %d", code)
	}
	if env["status"] != "warning" {
		t.Fatalf("status = %v", env["status"])
	}
	data := env["data"].(map[string]any)
	findings := data["findings"].([]any)
	if len(findings) != 1 || findings[0].(map[string]any)["name"] != "unknown_config_key" {
		t.Fatalf("findings = %#v", findings)
	}
	summary := data["summary"].(map[string]any)
	if summary["warnings"] != float64(1) || summary["errors"] != float64(0) {
		t.Fatalf("summary = %#v", summary)
	}
	if _, ok := data["config"]; !ok {
		t.Fatal("valid config must include the effective report")
	}
}

func TestConfigValidateErrorsExitFour(t *testing.T) {
	t.Parallel()
	_, dirs := policyFlagsForTest(t, `{"schema_version": "canon.v9"}`)
	args := append(append([]string{}, dirs...), "config", "validate")
	code, env := runForTest(t, args...)
	if code != exitState {
		t.Fatalf("code = %d, want %d", code, exitState)
	}
	if env["status"] != "error" {
		t.Fatalf("status = %v", env["status"])
	}
	data := env["data"].(map[string]any)
	findings := data["findings"].([]any)
	if len(findings) != 1 || findings[0].(map[string]any)["name"] != "unsupported_config_schema" {
		t.Fatalf("findings = %#v", findings)
	}
	if _, ok := data["config"]; ok {
		t.Fatal("invalid config must not include the effective report")
	}
}

func TestConfigCommandsRejectContextFormat(t *testing.T) {
	t.Parallel()
	for _, sub := range []string{"show", "validate"} {
		code, out := runRawForTest(t, "--format", "context", "config", sub)
		if code != exitUsage {
			t.Fatalf("config %s context code = %d", sub, code)
		}
		if !strings.Contains(out, "unsupported_context_format") {
			t.Fatalf("config %s context output: %s", sub, out)
		}
	}
}

func TestConfigCommandsRegistered(t *testing.T) {
	t.Parallel()
	code, env := runForTest(t, "commands")
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	seen := map[string]bool{}
	for _, raw := range env["data"].(map[string]any)["commands"].([]any) {
		command := raw.(map[string]any)
		name := command["name"].(string)
		if name == "config show" || name == "config validate" {
			if command["safety"] != "read-only" || command["mutating"] != false {
				t.Fatalf("%s must be read-only: %#v", name, command)
			}
			seen[name] = true
		}
	}
	if !seen["config show"] || !seen["config validate"] {
		t.Fatalf("config commands missing from registry: %v", seen)
	}
}

func TestRequiredKindsMakeOptionalSpecHealthy(t *testing.T) {
	t.Parallel()
	config := `{"conventions": {"required_kinds": ["adr", "domain"]}}`
	_, dirs := policyFlagsForTest(t, config, "adr", "domain")

	// Doctor reports the absent optional SPEC store as ok and never suggests
	// initializing it.
	args := append(append([]string{}, dirs...), "doctor")
	code, env := runForTest(t, args...)
	if code != exitOK {
		t.Fatalf("doctor code = %d", code)
	}
	if env["status"] != "ok" {
		t.Fatalf("doctor status = %v, data = %#v", env["status"], env["data"])
	}
	sawOptionalOK := false
	for _, raw := range env["data"].(map[string]any)["diagnostics"].([]any) {
		d := raw.(map[string]any)
		if d["name"] == "spec_directory" {
			if d["status"] != "ok" {
				t.Fatalf("spec_directory = %#v", d)
			}
			if _, hasFix := d["suggested_fix"]; hasFix {
				t.Fatalf("optional store must not suggest init: %#v", d)
			}
			sawOptionalOK = true
		}
	}
	if !sawOptionalOK {
		t.Fatalf("expected a spec_directory diagnostic: %#v", env["data"])
	}
	for _, raw := range env["next_actions"].([]any) {
		if strings.Contains(raw.(map[string]any)["command"].(string), "spec init") {
			t.Fatalf("doctor must not suggest spec init: %#v", raw)
		}
	}

	// Corpus validation is healthy with zero spec files.
	args = append(append([]string{}, dirs...), "validate")
	code, env = runForTest(t, args...)
	if code != exitOK {
		t.Fatalf("validate code = %d", code)
	}
	if env["status"] != "ok" {
		t.Fatalf("validate status = %v, data = %#v", env["status"], env["data"])
	}

	// Kind-scoped validation of the absent optional store is healthy.
	args = append(append([]string{}, dirs...), "spec", "validate")
	code, env = runForTest(t, args...)
	if code != exitOK || env["status"] != "ok" {
		t.Fatalf("spec validate code = %d status = %v", code, env["status"])
	}
	data := env["data"].(map[string]any)
	if data["summary"].(map[string]any)["files_checked"] != float64(0) {
		t.Fatalf("summary = %#v", data["summary"])
	}

	// An existing optional SPEC store is still scanned. The hand-written doc
	// references a missing document, which deep validation must catch.
	if code, _ := runForTest(t, append(append([]string{}, dirs...), "spec", "init")...); code != exitOK {
		t.Fatal("spec init failed")
	}
	writeRawDocForTest(t, dirs[3], "0001-broken.md", "SPEC-0001", "accepted", "tags: fine\nsuperseded_by: SPEC-0099\n")
	args = append(append([]string{}, dirs...), "spec", "validate")
	code, env = runForTest(t, args...)
	findings := env["data"].(map[string]any)["findings"].([]any)
	sawBroken := false
	for _, raw := range findings {
		if raw.(map[string]any)["name"] == "broken_reference" {
			sawBroken = true
		}
	}
	if !sawBroken || code == exitOK {
		t.Fatalf("present optional store must be validated: code = %d findings = %#v", code, findings)
	}
}

func TestConfiguredStrictnessComposesWithFlag(t *testing.T) {
	t.Parallel()
	// A corpus whose only finding is a warning (malformed date).
	setup := func(strict bool) []string {
		config := "{}"
		if strict {
			config = `{"conventions": {"validation": {"strict": true}}}`
		}
		_, dirs := policyFlagsForTest(t, config, "adr")
		writeRawDocForTest(t, dirs[1], "0001-bad-date.md", "ADR-0001", "accepted", "date: 24-08-2026\n")
		return dirs
	}

	dirs := setup(false)
	args := append(append([]string{}, dirs...), "validate")
	code, env := runForTest(t, args...)
	if code != exitOK || env["status"] != "warning" {
		t.Fatalf("without strictness warnings must exit 0: code = %d status = %v", code, env["status"])
	}
	code, env = runForTest(t, append(append([]string{}, dirs...), "validate", "--strict")...)
	if code != exitState {
		t.Fatalf("--strict must exit 4 on warnings: code = %d", code)
	}
	if env["status"] != "warning" {
		t.Fatalf("strictness must not rewrite severity: status = %v", env["status"])
	}

	dirs = setup(true)
	code, env = runForTest(t, append(append([]string{}, dirs...), "validate")...)
	if code != exitState {
		t.Fatalf("configured strictness must exit 4 on warnings: code = %d", code)
	}
	if env["status"] != "warning" {
		t.Fatalf("configured strictness must not rewrite severity: status = %v", env["status"])
	}
}

func TestLifecycleReasonRequiredByConfig(t *testing.T) {
	t.Parallel()
	config := `{"conventions": {"lifecycle": {"require_reason": true}}}`
	_, dirs := policyFlagsForTest(t, config, "adr", "domain")
	base := append([]string{}, dirs...)
	create := func() {
		t.Helper()
		if code, env := runForTest(t, append(base, "adr", "new", "--title", "Gate check")...); code != exitOK {
			t.Fatalf("adr new code = %d env = %#v", code, env["error"])
		}
	}
	create() // ADR-0001
	create() // ADR-0002

	// accept, reject, and deprecate reject blank reasons in both modes.
	for _, command := range []string{"accept", "reject", "deprecate"} {
		for _, extra := range []string{"", "--dry-run"} {
			args := append(append([]string{}, base...), command, "--id", "ADR-0001")
			if extra != "" {
				args = append(args, extra)
			}
			code, env := runForTest(t, args...)
			if code != exitState || errorCode(env) != "reason_required_by_config" {
				t.Fatalf("%s %s code = %d error = %#v", command, extra, code, env["error"])
			}
			if env["error"].(map[string]any)["category"] != "config" {
				t.Fatalf("%s category = %#v", command, env["error"])
			}
		}
	}
	for _, extra := range []string{"", "--dry-run"} {
		args := append(append([]string{}, base...), "supersede", "--id", "ADR-0001", "--by", "ADR-0002")
		if extra != "" {
			args = append(args, extra)
		}
		code, env := runForTest(t, args...)
		if code != exitState || errorCode(env) != "reason_required_by_config" {
			t.Fatalf("supersede %s code = %d error = %#v", extra, code, env["error"])
		}
	}

	// The document must be untouched. Reasoned calls succeed.
	code, env := runForTest(t, append(base, "show", "--id", "ADR-0001")...)
	if code != exitOK {
		t.Fatalf("show code = %d", code)
	}
	if env["data"].(map[string]any)["adr"].(map[string]any)["status"] != "proposed" {
		t.Fatalf("doc mutated despite rejection: %#v", env["data"])
	}
	if code, _ := runForTest(t, append(base, "accept", "--id", "ADR-0001", "--reason", "Approved.")...); code != exitOK {
		t.Fatal("accept with reason must succeed")
	}
}

func TestProposedCreationRequiredByConfig(t *testing.T) {
	t.Parallel()
	config := `{"conventions": {"lifecycle": {"new_documents_must_be_proposed": true}}}`
	_, dirs := policyFlagsForTest(t, config, "adr", "spec", "domain")
	base := append([]string{}, dirs...)
	for _, kind := range []string{"adr", "spec", "domain"} {
		for _, extra := range []string{"", "--dry-run"} {
			args := append(append([]string{}, base...), kind, "new", "--title", "Gate check", "--status", "accepted")
			if extra != "" {
				args = append(args, extra)
			}
			code, env := runForTest(t, args...)
			if code != exitState || errorCode(env) != "initial_status_restricted" {
				t.Fatalf("%s new %s code = %d error = %#v", kind, extra, code, env["error"])
			}
		}
	}
	// No files were created.
	entries, err := os.ReadDir(dirs[1])
	if err != nil || len(entries) != 0 {
		t.Fatalf("rejected creations must not write: err = %v entries = %v", err, entries)
	}
	// Proposed creation still works.
	if code, env := runForTest(t, append(base, "adr", "new", "--title", "Gate check")...); code != exitOK {
		t.Fatalf("proposed creation code = %d env = %#v", code, env["error"])
	}
}

func TestTagPolicyEnforcement(t *testing.T) {
	t.Parallel()
	config := `{"conventions": {"tags": {"adr": {"allowed": ["cli", "config"]}}}}`
	_, dirs := policyFlagsForTest(t, config, "adr", "spec")
	base := append([]string{}, dirs...)

	// Allowed tags pass.
	if code, env := runForTest(t, append(base, "adr", "new", "--title", "Use tags", "--tags", "cli, config")...); code != exitOK {
		t.Fatalf("allowed tags code = %d env = %#v", code, env["error"])
	}
	// Disallowed tags fail before write, in both modes and with all offenders listed.
	for _, extra := range []string{"", "--dry-run"} {
		args := append(append([]string{}, base...), "adr", "new", "--title", "Bad tags", "--tags", "random, cli, zeta")
		if extra != "" {
			args = append(args, extra)
		}
		code, env := runForTest(t, args...)
		if code != exitState || errorCode(env) != "disallowed_tag" {
			t.Fatalf("disallowed %s code = %d error = %#v", extra, code, env["error"])
		}
		message := env["error"].(map[string]any)["message"].(string)
		if !strings.Contains(message, "random") || !strings.Contains(message, "zeta") {
			t.Fatalf("error must list every offending tag: %s", message)
		}
	}
	entries, err := os.ReadDir(dirs[1])
	if err != nil || len(entries) != 1 {
		t.Fatalf("rejected creation must not write: err = %v entries = %v", err, entries)
	}
	// Unrestricted kinds keep free-form tags.
	if code, env := runForTest(t, append(base, "spec", "new", "--title", "Free tags", "--tags", "whatever")...); code != exitOK {
		t.Fatalf("unrestricted spec code = %d env = %#v", code, env["error"])
	}
	// Hand-edited violations are found by deep validation.
	writeRawDocForTest(t, dirs[1], "0002-handmade.md", "ADR-0002", "accepted", "tags: rogue\n")
	code, env := runForTest(t, append(base, "validate")...)
	findings := env["data"].(map[string]any)["findings"].([]any)
	saw := false
	for _, raw := range findings {
		f := raw.(map[string]any)
		if f["name"] == "disallowed_tag" && f["status"] == "error" && f["id"] == "ADR-0002" {
			saw = true
		}
	}
	if !saw || code == exitOK {
		t.Fatalf("expected disallowed_tag error finding: code = %d findings = %#v", code, findings)
	}
	// The CLI-created document stayed clean.
	for _, raw := range findings {
		f := raw.(map[string]any)
		if f["name"] == "disallowed_tag" && f["id"] == "ADR-0001" {
			t.Fatalf("allowed tags must not be flagged: %#v", f)
		}
	}
}

func TestMalformedConfigFailsPolicyCommands(t *testing.T) {
	t.Parallel()
	// Seed a document while the configuration is still valid, then corrupt it.
	root, dirs := policyFlagsForTest(t, `{}`, "adr", "domain")
	base := append([]string{}, dirs...)
	if code, _ := runForTest(t, append(base, "adr", "new", "--title", "Seeded")...); code != exitOK {
		t.Fatal("seed creation failed")
	}
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"config", "show"},
		{"doctor"},
		{"validate"},
		{"adr", "validate"},
		{"validate", "--id", "ADR-0001"},
		{"show", "--id", "ADR-0001"},
		{"accept", "--id", "ADR-0001", "--reason", "x"},
		{"append", "--id", "ADR-0001", "--title", "T", "--body", "B"},
	} {
		code, env := runForTest(t, append(append([]string{}, base...), args...)...)
		if code != exitState {
			t.Fatalf("%v code = %d, want %d", args, code, exitState)
		}
		if env["error"].(map[string]any)["category"] != "config" {
			t.Fatalf("%v error = %#v", args, env["error"])
		}
	}
	// init stays usable for recovery even with a malformed configuration.
	if code, env := runForTest(t, append(base, "spec", "init")...); code != exitOK {
		t.Fatalf("spec init must work under malformed config: code = %d env = %#v", code, env["error"])
	}
	// Nothing else wrote or mutated files.
	entries, err := os.ReadDir(dirs[1])
	if err != nil || len(entries) != 1 {
		t.Fatalf("failed policy commands must not write: err = %v entries = %v", err, entries)
	}
}

func TestConfigTextOutput(t *testing.T) {
	t.Parallel()
	_, dirs := policyFlagsForTest(t, `{"conventions": {"append": false, "required_kinds": ["adr", "domain"]}}`, "adr", "domain")
	args := append([]string{"--format", "text"}, append(append([]string{}, dirs...), "config", "show")...)
	code, out := runRawForTest(t, args...)
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	for _, want := range []string{
		"config show: ok",
		"source: file",
		"  append: false",
		"  required_kinds: adr, domain",
		"recognized keys:",
		"unknown keys: -",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("text output missing %q:\n%s", want, out)
		}
	}
	args = append([]string{"--format", "text"}, append(append([]string{}, dirs...), "config", "validate")...)
	code, out = runRawForTest(t, args...)
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(out, "config validate: ok") || !strings.Contains(out, "summary: files_checked=1 errors=0 warnings=0") {
		t.Fatalf("text validate output:\n%s", out)
	}
}
