package canon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Store struct {
	Dir  string
	Kind string
}

func NewStore(dir, kind string) Store {
	if kind == "" {
		kind = KindADR
	}
	if dir == "" {
		switch kind {
		case KindSPEC:
			dir = defaultSpecDir
		case KindDomain:
			dir = defaultDomainDir
		default:
			dir = defaultADRDir
		}
	}
	return Store{Dir: dir, Kind: kind}
}

func (s Store) Exists() bool {
	info, err := os.Stat(s.Dir)
	return err == nil && info.IsDir()
}

func (s Store) Init() error {
	return os.MkdirAll(s.Dir, 0o755)
}

func (s Store) List() ([]ADR, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}
	var adrs []ADR
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		adr, err := s.ReadPath(filepath.Join(s.Dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		adrs = append(adrs, adr)
	}
	sort.Slice(adrs, func(i, j int) bool {
		if adrs[i].Kind != adrs[j].Kind {
			return adrs[i].Kind < adrs[j].Kind
		}
		return adrs[i].Number < adrs[j].Number
	})
	return adrs, nil
}

func (s Store) Read(id string) (ADR, error) {
	kind, normalized, err := normalizeID(id)
	if err != nil {
		return ADR{}, err
	}
	if kind != "" && kind != s.Kind {
		return ADR{}, os.ErrNotExist
	}
	adrs, err := s.List()
	if err != nil {
		return ADR{}, err
	}
	for _, adr := range adrs {
		if adr.ID == normalized {
			return adr, nil
		}
	}
	return ADR{}, os.ErrNotExist
}

func (s Store) ReadPath(path string) (ADR, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return ADR{}, err
	}
	adr, err := parseADR(string(b))
	if err != nil {
		return ADR{}, fmt.Errorf("%s: %w", path, err)
	}
	if adr.Kind == "" {
		adr.Kind = s.Kind
	}
	adr.Path = path
	return adr, nil
}

func (s Store) NextNumber() (int, error) {
	adrs, err := s.List()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 1, nil
		}
		return 0, err
	}
	max := 0
	for _, adr := range adrs {
		if adr.Number > max {
			max = adr.Number
		}
	}
	return max + 1, nil
}

func (s Store) WriteNew(title, status string, tags []string, sections map[string]string) (ADR, error) {
	if err := s.Init(); err != nil {
		return ADR{}, err
	}
	next, err := s.NextNumber()
	if err != nil {
		return ADR{}, err
	}
	adr := ADR{
		Kind:   s.Kind,
		ID:     formatID(s.Kind, next),
		Number: next,
		Title:  title,
		Status: status,
		Date:   time.Now().Format("2006-01-02"),
		Tags:   cleanList(tags),
	}
	adr.Path = filepath.Join(s.Dir, fmt.Sprintf("%04d-%s.md", next, slugify(title)))
	var body string
	switch s.Kind {
	case KindSPEC:
		body = renderSPEC(adr, sections)
	case KindDomain:
		body = renderDomain(adr, sections)
	default:
		body = renderADR(adr, sections)
	}
	if err := os.WriteFile(adr.Path, []byte(body), 0o644); err != nil {
		return ADR{}, err
	}
	adr.Content = body
	return adr, nil
}

func (s Store) Save(adr ADR) error {
	if adr.Path == "" {
		return fmt.Errorf("document has no path")
	}
	body := renderExisting(adr)
	return os.WriteFile(adr.Path, []byte(body), 0o644)
}

func parseADR(body string) (ADR, error) {
	if !strings.HasPrefix(body, "---\n") {
		return ADR{}, fmt.Errorf("missing front matter")
	}
	rest := strings.TrimPrefix(body, "---\n")
	parts := strings.SplitN(rest, "\n---\n", 2)
	if len(parts) != 2 {
		return ADR{}, fmt.Errorf("unterminated front matter")
	}
	meta := map[string]string{}
	for _, line := range strings.Split(parts[0], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		meta[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	id := meta["id"]
	kind := normalizeKind(meta["kind"])
	if kind == "" {
		kind = kindFromID(id)
	}
	if kind == "" {
		kind = KindADR
	}
	num, err := numberFromID(id)
	if err != nil {
		return ADR{}, err
	}
	adr := ADR{
		Kind:         kind,
		ID:           id,
		Number:       num,
		Title:        meta["title"],
		Status:       meta["status"],
		Date:         meta["date"],
		Tags:         parseList(meta["tags"]),
		Supersedes:   parseList(meta["supersedes"]),
		SupersededBy: meta["superseded_by"],
		DeprecatedBy: meta["deprecated_by"],
		Content:      parts[1],
	}
	return adr, nil
}

func renderADR(adr ADR, sections map[string]string) string {
	context := sections["context"]
	decision := sections["decision"]
	consequences := sections["consequences"]
	return renderFrontMatter(adr) + fmt.Sprintf(`# %s: %s

## Status

%s

## Context

%s

## Decision

%s

## Consequences

%s
`, adr.ID, adr.Title, adr.Status, defaultText(context, "TBD"), defaultText(decision, "TBD"), defaultText(consequences, "TBD"))
}

func renderSPEC(adr ADR, sections map[string]string) string {
	context := sections["context"]
	requirements := sections["requirements"]
	constraints := sections["constraints"]
	acceptance := sections["acceptance"]
	return renderFrontMatter(adr) + fmt.Sprintf(`# %s: %s

## Status

%s

## Context

%s

## Requirements

%s

## Constraints

%s

## Acceptance Criteria

%s
`, adr.ID, adr.Title, adr.Status, defaultText(context, "TBD"), defaultText(requirements, "TBD"), defaultText(constraints, "TBD"), defaultText(acceptance, "TBD"))
}

func renderDomain(adr ADR, sections map[string]string) string {
	definition := sections["definition"]
	avoid := renderAvoidList(sections["avoid"])
	relationships := sections["relationships"]
	return renderFrontMatter(adr) + fmt.Sprintf(`# %s: %s

## Status

%s

## Definition

%s

## Avoid

%s

## Relationships

%s
`, adr.ID, adr.Title, adr.Status, defaultText(definition, "TBD"), avoid, defaultText(relationships, "TBD"))
}

// renderAvoidList renders the --avoid flag value as a markdown bullet list.
// Entries are separated by semicolons; each entry is either "term" or
// "term: reason", rendered as "- **term** — reason" so every avoided term
// carries the reason it is not canonical.
func renderAvoidList(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "TBD"
	}
	var b strings.Builder
	for _, entry := range strings.Split(value, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		term, reason, hasReason := strings.Cut(entry, ":")
		term = strings.TrimSpace(term)
		reason = strings.TrimSpace(reason)
		if hasReason && reason != "" {
			fmt.Fprintf(&b, "- **%s** — %s\n", term, reason)
			continue
		}
		fmt.Fprintf(&b, "- **%s**\n", term)
	}
	out := strings.TrimRight(b.String(), "\n")
	if out == "" {
		return "TBD"
	}
	return out
}

func renderExisting(adr ADR) string {
	content := adr.Content
	if !strings.HasPrefix(content, "# ") {
		content = fmt.Sprintf("# %s: %s\n\n%s", adr.ID, adr.Title, content)
	}
	return renderFrontMatter(adr) + content
}

func renderFrontMatter(adr ADR) string {
	return fmt.Sprintf(`---
kind: %s
id: %s
title: %s
status: %s
date: %s
tags: %s
supersedes: %s
superseded_by: %s
deprecated_by: %s
---
`, adr.Kind, adr.ID, adr.Title, adr.Status, adr.Date, strings.Join(adr.Tags, ", "), strings.Join(adr.Supersedes, ", "), adr.SupersededBy, adr.DeprecatedBy)
}

// normalizeID parses a user-supplied id and returns the kind it refers to
// (empty when the input is a bare number, which defaults to ADR), the
// normalized id, and an error.
func normalizeID(id string) (kind, normalized string, err error) {
	id = strings.TrimSpace(strings.ToUpper(id))
	id = strings.TrimPrefix(id, "#")
	for _, k := range []string{KindADR, KindSPEC, KindDomain} {
		prefix := kindPrefix(k)
		if strings.HasPrefix(id, prefix) {
			num, parseErr := strconv.Atoi(strings.TrimPrefix(id, prefix))
			if parseErr != nil {
				return "", "", fmt.Errorf("invalid id %q", id)
			}
			return k, formatID(k, num), nil
		}
	}
	num, parseErr := strconv.Atoi(id)
	if parseErr != nil {
		return "", "", fmt.Errorf("invalid id %q", id)
	}
	return "", formatID(KindADR, num), nil
}

func numberFromID(id string) (int, error) {
	_, normalized, err := normalizeID(id)
	if err != nil {
		return 0, err
	}
	for _, k := range []string{KindADR, KindSPEC, KindDomain} {
		prefix := kindPrefix(k)
		if strings.HasPrefix(normalized, prefix) {
			return strconv.Atoi(strings.TrimPrefix(normalized, prefix))
		}
	}
	return 0, fmt.Errorf("invalid id %q", id)
}

func formatID(kind string, n int) string {
	return fmt.Sprintf("%s%04d", kindPrefix(kind), n)
}

func kindPrefix(kind string) string {
	switch kind {
	case KindSPEC:
		return PrefixSPEC
	case KindDomain:
		return PrefixDomain
	default:
		return PrefixADR
	}
}

func kindFromID(id string) string {
	upper := strings.ToUpper(strings.TrimSpace(id))
	switch {
	case strings.HasPrefix(upper, PrefixSPEC):
		return KindSPEC
	case strings.HasPrefix(upper, PrefixDomain):
		return KindDomain
	case strings.HasPrefix(upper, PrefixADR):
		return KindADR
	default:
		return ""
	}
}

func normalizeKind(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case KindADR, KindSPEC, KindDomain:
		return value
	default:
		return ""
	}
}

func parseList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return cleanList(strings.Split(value, ","))
}

func cleanList(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "untitled"
	}
	return slug
}

func defaultText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
