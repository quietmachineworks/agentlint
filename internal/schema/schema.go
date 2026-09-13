// Package schema exposes the published Claude Code settings schema.
//
// The schema is authoritative for what settings.json may contain, but nothing
// in a normal workflow ever runs it against a real file.
package schema

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"sync"
)

//go:embed settings.schema.json
var raw []byte

type document struct {
	Properties map[string]struct {
		Description string                     `json:"description"`
		Properties  map[string]json.RawMessage `json:"properties"`
	} `json:"properties"`
}

// Restriction is the set of scopes a key is read from, when the schema says a
// key is not read everywhere.
type Restriction struct {
	ManagedOnly bool
	Scopes      []string
}

var (
	once        sync.Once
	settingKey  map[string]bool
	hookEvent   map[string]bool
	restriction map[string]Restriction
)

var (
	managedOnly  = regexp.MustCompile(`(?i)\((?:admin/)?managed[^)]*only\)`)
	notInProject = regexp.MustCompile(`(?i)not from \.claude/settings\.json|ignored in project and local settings`)
	fromUserOnly = regexp.MustCompile(`(?i)honored (?:only )?from (?:mdm|user, managed)`)
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
		restriction = make(map[string]Restriction)
		for name, property := range doc.Properties {
			switch {
			case managedOnly.MatchString(property.Description):
				restriction[name] = Restriction{ManagedOnly: true, Scopes: []string{"managed"}}
			case notInProject.MatchString(property.Description) || fromUserOnly.MatchString(property.Description):
				restriction[name] = Restriction{Scopes: []string{"user", "managed"}}
			}
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

// Restrictions are the keys the schema says are read from some scopes only.
func Restrictions() map[string]Restriction {
	load()
	return restriction
}

// Raw is the embedded schema, for callers that validate against it in full.
func Raw() []byte { return raw }
