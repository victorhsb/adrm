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

func runRawForTest(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if stderr.Len() > 0 {
		t.Logf("stderr: %s", stderr.String())
	}
	return code, stdout.String()
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
	var sawNew, sawSpecNew, sawDomainNew, sawSkillInstall, sawSkillUpdate bool
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
		if command["name"] == "domain new" {
			sawDomainNew = true
			if command["mutating"] != true || command["has_dry_run"] != true {
				t.Fatalf("domain new command metadata = %#v", command)
			}
		}
		if command["name"] == "skill install" {
			sawSkillInstall = true
			if command["mutating"] != true || command["has_dry_run"] != true {
				t.Fatalf("skill install command metadata = %#v", command)
			}
			for _, selector := range []string{"--skill-dir", "--only", "--agent"} {
				if !commandHasSelector(command, selector) {
					t.Fatalf("skill install missing selector %s: %#v", selector, command)
				}
			}
		}
		if command["name"] == "skill update" {
			sawSkillUpdate = true
			if command["mutating"] != true || command["has_dry_run"] != true {
				t.Fatalf("skill update command metadata = %#v", command)
			}
			for _, selector := range []string{"--skill-dir", "--only", "--agent", "--force"} {
				if !commandHasSelector(command, selector) {
					t.Fatalf("skill update missing selector %s: %#v", selector, command)
				}
			}
		}
	}
	if !sawNew || !sawSpecNew || !sawDomainNew {
		t.Fatalf("missing new commands: adr=%v spec=%v domain=%v", sawNew, sawSpecNew, sawDomainNew)
	}
	if !sawSkillInstall || !sawSkillUpdate {
		t.Fatalf("missing skill install/update commands: install=%v update=%v", sawSkillInstall, sawSkillUpdate)
	}
}

func TestVersionCommand(t *testing.T) {
	code, env := runForTest(t, "version")
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	if env["schema_version"] != SchemaVersion {
		t.Fatalf("schema_version = %v", env["schema_version"])
	}
	data := env["data"].(map[string]any)
	if data["version"] != Version {
		t.Fatalf("version = %v, want %q", data["version"], Version)
	}
	code, text := runRawForTest(t, "--format", "text", "version")
	if code != exitOK {
		t.Fatalf("text code = %d", code)
	}
	if !strings.Contains(text, "version: "+Version) {
		t.Fatalf("text output missing version: %q", text)
	}
	if strings.Contains(text, "schema_version") {
		t.Fatalf("text output contaminated with schema_version: %q", text)
	}
}

func commandHasSelector(command map[string]any, want string) bool {
	selectors, ok := command["selectors"].([]any)
	if !ok {
		return false
	}
	for _, selector := range selectors {
		if selector == want {
			return true
		}
	}
	return false
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

func TestSkillReturnsAssetCatalog(t *testing.T) {
	code, env := runForTest(t, "skill")
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	data := env["data"].(map[string]any)
	if data["default_skill_dir"] != skill.DefaultInstallDir {
		t.Fatalf("default_skill_dir = %v", data["default_skill_dir"])
	}
	assets := data["assets"].([]any)
	if len(assets) != 2 {
		t.Fatalf("asset count = %d", len(assets))
	}
	for i, wantName := range []string{"canon", "canon-record-gate"} {
		asset := assets[i].(map[string]any)
		if asset["name"] != wantName || asset["kind"] != skill.KindSkill {
			t.Fatalf("asset %d = %#v", i, asset)
		}
		if asset["version"] == "" || !strings.HasPrefix(asset["hash"].(string), "sha256:") {
			t.Fatalf("asset missing version/hash: %#v", asset)
		}
		if len(asset["target_paths"].([]any)) == 0 {
			t.Fatalf("asset missing target paths: %#v", asset)
		}
		if wantName == "canon-record-gate" {
			var foundCodex bool
			for _, rawPath := range asset["target_paths"].([]any) {
				if rawPath == filepath.Join(".codex", "agents", "canon-critic.toml") {
					foundCodex = true
					break
				}
			}
			if !foundCodex {
				t.Fatalf("record-gate catalog missing Codex agent path: %#v", asset)
			}
		}
	}
	if _, ok := data["content"]; ok {
		t.Fatal("catalog unexpectedly includes embedded content")
	}
}

func TestSkillInstallDryRunFallbackAndInstall(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)

	code, env := runForTest(t, "skill", "install", "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("dry-run code=%d env=%#v", code, env)
	}
	plan := envelopePlan(env)
	operations := plan["operations"].([]any)
	if len(operations) != 4 {
		t.Fatalf("operation count = %d, want 4: %#v", len(operations), operations)
	}
	assertPlanPaths(t, operations, []string{
		filepath.Join(".agents", "skills", "canon-record-gate", "SKILL.md"),
		filepath.Join(".agents", "skills", "canon-record-gate", "references", "boundary-examples.md"),
		filepath.Join(".agents", "skills", "canon", "SKILL.md"),
		filepath.Join(".opencode", "agents", "canon-critic.md"),
	})
	if plan["dry_run"] != true {
		t.Fatalf("dry_run = %v", plan["dry_run"])
	}
	assertNoChangesWarning(t, env)
	if _, err := os.Stat(".agents"); !os.IsNotExist(err) {
		t.Fatalf("dry-run created .agents: %v", err)
	}
	if _, err := os.Stat(".opencode"); !os.IsNotExist(err) {
		t.Fatalf("dry-run created .opencode: %v", err)
	}

	code, env = runForTest(t, "skill", "install")
	if code != exitOK {
		t.Fatalf("install code=%d env=%#v", code, env)
	}
	assertInstalledFiles(t, "", []string{skill.TargetOpenCode})

	code, env = runForTest(t, "skill", "install")
	if code != exitState {
		t.Fatalf("repeat install code=%d env=%#v", code, env)
	}
	if env["error"].(map[string]any)["code"] != "skill_already_installed" {
		t.Fatalf("error = %#v", env["error"])
	}
}

func TestSkillInstallOnlyCanon(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)

	code, env := runForTest(t, "skill", "install", "--only", "canon")
	if code != exitOK {
		t.Fatalf("install code=%d env=%#v", code, env)
	}
	operations := envelopePlan(env)["operations"].([]any)
	assertPlanPaths(t, operations, []string{filepath.Join(".agents", "skills", "canon", "SKILL.md")})
	if _, err := os.Stat(filepath.Join(".agents", "skills", "canon", "SKILL.md")); err != nil {
		t.Fatalf("canon skill not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(".agents", "skills", "canon-record-gate")); !os.IsNotExist(err) {
		t.Fatalf("record gate unexpectedly installed: %v", err)
	}
	if _, err := os.Stat(".opencode"); !os.IsNotExist(err) {
		t.Fatalf("agent unexpectedly installed: %v", err)
	}
}

func TestSkillInstallClaudeTarget(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)

	code, env := runForTest(t, "skill", "install", "--agent", "claude")
	if code != exitOK {
		t.Fatalf("install code=%d env=%#v", code, env)
	}
	agentPath := filepath.Join(".claude", "agents", "canon-critic.md")
	content, err := os.ReadFile(agentPath)
	if err != nil {
		t.Fatalf("read claude agent: %v", err)
	}
	for _, want := range []string{"tools: Read, Grep, Glob, Bash", "model: inherit", "canon-skill-version: 3"} {
		if !strings.Contains(string(content), want) {
			t.Fatalf("claude agent missing %q:\n%s", want, content)
		}
	}
	if _, err := os.Stat(".opencode"); !os.IsNotExist(err) {
		t.Fatalf("opencode target unexpectedly created: %v", err)
	}
	targets := env["data"].(map[string]any)["targets"].([]any)
	if len(targets) != 1 || targets[0] != skill.TargetClaude {
		t.Fatalf("targets = %#v", targets)
	}
}

func TestSkillInstallInfersTargets(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)
	if err := os.Mkdir(".claude", 0o755); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}
	if err := os.Mkdir(".codex", 0o755); err != nil {
		t.Fatalf("mkdir .codex: %v", err)
	}

	code, env := runForTest(t, "skill", "install", "--dry-run")
	if code != exitOK {
		t.Fatalf("dry-run code=%d env=%#v", code, env)
	}
	operations := envelopePlan(env)["operations"].([]any)
	assertPlanPaths(t, operations, []string{
		filepath.Join(".agents", "skills", "canon-record-gate", "SKILL.md"),
		filepath.Join(".agents", "skills", "canon-record-gate", "references", "boundary-examples.md"),
		filepath.Join(".agents", "skills", "canon", "SKILL.md"),
		filepath.Join(".claude", "agents", "canon-critic.md"),
		filepath.Join(".codex", "agents", "canon-critic.toml"),
	})
}

func TestSkillInstallRepeatedTargetsAndCodex(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)

	code, env := runForTest(t, "skill", "install", "--agent", "claude", "--agent", "opencode", "--agent", "claude", "--dry-run")
	if code != exitOK {
		t.Fatalf("dry-run code=%d env=%#v", code, env)
	}
	operations := envelopePlan(env)["operations"].([]any)
	if len(operations) != 5 {
		t.Fatalf("operation count = %d, want 5", len(operations))
	}
	targets := env["data"].(map[string]any)["assets"].([]any)[1].(map[string]any)["target_paths"].([]any)
	if len(targets) != 4 {
		t.Fatalf("selected record-gate target paths = %#v", targets)
	}

	project = t.TempDir()
	t.Chdir(project)
	code, env = runForTest(t, "skill", "install", "--agent", "codex", "--dry-run")
	if code != exitOK {
		t.Fatalf("codex dry-run code=%d env=%#v", code, env)
	}
	operations = envelopePlan(env)["operations"].([]any)
	assertPlanPaths(t, operations, []string{
		filepath.Join(".agents", "skills", "canon-record-gate", "SKILL.md"),
		filepath.Join(".agents", "skills", "canon-record-gate", "references", "boundary-examples.md"),
		filepath.Join(".agents", "skills", "canon", "SKILL.md"),
		filepath.Join(".codex", "agents", "canon-critic.toml"),
	})

	code, env = runForTest(t, "skill", "install", "--agent", "codex")
	if code != exitOK {
		t.Fatalf("codex install code=%d env=%#v", code, env)
	}
	codexPath := filepath.Join(".codex", "agents", "canon-critic.toml")
	content, err := os.ReadFile(codexPath)
	if err != nil {
		t.Fatalf("read codex agent: %v", err)
	}
	for _, want := range []string{
		`name = "canon-critic"`,
		`sandbox_mode = "read-only"`,
		`developer_instructions = "You are a canon corpus critic:`,
		`# canon-skill-version: 3`,
		`# canon-skill-hash: sha256:`,
		"canon-record-gate",
	} {
		if !strings.Contains(string(content), want) {
			t.Fatalf("codex agent missing %q:\n%s", want, content)
		}
	}
	for _, path := range []string{filepath.Join(".claude", "agents", "canon-critic.md"), filepath.Join(".opencode", "agents", "canon-critic.md")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("unselected agent target %s exists: %v", path, err)
		}
	}
}

func TestSkillInstallInfersCodexTarget(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)
	if err := os.Mkdir(".codex", 0o755); err != nil {
		t.Fatalf("mkdir .codex: %v", err)
	}

	code, env := runForTest(t, "skill", "install", "--dry-run")
	if code != exitOK {
		t.Fatalf("dry-run code=%d env=%#v", code, env)
	}
	operations := envelopePlan(env)["operations"].([]any)
	assertPlanPaths(t, operations, []string{
		filepath.Join(".agents", "skills", "canon-record-gate", "SKILL.md"),
		filepath.Join(".agents", "skills", "canon-record-gate", "references", "boundary-examples.md"),
		filepath.Join(".agents", "skills", "canon", "SKILL.md"),
		filepath.Join(".codex", "agents", "canon-critic.toml"),
	})
}

func TestSkillRejectsInvalidAssetAndTarget(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)

	code, env := runForTest(t, "skill", "install", "--only", "canon-critic", "--dry-run")
	if code != exitUsage || env["error"].(map[string]any)["category"] != "usage" {
		t.Fatalf("invalid asset code=%d env=%#v", code, env)
	}
	code, env = runForTest(t, "skill", "install", "--agent", "other", "--dry-run")
	if code != exitUsage || env["error"].(map[string]any)["category"] != "usage" {
		t.Fatalf("invalid target code=%d env=%#v", code, env)
	}
}

func TestSkillInstallCollisionPreflightWritesNothing(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)
	conflict := filepath.Join(".agents", "skills", "canon-record-gate", "references", "boundary-examples.md")
	if err := os.MkdirAll(filepath.Dir(conflict), 0o755); err != nil {
		t.Fatalf("mkdir conflict: %v", err)
	}
	if err := os.WriteFile(conflict, []byte("local"), 0o644); err != nil {
		t.Fatalf("write conflict: %v", err)
	}

	code, env := runForTest(t, "skill", "install")
	if code != exitState {
		t.Fatalf("install code=%d env=%#v", code, env)
	}
	if env["error"].(map[string]any)["code"] != "skill_already_installed" {
		t.Fatalf("error = %#v", env["error"])
	}
	if _, err := os.Stat(filepath.Join(".agents", "skills", "canon", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("preflight still wrote canon skill: %v", err)
	}
	if _, err := os.Stat(".opencode"); !os.IsNotExist(err) {
		t.Fatalf("preflight still wrote agent: %v", err)
	}
}

func TestSkillInstallPreflightsDirectoryCreation(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-based preflight test is not meaningful as root")
	}
	project := t.TempDir()
	t.Chdir(project)
	if err := os.Mkdir(".claude", 0o500); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}

	code, env := runForTest(t, "skill", "install", "--agent", "claude")
	if code != exitIO {
		t.Fatalf("install code=%d env=%#v", code, env)
	}
	if env["error"].(map[string]any)["code"] != "skill_directory_create_failed" {
		t.Fatalf("error = %#v", env["error"])
	}
	if _, err := os.Stat(filepath.Join(".agents", "skills", "canon", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("preflight still wrote skill files: %v", err)
	}
}

func TestSkillUpdateCurrentNoops(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)
	if code, env := runForTest(t, "skill", "install"); code != exitOK {
		t.Fatalf("install code=%d env=%#v", code, env)
	}

	code, env := runForTest(t, "skill", "update", "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("update dry-run code=%d env=%#v", code, env)
	}
	plan := envelopePlan(env)
	if plan["dry_run"] != true {
		t.Fatalf("dry_run = %v", plan["dry_run"])
	}
	assertNoChangesWarning(t, env)
	for _, raw := range plan["operations"].([]any) {
		if raw.(map[string]any)["action"] != "noop" {
			t.Fatalf("operation is not noop: %#v", raw)
		}
	}

	code, env = runForTest(t, "skill", "update")
	if code != exitOK || env["status"] != "ok" {
		t.Fatalf("update code=%d env=%#v", code, env)
	}
	plan = envelopePlan(env)
	if plan["dry_run"] != false || plan["changes_made"] != false {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestSkillUpdateManagedOlderVersion(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)
	target := filepath.Join(".agents", "skills", "canon", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(target, []byte(testManagedSkillContent("older instructions")), 0o644); err != nil {
		t.Fatalf("write old skill: %v", err)
	}

	code, env := runForTest(t, "skill", "update", "--only", "canon", "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("dry-run code=%d env=%#v", code, env)
	}
	operation := envelopePlan(env)["operations"].([]any)[0].(map[string]any)
	if operation["action"] != "update_file" {
		t.Fatalf("operation = %#v", operation)
	}
	if content, _ := os.ReadFile(target); !strings.Contains(string(content), "older instructions") {
		t.Fatal("dry-run updated target")
	}

	code, env = runForTest(t, "skill", "update", "--only", "canon")
	if code != exitOK {
		t.Fatalf("update code=%d env=%#v", code, env)
	}
	desired := desiredSkillFilesForTest(t, []string{"canon"}, "")[0]
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read updated skill: %v", err)
	}
	if string(content) != desired.Content() {
		t.Fatal("updated content differs from bundled content")
	}
}

func TestSkillUpdatePreflightsReadOnlyManagedFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-based preflight test is not meaningful as root")
	}
	project := t.TempDir()
	t.Chdir(project)
	target := filepath.Join(".agents", "skills", "canon", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	oldContent := testManagedSkillContent("older instructions")
	if err := os.WriteFile(target, []byte(oldContent), 0o444); err != nil {
		t.Fatalf("write old skill: %v", err)
	}

	code, env := runForTest(t, "skill", "update", "--only", "canon")
	if code != exitIO {
		t.Fatalf("update code=%d env=%#v", code, env)
	}
	if env["error"].(map[string]any)["code"] != "skill_write_failed" {
		t.Fatalf("error = %#v", env["error"])
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read old skill: %v", err)
	}
	if string(content) != oldContent {
		t.Fatal("preflight failure still modified the target")
	}
}

func TestSkillUpdateAddsMissingBundleFiles(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)
	if code, env := runForTest(t, "skill", "install", "--only", "canon"); code != exitOK {
		t.Fatalf("install canon code=%d env=%#v", code, env)
	}

	code, env := runForTest(t, "skill", "update", "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("update dry-run code=%d env=%#v", code, env)
	}
	operations := envelopePlan(env)["operations"].([]any)
	if len(operations) != 4 {
		t.Fatalf("operation count = %d, want 4", len(operations))
	}
	for _, raw := range operations[:2] {
		if raw.(map[string]any)["action"] != "write_file" {
			t.Fatalf("missing record-gate operation = %#v", raw)
		}
	}
	if operations[2].(map[string]any)["action"] != "noop" {
		t.Fatalf("existing canon operation = %#v", operations[2])
	}
	if operations[3].(map[string]any)["action"] != "write_file" {
		t.Fatalf("missing agent operation = %#v", operations[3])
	}

	code, env = runForTest(t, "skill", "update")
	if code != exitOK {
		t.Fatalf("update code=%d env=%#v", code, env)
	}
	assertInstalledFiles(t, "", []string{skill.TargetOpenCode})
}

func TestSkillUpdateRefusesLocalModificationWithoutForce(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)
	target := filepath.Join(".agents", "skills", "canon", "SKILL.md")
	if code, env := runForTest(t, "skill", "install", "--only", "canon"); code != exitOK {
		t.Fatalf("install code=%d env=%#v", code, env)
	}
	original, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read installed skill: %v", err)
	}
	if err := os.WriteFile(target, append(original, []byte("local edit\n")...), 0o644); err != nil {
		t.Fatalf("write local edit: %v", err)
	}

	code, env := runForTest(t, "skill", "update", "--only", "canon", "--dry-run")
	if code != exitState {
		t.Fatalf("update code=%d env=%#v", code, env)
	}
	if env["error"].(map[string]any)["code"] != "local_skill_modified" {
		t.Fatalf("error = %#v", env["error"])
	}

	code, env = runForTest(t, "skill", "update", "--only", "canon", "--force", "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("force dry-run code=%d env=%#v", code, env)
	}
	assertNoChangesWarning(t, env)

	code, env = runForTest(t, "skill", "update", "--only", "canon", "--force")
	if code != exitOK {
		t.Fatalf("force update code=%d env=%#v", code, env)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read forced skill: %v", err)
	}
	if strings.Contains(string(content), "local edit") {
		t.Fatal("force update retained local edit")
	}
}

func TestSkillUpdateProtectsCodexAgent(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)
	if code, env := runForTest(t, "skill", "install", "--agent", "codex"); code != exitOK {
		t.Fatalf("install code=%d env=%#v", code, env)
	}

	code, env := runForTest(t, "skill", "update", "--agent", "codex", "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("current update code=%d env=%#v", code, env)
	}
	for _, raw := range envelopePlan(env)["operations"].([]any) {
		if raw.(map[string]any)["action"] != "noop" {
			t.Fatalf("current bundle operation = %#v", raw)
		}
	}

	target := filepath.Join(".codex", "agents", "canon-critic.toml")
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read codex agent: %v", err)
	}
	older := testManagedCodexAgentVersion(t, string(content), "1")
	if err := os.WriteFile(target, []byte(older), 0o644); err != nil {
		t.Fatalf("write older codex agent: %v", err)
	}

	code, env = runForTest(t, "skill", "update", "--agent", "codex", "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("older update code=%d env=%#v", code, env)
	}
	var sawCodexUpdate bool
	for _, raw := range envelopePlan(env)["operations"].([]any) {
		operation := raw.(map[string]any)
		if operation["path"] == target && operation["action"] == "update_file" {
			sawCodexUpdate = true
		}
	}
	if !sawCodexUpdate {
		t.Fatalf("older Codex agent was not planned for update: %#v", envelopePlan(env)["operations"])
	}
	if code, env = runForTest(t, "skill", "update", "--agent", "codex"); code != exitOK {
		t.Fatalf("apply older update code=%d env=%#v", code, env)
	}
	content, err = os.ReadFile(target)
	if err != nil {
		t.Fatalf("read updated codex agent: %v", err)
	}
	if err := os.WriteFile(target, append(content, []byte("# local edit\n")...), 0o644); err != nil {
		t.Fatalf("edit codex agent: %v", err)
	}

	code, env = runForTest(t, "skill", "update", "--agent", "codex", "--dry-run")
	if code != exitState {
		t.Fatalf("modified update code=%d env=%#v", code, env)
	}
	if env["error"].(map[string]any)["code"] != "local_skill_modified" {
		t.Fatalf("error = %#v", env["error"])
	}
}

func TestSkillUpdateRequiresAnInstalledSelection(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)
	code, env := runForTest(t, "skill", "update")
	if code != exitNotFound {
		t.Fatalf("update code=%d env=%#v", code, env)
	}
	if env["error"].(map[string]any)["code"] != "skill_not_installed" {
		t.Fatalf("error = %#v", env["error"])
	}
}

func TestSkillInstallCustomSkillRoot(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)
	code, env := runForTest(t, "skill", "install", "--skill-dir", filepath.Join("custom", "skills"), "--only", "canon")
	if code != exitOK {
		t.Fatalf("install code=%d env=%#v", code, env)
	}
	if _, err := os.Stat(filepath.Join("custom", "skills", "canon", "SKILL.md")); err != nil {
		t.Fatalf("custom skill not installed: %v", err)
	}
}

func TestSkillTextOutputRendersCatalogAndPlans(t *testing.T) {
	project := t.TempDir()
	t.Chdir(project)
	code, output := runRawForTest(t, "-t", "skill")
	if code != exitOK {
		t.Fatalf("skill code=%d output:\n%s", code, output)
	}
	for _, want := range []string{"assets:", "canon [skill]", "canon-record-gate [skill]", "target paths:"} {
		if !strings.Contains(output, want) {
			t.Fatalf("skill text missing %q:\n%s", want, output)
		}
	}
	code, output = runRawForTest(t, "-t", "skill", "install", "--dry-run")
	if code != exitOK {
		t.Fatalf("install code=%d output:\n%s", code, output)
	}
	for _, want := range []string{"plan:", "write_file:", "dry_run: true", "No changes were made."} {
		if !strings.Contains(output, want) {
			t.Fatalf("install text missing %q:\n%s", want, output)
		}
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

func TestContextFormatRendersAcceptedADRList(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Use SQLite", "--status", "accepted"); code != exitOK {
		t.Fatalf("accepted new code = %d", code)
	}
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Evaluate Postgres"); code != exitOK {
		t.Fatalf("proposed new code = %d", code)
	}

	code, output := runRawForTest(t, "--adr-dir", dir, "--format", "context", "adr", "list", "--status", "accepted")
	if code != exitOK {
		t.Fatalf("code = %d, output:\n%s", code, output)
	}
	want := "## Architecture Decision Records\n\n- `ADR-0001`: Use SQLite\n"
	if output != want {
		t.Fatalf("context output mismatch\nwant:\n%s\ngot:\n%s", want, output)
	}
}

func TestContextFormatSupportsCombinedAndSpecLists(t *testing.T) {
	adrDir := filepath.Join(t.TempDir(), "adr")
	specDir := filepath.Join(t.TempDir(), "spec")
	if code, _ := runForTest(t, "--adr-dir", adrDir, "--spec-dir", specDir, "adr", "new", "--title", "Use SQLite", "--status", "accepted"); code != exitOK {
		t.Fatalf("adr new code = %d", code)
	}
	if code, _ := runForTest(t, "--adr-dir", adrDir, "--spec-dir", specDir, "spec", "new", "--title", "Query storage", "--status", "accepted"); code != exitOK {
		t.Fatalf("spec new code = %d", code)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "combined",
			args: []string{"list", "--status", "accepted"},
			want: "## Project Documents\n\n- `ADR-0001`: Use SQLite\n- `SPEC-0001`: Query storage\n",
		},
		{
			name: "spec",
			args: []string{"spec", "list", "--status", "accepted"},
			want: "## Specifications\n\n- `SPEC-0001`: Query storage\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"--adr-dir", adrDir, "--spec-dir", specDir, "--format", "context"}
			code, output := runRawForTest(t, append(args, tt.args...)...)
			if code != exitOK || output != tt.want {
				t.Fatalf("code = %d\nwant:\n%s\ngot:\n%s", code, tt.want, output)
			}
		})
	}
}

func TestContextFormatRendersEmptyList(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "adr")
	code, output := runRawForTest(t, "--adr-dir", dir, "--format", "context", "adr", "list", "--status", "accepted")
	if code != exitOK {
		t.Fatalf("code = %d, output:\n%s", code, output)
	}
	want := "## Architecture Decision Records\n\n_No matching documents._\n"
	if output != want {
		t.Fatalf("context output mismatch\nwant:\n%s\ngot:\n%s", want, output)
	}
}

func TestContextFormatRejectsUnsupportedCommands(t *testing.T) {
	code, output := runRawForTest(t, "--format", "context", "show", "--id", "ADR-0001")
	if code != exitUsage {
		t.Fatalf("code = %d, output:\n%s", code, output)
	}
	want := "## Canon Error\n\n- `unsupported_context_format`: context format is not supported by show\n"
	if output != want {
		t.Fatalf("context error mismatch\nwant:\n%s\ngot:\n%s", want, output)
	}
}

func envelopePlan(env map[string]any) map[string]any {
	return env["data"].(map[string]any)["plan"].(map[string]any)
}

func assertNoChangesWarning(t *testing.T, env map[string]any) {
	t.Helper()
	warnings, ok := env["warnings"].([]any)
	if !ok || len(warnings) != 1 || warnings[0] != "No changes were made." {
		t.Fatalf("warnings = %#v", env["warnings"])
	}
}

func assertPlanPaths(t *testing.T, operations []any, want []string) {
	t.Helper()
	if len(operations) != len(want) {
		t.Fatalf("operation count = %d, want %d: %#v", len(operations), len(want), operations)
	}
	for i, raw := range operations {
		operation := raw.(map[string]any)
		if operation["path"] != want[i] {
			t.Fatalf("operation %d path = %v, want %s (all operations: %#v)", i, operation["path"], want[i], operations)
		}
	}
}

func desiredSkillFilesForTest(t *testing.T, assets []string, skillsRoot string, targets ...string) []skill.ManagedFile {
	t.Helper()
	files, err := skill.ManagedFiles(assets, skillsRoot, targets)
	if err != nil {
		t.Fatalf("managed files: %v", err)
	}
	return files
}

func assertInstalledFiles(t *testing.T, skillsRoot string, targets []string) {
	t.Helper()
	for _, file := range desiredSkillFilesForTest(t, nil, skillsRoot, targets...) {
		content, err := os.ReadFile(file.Path)
		if err != nil {
			t.Fatalf("read installed file %s: %v", file.Path, err)
		}
		if string(content) != file.Content() {
			t.Fatalf("installed file %s differs from bundle", file.Path)
		}
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

func testManagedCodexAgentVersion(t *testing.T, content, version string) string {
	t.Helper()
	lines := strings.Split(content, "\n")
	baseLines := make([]string, 0, len(lines))
	versionIndex := -1
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "# canon-skill-version:"):
			versionIndex = len(baseLines)
			baseLines = append(baseLines, "# canon-skill-version: "+version)
		case strings.HasPrefix(line, "# canon-skill-hash:"):
			continue
		default:
			baseLines = append(baseLines, line)
		}
	}
	if versionIndex < 0 {
		t.Fatal("Codex agent has no managed version marker")
	}
	base := strings.Join(baseLines, "\n")
	sum := sha256.Sum256([]byte(base))
	hashLine := "# canon-skill-hash: sha256:" + fmt.Sprintf("%x", sum[:])
	baseLines = append(baseLines[:versionIndex+1], append([]string{hashLine}, baseLines[versionIndex+1:]...)...)
	return strings.Join(baseLines, "\n")
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

func TestNewDomainDryRunDoesNotWriteFile(t *testing.T) {
	domain := filepath.Join(t.TempDir(), "domain")
	code, env := runKindTest(t, "--domain-dir", domain, "domain", "new", "--title", "ADR", "--dry-run")
	if code != exitOK {
		t.Fatalf("code = %d", code)
	}
	if env["status"] != "planned" {
		t.Fatalf("status = %v", env["status"])
	}
	if _, err := os.Stat(domain); !os.IsNotExist(err) {
		t.Fatalf("dry-run created domain directory: %v", err)
	}
	data := env["data"].(map[string]any)
	created := data["adr"].(map[string]any)
	if created["kind"] != "domain" || created["id"] != "DM-0001" {
		t.Fatalf("domain preview = %#v", created)
	}
	warnings := env["warnings"].([]any)
	if warnings[0] != "No changes were made." {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestCreateListSearchAndShowDomain(t *testing.T) {
	domain := filepath.Join(t.TempDir(), "domain")
	code, env := runKindTest(t, "--domain-dir", domain, "domain", "new", "--title", "ADR", "--tags", "glossary", "--definition", "A dated, narrowly-scoped architecture commitment.", "--avoid", "design doc: too broad; ticket: tracks work, not decisions", "--relationships", "See [SPEC](0002-spec.md).")
	if code != exitOK {
		t.Fatalf("new domain code = %d", code)
	}
	created := env["data"].(map[string]any)["adr"].(map[string]any)
	if created["id"] != "DM-0001" || created["kind"] != "domain" {
		t.Fatalf("created domain entry = %#v", created)
	}

	code, env = runKindTest(t, "--domain-dir", domain, "domain", "list", "--tag", "glossary")
	if code != exitOK {
		t.Fatalf("list code = %d", code)
	}
	if env["data"].(map[string]any)["count"] != float64(1) {
		t.Fatalf("list data = %#v", env["data"])
	}

	code, env = runKindTest(t, "--domain-dir", domain, "domain", "search", "--query", "commitment")
	if code != exitOK {
		t.Fatalf("search code = %d", code)
	}
	if env["data"].(map[string]any)["count"] != float64(1) {
		t.Fatalf("search data = %#v", env["data"])
	}

	code, env = runKindTest(t, "--domain-dir", domain, "show", "--id", "DM-0001")
	if code != exitOK {
		t.Fatalf("show code = %d", code)
	}
	shown := env["data"].(map[string]any)["adr"].(map[string]any)
	if shown["kind"] != "domain" || shown["content"] == "" {
		t.Fatalf("show domain entry = %#v", shown)
	}
	content := shown["content"].(string)
	for _, section := range []string{"## Definition", "## Avoid", "## Relationships"} {
		if !strings.Contains(content, section) {
			t.Fatalf("domain content missing %s:\n%s", section, content)
		}
	}
	if !strings.Contains(content, "- **design doc** — too broad") || !strings.Contains(content, "- **ticket** — tracks work, not decisions") {
		t.Fatalf("domain avoid list not rendered as reasoned bullets:\n%s", content)
	}
}

func TestThreeKindsNumberIndependently(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	if code, _ := runKindTest(t, append(dirs, "adr", "new", "--title", "First ADR")...); code != exitOK {
		t.Fatalf("adr new code = %d", code)
	}
	if code, _ := runKindTest(t, append(dirs, "domain", "new", "--title", "First entry")...); code != exitOK {
		t.Fatalf("domain new code = %d", code)
	}
	if code, _ := runKindTest(t, append(dirs, "spec", "new", "--title", "First SPEC")...); code != exitOK {
		t.Fatalf("spec new code = %d", code)
	}
	if code, _ := runKindTest(t, append(dirs, "adr", "new", "--title", "Second ADR")...); code != exitOK {
		t.Fatalf("second adr code = %d", code)
	}
	code, env := runKindTest(t, append(dirs, "list")...)
	if code != exitOK {
		t.Fatalf("list code = %d", code)
	}
	data := env["data"].(map[string]any)
	if data["count"] != float64(4) {
		t.Fatalf("count = %v", data["count"])
	}
	adrs := data["adrs"].([]any)
	ids := []string{}
	for _, raw := range adrs {
		ids = append(ids, raw.(map[string]any)["id"].(string))
	}
	want := []string{"ADR-0001", "ADR-0002", "DM-0001", "SPEC-0001"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("ordered ids = %v, want %v", ids, want)
	}
}

func TestSupersedeRejectsCrossKindDomainReplacement(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--domain-dir", domain}
	if code, _ := runKindTest(t, append(dirs, "domain", "new", "--title", "Old entry")...); code != exitOK {
		t.Fatalf("domain new code = %d", code)
	}
	if code, _ := runKindTest(t, append(dirs, "adr", "new", "--title", "Replacement ADR")...); code != exitOK {
		t.Fatalf("adr new code = %d", code)
	}
	code, env := runKindTest(t, append(dirs, "supersede", "--id", "DM-0001", "--by", "ADR-0001", "--dry-run")...)
	if code != exitState {
		t.Fatalf("cross-kind supersede code = %d env = %#v", code, env)
	}
	errData := env["error"].(map[string]any)
	if errData["code"] != "cross_kind_supersede" {
		t.Fatalf("error = %#v", errData)
	}
}

func TestAcceptMutatesDomainEntry(t *testing.T) {
	domain := filepath.Join(t.TempDir(), "domain")
	if code, _ := runKindTest(t, "--domain-dir", domain, "domain", "new", "--title", "Candidate entry"); code != exitOK {
		t.Fatalf("domain new code = %d", code)
	}
	if code, env := runKindTest(t, "--domain-dir", domain, "accept", "--id", "DM-0001", "--reason", "Canonized.", "--dry-run"); code != exitOK || env["status"] != "planned" {
		t.Fatalf("accept dry-run code=%d env=%#v", code, env)
	}
	if code, _ := runKindTest(t, "--domain-dir", domain, "accept", "--id", "DM-0001", "--reason", "Canonized."); code != exitOK {
		t.Fatalf("accept code = %d", code)
	}
	code, env := runKindTest(t, "--domain-dir", domain, "show", "--id", "DM-0001")
	if code != exitOK {
		t.Fatalf("show code = %d", code)
	}
	shown := env["data"].(map[string]any)["adr"].(map[string]any)
	if shown["status"] != "accepted" {
		t.Fatalf("status = %v", shown["status"])
	}
	content := shown["content"].(string)
	if !strings.Contains(content, "## History: Accepted") || !strings.Contains(content, "Canonized.") {
		t.Fatalf("domain content missing accepted history:\n%s", content)
	}
}

func doctorDiagnostics(env map[string]any) []map[string]any {
	var out []map[string]any
	for _, raw := range env["data"].(map[string]any)["diagnostics"].([]any) {
		out = append(out, raw.(map[string]any))
	}
	return out
}

func TestDoctorReportsMissingDomainDirectory(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	if code, _ := runKindTest(t, append(dirs, "adr", "init")...); code != exitOK {
		t.Fatalf("adr init code = %d", code)
	}
	if code, _ := runKindTest(t, append(dirs, "spec", "init")...); code != exitOK {
		t.Fatalf("spec init code = %d", code)
	}
	code, env := runKindTest(t, append(dirs, "doctor")...)
	if code != exitOK {
		t.Fatalf("doctor code = %d", code)
	}
	if env["status"] != "warning" {
		t.Fatalf("doctor status = %v", env["status"])
	}
	var sawDomainWarning bool
	for _, d := range doctorDiagnostics(env) {
		if d["name"] == "domain_directory" && d["status"] == "warning" {
			sawDomainWarning = true
		}
	}
	if !sawDomainWarning {
		t.Fatalf("doctor missing domain_directory warning: %#v", env["data"])
	}
}

func TestDoctorFlagsDuplicateAcceptedDomainTitles(t *testing.T) {
	domain := filepath.Join(t.TempDir(), "domain")
	if code, _ := runKindTest(t, "--domain-dir", domain, "domain", "new", "--title", "Order", "--status", "accepted"); code != exitOK {
		t.Fatalf("first new code = %d", code)
	}
	if code, _ := runKindTest(t, "--domain-dir", domain, "domain", "new", "--title", "Order", "--status", "accepted"); code != exitOK {
		t.Fatalf("second new code = %d", code)
	}
	code, env := runKindTest(t, "--domain-dir", domain, "doctor")
	if code != exitOK {
		t.Fatalf("doctor code = %d", code)
	}
	if env["status"] != "warning" {
		t.Fatalf("doctor status = %v", env["status"])
	}
	var sawDuplicate bool
	for _, d := range doctorDiagnostics(env) {
		if d["name"] == "domain_duplicate_title" && d["status"] == "warning" {
			sawDuplicate = true
			if !strings.Contains(d["message"].(string), "DM-0001") || !strings.Contains(d["message"].(string), "DM-0002") {
				t.Fatalf("duplicate diagnostic missing ids: %#v", d)
			}
		}
	}
	if !sawDuplicate {
		t.Fatalf("doctor missing domain_duplicate_title warning: %#v", env["data"])
	}

	// A single accepted entry for the title clears the finding.
	if code, _ := runKindTest(t, "--domain-dir", domain, "deprecate", "--id", "DM-0002", "--reason", "Duplicate of DM-0001."); code != exitOK {
		t.Fatalf("deprecate code = %d", code)
	}
	code, env = runKindTest(t, "--domain-dir", domain, "doctor")
	if code != exitOK {
		t.Fatalf("doctor code = %d", code)
	}
	for _, d := range doctorDiagnostics(env) {
		if d["name"] == "domain_duplicate_title" {
			t.Fatalf("duplicate finding persists after deprecating one entry: %#v", d)
		}
	}
}

func TestDoctorFlagsDeadDomainReferences(t *testing.T) {
	domain := filepath.Join(t.TempDir(), "domain")
	if code, _ := runKindTest(t, "--domain-dir", domain, "domain", "new", "--title", "Session"); code != exitOK {
		t.Fatalf("first new code = %d", code)
	}
	if code, _ := runKindTest(t, "--domain-dir", domain, "domain", "new", "--title", "Connection", "--relationships", "Replaces [Session](0001-session.md)."); code != exitOK {
		t.Fatalf("second new code = %d", code)
	}

	// While DM-0001 is live, references to it are healthy.
	code, env := runKindTest(t, "--domain-dir", domain, "doctor")
	if code != exitOK {
		t.Fatalf("doctor code = %d", code)
	}
	for _, d := range doctorDiagnostics(env) {
		if d["name"] == "domain_dead_reference" {
			t.Fatalf("unexpected dead reference finding while target is live: %#v", d)
		}
	}

	if code, _ := runKindTest(t, "--domain-dir", domain, "supersede", "--id", "DM-0001", "--by", "DM-0002", "--reason", "Redefined."); code != exitOK {
		t.Fatalf("supersede code = %d", code)
	}
	code, env = runKindTest(t, "--domain-dir", domain, "doctor")
	if code != exitOK {
		t.Fatalf("doctor code = %d", code)
	}
	if env["status"] != "warning" {
		t.Fatalf("doctor status = %v", env["status"])
	}
	var sawDeadReference bool
	for _, d := range doctorDiagnostics(env) {
		if d["name"] == "domain_dead_reference" && d["status"] == "warning" {
			sawDeadReference = true
			message := d["message"].(string)
			if !strings.Contains(message, "DM-0002") || !strings.Contains(message, "DM-0001") {
				t.Fatalf("dead reference diagnostic missing ids: %#v", d)
			}
			if !strings.Contains(d["suggested_fix"].(string), "DM-0002") {
				t.Fatalf("dead reference fix should point at the successor: %#v", d)
			}
		}
	}
	if !sawDeadReference {
		t.Fatalf("doctor missing domain_dead_reference warning: %#v", env["data"])
	}
}

func writeRawDocForTest(t *testing.T, dir, filename, id, status, extraFrontMatter string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	kind := "adr"
	if strings.HasPrefix(id, "SPEC-") {
		kind = "spec"
	}
	if strings.HasPrefix(id, "DM-") {
		kind = "domain"
	}
	content := fmt.Sprintf(`---
kind: %s
id: %s
title: %s
status: %s
date: 2026-07-01
%s---
# %s: %s

## Status

%s
`, kind, id, id, status, extraFrontMatter, id, id, status)
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
	return path
}

func validateFindings(env map[string]any) []map[string]any {
	var out []map[string]any
	for _, raw := range env["data"].(map[string]any)["findings"].([]any) {
		out = append(out, raw.(map[string]any))
	}
	return out
}

func findingNames(findings []map[string]any) []string {
	names := []string{}
	for _, f := range findings {
		names = append(names, f["name"].(string))
	}
	return names
}

func TestValidateHealthyCorpus(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	if code, _ := runKindTest(t, append(dirs, "adr", "new", "--title", "One")...); code != exitOK {
		t.Fatalf("adr new code = %d", code)
	}
	if code, _ := runKindTest(t, append(dirs, "spec", "new", "--title", "Two")...); code != exitOK {
		t.Fatalf("spec new code = %d", code)
	}
	if code, _ := runKindTest(t, append(dirs, "domain", "new", "--title", "Three")...); code != exitOK {
		t.Fatalf("domain new code = %d", code)
	}
	code, env := runKindTest(t, append(dirs, "validate")...)
	if code != exitOK {
		t.Fatalf("validate code = %d", code)
	}
	if env["status"] != "ok" {
		t.Fatalf("validate status = %v", env["status"])
	}
	summary := env["data"].(map[string]any)["summary"].(map[string]any)
	if summary["files_checked"] != float64(3) || summary["errors"] != float64(0) || summary["warnings"] != float64(0) {
		t.Fatalf("summary = %#v", summary)
	}
	if len(validateFindings(env)) != 0 {
		t.Fatalf("expected no findings: %#v", env["data"])
	}
}

func TestValidateDuplicateID(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	first := writeRawDocForTest(t, adr, "0001-one.md", "ADR-0001", "accepted", "")
	second := writeRawDocForTest(t, adr, "0001-copy.md", "ADR-0001", "accepted", "")
	code, env := runKindTest(t, append(dirs, "validate")...)
	if code != exitState {
		t.Fatalf("validate code = %d", code)
	}
	if env["status"] != "error" {
		t.Fatalf("validate status = %v", env["status"])
	}
	var sawDuplicate bool
	for _, f := range validateFindings(env) {
		if f["name"] == "duplicate_id" && f["status"] == "error" {
			sawDuplicate = true
			message := f["message"].(string)
			if !strings.Contains(message, first) || !strings.Contains(message, second) {
				t.Fatalf("duplicate_id finding must name both paths: %#v", f)
			}
			if f["suggested_fix"] == "" || f["suggested_fix"] == nil {
				t.Fatalf("duplicate_id finding missing suggested_fix: %#v", f)
			}
		}
	}
	if !sawDuplicate {
		t.Fatalf("missing duplicate_id finding: %#v", env["data"])
	}
}

func TestValidateBrokenReference(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	writeRawDocForTest(t, adr, "0001-one.md", "ADR-0001", "superseded", "superseded_by: ADR-0009\n")
	code, env := runKindTest(t, append(dirs, "validate")...)
	if code != exitState {
		t.Fatalf("validate code = %d", code)
	}
	if env["status"] != "error" {
		t.Fatalf("validate status = %v", env["status"])
	}
	var sawBroken bool
	for _, f := range validateFindings(env) {
		if f["name"] == "broken_reference" && f["status"] == "error" {
			sawBroken = true
			if !strings.Contains(f["message"].(string), "ADR-0009") {
				t.Fatalf("broken_reference finding missing target id: %#v", f)
			}
		}
	}
	if !sawBroken {
		t.Fatalf("missing broken_reference finding: %#v", env["data"])
	}
}

func TestValidateMissingSpecDirectoryWarning(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	writeRawDocForTest(t, adr, "0001-one.md", "ADR-0001", "accepted", "")
	if err := os.MkdirAll(domain, 0o755); err != nil {
		t.Fatalf("mkdir domain: %v", err)
	}
	code, env := runKindTest(t, append(dirs, "validate")...)
	if code != exitOK {
		t.Fatalf("validate code = %d", code)
	}
	if env["status"] != "warning" {
		t.Fatalf("validate status = %v", env["status"])
	}
	names := findingNames(validateFindings(env))
	if len(names) != 1 || names[0] != "missing_directory" {
		t.Fatalf("findings = %v", names)
	}
	code, env = runKindTest(t, append(dirs, "validate", "--strict")...)
	if code != exitState {
		t.Fatalf("validate --strict code = %d", code)
	}
	if env["status"] != "warning" {
		t.Fatalf("validate --strict status = %v", env["status"])
	}
}

func TestValidateByIDScopesToOneDocument(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	writeRawDocForTest(t, adr, "0001-clean.md", "ADR-0001", "accepted", "")
	writeRawDocForTest(t, adr, "0002-broken.md", "ADR-0002", "superseded", "superseded_by: ADR-0009\n")

	code, env := runKindTest(t, append(dirs, "validate", "--id", "ADR-0001")...)
	if code != exitOK {
		t.Fatalf("validate --id clean code = %d", code)
	}
	if env["status"] != "ok" {
		t.Fatalf("validate --id clean status = %v", env["status"])
	}
	summary := env["data"].(map[string]any)["summary"].(map[string]any)
	if summary["files_checked"] != float64(1) {
		t.Fatalf("summary = %#v", summary)
	}

	code, env = runKindTest(t, append(dirs, "validate", "--id", "ADR-0002")...)
	if code != exitState {
		t.Fatalf("validate --id broken code = %d", code)
	}
	names := findingNames(validateFindings(env))
	if len(names) != 1 || names[0] != "broken_reference" {
		t.Fatalf("findings = %v", names)
	}

	code, _ = runKindTest(t, append(dirs, "validate", "--id", "ADR-0099")...)
	if code != exitNotFound {
		t.Fatalf("validate --id missing code = %d", code)
	}
	code, _ = runKindTest(t, append(dirs, "validate", "--id", "bogus")...)
	if code != exitUsage {
		t.Fatalf("validate --id invalid code = %d", code)
	}
	code, _ = runKindTest(t, append(dirs, "adr", "validate", "--id", "ADR-0001")...)
	if code != exitUsage {
		t.Fatalf("adr validate --id code = %d", code)
	}
}

func TestValidateKindScoped(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	writeRawDocForTest(t, adr, "0001-broken.md", "ADR-0001", "superseded", "superseded_by: ADR-0009\n")
	writeRawDocForTest(t, spec, "0001-fine.md", "SPEC-0001", "accepted", "")

	code, env := runKindTest(t, append(dirs, "adr", "validate")...)
	if code != exitState {
		t.Fatalf("adr validate code = %d", code)
	}
	names := findingNames(validateFindings(env))
	if len(names) != 1 || names[0] != "broken_reference" {
		t.Fatalf("adr validate findings = %v", names)
	}

	code, env = runKindTest(t, append(dirs, "spec", "validate")...)
	if code != exitOK {
		t.Fatalf("spec validate code = %d", code)
	}
	if env["status"] != "ok" {
		t.Fatalf("spec validate status = %v findings = %#v", env["status"], env["data"])
	}
}

func TestValidateDomainScoped(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	writeRawDocForTest(t, domain, "0001-fine.md", "DM-0001", "accepted", "")
	code, env := runKindTest(t, append(dirs, "domain", "validate")...)
	if code != exitOK {
		t.Fatalf("domain validate code = %d", code)
	}
	if env["status"] != "ok" {
		t.Fatalf("domain validate status = %v", env["status"])
	}
	summary := env["data"].(map[string]any)["summary"].(map[string]any)
	if summary["files_checked"] != float64(1) {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestValidateMalformedFileDoesNotMaskCorpus(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	writeRawDocForTest(t, adr, "0001-one.md", "ADR-0001", "superseded", "")
	badPath := filepath.Join(adr, "0002-bad.md")
	if err := os.WriteFile(badPath, []byte("no front matter here\n"), 0o644); err != nil {
		t.Fatalf("write malformed: %v", err)
	}
	code, env := runKindTest(t, append(dirs, "validate")...)
	if code != exitState {
		t.Fatalf("validate code = %d", code)
	}
	var sawMalformed, sawConsistency bool
	for _, f := range validateFindings(env) {
		if f["name"] == "malformed_file" && f["status"] == "error" && f["path"] == badPath {
			sawMalformed = true
		}
		// The parseable file is still checked: superseded without
		// superseded_by is a warning.
		if f["name"] == "status_reference_inconsistency" && f["id"] == "ADR-0001" {
			sawConsistency = true
		}
	}
	if !sawMalformed {
		t.Fatalf("missing malformed_file finding: %#v", env["data"])
	}
	if !sawConsistency {
		t.Fatalf("malformed file masked the rest of the corpus: %#v", env["data"])
	}
	summary := env["data"].(map[string]any)["summary"].(map[string]any)
	if summary["files_checked"] != float64(2) {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestValidateReciprocityViolation(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	writeRawDocForTest(t, adr, "0001-old.md", "ADR-0001", "superseded", "superseded_by: ADR-0002\n")
	writeRawDocForTest(t, adr, "0002-new.md", "ADR-0002", "accepted", "")
	code, env := runKindTest(t, append(dirs, "validate")...)
	if code != exitState {
		t.Fatalf("validate code = %d", code)
	}
	names := findingNames(validateFindings(env))
	if !contains(names, "reciprocity_violation") {
		t.Fatalf("findings = %v", names)
	}
}

func TestValidateKindAndDirectoryCoherence(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	// SPEC-prefixed id living in the ADR directory.
	writeRawDocForTest(t, adr, "0001-misplaced.md", "SPEC-0001", "accepted", "")
	code, env := runKindTest(t, append(dirs, "validate")...)
	if code != exitState {
		t.Fatalf("validate code = %d", code)
	}
	names := findingNames(validateFindings(env))
	if !contains(names, "directory_mismatch") {
		t.Fatalf("findings = %v", names)
	}
}

func TestValidateInvalidStatusAndMalformedDate(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	path := writeRawDocForTest(t, adr, "0001-one.md", "ADR-0001", "accepted", "")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	mutated := strings.Replace(string(content), "status: accepted", "status: someday", 1)
	mutated = strings.Replace(mutated, "date: 2026-07-01", "date: last tuesday", 1)
	mutated = strings.Replace(mutated, "## Status\n\naccepted", "## Status\n\nsomeday", 1)
	if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
	code, env := runKindTest(t, append(dirs, "validate")...)
	if code != exitState {
		t.Fatalf("validate code = %d", code)
	}
	names := findingNames(validateFindings(env))
	if !contains(names, "invalid_status") || !contains(names, "malformed_date") {
		t.Fatalf("findings = %v", names)
	}
	for _, f := range validateFindings(env) {
		if f["name"] == "malformed_date" && f["status"] != "warning" {
			t.Fatalf("malformed_date must be a warning: %#v", f)
		}
		if f["name"] == "invalid_status" && f["status"] != "error" {
			t.Fatalf("invalid_status must be an error: %#v", f)
		}
	}
}

func TestValidateNeverMutatesAndDeterministicOrder(t *testing.T) {
	adr := filepath.Join(t.TempDir(), "adr")
	spec := filepath.Join(t.TempDir(), "spec")
	domain := filepath.Join(t.TempDir(), "domain")
	dirs := []string{"--adr-dir", adr, "--spec-dir", spec, "--domain-dir", domain}
	writeRawDocForTest(t, adr, "0002-b.md", "ADR-0002", "superseded", "superseded_by: ADR-0009\n")
	writeRawDocForTest(t, adr, "0001-a.md", "ADR-0001", "superseded", "")

	before, err := os.ReadFile(filepath.Join(adr, "0001-a.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	code, env := runKindTest(t, append(dirs, "validate")...)
	if code != exitState {
		t.Fatalf("validate code = %d", code)
	}
	after, err := os.ReadFile(filepath.Join(adr, "0001-a.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("validate mutated the corpus")
	}

	// Findings are ordered by path then check name.
	var prevPath, prevName string
	for _, f := range validateFindings(env) {
		path, _ := f["path"].(string)
		name := f["name"].(string)
		if path < prevPath || (path == prevPath && name < prevName) {
			t.Fatalf("findings out of order at %#v after %s/%s", f, prevPath, prevName)
		}
		prevPath, prevName = path, name
	}

	// A repeated run produces identical findings.
	_, envAgain := runKindTest(t, append(dirs, "validate")...)
	firstFindings, _ := json.Marshal(env["data"].(map[string]any)["findings"])
	secondFindings, _ := json.Marshal(envAgain["data"].(map[string]any)["findings"])
	if !bytes.Equal(firstFindings, secondFindings) {
		t.Fatal("validate findings are not deterministic across runs")
	}
}

func TestContextFormatRendersDomainList(t *testing.T) {
	domain := filepath.Join(t.TempDir(), "domain")
	if code, _ := runKindTest(t, "--domain-dir", domain, "domain", "new", "--title", "ADR", "--status", "accepted"); code != exitOK {
		t.Fatalf("domain new code = %d", code)
	}
	code, output := runRawForTest(t, "--domain-dir", domain, "--format", "context", "domain", "list", "--status", "accepted")
	if code != exitOK {
		t.Fatalf("code = %d, output:\n%s", code, output)
	}
	want := "## Domain Model\n\n- `DM-0001`: ADR\n"
	if output != want {
		t.Fatalf("context output mismatch\nwant:\n%s\ngot:\n%s", want, output)
	}
}
