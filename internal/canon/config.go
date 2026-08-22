package canon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// configFileName is the repo-root configuration file. It is discovered by
// walking up from a document store directory (for example docs/adr), so the
// configuration always belongs to the corpus it sits above. There is no
// global configuration; a repository without the file gets all defaults.
const configFileName = ".canon.jsonc"

// Config holds repository-level conventions loaded from .canon.jsonc. The
// file is JSONC (JSON with // and /* */ comments); pointers distinguish an
// unset key from an explicit value so defaults stay in code, not in the file.
type Config struct {
	SchemaVersion string `json:"schema_version"`
	Conventions   struct {
		Append *bool `json:"append"`
	} `json:"conventions"`
}

// AppendEnabled reports whether the append command is allowed. Append is
// enabled unless the repository config explicitly sets
// conventions.append to false.
func (c Config) AppendEnabled() bool {
	return c.Conventions.Append == nil || *c.Conventions.Append
}

// LoadConfig searches for .canon.jsonc starting at startDir and walking up
// toward the filesystem root. It returns the parsed config and the path it
// came from; when no file exists anywhere up the chain it returns a zero
// Config (all defaults enabled) with an empty path.
func LoadConfig(startDir string) (Config, string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return Config{}, "", fmt.Errorf("resolve %s: %w", startDir, err)
	}
	for {
		path := filepath.Join(dir, configFileName)
		data, err := os.ReadFile(path)
		switch {
		case err == nil:
			cfg, err := parseConfig(data)
			if err != nil {
				return Config{}, path, err
			}
			return cfg, path, nil
		case errors.Is(err, os.ErrNotExist):
			parent := filepath.Dir(dir)
			if parent == dir {
				return Config{}, "", nil
			}
			dir = parent
		default:
			return Config{}, path, fmt.Errorf("read %s: %w", path, err)
		}
	}
}

// parseConfig decodes JSONC content into a Config after stripping comments.
func parseConfig(data []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(stripJSONComments(data), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// stripJSONComments removes // line comments and /* */ block comments that
// appear outside string literals. Newlines inside block comments are kept so
// parse error positions still line up with the source file.
func stripJSONComments(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	escaped := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		if c == '/' && i+1 < len(data) {
			switch data[i+1] {
			case '/':
				for i+1 < len(data) && data[i+1] != '\n' {
					i++
				}
				continue
			case '*':
				i += 2
				for i < len(data) && !(data[i] == '*' && i+1 < len(data) && data[i+1] == '/') {
					if data[i] == '\n' {
						out = append(out, '\n')
					}
					i++
				}
				i++
				continue
			}
		}
		out = append(out, c)
	}
	return out
}
