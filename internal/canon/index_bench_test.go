package canon

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// benchCorpus generates a corpus large enough to expose Markdown parsing and
// filesystem costs: documents per managed kind with multi-kilobyte bodies.
func benchCorpus(b *testing.B, docsPerKind int) Repo {
	b.Helper()
	root := b.TempDir()
	repo := Repo{
		ADR:    NewStore(filepath.Join(root, "adr"), KindADR),
		Spec:   NewStore(filepath.Join(root, "spec"), KindSPEC),
		Domain: NewStore(filepath.Join(root, "domain"), KindDomain),
	}
	body := strings.Repeat("Architecture context and consequences for the decision corpus benchmark. ", 60)
	for _, store := range []Store{repo.ADR, repo.Spec, repo.Domain} {
		for i := 1; i <= docsPerKind; i++ {
			if _, err := store.WriteNew(fmt.Sprintf("Decision %s %04d", store.Kind, i), "accepted", []string{"bench"}, map[string]string{"context": body}); err != nil {
				b.Fatalf("seed %s %d: %v", store.Kind, i, err)
			}
		}
	}
	return repo
}

func benchMarkdownSearch(b *testing.B, perKind int) {
	repo := benchCorpus(b, perKind)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		docs, err := docsForKind(repo, "")
		if err != nil {
			b.Fatalf("load: %v", err)
		}
		_ = searchResults(filterADRs(docs, "", "", "spec 0100"), "spec 0100")
	}
}

// benchIndexedSearch mirrors the `search --use-index` path on a fresh index:
// raw-line load, freshness check, prescan, decode of matching records, and
// exact filtering.
func benchIndexedSearch(b *testing.B, perKind int) {
	repo := benchCorpus(b, perKind)
	idx, err := buildSearchIndex(repo)
	if err != nil {
		b.Fatalf("build: %v", err)
	}
	path := filepath.Join(b.TempDir(), "index.jsonl")
	if err := writeSearchIndex(path, idx); err != nil {
		b.Fatalf("write: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lines, state, reason, _ := readIndexLines(path)
		if state != "" {
			b.Fatalf("state = %q (%s)", state, reason)
		}
		if state, reason := indexFreshness(lines.manifest, repo); state != indexStateFresh {
			b.Fatalf("freshness = %q (%s)", state, reason)
		}
		docs, warning := searchIndexDocuments(lines, "", "", "", "spec 0100")
		if warning != "" {
			b.Fatalf("warning = %s", warning)
		}
		_ = searchResults(docs, "spec 0100")
	}
}

func BenchmarkSearchMarkdown600Docs(b *testing.B)  { benchMarkdownSearch(b, 200) }
func BenchmarkSearchIndexed600Docs(b *testing.B)   { benchIndexedSearch(b, 200) }
func BenchmarkSearchMarkdown30Docs(b *testing.B)   { benchMarkdownSearch(b, 10) }
func BenchmarkSearchIndexed30Docs(b *testing.B)    { benchIndexedSearch(b, 10) }
func BenchmarkSearchMarkdown2000Docs(b *testing.B) { benchMarkdownSearch(b, 667) }
func BenchmarkSearchIndexed2000Docs(b *testing.B)  { benchIndexedSearch(b, 667) }
