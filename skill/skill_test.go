package skill

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestCatalogContainsOnlySkillAssets(t *testing.T) {
	assets := Catalog()
	if len(assets) != 2 {
		t.Fatalf("catalog asset count = %d, want 2", len(assets))
	}
	if assets[0].Name != "canon" || assets[1].Name != "canon-record-gate" {
		t.Fatalf("catalog order = %#v", assets)
	}
	for _, asset := range assets {
		if asset.Kind != KindSkill {
			t.Fatalf("asset %s kind = %q", asset.Name, asset.Kind)
		}
		if asset.Version == "" || !strings.HasPrefix(asset.Hash, "sha256:") {
			t.Fatalf("asset missing version or hash: %#v", asset)
		}
		if len(asset.TargetPaths) == 0 || !sort.StringsAreSorted(asset.TargetPaths) {
			t.Fatalf("asset %s target paths not deterministic: %#v", asset.Name, asset.TargetPaths)
		}
	}
	if !containsString(assets[0].TargetPaths, filepath.Join(".agents", "skills", "canon", "SKILL.md")) {
		t.Fatalf("canon target paths = %#v", assets[0].TargetPaths)
	}
	for _, want := range []string{
		filepath.Join(".agents", "skills", "canon-record-gate", "SKILL.md"),
		filepath.Join(".agents", "skills", "canon-record-gate", "references", "boundary-examples.md"),
		filepath.Join(".claude", "agents", "canon-critic.md"),
		filepath.Join(".opencode", "agents", "canon-critic.md"),
	} {
		if !containsString(assets[1].TargetPaths, want) {
			t.Fatalf("record gate target paths missing %s: %#v", want, assets[1].TargetPaths)
		}
	}
}

func TestManagedFilesRenderWholeBundleDeterministically(t *testing.T) {
	files, err := ManagedFiles(nil, "", []string{TargetCodex, TargetClaude, TargetOpenCode, TargetOpenCode})
	if err != nil {
		t.Fatalf("managed files: %v", err)
	}
	if len(files) != 5 {
		t.Fatalf("managed file count = %d, want 5", len(files))
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
		if file.Version == "" || !strings.HasPrefix(file.Hash(), "sha256:") {
			t.Fatalf("file missing version/hash: %#v", file)
		}
		if !strings.Contains(file.Content(), "<!-- canon-skill-version: "+file.Version+" -->") ||
			!strings.Contains(file.Content(), "<!-- canon-skill-hash: "+file.Hash()+" -->") {
			t.Fatalf("file missing management markers: %s", file.Path)
		}
	}
	if !sort.StringsAreSorted(paths) {
		t.Fatalf("paths not sorted: %#v", paths)
	}
	if containsPath(paths, filepath.Join(".codex", "agents", "canon-critic.md")) {
		t.Fatalf("codex unexpectedly rendered an agent: %#v", paths)
	}
}

func TestAgentRenderingIsTargetSpecificAndProjectAgnostic(t *testing.T) {
	files, err := ManagedFiles([]string{"canon-record-gate"}, "", []string{TargetOpenCode, TargetClaude})
	if err != nil {
		t.Fatalf("managed files: %v", err)
	}
	byPath := make(map[string]ManagedFile, len(files))
	for _, file := range files {
		byPath[file.Path] = file
	}

	opencode := byPath[filepath.Join(".opencode", "agents", "canon-critic.md")].Content()
	for _, want := range []string{"mode: subagent", "permission:", "  edit: deny", "  skill: allow"} {
		if !strings.Contains(opencode, want) {
			t.Fatalf("opencode agent missing %q:\n%s", want, opencode)
		}
	}

	claude := byPath[filepath.Join(".claude", "agents", "canon-critic.md")].Content()
	for _, want := range []string{"tools: Read, Grep, Glob, Bash", "model: inherit"} {
		if !strings.Contains(claude, want) {
			t.Fatalf("claude agent missing %q:\n%s", want, claude)
		}
	}

	for name, content := range map[string]string{"opencode": opencode, "claude": claude} {
		for _, unwanted := range []string{"go run ./cmd/canon", "canon repository", "corpus of 16 entries"} {
			if strings.Contains(content, unwanted) {
				t.Fatalf("%s agent contains project-specific text %q", name, unwanted)
			}
		}
		if !strings.Contains(content, "canon doctor") || !strings.Contains(content, "canon-record-gate") {
			t.Fatalf("%s agent missing generic canon guidance:\n%s", name, content)
		}
	}
}

func TestAssetSelectionAndTargetValidation(t *testing.T) {
	selected, err := SelectAssets([]string{"canon-record-gate", "canon", "canon"})
	if err != nil {
		t.Fatalf("select assets: %v", err)
	}
	if strings.Join(selected, ",") != "canon,canon-record-gate" {
		t.Fatalf("selected assets = %#v", selected)
	}
	if _, err := SelectAssets([]string{"canon-critic"}); err == nil {
		t.Fatal("agent component must not be selectable as a top-level asset")
	}
	targets, err := NormalizeTargets([]string{TargetCodex, TargetOpenCode, TargetCodex})
	if err != nil {
		t.Fatalf("normalize targets: %v", err)
	}
	if strings.Join(targets, ",") != "opencode,codex" {
		t.Fatalf("targets = %#v", targets)
	}
	if _, err := NormalizeTargets([]string{"other"}); err == nil {
		t.Fatal("unsupported target accepted")
	}
}

func TestEmbeddedSourcesDoNotContainManagementMarkers(t *testing.T) {
	for _, spec := range assetSpecs {
		for _, payload := range spec.files {
			content := readAsset(payload.sourcePath)
			if strings.Contains(content, "canon-skill-version:") || strings.Contains(content, "canon-skill-hash:") {
				t.Fatalf("source %s contains a management marker", payload.sourcePath)
			}
		}
		if spec.agent != nil {
			content := readAsset(spec.agent.sourcePath)
			if strings.Contains(content, "canon-skill-version:") || strings.Contains(content, "canon-skill-hash:") {
				t.Fatalf("source %s contains a management marker", spec.agent.sourcePath)
			}
		}
	}
}

func TestInspectionDistinguishesCurrentOldAndModifiedFiles(t *testing.T) {
	files, err := ManagedFiles([]string{"canon"}, "", nil)
	if err != nil {
		t.Fatalf("managed files: %v", err)
	}
	file := files[0]

	current := Inspect(file.Content(), file)
	if !current.Managed || !current.Current || current.Modified {
		t.Fatalf("current inspection = %#v", current)
	}

	old := strings.Replace(file.Content(), versionComment(file.Version), versionComment("0"), 1)
	old = strings.Replace(old, hashComment(file.Hash()), hashComment(hashWithoutHashComment(old)), 1)
	oldInspection := Inspect(old, file)
	if !oldInspection.Managed || oldInspection.Current || oldInspection.Modified {
		t.Fatalf("old inspection = %#v", oldInspection)
	}

	modifiedInspection := Inspect(file.Content()+"local edit\n", file)
	if modifiedInspection.Managed || modifiedInspection.Current || !modifiedInspection.Modified {
		t.Fatalf("modified inspection = %#v", modifiedInspection)
	}
}

func TestCanonSkillContentGuidance(t *testing.T) {
	files, err := ManagedFiles([]string{"canon"}, "", nil)
	if err != nil {
		t.Fatalf("managed files: %v", err)
	}
	content := files[0].Content()
	for _, want := range []string{
		"canon --format context adr list --status accepted",
		"## When to create or change an ADR",
		"commitment, not an intention",
		"project process, agent workflow, or skill instruction",
		"canon skill install",
		"canon skill update",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("canon skill content missing %q:\n%s", want, content)
		}
	}
	const marker = "## Common commands"
	idx := strings.Index(content, marker)
	if idx < 0 {
		t.Fatalf("canon skill content missing %q", marker)
	}
	for _, line := range strings.Split(content[idx:], "\n") {
		if strings.HasPrefix(line, "canon ") && strings.Contains(line, "--dry-run") {
			t.Fatalf("command example repeats --dry-run: %q", line)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsPath(paths []string, want string) bool {
	return containsString(paths, want)
}
