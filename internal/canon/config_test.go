package canon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripJSONComments(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"line comment", "{\n// note\n}\n", "{\n\n}\n"},
		{"trailing line comment", "{\"a\": 1} // done", "{\"a\": 1} "},
		{"block comment", "{/* hidden */}", "{}"},
		{"block comment keeps newlines", "{/* a\nb */}", "{\n}"},
		{"slashes inside string stay", "{\"url\": \"https://x/*y\"}", "{\"url\": \"https://x/*y\"}"},
		{"escaped quote in string", "{\"a\": \"\\\"//\"}", "{\"a\": \"\\\"//\"}"},
		{"no comments", "{\"a\": 1}", "{\"a\": 1}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := string(stripJSONComments([]byte(tc.input))); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLoadConfigDefaultsWhenAbsent(t *testing.T) {
	t.Parallel()
	cfg, path, err := LoadConfig(t.TempDir())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if path != "" {
		t.Fatalf("path = %q, want empty", path)
	}
	if !cfg.AppendEnabled() {
		t.Fatal("append should default to enabled")
	}
}

func TestLoadConfigDiscoversAncestor(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	deep := filepath.Join(root, "docs", "adr")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	data := "{\n// no appendices here\n\"conventions\": {\"append\": false}\n}\n"
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, path, err := LoadConfig(deep)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if path != filepath.Join(root, configFileName) {
		t.Fatalf("path = %q", path)
	}
	if cfg.AppendEnabled() {
		t.Fatal("append should be disabled")
	}
}

func TestLoadConfigReportsMalformedFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, path, err := LoadConfig(root)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if path == "" {
		t.Fatal("expected the offending path")
	}
}

// TestAppendDisabledByConfig exercises the append gate end to end: a corpus
// whose repo-root config sets conventions.append=false rejects both apply and
// dry-run attempts with a config error.
func TestAppendDisabledByConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	adrDir := filepath.Join(root, "docs", "adr")
	if err := os.MkdirAll(adrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := "{\n// edit documents directly; git is the history\n\"schema_version\": \"canon.v1\",\n\"conventions\": {\"append\": false}\n}\n"
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _ := runForTest(t, "--adr-dir", adrDir, "adr", "new", "--title", "Use SQLite"); code != exitOK {
		t.Fatalf("adr new code = %d", code)
	}
	code, env := runForTest(t, "--adr-dir", adrDir, "append", "--id", "ADR-0001", "--title", "Review", "--body", "Still valid.")
	if code != exitState {
		t.Fatalf("code = %d, want %d", code, exitState)
	}
	errObj := env["error"].(map[string]any)
	if errObj["code"] != "append_disabled" || errObj["category"] != "config" {
		t.Fatalf("error = %#v", errObj)
	}
	code, env = runForTest(t, "--adr-dir", adrDir, "append", "--id", "ADR-0001", "--title", "Review", "--body", "Still valid.", "--dry-run")
	if code != exitState {
		t.Fatalf("dry-run code = %d, want %d", code, exitState)
	}
	if env["error"].(map[string]any)["code"] != "append_disabled" {
		t.Fatalf("dry-run error = %#v", env["error"])
	}
	code, env = runForTest(t, "--adr-dir", adrDir, "show", "--id", "ADR-0001")
	if code != exitOK {
		t.Fatalf("show code = %d", code)
	}
	if actions, ok := env["next_actions"].([]any); ok {
		for _, raw := range actions {
			if strings.Contains(raw.(map[string]any)["command"].(string), "append") {
				t.Fatalf("show suggests disabled append: %#v", raw)
			}
		}
	}
}

func TestAppendEnabledWithoutConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if code, _ := runForTest(t, "--adr-dir", dir, "adr", "new", "--title", "Use SQLite"); code != exitOK {
		t.Fatalf("adr new code = %d", code)
	}
	if code, env := runForTest(t, "--adr-dir", dir, "append", "--id", "ADR-0001", "--title", "Review", "--body", "Still valid."); code != exitOK {
		t.Fatalf("append code = %d, error = %#v", code, env["error"])
	}
}
