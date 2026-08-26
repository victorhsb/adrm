package canon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/victorhsb/canon/skill"
)

// This file pins the current canon.v1 JSON contract before the output
// renderer refactor. Golden files under testdata/output/golden record the
// exact bytes the CLI emits today; the corpus under testdata/output/corpus
// uses fixed dates so list, show, search, and mutation envelopes are fully
// deterministic. Payloads that include runtime values (today's date, the
// build version) use focused decoded comparisons instead of goldens.

var corpusFlagSet = []string{"--adr-dir", "t-adr", "--spec-dir", "t-spec", "--domain-dir", "t-domain"}

// stageCorpus copies the static corpus into a temporary directory and
// switches the test to it so store paths and skill target paths stay
// relative and deterministic. It must run before t.Chdir changes the
// working directory, or testdata would no longer resolve.
func stageCorpus(t *testing.T) string {
	t.Helper()
	corpusDir := filepath.Join("testdata", "output", "corpus")
	stores := []string{"t-adr", "t-spec", "t-domain"}
	tmp := t.TempDir()
	for _, store := range stores {
		dst := filepath.Join(tmp, store)
		if err := os.MkdirAll(dst, 0o755); err != nil {
			t.Fatalf("stage corpus store %s: %v", store, err)
		}
		files, err := os.ReadDir(filepath.Join(corpusDir, store))
		if err != nil {
			t.Fatalf("read corpus store %s: %v", store, err)
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(corpusDir, store, file.Name()))
			if err != nil {
				t.Fatalf("read corpus file %s/%s: %v", store, file.Name(), err)
			}
			if err := os.WriteFile(filepath.Join(dst, file.Name()), data, 0o644); err != nil {
				t.Fatalf("write corpus file %s/%s: %v", store, file.Name(), err)
			}
		}
	}
	t.Chdir(tmp)
	return tmp
}

func goldenBytes(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "output", "golden", name+".json"))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(data)
}

func TestJSONMatchesGoldenBytes(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
		setup    func(t *testing.T)
	}{
		{name: "root", args: nil, wantExit: exitOK},
		{name: "commands", args: []string{"commands"}, wantExit: exitOK},
		{name: "skill", args: []string{"skill"}, wantExit: exitOK},
		{
			name:     "skill-install-dryrun",
			args:     []string{"skill", "install", "--skill-dir", "t-skills", "--agent", "claude", "--dry-run"},
			wantExit: exitOK,
		},
		{
			name:     "skill-install",
			args:     []string{"skill", "install", "--skill-dir", "t-skills", "--agent", "claude"},
			wantExit: exitOK,
		},
		{
			name: "skill-update-dryrun",
			args: []string{"skill", "update", "--skill-dir", "t-skills", "--agent", "claude", "--dry-run"},
			setup: func(t *testing.T) {
				code, _ := runRawForTest(t, "skill", "install", "--skill-dir", "t-skills", "--agent", "claude")
				if code != exitOK {
					t.Fatalf("install before update code = %d", code)
				}
			},
			wantExit: exitOK,
		},
		{
			name:     "init-dryrun",
			args:     []string{"--adr-dir", "t-newstore", "--spec-dir", "t-spec", "--domain-dir", "t-domain", "adr", "init", "--dry-run"},
			wantExit: exitOK,
		},
		{
			name:     "init-applied",
			args:     []string{"--adr-dir", "t-newstore", "--spec-dir", "t-spec", "--domain-dir", "t-domain", "adr", "init"},
			wantExit: exitOK,
		},
		{name: "list", args: append(corpusFlagSet, "list"), wantExit: exitOK},
		{name: "adr-list", args: append(corpusFlagSet, "adr", "list"), wantExit: exitOK},
		{name: "show", args: append(corpusFlagSet, "show", "--id", "ADR-0001"), wantExit: exitOK},
		{name: "search", args: append(corpusFlagSet, "search", "--query", "querying"), wantExit: exitOK},
		{name: "domain-search", args: append(corpusFlagSet, "domain", "search", "--query", "commitment"), wantExit: exitOK},
		{name: "search-empty", args: append(corpusFlagSet, "search", "--query", "zzzz"), wantExit: exitOK},
		{name: "doctor-ok", args: append(corpusFlagSet, "doctor"), wantExit: exitOK},
		{
			name: "doctor-warning",
			args: append(corpusFlagSet, "doctor"),
			setup: func(t *testing.T) {
				if err := os.RemoveAll("t-spec"); err != nil {
					t.Fatalf("remove t-spec: %v", err)
				}
			},
			wantExit: exitOK,
		},
		{name: "validate-ok", args: append(corpusFlagSet, "validate"), wantExit: exitOK},
		{name: "config-show", args: append(corpusFlagSet, "config", "show"), wantExit: exitOK},
		{name: "config-validate", args: append(corpusFlagSet, "config", "validate"), wantExit: exitOK},
		{
			name:     "accept-dryrun",
			args:     append(corpusFlagSet, "accept", "--id", "ADR-0001", "--reason", "Approved for review.", "--dry-run"),
			wantExit: exitOK,
		},
		{
			name:     "accept-applied",
			args:     append(corpusFlagSet, "accept", "--id", "SPEC-0001", "--reason", "Ratified."),
			wantExit: exitOK,
		},
		{
			name:     "error-show-missing-id",
			args:     append(corpusFlagSet, "show"),
			wantExit: exitUsage,
		},
		{name: "error-unknown-command", args: []string{"frobnicate"}, wantExit: exitUsage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := goldenBytes(t, tt.name)
			stageCorpus(t)
			if tt.setup != nil {
				tt.setup(t)
			}
			code, output := runRawForTest(t, tt.args...)
			if code != tt.wantExit {
				t.Fatalf("exit = %d, want %d\n%s", code, tt.wantExit, output)
			}
			if output != want {
				t.Fatalf("output differs from golden %s\nwant:\n%s\ngot:\n%s", tt.name, want, output)
			}
		})
	}
}

// runtimePayloadKeys asserts the decoded shape of payloads whose exact bytes
// depend on runtime values: `adr new --dry-run` stamps today's date, and
// `version` reflects build metadata.
func TestNewDryRunJSONShape(t *testing.T) {
	stageCorpus(t)
	code, output := runRawForTest(t, "--adr-dir", "t-adr", "--spec-dir", "t-spec", "--domain-dir", "t-domain", "adr", "new", "--title", "Tentative Plan", "--tags", "storage", "--dry-run")
	if code != exitOK {
		t.Fatalf("code = %d\n%s", code, output)
	}
	var env map[string]any
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if env["status"] != "planned" {
		t.Fatalf("status = %v", env["status"])
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatalf("data missing or wrong type: %#v", env["data"])
	}
	if len(data) != 2 || data["adr"] == nil || data["plan"] == nil {
		t.Fatalf("data keys = %#v", keysOf(data))
	}
	adr := data["adr"].(map[string]any)
	wantADRKeys := []string{"date", "id", "kind", "number", "path", "status", "tags", "title"}
	if got := keysOf(adr); !equalStrings(got, wantADRKeys) {
		t.Fatalf("adr keys = %v, want %v", got, wantADRKeys)
	}
	if adr["date"] == "" {
		t.Fatal("adr.date missing")
	}
	plan := data["plan"].(map[string]any)
	wantPlanKeys := []string{"changes_made", "dry_run", "operations"}
	if got := keysOf(plan); !equalStrings(got, wantPlanKeys) {
		t.Fatalf("plan keys = %v, want %v", got, wantPlanKeys)
	}
	warnings, ok := env["warnings"].([]any)
	if !ok || len(warnings) != 1 || warnings[0] != "No changes were made." {
		t.Fatalf("warnings = %#v", env["warnings"])
	}
}

func TestVersionJSONShape(t *testing.T) {
	stageCorpus(t)
	code, output := runRawForTest(t, "version")
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	want := fmt.Sprintf(`{
  "schema_version": "canon.v1",
  "command": "version",
  "status": "ok",
  "data": {
    "version": %q
  },
  "next_actions": [
    {
      "command": "canon commands",
      "description": "Inspect all available commands and safety rules.",
      "safety": "read-only"
    }
  ]
}
`, versionString())
	if output != want {
		t.Fatalf("version output mismatch\nwant:\n%s\ngot:\n%s", want, output)
	}
}

// TestPayloadTextProjections pins the exact text rendering of each payload
// family with representative data. A new payload type must be added here and
// to the compile-time assertions in payloads.go.
func TestPayloadTextProjections(t *testing.T) {
	doc := ADR{
		Kind:   KindADR,
		ID:     "ADR-0001",
		Number: 1,
		Title:  "Use SQLite",
		Status: "accepted",
		Date:   "2025-01-06",
		Tags:   []string{"storage"},
		Path:   "t-adr/0001-use-sqlite.md",
	}
	docWithContent := doc
	docWithContent.Content = "# ADR-0001: Use SQLite\n\n## Status\n\naccepted\n"
	plan := Plan{
		DryRun: true,
		Operations: []OpPlan{
			{Action: "write_file", Path: "t-adr/0002-plan.md", Description: "Create new adr markdown file."},
		},
	}
	tests := []struct {
		name    string
		payload outputPayload
		want    string
	}{
		{
			name:    "root",
			payload: rootPayload{Commands: []string{"list", "show"}, Purpose: "Manage documents."},
			want:    "purpose: Manage documents.\ncommands:\n  list\n  show\n",
		},
		{
			name: "commands",
			payload: commandsPayload{
				Commands: []CommandInfo{{
					Name:      "list",
					Purpose:   "List documents.",
					Safety:    "read-only",
					Selectors: []string{"--status"},
					Examples:  []string{"canon list"},
				}},
				GlobalFlags: []globalFlag{{Name: "--adr-dir", Default: "docs/adr", Purpose: "Select ADR storage directory."}},
			},
			want: "commands:\n  list\n    purpose: List documents.\n    safety: read-only\n    selectors: --status\n    example: canon list\nglobal flags:\n  --adr-dir (default: docs/adr)\n    Select ADR storage directory.\n",
		},
		{
			name:    "version",
			payload: versionPayload{Version: "v1.2.3"},
			want:    "version: v1.2.3\n",
		},
		{
			name:    "doctor",
			payload: doctorPayload{Diagnostics: []Diagnostic{{Name: "adr_directory", Status: "ok", Message: "t-adr exists"}}},
			want:    "diagnostics:\n  adr_directory: ok - t-adr exists\n",
		},
		{
			name: "validate",
			payload: validatePayload{
				Findings: []Diagnostic{{Name: "duplicate_id", Status: "error", Message: "duplicate id", Path: "t-adr/0001-x.md", ID: "ADR-0001"}},
				Summary:  validationSummary{FilesChecked: 2, Errors: 1, Warnings: 0},
			},
			want: "findings:\n  duplicate_id: error - duplicate id (t-adr/0001-x.md ADR-0001)\nsummary: files_checked=2 errors=1 warnings=0\n",
		},
		{
			name:    "empty plan renders nothing",
			payload: Plan{},
			want:    "",
		},
		{
			name:    "bare plan",
			payload: plan,
			want:    "plan:\n  write_file: t-adr/0002-plan.md\n    Create new adr markdown file.\ndry_run: true\nchanges_made: false\n",
		},
		{
			name:    "dry-run plan payload",
			payload: planDryRunPayload{Plan: plan, TargetID: "ADR-0001"},
			want:    "plan:\n  write_file: t-adr/0002-plan.md\n    Create new adr markdown file.\ndry_run: true\nchanges_made: false\n",
		},
		{
			name:    "mutation with document",
			payload: mutationPayload{Plan: plan, ADR: doc},
			want:    "plan:\n  write_file: t-adr/0002-plan.md\n    Create new adr markdown file.\ndry_run: true\nchanges_made: false\nadr:\n  kind: adr\n  id: ADR-0001\n  title: Use SQLite\n  status: accepted\n  date: 2025-01-06\n  tags: storage\n  path: t-adr/0001-use-sqlite.md\n",
		},
		{
			name:    "show includes content",
			payload: showPayload{ADR: docWithContent},
			want:    "adr:\n  kind: adr\n  id: ADR-0001\n  title: Use SQLite\n  status: accepted\n  date: 2025-01-06\n  tags: storage\n  path: t-adr/0001-use-sqlite.md\n  content:\n    # ADR-0001: Use SQLite\n    \n    ## Status\n    \n    accepted\n    \n",
		},
		{
			name:    "list",
			payload: listPayload{ADRs: []ADR{doc}, Count: 1},
			want:    "count: 1\nadrs:\n  ADR-0001: Use SQLite [accepted] (storage)\n",
		},
		{
			name:    "empty list",
			payload: listPayload{ADRs: []ADR{}, Count: 0},
			want:    "count: 0\n",
		},
		{
			name: "search",
			payload: searchPayload{
				Count:   1,
				Query:   "querying",
				Results: []searchResult{{ADR: doc, Snippet: "Need local querying"}},
			},
			want: "query: querying\ncount: 1\nresults:\n  ADR-0001: Use SQLite [accepted]\n    snippet: Need local querying\n",
		},
		{
			name:    "config validate without config",
			payload: configValidatePayload{Findings: []Diagnostic{}, Summary: validationSummary{}},
			want:    "findings:\nsummary: files_checked=0 errors=0 warnings=0\n",
		},
		{
			name: "skill catalog",
			payload: skillCatalogPayload{
				Assets:          []skill.CatalogAsset{{Name: "canon", Kind: "skill", Version: "1", Hash: "sha256:abc", TargetPaths: []string{".agents/skills/canon/SKILL.md"}}},
				DefaultSkillDir: ".agents/skills",
			},
			want: "default_skill_dir: .agents/skills\nassets:\n  canon [skill]\n    version: 1\n    hash: sha256:abc\n    target paths:\n      .agents/skills/canon/SKILL.md\n",
		},
		{
			name: "skill mutation",
			payload: skillMutationPayload{
				Assets:  []skillManagedAsset{{Hash: "sha256:abc", Kind: "skill", Name: "canon", TargetPaths: []string{"t-skills/canon/SKILL.md"}, Version: "1"}},
				Plan:    plan,
				Targets: []string{"claude"},
			},
			want: "plan:\n  write_file: t-adr/0002-plan.md\n    Create new adr markdown file.\ndry_run: true\nchanges_made: false\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			tt.payload.renderText(&out)
			if out.String() != tt.want {
				t.Fatalf("text projection mismatch\nwant:\n%s\ngot:\n%s", tt.want, out.String())
			}
		})
	}
}

// TestContextProjections pins the bounded Markdown list projections and their
// heading per scope.
func TestContextProjections(t *testing.T) {
	doc := ADR{ID: "ADR-0001", Title: "Use SQLite"}
	tests := []struct {
		name    string
		payload contextPayload
		want    string
	}{
		{name: "combined", payload: listPayload{ADRs: []ADR{doc}, Count: 1}, want: "## Project Documents\n\n- `ADR-0001`: Use SQLite\n"},
		{name: "adr", payload: listPayload{ADRs: []ADR{doc}, Count: 1, scope: KindADR}, want: "## Architecture Decision Records\n\n- `ADR-0001`: Use SQLite\n"},
		{name: "spec", payload: listPayload{ADRs: []ADR{doc}, Count: 1, scope: KindSPEC}, want: "## Specifications\n\n- `ADR-0001`: Use SQLite\n"},
		{name: "domain", payload: listPayload{ADRs: []ADR{doc}, Count: 1, scope: KindDomain}, want: "## Domain Model\n\n- `ADR-0001`: Use SQLite\n"},
		{name: "empty", payload: listPayload{ADRs: []ADR{}, Count: 0, scope: KindADR}, want: "## Architecture Decision Records\n\n_No matching documents._\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			tt.payload.renderContext(&out)
			if out.String() != tt.want {
				t.Fatalf("context projection mismatch\nwant:\n%s\ngot:\n%s", tt.want, out.String())
			}
		})
	}
}

// TestTextCommandsEmitPayloadData guards the repaired behavior: every
// successful command emits command-specific text after the status line, for
// unprefixed and kind-prefixed forms alike. Mutation paths run against the
// staged corpus or a skill dir under a temporary directory.
func TestTextCommandsEmitPayloadData(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		contains []string
		setup    func(t *testing.T)
	}{
		{
			name:     "root",
			args:     []string{"--format", "text"},
			contains: []string{"purpose: Manage Architecture Decision Records", "commands:"},
		},
		{
			name:     "combined list",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "list"),
			contains: []string{"count: 3", "ADR-0001: Use SQLite", "SPEC-0001: Storage Requirements", "DM-0001: Decision Record"},
		},
		{
			name:     "adr list",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "adr", "list"),
			contains: []string{"count: 1", "ADR-0001: Use SQLite"},
		},
		{
			name:     "spec list",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "spec", "list"),
			contains: []string{"count: 1", "SPEC-0001: Storage Requirements"},
		},
		{
			name:     "domain list",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "domain", "list"),
			contains: []string{"count: 1", "DM-0001: Decision Record"},
		},
		{
			name:     "combined search",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "search", "--query", "querying"),
			contains: []string{"query: querying", "count: 1", "snippet:"},
		},
		{
			name:     "adr search",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "adr", "search", "--query", "querying"),
			contains: []string{"query: querying", "ADR-0001: Use SQLite"},
		},
		{
			name:     "spec search",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "spec", "search", "--query", "requirements"),
			contains: []string{"query: requirements", "SPEC-0001: Storage Requirements"},
		},
		{
			name:     "domain search",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "domain", "search", "--query", "commitment"),
			contains: []string{"query: commitment", "DM-0001: Decision Record"},
		},
		{
			name:     "show",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "show", "--id", "ADR-0001"),
			contains: []string{"adr:", "id: ADR-0001", "content:"},
		},
		{
			name:     "doctor",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "doctor"),
			contains: []string{"diagnostics:", "adr_directory: ok"},
		},
		{
			name:     "validate",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "validate"),
			contains: []string{"findings:", "summary: files_checked=3 errors=0 warnings=0"},
		},
		{
			name:     "adr validate",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "adr", "validate"),
			contains: []string{"findings:", "summary: files_checked=1 errors=0 warnings=0"},
		},
		{
			name:     "config show",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "config", "show"),
			contains: []string{"source: defaults", "effective configuration:"},
		},
		{
			name:     "config validate",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "config", "validate"),
			contains: []string{"summary: files_checked=0 errors=0 warnings=0", "source: defaults"},
		},
		{
			name:     "adr new dry run",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "adr", "new", "--title", "Tentative Plan", "--dry-run"),
			contains: []string{"plan:", "write_file:", "adr:", "dry_run: true"},
		},
		{
			name:     "accept dry run",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "accept", "--id", "ADR-0001", "--reason", "Approved.", "--dry-run"),
			contains: []string{"accept: planned", "plan:", "update_file:", "dry_run: true"},
		},
		{
			name:     "deprecate dry run",
			args:     append(append([]string{"--format", "text"}, corpusFlagSet...), "deprecate", "--id", "ADR-0001", "--reason", "Retired.", "--dry-run"),
			contains: []string{"deprecate: planned", "plan:", "update_file:"},
		},
		{
			name:     "adr init dry run",
			args:     []string{"--format", "text", "--adr-dir", "t-newstore", "--spec-dir", "t-spec", "--domain-dir", "t-domain", "adr", "init", "--dry-run"},
			contains: []string{"adr init: planned", "plan:", "mkdir: t-newstore"},
		},
		{
			name:     "spec init dry run",
			args:     []string{"--format", "text", "--spec-dir", "t-newspec", "spec", "init", "--dry-run"},
			contains: []string{"spec init: planned", "plan:", "mkdir: t-newspec"},
		},
		{
			name:     "domain init dry run",
			args:     []string{"--format", "text", "--domain-dir", "t-newdomain", "domain", "init", "--dry-run"},
			contains: []string{"domain init: planned", "plan:", "mkdir: t-newdomain"},
		},
		{
			name:     "skill catalog",
			args:     []string{"--format", "text", "skill"},
			contains: []string{"default_skill_dir: .agents/skills", "assets:", "canon-record-gate [skill]"},
		},
		{
			name:     "skill install dry run",
			args:     []string{"--format", "text", "skill", "install", "--skill-dir", "t-skills", "--agent", "claude", "--dry-run"},
			contains: []string{"plan:", "write_file: t-skills/canon/SKILL.md"},
		},
		{
			name: "skill update dry run",
			args: []string{"--format", "text", "skill", "update", "--skill-dir", "t-skills", "--agent", "claude", "--dry-run"},
			setup: func(t *testing.T) {
				code, _ := runRawForTest(t, "skill", "install", "--skill-dir", "t-skills", "--agent", "claude")
				if code != exitOK {
					t.Fatalf("install before update code = %d", code)
				}
			},
			contains: []string{"plan:", "noop: t-skills/canon/SKILL.md"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stageCorpus(t)
			if tt.setup != nil {
				tt.setup(t)
			}
			code, output := runRawForTest(t, tt.args...)
			if code != exitOK {
				t.Fatalf("code = %d, output:\n%s", code, output)
			}
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Fatalf("text output missing %q:\n%s", want, output)
				}
			}
		})
	}
}

// TestUnsupportedContextCommandsKeepEarlyRejection confirms context requests
// for unsupported commands exit 2 with the usage error before any command
// flag parsing, store reads, or mutations run.
func TestUnsupportedContextCommandsKeepEarlyRejection(t *testing.T) {
	stageCorpus(t)
	for _, args := range [][]string{
		{"--format", "context"},
		{"--format", "context", "show", "--id", "ADR-0001"},
		{"--format", "context", "doctor"},
		{"--format", "context", "adr", "new", "--title", "Sneaky"},
	} {
		code, output := runRawForTest(t, args...)
		if code != exitUsage {
			t.Fatalf("args %v: code = %d, output:\n%s", args, code, output)
		}
		if !strings.Contains(output, "unsupported_context_format") {
			t.Fatalf("args %v: expected unsupported_context_format error, got:\n%s", args, output)
		}
	}
	files, err := os.ReadDir("t-adr")
	if err != nil {
		t.Fatalf("read t-adr: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("context rejection created files: %v", files)
	}
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
