// Package schema exposes the published Claude Code settings schema.
//
// The schema is authoritative for what settings.json may contain, but nothing
// in a normal workflow ever runs it against a real file.
package schema

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed settings.schema.json
var raw []byte

type document struct {
	Properties map[string]struct {
		Properties map[string]json.RawMessage `json:"properties"`
	} `json:"properties"`
}

var (
	once       sync.Once
	settingKey map[string]bool
	hookEvent  map[string]bool
)

func load() {
	once.Do(func() {
		var doc document
		if err := json.Unmarshal(raw, &doc); err != nil {
			panic("embedded settings schema does not parse: " + err.Error())
		}
		settingKey = make(map[string]bool, len(doc.Properties))
		for name := range doc.Properties {
			settingKey[name] = true
		}
		hookEvent = make(map[string]bool)
		for name := range doc.Properties["hooks"].Properties {
			hookEvent[name] = true
		}
	})
}

// SettingKeys are the top-level keys settings.json is defined to carry.
func SettingKeys() map[string]bool {
	load()
	return settingKey
}

// HookEvents are the lifecycle events a hooks block may key on.
func HookEvents() map[string]bool {
	load()
	return hookEvent
}

// Raw is the embedded schema, for callers that validate against it in full.
func Raw() []byte { return raw }
