package check

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/quietmachineworks/agentlint/internal/inventory"
	"github.com/quietmachineworks/agentlint/internal/schema"
)

// SettingsScope reports a key written where the runtime does not read it.
//
// Precedence explains a key that loses to another copy. This is the other
// failure: a key with no competing copy at all, sitting in a file that is read,
// under a name that is read, doing nothing because the level is wrong. Nothing
// warns about it, and the file it sits in looks correct.
type SettingsScope struct{}

func (SettingsScope) Name() string { return "settings-scope" }

// honoured maps a scope onto the name the schema uses for it.
func honoured(scope inventory.Scope) string {
	switch scope {
	case inventory.ScopeManaged:
		return "managed"
	case inventory.ScopeUser, inventory.ScopeUserLocal:
		return "user"
	default:
		return "project"
	}
}

func (SettingsScope) Run(inv *inventory.Inventory) []Finding {
	restrictions := schema.Restrictions()
	var findings []Finding
	for _, settings := range inv.Settings {
		if !settings.Parsed {
			continue
		}
		level := honoured(settings.Scope)
		keys := make([]string, 0, len(settings.Raw))
		for key := range settings.Raw {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			restriction, restricted := restrictions[key]
			if !restricted || slices.Contains(restriction.Scopes, level) {
				continue
			}
			findings = append(findings, Finding{
				Check: "settings-scope", Severity: Error, File: settings.Path, Where: key,
				Message: fmt.Sprintf("is read from %s settings only, and this file is %s settings, so it does nothing here",
					strings.Join(restriction.Scopes, " or "), settings.Scope.Name),
				Fix: fmt.Sprintf("move %q to a settings file the runtime reads it from, or drop it", key),
			})
		}
	}
	return findings
}
