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
// set in both files is two contributions, not a contradiction.
var merged = map[string]bool{"hooks": true, "permissions": true, "env": true}

func (SettingsPrecedence) Run(inv *inventory.Inventory) []Finding {
	if len(inv.Settings) < 2 {
		return nil
	}
	var findings []Finding
	for i := 0; i < len(inv.Settings)-1; i++ {
		loser := inv.Settings[i]
		for _, winner := range inv.Settings[i+1:] {
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
			Message: fmt.Sprintf("%s sets this too, and wins; the value here decides nothing", filepath.Base(winner.Path)),
			Fix:     fmt.Sprintf("drop %q from %s, or change the one that is read", key, filepath.Base(loser.Path)),
		})
	}
	return findings
}

// scalar reports whether a value is replaced wholesale rather than merged.
func scalar(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] != '{' && trimmed[0] != '['
}
