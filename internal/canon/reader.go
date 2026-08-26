package canon

import (
	"errors"
	"sort"
)

// DocumentReader is the narrow read seam for parsed documents. It covers
// only the two query capabilities that read aggregation needs: listing every
// document and reading one by id. Readiness checks, directory inspection,
// writes, and number allocation stay on the concrete Store, because they are
// filesystem operations rather than document queries.
type DocumentReader interface {
	List() ([]Document, error)
	Read(string) (Document, error)
}

// Store satisfies DocumentReader without an adapter.
var _ DocumentReader = Store{}

// Backend-neutral read errors. Read-only consumers classify failures with
// these sentinels instead of filesystem-specific errors such as
// os.ErrNotExist.
var (
	// ErrDocumentNotFound wraps a read for a valid but absent id, including
	// an id that belongs to another kind.
	ErrDocumentNotFound = errors.New("document not found")
	// ErrStoreUnavailable wraps a list against a store whose backing
	// directory is absent. Missing optional stores are healthy; aggregation
	// helpers skip them instead of failing.
	ErrStoreUnavailable = errors.New("store unavailable")
)

// listDocuments combines documents from the given readers in stable kind and
// number order. A reader that reports ErrStoreUnavailable is skipped, so a
// missing optional store lists as empty; every other error aborts.
func listDocuments(readers ...DocumentReader) ([]Document, error) {
	var docs []Document
	for _, reader := range readers {
		listed, err := reader.List()
		if err != nil {
			if errors.Is(err, ErrStoreUnavailable) {
				continue
			}
			return nil, err
		}
		docs = append(docs, listed...)
	}
	sortDocuments(docs)
	return docs, nil
}

// sortDocuments orders documents by kind, then number, matching the
// deterministic ordering the combined list and search commands have always
// produced.
func sortDocuments(docs []Document) {
	sort.Slice(docs, func(i, j int) bool {
		if docs[i].Kind != docs[j].Kind {
			return docs[i].Kind < docs[j].Kind
		}
		return docs[i].Number < docs[j].Number
	})
}
