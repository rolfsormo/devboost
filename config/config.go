// Package config reads devboost's user-facing ~/.devboost.yaml. This stays
// YAML deliberately — it's a pre-existing, user-authored file, unrelated to
// the "module resource declarations are Go struct literals, not YAML"
// decision, which only concerns how devboost's own modules declare
// resources internally.
package config

import (
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

var loaded map[string]any
var loadErr error
var didLoad bool

func path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".devboost.yaml"
	}
	return filepath.Join(home, ".devboost.yaml")
}

func load() {
	didLoad = true
	data, err := os.ReadFile(path())
	if err != nil {
		if os.IsNotExist(err) {
			loaded = map[string]any{}
			return
		}
		loadErr = err
		return
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		loadErr = err
		return
	}
	loaded = m
}

// Get reads a dotted key path (e.g. "zsh.znap_path") from
// ~/.devboost.yaml, returning def if the file, key, or any intermediate
// segment is absent. It never errors on a missing file or key — only a
// malformed file is surfaced, via GetErr.
func Get(dottedKey string, def string) string {
	return expandHome(get(dottedKey, def))
}

// get returns the raw (un-expanded) value, so expansion happens exactly
// once regardless of which return path was taken — including the default,
// since defaults like "~/.zsh-snap" need expansion too.
func get(dottedKey string, def string) string {
	if !didLoad {
		load()
	}
	if loadErr != nil {
		return def
	}
	cur := any(loaded)
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
	s, ok := cur.(string)
	if !ok {
		return def
	}
	return s
}

// GetErr reports a parse error from the last Get call, if any (e.g. a
// malformed ~/.devboost.yaml). Callers that want to distinguish "using the
// default because nothing was configured" from "using the default because
// the config file is broken" should check this after calling Get.
func GetErr() error {
	if !didLoad {
		load()
	}
	return loadErr
}

func expandHome(s string) string {
	if s == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
		return s
	}
	if len(s) >= 2 && s[0] == '~' && s[1] == '/' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, s[2:])
		}
	}
	return s
}
