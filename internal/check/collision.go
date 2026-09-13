package check

import (
	"fmt"
	"sort"

	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// NameCollision reports one name installed twice. The runtime picks one of the
// two and says nothing, so the copy that loses looks dormant while being an
// exact duplicate of the one that works.
//
// Two names are a collision whatever their descriptions say, which is why this
// does not measure them.
type NameCollision struct{}

func (NameCollision) Name() string { return "name-collision" }

func (NameCollision) Run(inv *inventory.Inventory) []Finding {
	var findings []Finding
	findings = append(findings, collisionsIn(inv.Root, inv.Skills, "skill")...)
	findings = append(findings, collisionsIn(inv.Root, inv.Agents, "subagent")...)
	return findings
}

func collisionsIn(root string, definitions []inventory.Definition, kind string) []Finding {
	byName := map[string][]inventory.Definition{}
	for _, definition := range definitions {
		byName[definition.Stem] = append(byName[definition.Stem], definition)
	}
	names := make([]string, 0, len(byName))
	for name, group := range byName {
		if len(group) > 1 {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var findings []Finding
	for _, name := range names {
		group := byName[name]
		sort.Slice(group, func(i, j int) bool { return group[i].Path < group[j].Path })
		others := make([]string, 0, len(group)-1)
		for _, definition := range group[1:] {
			others = append(others, trim(root, definition.Path))
		}
		findings = append(findings, Finding{
			Check: "name-collision", Severity: Warning, File: group[0].Path, Where: name,
			Message: fmt.Sprintf("%d %ss are installed under this name; the runtime reaches one of them and the rest never fire", len(group), kind),
			Fix:     fmt.Sprintf("keep the one you meant: the others are %v", others),
		})
	}
	return findings
}
