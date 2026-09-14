// Package check holds the checks agentlint runs.
//
// Every check here is deterministic: it resolves what the configuration
// declares against what is on the machine, and never asks a model.
package check

import (
	"sort"

	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// Severity ranks a finding. Errors fail a run; warnings fail only under strict.
type Severity int

const (
	Warning Severity = iota
	Error
)

func (s Severity) String() string {
	if s == Error {
		return "error"
	}
	return "warning"
}

// Finding is one thing wrong, named where it is and what to do about it.
type Finding struct {
	Check    string   `json:"check"`
	Severity Severity `json:"-"`
	Level    string   `json:"severity"`
	File     string   `json:"file"`
	Where    string   `json:"where,omitempty"`
	Message  string   `json:"message"`
	Fix      string   `json:"fix,omitempty"`
}

// Checker judges one inventory and returns what it found.
type Checker interface {
	Name() string
	Run(inv *inventory.Inventory) []Finding
}

// All returns every check, in report order.
func All() []Checker {
	return []Checker{
		HookResolvable{},
		HookCost{},
		MCPResolvable{},
		SecretInConfig{},
		SettingsSchema{},
		SettingsKeys{},
		PermissionRules{},
		SettingsPrecedence{},
		SettingsScope{},
		SettingsHookEvents{},
		AgentFrontmatter{},
		SkillFrontmatter{},
		NameCollision{},
		DescriptionShadowing{},
		SkillLock{},
	}
}

// Run applies every check and returns the findings, errors first.
func Run(inv *inventory.Inventory, checkers []Checker) []Finding {
	var findings []Finding
	for _, checker := range checkers {
		for _, finding := range checker.Run(inv) {
			finding.Level = finding.Severity.String()
			findings = append(findings, finding)
		}
	}
	sort.SliceStable(findings, func(i, j int) bool {
		return findings[i].Severity > findings[j].Severity
	})
	return findings
}
