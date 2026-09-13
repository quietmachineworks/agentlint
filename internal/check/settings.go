package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/quietmachineworks/agentlint/internal/inventory"
	"github.com/quietmachineworks/agentlint/internal/schema"
)

// SettingsKeys catches a misspelt top-level key. The published schema sets
// additionalProperties to true, so nothing else ever will.
type SettingsKeys struct{}

func (SettingsKeys) Name() string { return "settings-unknown-key" }

func (SettingsKeys) Run(inv *inventory.Inventory) []Finding {
	known := schema.SettingKeys()
	var findings []Finding
	for _, settings := range inv.Settings {
		if !settings.Parsed {
			continue
		}
		keys := make([]string, 0, len(settings.Raw))
		for key := range settings.Raw {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if known[key] || strings.HasPrefix(key, "$") {
				continue
			}
			if suggestion, ok := nearest(key, known); ok {
				findings = append(findings, Finding{
					Check: "settings-unknown-key", Severity: Error, File: settings.Path, Where: key,
					Message: fmt.Sprintf("%q is not a setting; %q differs by one edit", key, suggestion),
					Fix:     fmt.Sprintf("rename %q to %q", key, suggestion),
				})
				continue
			}
			findings = append(findings, Finding{
				Check: "settings-unknown-key", Severity: Warning, File: settings.Path, Where: key,
				Message: fmt.Sprintf("%q is not in the published schema; the runtime reads it only if it is undocumented", key),
			})
		}
	}
	return findings
}

// SettingsHookEvents catches a hook keyed on an event that will never fire.
type SettingsHookEvents struct{}

func (SettingsHookEvents) Name() string { return "settings-unknown-hook-event" }

func (SettingsHookEvents) Run(inv *inventory.Inventory) []Finding {
	events := schema.HookEvents()
	var findings []Finding
	for _, settings := range append(append([]inventory.Settings{}, inv.Settings...), inv.HookSources...) {
		names := make([]string, 0, len(settings.Hooks))
		for name := range settings.Hooks {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if events[name] {
				continue
			}
			finding := Finding{
				Check: "settings-unknown-hook-event", Severity: Error, File: settings.Path, Where: name,
				Message: fmt.Sprintf("%q is not a hook event; every hook under it is ignored", name),
			}
			if suggestion, ok := nearest(name, events); ok {
				finding.Fix = fmt.Sprintf("rename %q to %q", name, suggestion)
			}
			findings = append(findings, finding)
		}
	}
	return findings
}

// nearest returns a known name within one edit of the given one.
func nearest(name string, known map[string]bool) (string, bool) {
	best, bestDistance := "", 3
	for candidate := range known {
		distance := editDistance(strings.ToLower(name), strings.ToLower(candidate))
		if distance < bestDistance || (distance == bestDistance && candidate < best) {
			best, bestDistance = candidate, distance
		}
	}
	if bestDistance <= 1 {
		return best, true
	}
	return "", false
}

func editDistance(a, b string) int {
	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(a); i++ {
		current[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			current[j] = min(previous[j]+1, min(current[j-1]+1, previous[j-1]+cost))
		}
		previous, current = current, previous
	}
	return previous[len(b)]
}
