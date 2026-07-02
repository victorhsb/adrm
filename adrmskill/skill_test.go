package adrmskill

import (
	"strings"
	"testing"
)

func TestVersionBumped(t *testing.T) {
	if Version != "2" {
		t.Fatalf("expected skill version 2, got %q", Version)
	}
}

func TestContentContainsADRGuidanceSection(t *testing.T) {
	content := Content()
	for _, want := range []string{
		"## When to create or change an ADR",
		"commitment, not an intention",
		"It is architectural",
		"It is non-obvious",
		"It is narrow",
		"## Technical vs product decisions",
		"### `adrm` trigger list",
		"## Anti-patterns",
		"roadmap",
		"ticket",
		"changelog entry",
		"bundle of unrelated decisions",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("skill content missing %q:\n%s", want, content)
		}
	}
}

func TestContentContainsVersionAndHashComments(t *testing.T) {
	content := Content()
	if !strings.Contains(content, "<!-- adrm-skill-version: "+Version+" -->") {
		t.Fatalf("skill content missing version comment")
	}
	if !strings.Contains(content, "<!-- adrm-skill-hash: sha256:") {
		t.Fatalf("skill content missing hash comment")
	}
}
