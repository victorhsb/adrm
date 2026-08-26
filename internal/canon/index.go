package canon

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// This file implements the optional rebuildable search index (ADR-0017). The
// index is a derived artifact under the user cache directory; Markdown files
// remain the authoritative corpus. Validation, list, show, and lifecycle
// commands never read it. Both the indexed and the Markdown-backed search
// paths apply the same selectors and produce equal results.

// indexSchemaVersion pins the JSONL format. A file with any other version is
// reported as unsupported and ignored, never migrated in place.
const indexSchemaVersion = "canon-index.v1"

// Index states reported by `canon index status` and used for search fallback.
const (
	indexStateAbsent      = "absent"
	indexStateFresh       = "fresh"
	indexStateStale       = "stale"
	indexStateCorrupt     = "corrupt"
	indexStateUnsupported = "unsupported"
)

// indexManifest is the first JSONL line: schema version, corpus identity,
// and per-source metadata used for freshness checks.
type indexManifest struct {
	Type          string        `json:"type"`
	SchemaVersion string        `json:"schema_version"`
	CorpusID      string        `json:"corpus_id"`
	Sources       []indexSource `json:"sources"`
}

// indexSource captures one Markdown source file's freshness metadata. Path is
// the cleaned absolute path so statting at search time is independent of the
// working directory.
type indexSource struct {
	Path            string `json:"path"`
	Size            int64  `json:"size"`
	ModTimeUnixNano int64  `json:"mod_time_unix_nano"`
	SHA256          string `json:"sha256"`
}

// indexRecord is one document line after the manifest. It carries every
// field the search payload needs plus the precomputed lowercase searchable
// text matching adrMatches, so the indexed path never touches Markdown.
// Content stays a raw encoded JSON string: filtering runs over the
// searchable text and metadata, and only matched documents pay the content
// decode cost for snippets.
type indexRecord struct {
	Type         string          `json:"type"`
	Kind         string          `json:"kind"`
	ID           string          `json:"id"`
	Number       int             `json:"number"`
	Title        string          `json:"title"`
	Status       string          `json:"status"`
	Date         string          `json:"date"`
	Tags         []string        `json:"tags,omitempty"`
	Supersedes   []string        `json:"supersedes,omitempty"`
	SupersededBy string          `json:"superseded_by,omitempty"`
	DeprecatedBy string          `json:"deprecated_by,omitempty"`
	Path         string          `json:"path"`
	AbsPath      string          `json:"abs_path"`
	Content      json.RawMessage `json:"content"`
	SearchText   string          `json:"search_text"`
}

// document projects a record back into the document shape the search payload
// renders, identical to what the Markdown reader returns. A record whose
// content fails to decode is treated as empty rather than failing the
// search; freshness checks already guarantee structural validity at load
// time.
func (r indexRecord) document() Document {
	var content string
	_ = json.Unmarshal(r.Content, &content)
	return Document{
		Kind:         r.Kind,
		ID:           r.ID,
		Number:       r.Number,
		Title:        r.Title,
		Status:       r.Status,
		Date:         r.Date,
		Tags:         r.Tags,
		Supersedes:   r.Supersedes,
		SupersededBy: r.SupersededBy,
		DeprecatedBy: r.DeprecatedBy,
		Path:         r.Path,
		Content:      content,
	}
}

// searchIndex is a fully parsed index file: manifest plus document records
// in stable kind and number order. The rebuild path constructs it; the
// status command parses into it. Search loads lines instead, so it only
// decodes records that can match the query.
type searchIndex struct {
	Manifest indexManifest
	Records  []indexRecord
}

// indexLines is the search-path view of an index file: the decoded manifest
// plus the raw record lines. Records are decoded on demand because decoding
// every record costs more than parsing the Markdown corpus at typical
// corpus sizes.
type indexLines struct {
	manifest indexManifest
	lines    [][]byte
}

// indexCorpusID identifies the configured corpus: the cleaned absolute paths
// of all three managed store directories joined by newlines. Absent stores
// still contribute their path, so the index identity is stable regardless of
// which kinds are initialized.
func indexCorpusID(repo Repo) (string, error) {
	dirs := make([]string, 0, 3)
	for _, store := range []Store{repo.ADR, repo.Spec, repo.Domain} {
		abs, err := filepath.Abs(store.Dir)
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", store.Dir, err)
		}
		dirs = append(dirs, filepath.Clean(abs))
	}
	return strings.Join(dirs, "\n"), nil
}

// indexPathFor locates the cache file for the configured corpus: the hex
// SHA-256 of the corpus id under os.UserCacheDir/canon/index.
func indexPathFor(repo Repo) (string, error) {
	corpusID, err := indexCorpusID(repo)
	if err != nil {
		return "", err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate user cache directory: %w", err)
	}
	sum := sha256.Sum256([]byte(corpusID))
	return filepath.Join(cache, "canon", "index", hex.EncodeToString(sum[:])+".jsonl"), nil
}

// searchableText reproduces the adrMatches haystack exactly, so the indexed
// filter and the Markdown filter apply the same case-insensitive substring
// query over the same fields.
func searchableText(doc Document) string {
	return strings.ToLower(strings.Join([]string{
		doc.Kind,
		doc.ID,
		doc.Title,
		doc.Status,
		strings.Join(doc.Tags, " "),
		doc.Content,
	}, " "))
}

// buildSearchIndex reads every present managed kind through DocumentReader
// and collects document records and source freshness metadata in stable kind
// and number order. Rebuilding unchanged Markdown yields an identical index.
func buildSearchIndex(repo Repo) (searchIndex, error) {
	corpusID, err := indexCorpusID(repo)
	if err != nil {
		return searchIndex{}, err
	}
	docs, err := listDocuments(repo.ADR, repo.Spec, repo.Domain)
	if err != nil {
		return searchIndex{}, err
	}
	idx := searchIndex{
		Manifest: indexManifest{
			Type:          "manifest",
			SchemaVersion: indexSchemaVersion,
			CorpusID:      corpusID,
			Sources:       []indexSource{},
		},
		Records: []indexRecord{},
	}
	for _, doc := range docs {
		abs, err := filepath.Abs(doc.Path)
		if err != nil {
			return searchIndex{}, fmt.Errorf("resolve %s: %w", doc.Path, err)
		}
		abs = filepath.Clean(abs)
		source, err := statSource(abs)
		if err != nil {
			return searchIndex{}, err
		}
		content, err := encodeJSONValue(doc.Content)
		if err != nil {
			return searchIndex{}, fmt.Errorf("encode content of %s: %w", doc.ID, err)
		}
		idx.Manifest.Sources = append(idx.Manifest.Sources, source)
		idx.Records = append(idx.Records, indexRecord{
			Type:         "document",
			Kind:         doc.Kind,
			ID:           doc.ID,
			Number:       doc.Number,
			Title:        doc.Title,
			Status:       doc.Status,
			Date:         doc.Date,
			Tags:         doc.Tags,
			Supersedes:   doc.Supersedes,
			SupersededBy: doc.SupersededBy,
			DeprecatedBy: doc.DeprecatedBy,
			Path:         doc.Path,
			AbsPath:      abs,
			Content:      content,
			SearchText:   searchableText(doc),
		})
	}
	return idx, nil
}

// encodeJSONValue encodes one value without HTML escaping, so searchable
// text in the index holds the same bytes as the lowered document text. The
// raw-line prescan relies on this: a query without JSON-special characters
// appears verbatim in the record line exactly when it appears in the
// searchable text.
func encodeJSONValue(v any) (json.RawMessage, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return json.RawMessage(bytes.TrimRight(buf.Bytes(), "\n")), nil
}

// statSource hashes one Markdown file and captures its size and modification
// time.
func statSource(abs string) (indexSource, error) {
	info, err := os.Stat(abs)
	if err != nil {
		return indexSource{}, fmt.Errorf("stat %s: %w", abs, err)
	}
	file, err := os.Open(abs)
	if err != nil {
		return indexSource{}, fmt.Errorf("open %s: %w", abs, err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return indexSource{}, fmt.Errorf("hash %s: %w", abs, err)
	}
	return indexSource{
		Path:            abs,
		Size:            info.Size(),
		ModTimeUnixNano: info.ModTime().UnixNano(),
		SHA256:          hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

// writeSearchIndex replaces the index file atomically: it writes a temporary
// file in the same cache directory, syncs, closes, and renames. The file
// holds document content, so the directory and file use restrictive
// permissions. An interrupted build leaves at most an orphaned temporary
// file; it never replaces a valid index with a partial one.
func writeSearchIndex(path string, idx searchIndex) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create index directory: %w", err)
	}
	tmp, err := os.OpenFile(path+".tmp", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create temporary index: %w", err)
	}
	encoder := json.NewEncoder(tmp)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(idx.Manifest); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("encode index manifest: %w", err)
	}
	for _, record := range idx.Records {
		if err := encoder.Encode(record); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			return fmt.Errorf("encode index record %s: %w", record.ID, err)
		}
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("sync index: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("close index: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("replace index: %w", err)
	}
	return nil
}

// readSearchIndexFile reads an index file, returning the raw bytes with the
// absent and empty states already classified. Callers decode either the full
// record set (status, tests) or raw lines (search).
func readSearchIndexFile(path string) (data []byte, state, reason string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, indexStateAbsent, "no index file"
		}
		return nil, indexStateCorrupt, fmt.Sprintf("cannot read index: %v", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, indexStateCorrupt, "index is empty"
	}
	return raw, "", ""
}

// decodeIndexManifest parses the first JSONL line and enforces the schema
// version gate.
func decodeIndexManifest(line []byte) (manifest indexManifest, state, reason string) {
	if err := json.Unmarshal(line, &manifest); err != nil || manifest.Type != "manifest" {
		return indexManifest{}, indexStateCorrupt, "index manifest is not valid JSON"
	}
	if manifest.SchemaVersion != indexSchemaVersion {
		return manifest, indexStateUnsupported, fmt.Sprintf("schema version %q is not supported (expected %q)", manifest.SchemaVersion, indexSchemaVersion)
	}
	return manifest, "", ""
}

// readSearchIndex parses an index file fully, validating every record. The
// returned state is empty on a structurally valid file, allowing the caller
// to run freshness checks; any other state is one of absent, corrupt, or
// unsupported with a human-readable reason. The schema version is reported
// whenever the first line decoded.
func readSearchIndex(path string) (idx searchIndex, state, reason, version string) {
	data, state, reason := readSearchIndexFile(path)
	if state != "" {
		return searchIndex{}, state, reason, ""
	}
	lines := bytes.Split(data, []byte{'\n'})
	manifest, state, reason := decodeIndexManifest(lines[0])
	if state != "" {
		return searchIndex{}, state, reason, manifest.SchemaVersion
	}
	idx = searchIndex{Manifest: manifest, Records: []indexRecord{}}
	for i, line := range lines[1:] {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var record indexRecord
		if err := json.Unmarshal(line, &record); err != nil || record.Type != "document" {
			return searchIndex{}, indexStateCorrupt, fmt.Sprintf("index record on line %d is not valid JSON", i+2), manifest.SchemaVersion
		}
		idx.Records = append(idx.Records, record)
	}
	return idx, "", "", manifest.SchemaVersion
}

// readIndexLines parses an index file for the search path: the manifest and
// the raw record lines, without decoding records. Every line the writer
// emits ends with a newline, so a final line without one is truncated and
// the file is corrupt. Mid-file structural corruption is caught when a
// matching record fails to decode; `index status` and rebuild validate every
// record.
func readIndexLines(path string) (lines indexLines, state, reason, version string) {
	data, state, reason := readSearchIndexFile(path)
	if state != "" {
		return indexLines{}, state, reason, ""
	}
	if data[len(data)-1] != '\n' {
		return indexLines{}, indexStateCorrupt, "index is truncated", ""
	}
	raw := bytes.Split(data, []byte{'\n'})
	manifest, state, reason := decodeIndexManifest(raw[0])
	if state != "" {
		return indexLines{}, state, reason, manifest.SchemaVersion
	}
	return indexLines{manifest: manifest, lines: raw[1 : len(raw)-1]}, "", "", manifest.SchemaVersion
}

// indexFreshness compares the current corpus with the manifest: the present
// source path set must match, and each file is trusted while its size and
// modification time match the manifest; a file whose size or modification
// time changed is rehashed and only invalidates the index when its content
// actually differs. A content change preserving both size and modification
// time is the accepted false-fresh limitation from ADR-0017.
func indexFreshness(manifest indexManifest, repo Repo) (state, reason string) {
	manifestSources := make(map[string]indexSource, len(manifest.Sources))
	for _, source := range manifest.Sources {
		manifestSources[source.Path] = source
	}
	current, err := currentSourcePaths(repo)
	if err != nil {
		return indexStateStale, err.Error()
	}
	if len(current) != len(manifestSources) {
		return indexStateStale, "the set of Markdown files changed"
	}
	for _, abs := range current {
		source, ok := manifestSources[abs]
		if !ok {
			return indexStateStale, "the set of Markdown files changed"
		}
		info, err := os.Stat(abs)
		if err != nil {
			return indexStateStale, fmt.Sprintf("source %s is unreadable: %v", abs, err)
		}
		if info.Size() == source.Size && info.ModTime().UnixNano() == source.ModTimeUnixNano {
			continue
		}
		fresh, err := statSource(abs)
		if err != nil {
			return indexStateStale, fmt.Sprintf("source %s is unreadable: %v", abs, err)
		}
		if fresh.SHA256 != source.SHA256 {
			return indexStateStale, fmt.Sprintf("source %s changed", abs)
		}
	}
	return indexStateFresh, ""
}

// currentSourcePaths returns the cleaned absolute path of every Markdown
// file in every present managed store.
func currentSourcePaths(repo Repo) ([]string, error) {
	paths := []string{}
	for _, store := range []Store{repo.ADR, repo.Spec, repo.Domain} {
		if !store.Exists() {
			continue
		}
		entries, err := os.ReadDir(store.Dir)
		if err != nil {
			return nil, fmt.Errorf("read %s: %v", store.Dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			abs, err := filepath.Abs(filepath.Join(store.Dir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("resolve %s: %v", entry.Name(), err)
			}
			paths = append(paths, filepath.Clean(abs))
		}
	}
	return paths, nil
}

// loadFreshIndexLines returns the index manifest and raw record lines when
// the index is present, structurally sound, supported, and fresh. Every
// other outcome yields a fallback warning naming the state, so indexed
// search can degrade to the Markdown reader without hiding results.
func loadFreshIndexLines(repo Repo) (indexLines, string) {
	path, err := indexPathFor(repo)
	if err != nil {
		return indexLines{}, fmt.Sprintf("Search index unavailable (%v); fell back to Markdown search.", err)
	}
	lines, state, reason, _ := readIndexLines(path)
	if state == indexStateAbsent {
		return indexLines{}, fmt.Sprintf("Search index absent at %s; fell back to Markdown search. Run `canon index rebuild` to create it.", path)
	}
	if state == "" {
		state, reason = indexFreshness(lines.manifest, repo)
	}
	if state != indexStateFresh {
		return indexLines{}, fmt.Sprintf("Search index %s (%s); fell back to Markdown search. Run `canon index rebuild` to refresh it.", state, reason)
	}
	return lines, ""
}

// searchIndexDocuments resolves the indexed search path over raw record
// lines: a case-folded byte prescan skips records that cannot match the
// query, matching lines decode into records, and the exact filter applies
// the selectors. A matching line that fails to decode means corruption that
// could hide results, so the search falls back to Markdown with a warning.
// The returned documents equal what the Markdown path returns.
func searchIndexDocuments(lines indexLines, kind, status, tag, query string) ([]Document, string) {
	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	prescan := lowerQuery != "" && prescanSafe(lowerQuery)
	needle := []byte(lowerQuery)
	var records []indexRecord
	for _, line := range lines.lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if prescan && !bytes.Contains(line, needle) {
			continue
		}
		var record indexRecord
		if err := json.Unmarshal(line, &record); err != nil || record.Type != "document" {
			return nil, "Search index corrupt (a matching record is not valid JSON); fell back to Markdown search. Run `canon index rebuild` to refresh it."
		}
		if kind != "" && record.Kind != kind {
			continue
		}
		records = append(records, record)
	}
	matched := filterIndexRecords(records, status, tag, query)
	docs := make([]Document, 0, len(matched))
	for _, record := range matched {
		docs = append(docs, record.document())
	}
	return docs, ""
}

// prescanSafe reports whether a lowered query can be byte-matched against
// raw record lines. The index writes searchable text without HTML escaping,
// so the only characters that would appear transformed in a line are the
// JSON escapes: quote, backslash, control characters, and U+2028/U+2029. A
// query without them byte-occurs in a line exactly when it occurs in the
// record's searchable text, so prescanning can only over-select lines;
// queries with them decode every record and filter exactly.
func prescanSafe(lowerQuery string) bool {
	if strings.ContainsAny(lowerQuery, "\"\\") {
		return false
	}
	if strings.ContainsRune(lowerQuery, ' ') || strings.ContainsRune(lowerQuery, ' ') {
		return false
	}
	return !strings.ContainsFunc(lowerQuery, unicode.IsControl)
}

// filterIndexRecords mirrors filterADRs over index records, applying the
// same status and tag filters with the same case behavior and the same
// case-insensitive substring query, but over the precomputed searchable
// text. Records stay in their stable kind and number order.
func filterIndexRecords(records []indexRecord, status, tag, query string) []indexRecord {
	status = normalizeStatus(status)
	tag = strings.TrimSpace(tag)
	query = strings.ToLower(strings.TrimSpace(query))
	var out []indexRecord
	for _, record := range records {
		if status != "" && record.Status != status {
			continue
		}
		if tag != "" && !contains(record.Tags, tag) {
			continue
		}
		if query != "" && !strings.Contains(record.SearchText, query) {
			continue
		}
		out = append(out, record)
	}
	return out
}
