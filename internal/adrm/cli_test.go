package adrm

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/victorhsb/adrm/adrmskill"
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
	var sawNew, sawSkillInstall, sawSkillUpdate bool
	for _, raw := range commands {
		command := raw.(map[string]any)
		if command["name"] == "new" {
			sawNew = true
			if command["mutating"] != true || command["has_dry_run"] != true {
				t.Fatalf("new command metadata = %#v", command)
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
	if !sawNew {
		t.Fatal("missing new command")
	}
	if !sawSkillInstall || !sawSkillUpdate {
		t.Fatalf("missing skill install/update commands: install=%v update=%v", sawSkillInstall, sawSkillUpdate)
	}
}

func TestNewDryRunDoesNotWriteFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	code, env := runForTest(t, "--adr-dir", dir, "new", "--title", "Use SQLite", "--dry-run")
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
	code, env := runForTest(t, "--adr-dir", dir, "new", "--title", "Use SQLite", "--tags", "storage, local", "--context", "Need local querying", "--decision", "Use SQLite")
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
	if code, _ := runForTest(t, "--adr-dir", dir, "new", "--title", "Temporary choice"); code != exitOK {
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
	if code, _ := runForTest(t, "--adr-dir", dir, "new", "--title", "Candidate decision"); code != exitOK {
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
	if code, _ := runForTest(t, "--adr-dir", dir, "new", "--title", "Candidate decision"); code != exitOK {
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
	if code, _ := runForTest(t, "--adr-dir", dir, "new", "--title", "Candidate decision"); code != exitOK {
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
	if code, _ := runForTest(t, "--adr-dir", dir, "new", "--title", "Candidate decision"); code != exitOK {
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
	if code, _ := runForTest(t, "--adr-dir", dir, "new", "--title", "Old decision"); code != exitOK {
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

func TestSkillReturnsManagedContent(t *testing.T) {
	code, env := runForTest(t, "skill")
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	data := env["data"].(map[string]any)
	if data["filename"] != adrmskill.FileName {
		t.Fatalf("filename = %v", data["filename"])
	}
	content := data["content"].(string)
	if !strings.Contains(content, "name: adrm") || !strings.Contains(content, "adrm-skill-hash: sha256:") {
		t.Fatalf("content missing skill metadata:\n%s", content)
	}
}

func TestSkillInstallDryRunAndInstall(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "skill")
	target := adrmskill.TargetPath(dir)

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
	if string(content) != adrmskill.Content() {
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
	target := adrmskill.TargetPath(dir)
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
	if string(content) != adrmskill.Content() {
		t.Fatalf("updated content differs from bundled content")
	}
}

func TestSkillUpdateRefusesLocalModificationWithoutForce(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "skill")
	target := adrmskill.TargetPath(dir)
	if code, env := runForTest(t, "skill", "install", "--skill-dir", dir); code != exitOK {
		t.Fatalf("install code=%d env=%#v", code, env)
	}
	if err := os.WriteFile(target, []byte(adrmskill.Content()+"\nlocal edit\n"), 0o644); err != nil {
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

func testManagedSkillContent(body string) string {
	payload := strings.TrimSpace(`---
name: adrm
description: Manage Architecture Decision Records with the adrm CLI in agent-led workflows.
---
<!-- adrm-skill-version: 0 -->

# ADRM Agent Skill

`+body) + "\n"
	sum := sha256.Sum256([]byte(payload))
	hash := "sha256:" + fmt.Sprintf("%x", sum[:])
	return strings.Replace(payload, "<!-- adrm-skill-version: 0 -->\n", "<!-- adrm-skill-version: 0 -->\n<!-- adrm-skill-hash: "+hash+" -->\n", 1)
}
