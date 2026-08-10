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
	cur, ok := c.lookup(dottedKey)
	if !ok {
		return def
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

// GetList reads a dotted key path expected to hold a YAML list of
// strings (e.g. "packages.base"), returning nil if the key is absent or
// isn't a list. Non-string list items are stringified the same way Get
// stringifies scalars; items that are neither a string nor a plain
// scalar are skipped rather than erroring, so one malformed entry
// doesn't take down reading the whole list.
func (c *Config) GetList(dottedKey string) []string {
	cur, ok := c.lookup(dottedKey)
	if !ok {
		return nil
	}
	items, ok := cur.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range items {
		switch v := item.(type) {
		case string:
			out = append(out, v)
		case bool, int, int64, float64:
			out = append(out, fmt.Sprintf("%v", v))
		}
	}
	return out
}

// lookup walks a dotted key path through the loaded config, returning the
// raw value at that path (whatever type it happens to be) and whether it
// was found at all.
func (c *Config) lookup(dottedKey string) (any, bool) {
	cur := any(c.data)
	for _, part := range strings.Split(strings.Trim(dottedKey, "."), ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := m[part]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// Set writes value at dottedKey, creating intermediate maps as needed —
// the write-side mirror of lookup's read-side walk, so a value set here
// is indistinguishable from Get's perspective from one the user wrote in
// ~/.devboost.yaml. Used for CLI-flag overrides (e.g. --no-optimizations)
// that need to behave exactly as if the user had written the equivalent
// config key: Get can't tell the difference afterward, which is the
// point — one code path (Get's existing default-handling) stays the only
// place that decides what a key's absence/presence means, rather than a
// second, parallel "was this overridden by a flag" concept.
func (c *Config) Set(dottedKey string, value any) {
	parts := strings.Split(strings.Trim(dottedKey, "."), ".")
	cur := c.data
	for _, part := range parts[:len(parts)-1] {
		next, ok := cur[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[part] = next
		}
		cur = next
	}
	cur[parts[len(parts)-1]] = value
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
