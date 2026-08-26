package canon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeReader is a DocumentReader implementation that serves in-memory
// documents and injected failures, so read aggregation and domain integrity
// are testable without touching the filesystem.
type fakeReader struct {
	docs    []Document
	listErr error
	readErr error
}

func (f fakeReader) List() ([]Document, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.docs, nil
}

func (f fakeReader) Read(id string) (Document, error) {
	if f.readErr != nil {
		return Document{}, f.readErr
	}
	_, normalized, err := normalizeID(id)
	if err != nil {
		return Document{}, err
	}
	for _, doc := range f.docs {
		if doc.ID == normalized {
			return doc, nil
		}
	}
	return Document{}, fmt.Errorf("%w: %s", ErrDocumentNotFound, normalized)
}

func TestStoreSatisfiesDocumentReader(t *testing.T) {
	var reader DocumentReader = NewStore(t.TempDir(), KindADR)
	if _, err := reader.List(); err != nil {
		t.Fatalf("List on empty dir: %v", err)
	}
}

func TestStoreListAbsentDirWrapsStoreUnavailable(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "missing"), KindADR)
	_, err := store.List()
	if !errors.Is(err, ErrStoreUnavailable) {
		t.Fatalf("expected ErrStoreUnavailable, got %v", err)
	}
	if errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("store-absent must not classify as document-absent: %v", err)
	}
}

func TestStoreReadAbsentIDWrapsDocumentNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, KindADR)
	if _, err := store.WriteNew("Some decision", "proposed", nil, map[string]string{}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := store.Read("ADR-0099"); !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("expected ErrDocumentNotFound, got %v", err)
	}
}

func TestStoreReadWrongKindWrapsDocumentNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, KindADR)
	if _, err := store.WriteNew("Some decision", "proposed", nil, map[string]string{}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := store.Read("DM-0001")
	if !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("expected ErrDocumentNotFound for wrong-kind id, got %v", err)
	}
}

func TestStoreListParseFailureIsNotSentinel(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "0001-broken.md"), []byte("no front matter"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	store := NewStore(dir, KindADR)
	_, err := store.List()
	if err == nil {
		t.Fatal("expected parse error")
	}
	if errors.Is(err, ErrStoreUnavailable) || errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("parse failure must stay a cause, not a sentinel: %v", err)
	}
}

func TestStoreListNonExistDirErrorIsNotSentinel(t *testing.T) {
	// A regular file used as the store dir triggers a ReadDir failure that is
	// neither absence nor a parse problem; it must pass through unwrapped.
	path := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	store := NewStore(path, KindADR)
	_, err := store.List()
	if err == nil {
		t.Fatal("expected ReadDir error")
	}
	if errors.Is(err, ErrStoreUnavailable) {
		t.Fatalf("non-absence directory error must not classify as unavailable: %v", err)
	}
}

func TestStoreNextNumberAbsentDirStartsAtOne(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "missing"), KindADR)
	next, err := store.NextNumber()
	if err != nil {
		t.Fatalf("NextNumber: %v", err)
	}
	if next != 1 {
		t.Fatalf("next = %d, want 1", next)
	}
}

func TestListDocumentsSkipsUnavailableReaders(t *testing.T) {
	present := fakeReader{docs: []Document{{Kind: KindADR, ID: "ADR-0001", Number: 1, Title: "One"}}}
	missing := fakeReader{listErr: fmt.Errorf("%w: ghost", ErrStoreUnavailable)}
	docs, err := listDocuments(present, missing)
	if err != nil {
		t.Fatalf("listDocuments: %v", err)
	}
	if len(docs) != 1 || docs[0].ID != "ADR-0001" {
		t.Fatalf("docs = %#v", docs)
	}
}

func TestListDocumentsPropagatesFailures(t *testing.T) {
	broken := fakeReader{listErr: errors.New("disk on fire")}
	if _, err := listDocuments(broken); err == nil || !strings.Contains(err.Error(), "disk on fire") {
		t.Fatalf("expected propagated error, got %v", err)
	}
	unparseable := fakeReader{listErr: errors.New("missing front matter")}
	if _, err := listDocuments(fakeReader{}, unparseable); err == nil {
		t.Fatal("expected propagated parse error")
	}
}

func TestListDocumentsDeterministicCrossReaderOrder(t *testing.T) {
	a := fakeReader{docs: []Document{
		{Kind: KindDomain, ID: "DM-0002", Number: 2, Title: "Two"},
		{Kind: KindADR, ID: "ADR-0002", Number: 2, Title: "B"},
	}}
	b := fakeReader{docs: []Document{
		{Kind: KindDomain, ID: "DM-0001", Number: 1, Title: "One"},
		{Kind: KindADR, ID: "ADR-0001", Number: 1, Title: "A"},
		{Kind: KindSPEC, ID: "SPEC-0001", Number: 1, Title: "S"},
	}}
	docs, err := listDocuments(a, b)
	if err != nil {
		t.Fatalf("listDocuments: %v", err)
	}
	want := []string{"ADR-0001", "ADR-0002", "DM-0001", "DM-0002", "SPEC-0001"}
	if len(docs) != len(want) {
		t.Fatalf("got %d docs, want %d", len(docs), len(want))
	}
	for i, id := range want {
		if docs[i].ID != id {
			t.Fatalf("docs[%d] = %s, want %s (full: %#v)", i, docs[i].ID, id, docs)
		}
	}
}

func TestDomainIntegrityChecksWithFakeReaders(t *testing.T) {
	domain := fakeReader{docs: []Document{
		{Kind: KindDomain, ID: "DM-0001", Number: 1, Title: "Corpus", Status: "accepted"},
		{Kind: KindDomain, ID: "DM-0002", Number: 2, Title: "corpus", Status: "accepted"},
		{Kind: KindDomain, ID: "DM-0003", Number: 3, Title: "Old", Status: "superseded", SupersededBy: "DM-0001"},
	}}
	live := fakeReader{docs: []Document{
		{Kind: KindADR, ID: "ADR-0001", Number: 1, Title: "Uses old", Status: "accepted", Content: "See DM-0003 for details."},
	}}
	missing := fakeReader{listErr: fmt.Errorf("%w: ghost", ErrStoreUnavailable)}

	checks := domainIntegrityChecks(domain, live, missing)
	var sawDuplicate, sawDeadRef bool
	for _, check := range checks {
		switch check.Name {
		case "domain_duplicate_title":
			sawDuplicate = true
		case "domain_dead_reference":
			sawDeadRef = true
		}
	}
	if !sawDuplicate {
		t.Fatalf("expected duplicate-title finding: %#v", checks)
	}
	if !sawDeadRef {
		t.Fatalf("expected dead-reference finding: %#v", checks)
	}
}

func TestDomainIntegrityChecksToleratesUnreadableDomain(t *testing.T) {
	broken := fakeReader{listErr: errors.New("permission denied")}
	if checks := domainIntegrityChecks(broken, fakeReader{}); checks != nil {
		t.Fatalf("expected nil checks, got %#v", checks)
	}
}
