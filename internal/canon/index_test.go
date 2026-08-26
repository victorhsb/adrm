package canon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// isolateCache points the user cache directory at a temp location on every
// supported platform: XDG_CACHE_HOME on Linux, HOME on macOS (UserCacheDir
// resolves to $HOME/Library/Caches there).
func isolateCache(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "xdg-cache"))
	t.Setenv("HOME", filepath.Join(root, "home"))
	return root
}

// seedCorpus writes deterministic documents into per-kind stores under a
// temp root and returns the repo pointing at them.
func seedCorpus(t *testing.T) (Repo, string) {
	t.Helper()
	root := t.TempDir()
	repo := Repo{
		ADR:    NewStore(filepath.Join(root, "adr"), KindADR),
		Spec:   NewStore(filepath.Join(root, "spec"), KindSPEC),
		Domain: NewStore(filepath.Join(root, "domain"), KindDomain),
	}
	seeds := []struct {
		store  Store
		title  string
		status string
		tags   []string
	}{
		{repo.ADR, "Use SQLite for storage", "accepted", []string{"storage"}},
		{repo.ADR, "Cache query results", "proposed", nil},
		{repo.Spec, "Search requirements", "accepted", []string{"search"}},
		{repo.Domain, "Corpus", "accepted", []string{"glossary"}},
	}
	for _, seed := range seeds {
		if _, err := seed.store.WriteNew(seed.title, seed.status, seed.tags, map[string]string{"context": "corpus context about storage engines"}); err != nil {
			t.Fatalf("seed %s: %v", seed.title, err)
		}
	}
	// Add lifecycle relationships so parity checks cover every summary field.
	old, err := repo.ADR.Read("ADR-0001")
	if err != nil {
		t.Fatalf("read old: %v", err)
	}
	old.Status = "superseded"
	old.SupersededBy = "ADR-0002"
	if err := repo.ADR.Save(old); err != nil {
		t.Fatalf("save old: %v", err)
	}
	replacement, err := repo.ADR.Read("ADR-0002")
	if err != nil {
		t.Fatalf("read replacement: %v", err)
	}
	replacement.Supersedes = []string{"ADR-0001"}
	if err := repo.ADR.Save(replacement); err != nil {
		t.Fatalf("save replacement: %v", err)
	}
	return repo, root
}

func TestIndexDeterministicRebuild(t *testing.T) {
	repo, _ := seedCorpus(t)
	path := filepath.Join(t.TempDir(), "index.jsonl")
	first, err := buildSearchIndex(repo)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := writeSearchIndex(path, first); err != nil {
		t.Fatalf("write: %v", err)
	}
	second, err := buildSearchIndex(repo)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if err := writeSearchIndex(path, second); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	want, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(want) != string(got) {
		t.Fatal("rebuilding unchanged Markdown produced a different index")
	}
}

func TestIndexAtomicReplacementAndPermissions(t *testing.T) {
	repo, _ := seedCorpus(t)
	path := filepath.Join(t.TempDir(), "canon", "index", "index.jsonl")
	idx, err := buildSearchIndex(repo)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := writeSearchIndex(path, idx); err != nil {
		t.Fatalf("write: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("index permissions = %o, want 600", info.Mode().Perm())
	}
	// A leftover temporary file must never survive a successful write, and a
	// second write replaces the file instead of appending.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file left behind: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := writeSearchIndex(path, idx); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reread: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("replacement changed index bytes for unchanged Markdown")
	}
}

func TestReadSearchIndexStates(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.jsonl")
	if _, state, _, _ := readSearchIndex(missing); state != indexStateAbsent {
		t.Fatalf("missing file state = %q", state)
	}

	garbage := filepath.Join(dir, "garbage.jsonl")
	if err := os.WriteFile(garbage, []byte("not json\n"), 0o600); err != nil {
		t.Fatalf("write garbage: %v", err)
	}
	if _, state, _, _ := readSearchIndex(garbage); state != indexStateCorrupt {
		t.Fatalf("garbage state = %q", state)
	}

	manifest := indexManifest{Type: "manifest", SchemaVersion: "canon-index.v0", CorpusID: "c", Sources: []indexSource{}}
	line, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	unsupported := filepath.Join(dir, "unsupported.jsonl")
	if err := os.WriteFile(unsupported, append(line, '\n'), 0o600); err != nil {
		t.Fatalf("write unsupported: %v", err)
	}
	_, state, _, version := readSearchIndex(unsupported)
	if state != indexStateUnsupported {
		t.Fatalf("unsupported state = %q", state)
	}
	if version != "canon-index.v0" {
		t.Fatalf("reported version = %q", version)
	}

	manifest.SchemaVersion = indexSchemaVersion
	line, err = json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	truncated := filepath.Join(dir, "truncated.jsonl")
	if err := os.WriteFile(truncated, append(append(line, '\n'), []byte(`{"type":"docu`)...), 0o600); err != nil {
		t.Fatalf("write truncated: %v", err)
	}
	if _, state, _, _ := readSearchIndex(truncated); state != indexStateCorrupt {
		t.Fatalf("truncated state = %q", state)
	}
	if _, state, _, _ := readIndexLines(truncated); state != indexStateCorrupt {
		t.Fatalf("truncated search-load state = %q", state)
	}
}

func TestIndexFreshnessTransitions(t *testing.T) {
	repo, root := seedCorpus(t)
	idx, err := buildSearchIndex(repo)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if state, reason := indexFreshness(idx.Manifest, repo); state != indexStateFresh {
		t.Fatalf("fresh build state = %q (%s)", state, reason)
	}
	target := filepath.Join(root, "adr", "0001-use-sqlite-for-storage.md")
	original, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	originalInfo, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}

	// A content change that alters size must mark the index stale.
	if err := os.WriteFile(target, append(original, "extra content\n"...), 0o644); err != nil {
		t.Fatalf("grow target: %v", err)
	}
	if state, _ := indexFreshness(idx.Manifest, repo); state != indexStateStale {
		t.Fatalf("edited content state = %q", state)
	}

	// A pure touch (mtime change, identical content) keeps the index usable.
	fixed := time.Unix(1_700_000_000, 0)
	if err := os.WriteFile(target, original, 0o644); err != nil {
		t.Fatalf("restore target: %v", err)
	}
	if err := os.Chtimes(target, fixed, fixed); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	if state, reason := indexFreshness(idx.Manifest, repo); state != indexStateFresh {
		t.Fatalf("touched-only state = %q (%s)", state, reason)
	}

	// A same-size content change with a moved mtime must go stale: the hash
	// differs even though the size matches.
	swapped := []byte(strings.Replace(string(original), "storage", "storagz", 1))
	if len(swapped) != len(original) {
		t.Fatal("same-size rewrite precondition failed")
	}
	if err := os.WriteFile(target, swapped, 0o644); err != nil {
		t.Fatalf("swap target: %v", err)
	}
	if state, _ := indexFreshness(idx.Manifest, repo); state != indexStateStale {
		t.Fatalf("same-size content change state = %q", state)
	}

	// Restoring both content and metadata preserves the accepted false-fresh
	// case: identical bytes at the manifest size and mtime stay fresh.
	if err := os.WriteFile(target, original, 0o644); err != nil {
		t.Fatalf("restore target: %v", err)
	}
	if err := os.Chtimes(target, originalInfo.ModTime(), originalInfo.ModTime()); err != nil {
		t.Fatalf("restore mtime: %v", err)
	}
	if state, reason := indexFreshness(idx.Manifest, repo); state != indexStateFresh {
		t.Fatalf("restored state = %q (%s)", state, reason)
	}

	// Adding a Markdown file changes the path set and must go stale.
	if _, err := repo.ADR.WriteNew("New decision", "proposed", nil, map[string]string{}); err != nil {
		t.Fatalf("add doc: %v", err)
	}
	if state, _ := indexFreshness(idx.Manifest, repo); state != indexStateStale {
		t.Fatalf("added file state = %q", state)
	}
}

func TestIndexFreshnessDeleteAndRename(t *testing.T) {
	repo, root := seedCorpus(t)
	idx, err := buildSearchIndex(repo)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	// Deleting a source changes the path set.
	victim := filepath.Join(root, "domain", "0001-corpus.md")
	data, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("read victim: %v", err)
	}
	if err := os.Remove(victim); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if state, _ := indexFreshness(idx.Manifest, repo); state != indexStateStale {
		t.Fatalf("delete state = %q", state)
	}

	// Restoring the identical bytes keeps the index fresh: the mtime moved,
	// but the size and content hash match the manifest.
	if err := os.WriteFile(victim, data, 0o644); err != nil {
		t.Fatalf("restore victim: %v", err)
	}
	if state, reason := indexFreshness(idx.Manifest, repo); state != indexStateFresh {
		t.Fatalf("restored state = %q (%s)", state, reason)
	}
	if err := os.Rename(filepath.Join(root, "adr", "0002-cache-query-results.md"), filepath.Join(root, "adr", "0002-renamed.md")); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if state, _ := indexFreshness(idx.Manifest, repo); state != indexStateStale {
		t.Fatalf("rename state = %q", state)
	}
}

func TestUnreadableCacheFile(t *testing.T) {
	repo, _ := seedCorpus(t)
	idx, err := buildSearchIndex(repo)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	path := filepath.Join(t.TempDir(), "index.jsonl")
	if err := writeSearchIndex(path, idx); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if os.Geteuid() == 0 {
		t.Skip("root can read mode-000 files")
	}
	_, state, reason, _ := readIndexLines(path)
	if state != indexStateCorrupt {
		t.Fatalf("unreadable state = %q (%s)", state, reason)
	}
}

func TestIndexRebuildRejectsMalformedMarkdown(t *testing.T) {
	isolateCache(t)
	repo, root := seedCorpus(t)
	flags := []string{"--adr-dir", repo.ADR.Dir, "--spec-dir", repo.Spec.Dir, "--domain-dir", repo.Domain.Dir}
	broken := filepath.Join(root, "adr", "0099-broken.md")
	if err := os.WriteFile(broken, []byte("no front matter"), 0o644); err != nil {
		t.Fatalf("seed broken: %v", err)
	}

	// Rebuild fails rather than indexing a partial corpus.
	code, env := runForTest(t, append(flags, "index", "rebuild")...)
	if code != exitIO {
		t.Fatalf("rebuild code = %d, want %d", code, exitIO)
	}
	if env["error"].(map[string]any)["code"] != "index_build_failed" {
		t.Fatalf("error = %#v", env["error"])
	}

	// Deep validation still isolates the malformed file through Markdown,
	// never through the index.
	code, env = runForTest(t, append(flags, "validate")...)
	if code != exitState {
		t.Fatalf("validate code = %d, want %d", code, exitState)
	}
	findings := env["data"].(map[string]any)["findings"].([]any)
	var sawMalformed bool
	for _, raw := range findings {
		if raw.(map[string]any)["name"] == "malformed_file" {
			sawMalformed = true
		}
	}
	if !sawMalformed {
		t.Fatalf("expected malformed_file finding: %#v", findings)
	}

	// The malformed file must not poison read commands either: like before
	// the index existed, list surfaces the parse failure.
	if code, _ := runForTest(t, append(flags, "list")...); code != exitIO {
		t.Fatalf("list code = %d, want %d", code, exitIO)
	}
}

func TestIndexRebuildDryRunCreatesNothing(t *testing.T) {
	cacheRoot := isolateCache(t)
	repo, root := seedCorpus(t)
	code, env := runForTest(t, "--adr-dir", repo.ADR.Dir, "--spec-dir", repo.Spec.Dir, "--domain-dir", repo.Domain.Dir, "index", "rebuild", "--dry-run")
	if code != exitOK || env["status"] != "planned" {
		t.Fatalf("dry-run code=%d env=%#v", code, env)
	}
	warnings, ok := env["warnings"].([]any)
	if !ok || len(warnings) == 0 || warnings[0] != "No changes were made." {
		t.Fatalf("dry-run warnings = %#v", env["warnings"])
	}
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		t.Fatalf("read cache root: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("dry run created cache entries: %#v", entries)
	}
	if _, err := os.Stat(filepath.Join(root, "canon")); err == nil {
		t.Fatal("rebuild wrote into the corpus root")
	}
}

func TestIndexStatusTransitionsThroughCLI(t *testing.T) {
	isolateCache(t)
	repo, _ := seedCorpus(t)
	flags := []string{"--adr-dir", repo.ADR.Dir, "--spec-dir", repo.Spec.Dir, "--domain-dir", repo.Domain.Dir}

	code, env := runForTest(t, append(flags, "index", "status")...)
	if code != exitOK {
		t.Fatalf("status code = %d", code)
	}
	data := env["data"].(map[string]any)
	if data["state"] != indexStateAbsent {
		t.Fatalf("state = %v", data["state"])
	}

	code, env = runForTest(t, append(flags, "index", "rebuild")...)
	if code != exitOK {
		t.Fatalf("rebuild code = %d env=%#v", code, env)
	}
	data = env["data"].(map[string]any)
	if data["documents"].(float64) != 4 {
		t.Fatalf("documents = %v", data["documents"])
	}

	code, env = runForTest(t, append(flags, "index", "status")...)
	if code != exitOK {
		t.Fatalf("status code = %d", code)
	}
	data = env["data"].(map[string]any)
	if data["state"] != indexStateFresh {
		t.Fatalf("state = %v, reason=%v", data["state"], data["reason"])
	}
	if data["documents"].(float64) != 4 {
		t.Fatalf("documents = %v", data["documents"])
	}
	if data["schema_version"] != indexSchemaVersion {
		t.Fatalf("schema_version = %v", data["schema_version"])
	}
}

func TestIndexedSearchParityWithMarkdown(t *testing.T) {
	isolateCache(t)
	repo, _ := seedCorpus(t)
	flags := []string{"--adr-dir", repo.ADR.Dir, "--spec-dir", repo.Spec.Dir, "--domain-dir", repo.Domain.Dir}
	if code, env := runForTest(t, append(flags, "index", "rebuild")...); code != exitOK {
		t.Fatalf("rebuild code = %d env=%#v", code, env)
	}

	cases := [][]string{
		{"search", "--query", "storage"},
		{"search", "--query", "STORAGE"},
		{"search", "--query", "adr"},
		{"search", "--query", "adr-0001"},
		{"search", "--status", "accepted"},
		{"search", "--tag", "storage"},
		{"search", "--query", "xyz-no-match"},
		{"adr", "search", "--query", "storage"},
		{"spec", "search", "--query", "requirements"},
		{"domain", "search", "--query", "corpus"},
		{"domain", "search", "--status", "accepted"},
	}
	for _, args := range cases {
		code, plain := runForTest(t, append(flags, args...)...)
		if code != exitOK {
			t.Fatalf("%v plain code = %d", args, code)
		}
		indexed := append(append([]string{}, args...), "--use-index")
		code, withIndex := runForTest(t, append(flags, indexed...)...)
		if code != exitOK {
			t.Fatalf("%v indexed code = %d env=%#v", args, code, withIndex)
		}
		if _, warned := withIndex["warnings"]; warned {
			t.Fatalf("%v indexed search warned on a fresh index: %#v", args, withIndex["warnings"])
		}
		plainData, _ := json.Marshal(plain["data"])
		indexedData, _ := json.Marshal(withIndex["data"])
		if string(plainData) != string(indexedData) {
			t.Fatalf("%v parity mismatch\nplain:   %s\nindexed: %s", args, plainData, indexedData)
		}
	}
}

func TestIndexedSearchFallsBackWithWarning(t *testing.T) {
	cacheRoot := isolateCache(t)
	repo, _ := seedCorpus(t)
	flags := []string{"--adr-dir", repo.ADR.Dir, "--spec-dir", repo.Spec.Dir, "--domain-dir", repo.Domain.Dir}

	// Absent index: identical results, non-fatal fallback warning.
	code, plain := runForTest(t, append(flags, "search", "--query", "storage")...)
	if code != exitOK {
		t.Fatalf("plain code = %d", code)
	}
	code, env := runForTest(t, append(flags, "search", "--query", "storage", "--use-index")...)
	if code != exitOK {
		t.Fatalf("fallback code = %d env=%#v", code, env)
	}
	warnings, _ := env["warnings"].([]any)
	if len(warnings) != 1 || !strings.Contains(warnings[0].(string), "fell back to Markdown") {
		t.Fatalf("fallback warning = %#v", env["warnings"])
	}
	plainData, _ := json.Marshal(plain["data"])
	envData, _ := json.Marshal(env["data"])
	if string(plainData) != string(envData) {
		t.Fatalf("absent index changed results\nplain:   %s\nfallback: %s", plainData, envData)
	}

	// Corrupt index at the resolved path: same behavior.
	path, err := indexPathFor(repo)
	if err != nil {
		t.Fatalf("index path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("broken\n"), 0o600); err != nil {
		t.Fatalf("corrupt index: %v", err)
	}
	code, env = runForTest(t, append(flags, "search", "--query", "storage", "--use-index")...)
	if code != exitOK {
		t.Fatalf("corrupt fallback code = %d", code)
	}
	warnings, _ = env["warnings"].([]any)
	if len(warnings) != 1 || !strings.Contains(warnings[0].(string), "corrupt") {
		t.Fatalf("corrupt fallback warning = %#v", env["warnings"])
	}
	envData, _ = json.Marshal(env["data"])
	if string(plainData) != string(envData) {
		t.Fatal("corrupt index changed results")
	}

	// Stale index: edit Markdown after a rebuild.
	if code, _ := runForTest(t, append(flags, "index", "rebuild")...); code != exitOK {
		t.Fatalf("rebuild before stale check")
	}
	target := filepath.Join(repo.ADR.Dir, "0001-use-sqlite-for-storage.md")
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if err := os.WriteFile(target, append(body, "fresh sentence\n"...), 0o644); err != nil {
		t.Fatalf("edit target: %v", err)
	}
	code, staleEnv := runForTest(t, append(flags, "search", "--query", "fresh sentence", "--use-index")...)
	if code != exitOK {
		t.Fatalf("stale fallback code = %d", code)
	}
	warnings, _ = staleEnv["warnings"].([]any)
	if len(warnings) != 1 || !strings.Contains(warnings[0].(string), "stale") {
		t.Fatalf("stale fallback warning = %#v", staleEnv["warnings"])
	}
	staleData := staleEnv["data"].(map[string]any)
	if staleData["count"].(float64) != 1 {
		t.Fatalf("stale fallback must see the edited Markdown, count = %v", staleData["count"])
	}

	// The index path lives under the isolated cache, never the corpus.
	if !strings.HasPrefix(path, cacheRoot) {
		t.Fatalf("index path %s not under cache root %s", path, cacheRoot)
	}
}

func TestIndexedSearchParityWithTrickyQueries(t *testing.T) {
	isolateCache(t)
	repo, _ := seedCorpus(t)
	// Seed content with JSON-special and HTML characters so prescan and
	// decode paths both get exercised across formats.
	tricky := NewStore(repo.ADR.Dir, KindADR)
	if _, err := tricky.WriteNew(`Quoted "decision" <tag> & more`, "accepted", nil, map[string]string{"context": "escapes: \"quoted\" back\\slash <html> & ampersand"}); err != nil {
		t.Fatalf("seed tricky: %v", err)
	}
	flags := []string{"--adr-dir", repo.ADR.Dir, "--spec-dir", repo.Spec.Dir, "--domain-dir", repo.Domain.Dir}
	if code, env := runForTest(t, append(flags, "index", "rebuild")...); code != exitOK {
		t.Fatalf("rebuild code = %d env=%#v", code, env)
	}
	queries := []string{
		`"quoted"`,        // JSON-escaped in the record line: decode-all path
		`back\slash`,      // backslash: decode-all path
		"<html>",          // HTML chars stay literal (SetEscapeHTML off)
		"adr adr-0001",    // query spanning the kind/id field boundary
		"STORAGE engines", // multi-word case-folded query
	}
	for _, query := range queries {
		code, plain := runForTest(t, append(flags, "search", "--query", query)...)
		if code != exitOK {
			t.Fatalf("%q plain code = %d", query, code)
		}
		code, indexed := runForTest(t, append(flags, "search", "--query", query, "--use-index")...)
		if code != exitOK {
			t.Fatalf("%q indexed code = %d env=%#v", query, code, indexed)
		}
		if _, warned := indexed["warnings"]; warned {
			t.Fatalf("%q warned on a fresh index: %#v", query, indexed["warnings"])
		}
		plainData, _ := json.Marshal(plain["data"])
		indexedData, _ := json.Marshal(indexed["data"])
		if string(plainData) != string(indexedData) {
			t.Fatalf("%q parity mismatch\nplain:   %s\nindexed: %s", query, plainData, indexedData)
		}
	}
}

func TestSearchIndexDocumentsCorruptMatchingLine(t *testing.T) {
	if _, err := encodeJSONValue(indexManifest{Type: "manifest", SchemaVersion: indexSchemaVersion, CorpusID: "c", Sources: []indexSource{}}); err != nil {
		t.Fatalf("encode manifest: %v", err)
	}
	record, err := encodeJSONValue(indexRecord{Type: "document", Kind: KindADR, ID: "ADR-0001", Number: 1, Title: "Broken", Content: json.RawMessage(`"broken content"`), SearchText: "adr adr-0001 broken broken content"})
	if err != nil {
		t.Fatalf("encode record: %v", err)
	}
	garbage := []byte(`{"type":"document","id":"ADR-0002" garbage broken`)

	// A malformed line that cannot match the query is skipped by the prescan;
	// only its bytes matter, and the exact result is still correct.
	lines := indexLines{manifest: indexManifest{}, lines: [][]byte{record, garbage}}
	docs, warning := searchIndexDocuments(lines, "", "", "", "content")
	if warning != "" {
		t.Fatalf("non-matching garbage warned: %s", warning)
	}
	if len(docs) != 1 || docs[0].ID != "ADR-0001" {
		t.Fatalf("docs = %#v", docs)
	}

	// The same garbage line contains the query bytes, so a skipped decode
	// could hide results; the search must fall back instead.
	if _, warning := searchIndexDocuments(lines, "", "", "", "garbage"); warning == "" {
		t.Fatal("matching garbage must fall back to Markdown with a warning")
	}
}

func TestIndexRebuildEmptyCorpus(t *testing.T) {
	isolateCache(t)
	root := t.TempDir()
	flags := []string{
		"--adr-dir", filepath.Join(root, "adr"),
		"--spec-dir", filepath.Join(root, "spec"),
		"--domain-dir", filepath.Join(root, "domain"),
	}
	code, env := runForTest(t, append(flags, "index", "rebuild")...)
	if code != exitOK {
		t.Fatalf("rebuild code = %d env=%#v", code, env)
	}
	if docs := env["data"].(map[string]any)["documents"].(float64); docs != 0 {
		t.Fatalf("documents = %v", docs)
	}
	code, env = runForTest(t, append(flags, "search", "--query", "anything", "--use-index")...)
	if code != exitOK {
		t.Fatalf("search code = %d", code)
	}
	if _, warned := env["warnings"]; warned {
		t.Fatalf("empty fresh index warned: %#v", env["warnings"])
	}
	if count := env["data"].(map[string]any)["count"].(float64); count != 0 {
		t.Fatalf("count = %v", count)
	}
}
