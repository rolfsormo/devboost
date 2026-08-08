// Package config reads devboost's user-facing ~/.devboost.yaml. This stays
// YAML deliberately — it's a pre-existing, user-authored file, unrelated to
// the "module resource declarations are Go struct literals, not YAML"
// decision, which only concerns how devboost's own modules declare
// resources internally.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Config is a loaded ~/.devboost.yaml. Load it once at CLI startup and
// pass it to whatever needs it — no package-level global state, so tests
// can load an arbitrary fixture path without env-var tricks.
type Config struct {
	data map[string]any
}

// Load reads and parses the YAML file at path. A missing file is not an
// error — it's treated as an empty config, so every Get call falls back
// to its default. Only a malformed file (present but unparsable) errors.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{data: map[string]any{}}, nil
		}
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return &Config{data: m}, nil
}

// DefaultPath returns ~/.devboost.yaml for the current user.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".devboost.yaml"
	}
	return filepath.Join(home, ".devboost.yaml")
}

// Get reads a dotted key path (e.g. "zsh.znap_path"), returning def if
// the key or any intermediate segment is absent. String values (both
// real ones and def itself) are expanded for a leading ~, since defaults
// like "~/.zsh-snap" need that too — expansion happens exactly once,
// regardless of which path returned the value.
//
// A non-string scalar (bool, int, float — e.g. "enable: false" written
// as a real YAML boolean, not a quoted string) is stringified the same
// way the bash tool's yq-based reader renders it in plain output ("true"/
// "false", plain decimal), so config authors can write natural YAML
// without needing to know devboost's Get treats everything as text
// underneath. A map or list value (wrong shape for a scalar key) falls
// back to def, same as a missing key.
func (c *Config) Get(dottedKey string, def string) string {
	return expandHome(c.get(dottedKey, def))
}

func (c *Config) get(dottedKey string, def string) string {
	cur := any(c.data)
	for _, part := range strings.Split(strings.Trim(dottedKey, "."), ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return def
		}
		v, ok := m[part]
		if !ok {
			return def
		}
		cur = v
	}
	switch v := cur.(type) {
	case string:
		return v
	case bool, int, int64, float64:
		return fmt.Sprintf("%v", v)
	default:
		return def
	}
}

func expandHome(s string) string {
	if s == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return s
	}
	if len(s) >= 2 && s[0] == '~' && s[1] == '/' {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, s[2:])
		}
	}
	return s
}
