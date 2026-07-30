package canon

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/victorhsb/canon/skill"
)

func runForTest(t *testing.T, args ...string) (int, map[string]any) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if stderr.Len() > 0 {
		t.Logf("stderr: %s", stderr.String())
	}
	var env map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, stdout.String())
	}
	return code, env
}

func TestCommandsExposeAgentMetadata(t *testing.T) {
	code, env := runForTest(t, "commands")
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	if env["schema_version"] != SchemaVersion {
		t.Fatalf("schema_version = %v", env["schema_version"])
	}
	data := env["data"].(map[string]any)
	commands := data["commands"].([]any)
	if len(commands) == 0 {
		t.Fatal("expected commands")
	}
	var sawNew, sawSpecNew, sawSkillInstall, sawSkillUpdate bool
	for _, raw := range commands {
		command := raw.(map[string]any)
		if command["name"] == "adr new" {
			sawNew = true
			if command["mutating"] != true || command["has_dry_run"] != true {
				t.Fatalf("adr new command metadata = %#v", command)
			}
		}
		if command["name"] == "spec new" {
			sawSpecNew = true
			if command["mutating"] != true || command["has_dry_run"] != true {
				t.Fatalf("spec new command metadata = %#v", command)
			}
		}
		if command["name"] == "skill install" {
			sawSkillInstall = true
			if command["mutating"] != true || command["has_dry_run"] != true {
				t.Fatalf("skill install command metadata = %#v", command)
			}
		}
		if command["name"] == "skill update" {
			sawSkillUpdate = true
			if command["mutating"] != true || command["has_dry_run"] != true {
				t.Fatalf("skill update command metadata = %#v", command)
			}
		}
	}
	if !sawNew || !sawSpecNew {
		t.Fatalf("missing new commands: adr=%v spec=%v", sawNew, sawSpecNew)
	}
	if !sawSkillInstall || !sawSkillUpdate {
		t.Fatalf("missing skill install/update commands: install=%v update=%v", sawSkillInstall, sawSkillUpdate)
	}
}

func TestNewDryRunDoesNotWriteFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	code, env := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Use SQLite", "--dry-run")
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	if env["status"] != "planned" {
		t.Fatalf("status = %v", env["status"])
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("dry-run created directory or unexpected stat error: %v", err)
	}
	warnings := env["warnings"].([]any)
	if warnings[0] != "No changes were made." {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestCreateListSearchAndShow(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	code, env := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Use SQLite", "--tags", "storage, local", "--context", "Need local querying", "--decision", "Use SQLite")
	if code != exitOK {
		t.Fatalf("new code = %d", code)
	}
	adr := env["data"].(map[string]any)["adr"].(map[string]any)
	if adr["id"] != "ADR-0001" {
		t.Fatalf("adr id = %v", adr["id"])
	}

	code, env = runForTest(t, "--adr-dir", dir, "list", "--tag", "storage")
	if code != exitOK {
		t.Fatalf("list code = %d", code)
	}
	if env["data"].(map[string]any)["count"] != float64(1) {
		t.Fatalf("list data = %#v", env["data"])
	}

	code, env = runForTest(t, "--adr-dir", dir, "search", "--query", "querying")
	if code != exitOK {
		t.Fatalf("search code = %d", code)
	}
	if env["data"].(map[string]any)["count"] != float64(1) {
		t.Fatalf("search data = %#v", env["data"])
	}

	code, env = runForTest(t, "--adr-dir", dir, "show", "--id", "1")
	if code != exitOK {
		t.Fatalf("show code = %d", code)
	}
	shown := env["data"].(map[string]any)["adr"].(map[string]any)
	if shown["id"] != "ADR-0001" || shown["content"] == "" {
		t.Fatalf("show adr = %#v", shown)
	}
}

func TestAppendAndDeprecateMutateADR(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Temporary choice"); code != exitOK {
		t.Fatalf("new code = %d", code)
	}
	if code, env := runForTest(t, "--adr-dir", dir, "append", "--id", "ADR-0001", "--title", "Review", "--body", "Still under review.", "--dry-run"); code != exitOK || env["status"] != "planned" {
		t.Fatalf("append dry-run code=%d env=%#v", code, env)
	}
	if code, _ := runForTest(t, "--adr-dir", dir, "append", "--id", "ADR-0001", "--title", "Review", "--body", "Still under review."); code != exitOK {
		t.Fatalf("append code = %d", code)
	}
	if code, _ := runForTest(t, "--adr-dir", dir, "deprecate", "--id", "ADR-0001", "--reason", "No longer relevant."); code != exitOK {
		t.Fatalf("deprecate code = %d", code)
	}
	code, env := runForTest(t, "--adr-dir", dir, "show", "--id", "ADR-0001")
	if code != exitOK {
		t.Fatalf("show code = %d", code)
	}
	adr := env["data"].(map[string]any)["adr"].(map[string]any)
	if adr["status"] != "deprecated" {
		t.Fatalf("status = %v", adr["status"])
	}
	content := adr["content"].(string)
	if !strings.Contains(content, "## Appendix: Review") || !strings.Contains(content, "## History: Deprecated") {
		t.Fatalf("content missing appendix/history:\n%s", content)
	}
}

func TestAcceptMutatesADR(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Candidate decision"); code != exitOK {
		t.Fatalf("new code = %d", code)
	}
	if code, env := runForTest(t, "--adr-dir", dir, "accept", "--id", "ADR-0001", "--reason", "Approved by the team.", "--dry-run"); code != exitOK || env["status"] != "planned" {
		t.Fatalf("accept dry-run code=%d env=%#v", code, env)
	}
	if code, _ := runForTest(t, "--adr-dir", dir, "accept", "--id", "ADR-0001", "--reason", "Approved by the team."); code != exitOK {
		t.Fatalf("accept code = %d", code)
	}
	code, env := runForTest(t, "--adr-dir", dir, "show", "--id", "ADR-0001")
	if code != exitOK {
		t.Fatalf("show code = %d", code)
	}
	adr := env["data"].(map[string]any)["adr"].(map[string]any)
	if adr["status"] != "accepted" {
		t.Fatalf("status = %v", adr["status"])
	}
	content := adr["content"].(string)
	if !strings.Contains(content, "## History: Accepted") || !strings.Contains(content, "Approved by the team.") {
		t.Fatalf("content missing accepted history:\n%s", content)
	}
}

func TestRejectMutatesADR(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Candidate decision"); code != exitOK {
		t.Fatalf("new code = %d", code)
	}
	if code, env := runForTest(t, "--adr-dir", dir, "reject", "--id", "ADR-0001", "--reason", "Chose a different approach.", "--dry-run"); code != exitOK || env["status"] != "planned" {
		t.Fatalf("reject dry-run code=%d env=%#v", code, env)
	}
	if code, _ := runForTest(t, "--adr-dir", dir, "reject", "--id", "ADR-0001", "--reason", "Chose a different approach."); code != exitOK {
		t.Fatalf("reject code = %d", code)
	}
	code, env := runForTest(t, "--adr-dir", dir, "show", "--id", "ADR-0001")
	if code != exitOK {
		t.Fatalf("show code = %d", code)
	}
	adr := env["data"].(map[string]any)["adr"].(map[string]any)
	if adr["status"] != "rejected" {
		t.Fatalf("status = %v", adr["status"])
	}
	content := adr["content"].(string)
	if !strings.Contains(content, "## History: Rejected") || !strings.Contains(content, "Chose a different approach.") {
		t.Fatalf("content missing rejected history:\n%s", content)
	}
}

func TestAcceptMissingID(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Candidate decision"); code != exitOK {
		t.Fatalf("new code = %d", code)
	}
	code, env := runForTest(t, "--adr-dir", dir, "accept")
	if code != exitUsage {
		t.Fatalf("code = %d env=%#v", code, env)
	}
	errData := env["error"].(map[string]any)
	if errData["code"] != "missing_id" {
		t.Fatalf("error = %#v", errData)
	}
}

func TestRejectMissingID(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Candidate decision"); code != exitOK {
		t.Fatalf("new code = %d", code)
	}
	code, env := runForTest(t, "--adr-dir", dir, "reject")
	if code != exitUsage {
		t.Fatalf("code = %d env=%#v", code, env)
	}
	errData := env["error"].(map[string]any)
	if errData["code"] != "missing_id" {
		t.Fatalf("error = %#v", errData)
	}
}

func TestSupersedeRequiresReplacementADR(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Old decision"); code != exitOK {
		t.Fatalf("new code = %d", code)
	}
	code, env := runForTest(t, "--adr-dir", dir, "supersede", "--id", "ADR-0001", "--by", "ADR-0002", "--dry-run")
	if code != exitNotFound {
		t.Fatalf("code = %d env=%#v", code, env)
	}
	errData := env["error"].(map[string]any)
	if errData["code"] != "superseding_adr_not_found" {
		t.Fatalf("error = %#v", errData)
	}
}

func TestSupersedeUpdatesBothADRs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Old decision"); code != exitOK {
		t.Fatalf("new code = %d", code)
	}
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "New decision"); code != exitOK {
		t.Fatalf("new code = %d", code)
	}

	// Dry-run previews two operations and writes nothing.
	code, env := runForTest(t, "--adr-dir", dir, "supersede", "--id", "ADR-0001", "--by", "ADR-0002", "--reason", "Replaced by current design.", "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("dry-run code=%d env=%#v", code, env)
	}
	plan := env["data"].(map[string]any)["plan"].(map[string]any)
	ops := plan["operations"].([]any)
	if len(ops) != 2 {
		t.Fatalf("expected 2 operations, got %d: %#v", len(ops), ops)
	}

	if code, _ := runForTest(t, "--adr-dir", dir, "supersede", "--id", "ADR-0001", "--by", "ADR-0002", "--reason", "Replaced by current design."); code != exitOK {
		t.Fatalf("supersede code = %d", code)
	}

	code, env = runForTest(t, "--adr-dir", dir, "show", "--id", "ADR-0001")
	if code != exitOK {
		t.Fatalf("show old code = %d", code)
	}
	old := env["data"].(map[string]any)["adr"].(map[string]any)
	if old["status"] != "superseded" || old["superseded_by"] != "ADR-0002" {
		t.Fatalf("old adr = %#v", old)
	}
	oldContent := old["content"].(string)
	if !strings.Contains(oldContent, "## History: Superseded") || !strings.Contains(oldContent, "Replaced by current design.") {
		t.Fatalf("old adr missing history:\n%s", oldContent)
	}

	code, env = runForTest(t, "--adr-dir", dir, "show", "--id", "ADR-0002")
	if code != exitOK {
		t.Fatalf("show new code = %d", code)
	}
	replacement := env["data"].(map[string]any)["adr"].(map[string]any)
	var sawOldID bool
	for _, raw := range replacement["supersedes"].([]any) {
		if raw == "ADR-0001" {
			sawOldID = true
		}
	}
	if !sawOldID {
		t.Fatalf("replacement adr missing ADR-0001 in supersedes: %#v", replacement)
	}

	// Second supersede is idempotent.
	if code, _ := runForTest(t, "--adr-dir", dir, "supersede", "--id", "ADR-0001", "--by", "ADR-0002"); code != exitOK {
		t.Fatalf("repeat supersede code = %d", code)
	}
	code, env = runForTest(t, "--adr-dir", dir, "show", "--id", "ADR-0002")
	if code != exitOK {
		t.Fatalf("show new after repeat code = %d", code)
	}
	replacement = env["data"].(map[string]any)["adr"].(map[string]any)
	if len(replacement["supersedes"].([]any)) != 1 {
		t.Fatalf("supersedes list not deduped: %#v", replacement["supersedes"])
	}
}

func TestSkillReturnsManagedContent(t *testing.T) {
	code, env := runForTest(t, "skill")
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	data := env["data"].(map[string]any)
	if data["filename"] != skill.FileName {
		t.Fatalf("filename = %v", data["filename"])
	}
	content := data["content"].(string)
	if !strings.Contains(content, "name: canon") || !strings.Contains(content, "canon-skill-hash: sha256:") {
		t.Fatalf("content missing skill metadata:\n%s", content)
	}
}

func TestSkillInstallDryRunAndInstall(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "skill")
	target := skill.TargetPath(dir)

	code, env := runForTest(t, "skill", "install", "--skill-dir", dir, "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("dry-run code=%d env=%#v", code, env)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote target or unexpected stat error: %v", err)
	}
	warnings := env["warnings"].([]any)
	if warnings[0] != "No changes were made." {
		t.Fatalf("warnings = %#v", warnings)
	}

	code, env = runForTest(t, "skill", "install", "--skill-dir", dir)
	if code != exitOK {
		t.Fatalf("install code=%d env=%#v", code, env)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read installed skill: %v", err)
	}
	if string(content) != skill.Content() {
		t.Fatalf("installed content differs from bundled content")
	}

	code, env = runForTest(t, "skill", "install", "--skill-dir", dir)
	if code != exitState {
		t.Fatalf("repeat install code=%d env=%#v", code, env)
	}
	errData := env["error"].(map[string]any)
	if errData["code"] != "skill_already_installed" {
		t.Fatalf("error = %#v", errData)
	}
}

func TestSkillUpdateCurrentNoops(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "skill")
	if code, env := runForTest(t, "skill", "install", "--skill-dir", dir); code != exitOK {
		t.Fatalf("install code=%d env=%#v", code, env)
	}
	code, env := runForTest(t, "skill", "update", "--skill-dir", dir, "--dry-run")
	if code != exitOK {
		t.Fatalf("update code=%d env=%#v", code, env)
	}
	if env["status"] != "ok" {
		t.Fatalf("status = %v", env["status"])
	}
	plan := env["data"].(map[string]any)["plan"].(map[string]any)
	operations := plan["operations"].([]any)
	if operations[0].(map[string]any)["action"] != "noop" {
		t.Fatalf("operations = %#v", operations)
	}
}

func TestSkillUpdateManagedOlderVersion(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "skill")
	target := skill.TargetPath(dir)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(target, []byte(testManagedSkillContent("older instructions")), 0o644); err != nil {
		t.Fatalf("write old skill: %v", err)
	}

	code, env := runForTest(t, "skill", "update", "--skill-dir", dir, "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("dry-run code=%d env=%#v", code, env)
	}
	if content, _ := os.ReadFile(target); strings.Contains(string(content), "Create an ADR:") {
		t.Fatalf("dry-run updated target")
	}

	code, env = runForTest(t, "skill", "update", "--skill-dir", dir)
	if code != exitOK {
		t.Fatalf("update code=%d env=%#v", code, env)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read updated skill: %v", err)
	}
	if string(content) != skill.Content() {
		t.Fatalf("updated content differs from bundled content")
	}
}

func TestSkillUpdateRefusesLocalModificationWithoutForce(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "skill")
	target := skill.TargetPath(dir)
	if code, env := runForTest(t, "skill", "install", "--skill-dir", dir); code != exitOK {
		t.Fatalf("install code=%d env=%#v", code, env)
	}
	if err := os.WriteFile(target, []byte(skill.Content()+"\nlocal edit\n"), 0o644); err != nil {
		t.Fatalf("write local edit: %v", err)
	}

	code, env := runForTest(t, "skill", "update", "--skill-dir", dir, "--dry-run")
	if code != exitState {
		t.Fatalf("update code=%d env=%#v", code, env)
	}
	errData := env["error"].(map[string]any)
	if errData["code"] != "local_skill_modified" {
		t.Fatalf("error = %#v", errData)
	}

	code, env = runForTest(t, "skill", "update", "--skill-dir", dir, "--force", "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("force dry-run code=%d env=%#v", code, env)
	}
}

func TestHumanReadableShorthandProducesText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"-t", "commands"}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	output := stdout.String()
	if strings.HasPrefix(output, "{") || strings.Contains(output, "\"schema_version\"") {
		t.Fatalf("expected text output with -t, got JSON:\n%s", output)
	}
	if !strings.Contains(output, "commands:") || !strings.Contains(output, "global flags:") {
		t.Fatalf("expected text-rendered commands output:\n%s", output)
	}
	if !strings.Contains(output, "-t") {
		t.Fatalf("expected -t to appear in global flags listing:\n%s", output)
	}
}

func TestHumanReadableShorthandOverridesJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--format", "json", "-t", "commands"}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	if strings.HasPrefix(stdout.String(), "{") {
		t.Fatalf("expected -t to override --format json, got JSON:\n%s", stdout.String())
	}
}

func TestCommandsExposeHumanReadableFlag(t *testing.T) {
	_, env := runForTest(t, "commands")
	data := env["data"].(map[string]any)
	flags := data["global_flags"].([]any)
	var sawHumanReadable bool
	for _, raw := range flags {
		flag := raw.(map[string]any)
		if flag["name"] == "-t" {
			sawHumanReadable = true
			if flag["purpose"] == "" {
				t.Fatalf("-t flag missing purpose: %#v", flag)
			}
		}
	}
	if !sawHumanReadable {
		t.Fatalf("global flags missing -t: %#v", flags)
	}
}

func testManagedSkillContent(body string) string {
	payload := strings.TrimSpace(`---
name: canon
description: Manage Architecture Decision Records with the canon CLI in agent-led workflows.
---
<!-- canon-skill-version: 0 -->

# CANON Agent Skill

`+body) + "\n"
	sum := sha256.Sum256([]byte(payload))
	hash := "sha256:" + fmt.Sprintf("%x", sum[:])
	return strings.Replace(payload, "<!-- canon-skill-version: 0 -->\n", "<!-- canon-skill-version: 0 -->\n<!-- canon-skill-hash: "+hash+" -->\n", 1)
}

func runKindTest(t *testing.T, args ...string) (int, map[string]any) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if stderr.Len() > 0 {
		t.Logf("stderr: %s", stderr.String())
	}
	var env map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, stdout.String())
	}
	return code, env
}

func TestNewSpecDryRunDoesNotWriteFile(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	code, env := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "spec", "new", "--title", "Local query index", "--dry-run")
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	if env["status"] != "planned" {
		t.Fatalf("status = %v", env["status"])
	}
	if _, err := os.Stat(spec); !os.IsNotExist(err) {
		t.Fatalf("dry-run created spec directory: %v", err)
	}
	data := env["data"].(map[string]any)
	created := data["adr"].(map[string]any)
	if created["kind"] != "spec" || created["id"] != "SPEC-0001" {
		t.Fatalf("spec preview = %#v", created)
	}
}

func TestCreateListSearchAndShowSpec(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	code, env := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "spec", "new", "--title", "Local query index", "--tags", "storage, query", "--context", "Agents need local lookup.", "--requirements", "Return ADRs by tag.", "--constraints", "No external deps.", "--acceptance", "list --tag storage works.")
	if code != exitOK {
		t.Fatalf("new spec code = %d", code)
	}
	created := env["data"].(map[string]any)["adr"].(map[string]any)
	if created["id"] != "SPEC-0001" || created["kind"] != "spec" {
		t.Fatalf("created spec = %#v", created)
	}

	code, env = runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "spec", "list", "--tag", "storage")
	if code != exitOK {
		t.Fatalf("list code = %d", code)
	}
	if env["data"].(map[string]any)["count"] != float64(1) {
		t.Fatalf("list data = %#v", env["data"])
	}

	code, env = runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "search", "--query", "requirements")
	if code != exitOK {
		t.Fatalf("search code = %d", code)
	}
	if env["data"].(map[string]any)["count"] != float64(1) {
		t.Fatalf("search data = %#v", env["data"])
	}

	code, env = runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "show", "--id", "SPEC-0001")
	if code != exitOK {
		t.Fatalf("show code = %d", code)
	}
	shown := env["data"].(map[string]any)["adr"].(map[string]any)
	if shown["kind"] != "spec" || shown["content"] == "" {
		t.Fatalf("show spec = %#v", shown)
	}
	content := shown["content"].(string)
	for _, section := range []string{"## Requirements", "## Constraints", "## Acceptance Criteria"} {
		if !strings.Contains(content, section) {
			t.Fatalf("spec content missing %s:\n%s", section, content)
		}
	}
}

func TestADRAndSPECNumberIndependently(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	if code, _ := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "adr", "new", "--title", "First ADR"); code != exitOK {
		t.Fatalf("adr new code = %d", code)
	}
	if code, _ := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "spec", "new", "--title", "First SPEC"); code != exitOK {
		t.Fatalf("spec new code = %d", code)
	}
	if code, _ := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "adr", "new", "--title", "Second ADR"); code != exitOK {
		t.Fatalf("second adr code = %d", code)
	}
	code, env := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "list")
	if code != exitOK {
		t.Fatalf("list code = %d", code)
	}
	data := env["data"].(map[string]any)
	if data["count"] != float64(3) {
		t.Fatalf("count = %v", data["count"])
	}
	adrs := data["adrs"].([]any)
	ids := []string{}
	for _, raw := range adrs {
		ids = append(ids, raw.(map[string]any)["id"].(string))
	}
	if ids[0] != "ADR-0001" || ids[1] != "ADR-0002" || ids[2] != "SPEC-0001" {
		t.Fatalf("ordered ids = %v", ids)
	}
}

func TestSupersedeRejectsCrossKindReplacement(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	if code, _ := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "adr", "new", "--title", "Old ADR"); code != exitOK {
		t.Fatalf("adr new code = %d", code)
	}
	if code, _ := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "spec", "new", "--title", "Replacement SPEC"); code != exitOK {
		t.Fatalf("spec new code = %d", code)
	}
	code, env := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "supersede", "--id", "ADR-0001", "--by", "SPEC-0001", "--dry-run")
	if code != exitState {
		t.Fatalf("cross-kind supersede code = %d env = %#v", code, env)
	}
	errData := env["error"].(map[string]any)
	if errData["code"] != "cross_kind_supersede" {
		t.Fatalf("error = %#v", errData)
	}
}

func TestAcceptMutatesSpec(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	if code, _ := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "spec", "new", "--title", "Candidate spec"); code != exitOK {
		t.Fatalf("spec new code = %d", code)
	}
	if code, env := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "accept", "--id", "SPEC-0001", "--reason", "Approved.", "--dry-run"); code != exitOK || env["status"] != "planned" {
		t.Fatalf("accept dry-run code=%d env=%#v", code, env)
	}
	if code, _ := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "accept", "--id", "SPEC-0001", "--reason", "Approved."); code != exitOK {
		t.Fatalf("accept code = %d", code)
	}
	code, env := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "show", "--id", "SPEC-0001")
	if code != exitOK {
		t.Fatalf("show code = %d", code)
	}
	shown := env["data"].(map[string]any)["adr"].(map[string]any)
	if shown["status"] != "accepted" {
		t.Fatalf("status = %v", shown["status"])
	}
	content := shown["content"].(string)
	if !strings.Contains(content, "## History: Accepted") || !strings.Contains(content, "Approved.") {
		t.Fatalf("spec content missing accepted history:\n%s", content)
	}
}

func TestDoctorReportsMissingSpecDirectory(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	if code, _ := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "adr", "init"); code != exitOK {
		t.Fatalf("init code = %d", code)
	}
	code, env := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, "doctor")
	if code != exitOK {
		t.Fatalf("doctor code = %d", code)
	}
	if env["status"] != "warning" {
		t.Fatalf("doctor status = %v", env["status"])
	}
	var sawSpecWarning bool
	for _, raw := range env["data"].(map[string]any)["diagnostics"].([]any) {
		d := raw.(map[string]any)
		if d["name"] == "spec_directory" && d["status"] == "warning" {
			sawSpecWarning = true
		}
	}
	if !sawSpecWarning {
		t.Fatalf("doctor missing spec_directory warning: %#v", env["data"])
	}
}

func TestNewAndInitRequireKindPrefix(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	for _, command := range []string{"new", "init"} {
		code, env := runKindTest(t, "--adr-dir", adr, "--spec-dir", spec, command)
		if code != exitUsage {
			t.Fatalf("%s code = %d env = %#v", command, code, env)
		}
		if env["error"].(map[string]any)["code"] != "kind_prefix_required" {
			t.Fatalf("%s error = %#v", command, env["error"])
		}
	}
}

func TestKindPrefixRequiresKnownSubcommand(t *testing.T) {
	code, env := runForTest(t, "adr")
	if code != exitUsage || env["error"].(map[string]any)["code"] != "missing_kind_subcommand" {
		t.Fatalf("bare adr code=%d env=%#v", code, env)
	}
	code, env = runForTest(t, "spec", "show", "--id", "SPEC-0001")
	if code != exitUsage || env["error"].(map[string]any)["code"] != "unknown_command" {
		t.Fatalf("spec show code=%d env=%#v", code, env)
	}
}
