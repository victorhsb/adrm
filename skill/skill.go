package skill

import (
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const (
	KindSkill = "skill"
	KindAgent = "agent"

	TargetOpenCode = "opencode"
	TargetClaude   = "claude"
	TargetCodex    = "codex"

	AgentName         = "canon-critic"
	DefaultSkillsRoot = ".agents/skills"
)

// DefaultInstallDir is kept as the CLI-facing name for the bundle's skill root.
const DefaultInstallDir = DefaultSkillsRoot

// ErrUnsupportedTarget marks an agent target outside the first-party set.
var ErrUnsupportedTarget = errors.New("unsupported agent target")

//go:embed assets/canon/SKILL.md assets/canon-record-gate/SKILL.md assets/canon-record-gate/references/boundary-examples.md assets/canon-record-gate/agent.md
var bundledAssets embed.FS

type payloadFile struct {
	relativePath string
	sourcePath   string
}

type agentSpec struct {
	name       string
	sourcePath string
}

type assetSpec struct {
	name    string
	version string
	files   []payloadFile
	agent   *agentSpec
}

var assetSpecs = []assetSpec{
	{
		name:    "canon",
		version: "10",
		files: []payloadFile{
			{relativePath: "SKILL.md", sourcePath: "assets/canon/SKILL.md"},
		},
	},
	{
		name:    "canon-record-gate",
		version: "1",
		files: []payloadFile{
			{relativePath: "SKILL.md", sourcePath: "assets/canon-record-gate/SKILL.md"},
			{relativePath: "references/boundary-examples.md", sourcePath: "assets/canon-record-gate/references/boundary-examples.md"},
		},
		agent: &agentSpec{name: AgentName, sourcePath: "assets/canon-record-gate/agent.md"},
	},
}

// CatalogAsset describes one public bundle asset. Agent renderings are
// components of their owning skill and therefore do not appear as assets.
type CatalogAsset struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Version     string   `json:"version"`
	Hash        string   `json:"hash"`
	TargetPaths []string `json:"target_paths"`
}

// ManagedFile is one installable file rendered from the bundle.
type ManagedFile struct {
	AssetName    string
	Kind         string
	RelativePath string
	Path         string
	Version      string

	baseContent string
	hash        string
}

func (f ManagedFile) Content() string {
	return insertHashMarker(f.baseContent, f.hash)
}

func (f ManagedFile) Hash() string {
	return f.hash
}

// Inspection describes how installed content relates to one desired file.
type Inspection struct {
	Version      string
	DeclaredHash string
	ActualHash   string
	Managed      bool
	Current      bool
	Modified     bool
}

// Catalog returns the bundle's public assets in deterministic order.
func Catalog() []CatalogAsset {
	assets := make([]CatalogAsset, 0, len(assetSpecs))
	for _, spec := range sortedAssetSpecs() {
		assets = append(assets, CatalogAsset{
			Name:        spec.name,
			Kind:        KindSkill,
			Version:     spec.version,
			Hash:        assetHash(spec),
			TargetPaths: catalogTargetPaths(spec),
		})
	}
	return assets
}

// AssetNames returns all public asset names in catalog order.
func AssetNames() []string {
	specs := sortedAssetSpecs()
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		names = append(names, spec.name)
	}
	return names
}

// SupportedTargets returns all first-party agent targets in deterministic order.
func SupportedTargets() []string {
	return []string{TargetOpenCode, TargetClaude, TargetCodex}
}

// SelectAssets validates and deduplicates asset names. An empty selection
// selects the full catalog.
func SelectAssets(names []string) ([]string, error) {
	valid := make(map[string]bool, len(assetSpecs))
	for _, name := range AssetNames() {
		valid[name] = true
	}
	if len(names) == 0 {
		return AssetNames(), nil
	}
	selected := make(map[string]bool, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if !valid[name] {
			return nil, fmt.Errorf("unknown bundled skill asset %q", name)
		}
		selected[name] = true
	}
	ordered := make([]string, 0, len(selected))
	for _, name := range AssetNames() {
		if selected[name] {
			ordered = append(ordered, name)
		}
	}
	return ordered, nil
}

// NormalizeTargets validates and deduplicates target names.
func NormalizeTargets(targets []string) ([]string, error) {
	valid := make(map[string]bool, len(SupportedTargets()))
	for _, target := range SupportedTargets() {
		valid[target] = true
	}
	selected := make(map[string]bool, len(targets))
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if !valid[target] {
			return nil, fmt.Errorf("%w %q", ErrUnsupportedTarget, target)
		}
		selected[target] = true
	}
	ordered := make([]string, 0, len(selected))
	for _, target := range SupportedTargets() {
		if selected[target] {
			ordered = append(ordered, target)
		}
	}
	return ordered, nil
}

// AgentPath returns the project-relative discovery path for a target. Codex is
// a valid target but currently has no managed agent discovery file.
func AgentPath(target string) (string, error) {
	switch target {
	case TargetOpenCode:
		return filepath.Join(".opencode", "agents", AgentName+".md"), nil
	case TargetClaude:
		return filepath.Join(".claude", "agents", AgentName+".md"), nil
	case TargetCodex:
		return "", nil
	default:
		return "", fmt.Errorf("%w %q", ErrUnsupportedTarget, target)
	}
}

// ManagedFiles renders all files for the selected assets and agent targets.
func ManagedFiles(assetNames []string, skillsRoot string, targets []string) ([]ManagedFile, error) {
	selectedAssets, err := SelectAssets(assetNames)
	if err != nil {
		return nil, err
	}
	selectedTargets, err := NormalizeTargets(targets)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(skillsRoot) == "" {
		skillsRoot = DefaultSkillsRoot
	}

	files := make([]ManagedFile, 0)
	for _, assetName := range selectedAssets {
		spec := mustAssetSpec(assetName)
		for _, payload := range spec.files {
			source := normalizeContent(readAsset(payload.sourcePath))
			path := filepath.Join(skillsRoot, spec.name, filepath.FromSlash(payload.relativePath))
			files = append(files, newManagedFile(spec.name, KindSkill, payload.relativePath, path, spec.version, source))
		}
		if spec.agent == nil {
			continue
		}
		agentSource := normalizeContent(readAsset(spec.agent.sourcePath))
		for _, target := range selectedTargets {
			path, err := AgentPath(target)
			if err != nil {
				return nil, err
			}
			if path == "" {
				continue
			}
			rendered, err := renderAgent(target, agentSource)
			if err != nil {
				return nil, err
			}
			relativePath := "agents/" + target + "/" + spec.agent.name + ".md"
			files = append(files, newManagedFile(spec.name, KindAgent, relativePath, path, spec.version, rendered))
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// Inspect compares installed content with one desired managed file.
func Inspect(content string, desired ManagedFile) Inspection {
	inspection := Inspection{ActualHash: hashWithoutHashComment(content)}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "<!-- canon-skill-version:") && strings.HasSuffix(line, "-->"):
			inspection.Version = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "<!-- canon-skill-version:"), "-->"))
		case strings.HasPrefix(line, "<!-- canon-skill-hash:") && strings.HasSuffix(line, "-->"):
			inspection.DeclaredHash = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "<!-- canon-skill-hash:"), "-->"))
		}
	}
	inspection.Managed = inspection.Version != "" && inspection.DeclaredHash != "" && inspection.DeclaredHash == inspection.ActualHash
	inspection.Current = content == desired.Content()
	inspection.Modified = inspection.DeclaredHash != "" && inspection.DeclaredHash != inspection.ActualHash
	return inspection
}

func sortedAssetSpecs() []assetSpec {
	specs := append([]assetSpec(nil), assetSpecs...)
	sort.Slice(specs, func(i, j int) bool { return specs[i].name < specs[j].name })
	return specs
}

func mustAssetSpec(name string) assetSpec {
	for _, spec := range assetSpecs {
		if spec.name == name {
			return spec
		}
	}
	panic("unknown embedded skill asset: " + name)
}

func readAsset(path string) string {
	content, err := bundledAssets.ReadFile(path)
	if err != nil {
		panic("read embedded skill asset " + path + ": " + err.Error())
	}
	return string(content)
}

func newManagedFile(assetName, kind, relativePath, path, version, source string) ManagedFile {
	base := insertVersionMarker(source, version)
	return ManagedFile{
		AssetName:    assetName,
		Kind:         kind,
		RelativePath: relativePath,
		Path:         path,
		Version:      version,
		baseContent:  base,
		hash:         hashContent(base),
	}
}

func catalogTargetPaths(spec assetSpec) []string {
	paths := make([]string, 0, len(spec.files)+len(SupportedTargets()))
	for _, payload := range spec.files {
		paths = append(paths, filepath.Join(DefaultSkillsRoot, spec.name, filepath.FromSlash(payload.relativePath)))
	}
	if spec.agent != nil {
		for _, target := range []string{TargetOpenCode, TargetClaude} {
			path, err := AgentPath(target)
			if err == nil && path != "" {
				paths = append(paths, path)
			}
		}
	}
	sort.Strings(paths)
	return paths
}

func assetHash(spec assetSpec) string {
	type hashedPayload struct {
		path string
		hash string
	}
	payloads := make([]hashedPayload, 0, len(spec.files)+1)
	for _, payload := range spec.files {
		payloads = append(payloads, hashedPayload{path: payload.relativePath, hash: hashContent(normalizeContent(readAsset(payload.sourcePath)))})
	}
	if spec.agent != nil {
		payloads = append(payloads, hashedPayload{
			path: "agents/" + spec.agent.name + ".md",
			hash: hashContent(normalizeContent(readAsset(spec.agent.sourcePath))),
		})
	}
	sort.Slice(payloads, func(i, j int) bool { return payloads[i].path < payloads[j].path })

	h := sha256.New()
	for _, payload := range payloads {
		_, _ = h.Write([]byte(payload.path))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(payload.hash))
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + fmt.Sprintf("%x", h.Sum(nil))
}

func renderAgent(target, body string) (string, error) {
	const description = "Judges whether an ADR, SPEC, or Domain entry in a canon corpus earns its place. Use when asked to review, audit, gate, or stress-test a canon document before creation or after acceptance. Read-only: returns a structured verdict and never mutates the corpus."

	var frontmatter string
	switch target {
	case TargetOpenCode:
		frontmatter = "---\n" +
			"name: " + AgentName + "\n" +
			"description: \"" + description + "\"\n" +
			"mode: subagent\n" +
			"permission:\n" +
			"  edit: deny\n" +
			"  skill: allow\n" +
			"---\n\n"
	case TargetClaude:
		frontmatter = "---\n" +
			"name: " + AgentName + "\n" +
			"description: " + description + "\n" +
			"tools: Read, Grep, Glob, Bash\n" +
			"model: inherit\n" +
			"---\n\n"
	default:
		return "", fmt.Errorf("no agent rendering for target %q", target)
	}
	return frontmatter + body, nil
}

func normalizeContent(content string) string {
	return strings.TrimSpace(content) + "\n"
}

func insertVersionMarker(source, version string) string {
	marker := versionComment(version)
	if strings.HasPrefix(source, "---\n") {
		if idx := strings.Index(source[len("---\n"):], "\n---\n"); idx >= 0 {
			insertAt := len("---\n") + idx + len("\n---\n")
			return source[:insertAt] + marker + "\n" + source[insertAt:]
		}
	}
	return marker + "\n\n" + source
}

func insertHashMarker(base, hash string) string {
	lines := strings.Split(base, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "<!-- canon-skill-version:") {
			lines = append(lines[:i+1], append([]string{hashComment(hash)}, lines[i+1:]...)...)
			break
		}
	}
	return strings.Join(lines, "\n")
}

func hashWithoutHashComment(content string) string {
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!-- canon-skill-hash:") && strings.HasSuffix(trimmed, "-->") {
			continue
		}
		kept = append(kept, line)
	}
	return hashContent(strings.Join(kept, "\n"))
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + fmt.Sprintf("%x", sum[:])
}

func versionComment(version string) string {
	return "<!-- canon-skill-version: " + version + " -->"
}

func hashComment(hash string) string {
	return "<!-- canon-skill-hash: " + hash + " -->"
}
