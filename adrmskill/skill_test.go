package adrmskill

import (
	"strings"
	"testing"
)

func TestVersionBumped(t *testing.T) {
	if Version != "3" {
		t.Fatalf("expected skill version 3, got %q", Version)
	}
}

func TestCommandExamplesDoNotRepeatDryRun(t *testing.T) {
	content := Content()
	const marker = "## Common commands"
	idx := strings.Index(content, marker)
	if idx < 0 {
		t.Fatalf("skill content missing %q section:\n%s", marker, content)
	}
	// Dry-run discipline is stated once in the operating rules; command
	// example lines must not repeat --dry-run variants.
	for _, line := range strings.Split(content[idx:], "\n") {
		if strings.HasPrefix(line, "adrm ") && strings.Contains(line, "--dry-run") {
			t.Fatalf("command example repeats --dry-run: %q", line)
		}
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
