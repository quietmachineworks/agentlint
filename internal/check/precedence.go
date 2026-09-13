package check

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// SettingsPrecedence reports a setting written twice, where one copy decides
// nothing. The file that loses says nothing about being overridden, and the
// one that wins is usually the one nobody remembers editing.
type SettingsPrecedence struct{}

func (SettingsPrecedence) Name() string { return "settings-precedence" }

// merged names the keys the runtime combines rather than replaces. A key here
// set at two levels is two contributions, not a contradiction.
var merged = map[string]bool{"hooks": true, "permissions": true}

func (SettingsPrecedence) Run(inv *inventory.Inventory) []Finding {
	var findings []Finding
	for _, loser := range inv.Settings {
		for _, winner := range inv.Settings {
			if !loser.Scope.Ordered() || !winner.Scope.Ordered() {
				continue
			}
			if winner.Scope.Rank <= loser.Scope.Rank {
				continue
			}
			findings = append(findings, compare(loser, winner)...)
		}
	}
	return findings
}

func compare(loser, winner inventory.Settings) []Finding {
	if !loser.Parsed || !winner.Parsed {
		return nil
	}
	keys := make([]string, 0, len(loser.Raw))
	for key := range loser.Raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var findings []Finding
	for _, key := range keys {
		if merged[key] {
			continue
		}
		overriding, present := winner.Raw[key]
		if !present || bytes.Equal(loser.Raw[key], overriding) {
			continue
		}
		if !scalar(loser.Raw[key]) || !scalar(overriding) {
			continue
		}
		findings = append(findings, Finding{
			Check: "settings-precedence", Severity: Warning, File: loser.Path, Where: key,
			Message: fmt.Sprintf("%s settings set this too, and sit higher in the stack; the value here decides nothing", winner.Scope.Name),
			Fix:     fmt.Sprintf("drop %q here, or change %s", key, filepath.Base(winner.Path)),
		})
	}
	return findings
}

// scalar reports whether a value is replaced wholesale rather than merged.
func scalar(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] != '{' && trimmed[0] != '['
}
