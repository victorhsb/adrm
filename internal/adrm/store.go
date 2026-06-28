package adrm

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
	Dir string
}

func NewStore(dir string) Store {
	if dir == "" {
		dir = defaultADRDir
	}
	return Store{Dir: dir}
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
		return adrs[i].Number < adrs[j].Number
	})
	return adrs, nil
}

func (s Store) Read(id string) (ADR, error) {
	normalized, err := normalizeID(id)
	if err != nil {
		return ADR{}, err
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

func (s Store) WriteNew(title, status string, tags []string, context, decision, consequences string) (ADR, error) {
	if err := s.Init(); err != nil {
		return ADR{}, err
	}
	next, err := s.NextNumber()
	if err != nil {
		return ADR{}, err
	}
	adr := ADR{
		ID:     formatID(next),
		Number: next,
		Title:  title,
		Status: status,
		Date:   time.Now().Format("2006-01-02"),
		Tags:   cleanList(tags),
	}
	adr.Path = filepath.Join(s.Dir, fmt.Sprintf("%04d-%s.md", next, slugify(title)))
	body := renderADR(adr, context, decision, consequences)
	if err := os.WriteFile(adr.Path, []byte(body), 0o644); err != nil {
		return ADR{}, err
	}
	adr.Content = body
	return adr, nil
}

func (s Store) Save(adr ADR) error {
	if adr.Path == "" {
		return fmt.Errorf("ADR has no path")
	}
	body := renderExistingADR(adr)
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
	num, err := numberFromID(id)
	if err != nil {
		return ADR{}, err
	}
	adr := ADR{
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

func renderADR(adr ADR, context, decision, consequences string) string {
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

func renderExistingADR(adr ADR) string {
	content := adr.Content
	if !strings.HasPrefix(content, "# ") {
		content = fmt.Sprintf("# %s: %s\n\n%s", adr.ID, adr.Title, content)
	}
	return renderFrontMatter(adr) + content
}

func renderFrontMatter(adr ADR) string {
	return fmt.Sprintf(`---
id: %s
title: %s
status: %s
date: %s
tags: %s
supersedes: %s
superseded_by: %s
deprecated_by: %s
---
`, adr.ID, adr.Title, adr.Status, adr.Date, strings.Join(adr.Tags, ", "), strings.Join(adr.Supersedes, ", "), adr.SupersededBy, adr.DeprecatedBy)
}

func normalizeID(id string) (string, error) {
	id = strings.TrimSpace(strings.ToUpper(id))
	id = strings.TrimPrefix(id, "#")
	if strings.HasPrefix(id, "ADR-") {
		num, err := strconv.Atoi(strings.TrimPrefix(id, "ADR-"))
		if err != nil {
			return "", fmt.Errorf("invalid ADR id %q", id)
		}
		return formatID(num), nil
	}
	num, err := strconv.Atoi(id)
	if err != nil {
		return "", fmt.Errorf("invalid ADR id %q", id)
	}
	return formatID(num), nil
}

func numberFromID(id string) (int, error) {
	normalized, err := normalizeID(id)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimPrefix(normalized, "ADR-"))
}

func formatID(n int) string {
	return fmt.Sprintf("ADR-%04d", n)
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
