package check

import (
	"fmt"

	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// descriptionMax is the budget a description is read against before the
// runtime truncates it.
const descriptionMax = 1024

// AgentFrontmatter judges subagent definitions, which `claude plugin validate`
// does not discover at all.
type AgentFrontmatter struct{}

func (AgentFrontmatter) Name() string { return "agent-frontmatter" }

func (AgentFrontmatter) Run(inv *inventory.Inventory) []Finding {
	return judgeDefinitions(inv.Agents, "agent-frontmatter", "subagent", "file")
}

// SkillFrontmatter judges skills on the fields the runtime validator passes
// over: the name, whether it matches its folder, and the description budget.
type SkillFrontmatter struct{}

func (SkillFrontmatter) Name() string { return "skill-frontmatter" }

func (SkillFrontmatter) Run(inv *inventory.Inventory) []Finding {
	return judgeDefinitions(inv.Skills, "skill-frontmatter", "skill", "folder")
}

func judgeDefinitions(definitions []inventory.Definition, check, kind, container string) []Finding {
	var findings []Finding
	for _, definition := range definitions {
		at := Finding{Check: check, File: definition.Path}
		if !definition.Frontmatter.Present {
			at.Severity = Error
			at.Message = fmt.Sprintf("no frontmatter, so this %s loads with no name and no description", kind)
			findings = append(findings, at)
			continue
		}
		name, hasName := definition.Frontmatter.Get("name")
		switch {
		case !hasName || name == "":
			findings = append(findings, Finding{
				Check: check, Severity: Error, File: definition.Path,
				Message: fmt.Sprintf("no name, so the %s cannot be invoked by name", kind),
				Fix:     fmt.Sprintf("add `name: %s`", definition.Stem),
			})
		case name != definition.Stem:
			findings = append(findings, Finding{
				Check: check, Severity: Error, File: definition.Path, Where: name,
				Message: fmt.Sprintf("declares %q, but its %s is %q; the two must agree", name, container, definition.Stem),
				Fix:     fmt.Sprintf("set `name: %s`, or move it to match", definition.Stem),
			})
		}
		description, hasDescription := definition.Frontmatter.Get("description")
		switch {
		case !hasDescription || description == "":
			findings = append(findings, Finding{
				Check: check, Severity: Error, File: definition.Path,
				Message: fmt.Sprintf("no description, so nothing can choose this %s", kind),
			})
		case len(description) > descriptionMax:
			findings = append(findings, Finding{
				Check: check, Severity: Warning, File: definition.Path,
				Message: fmt.Sprintf("description is %d characters, over the %d budget", len(description), descriptionMax),
			})
		}
	}
	return findings
}
