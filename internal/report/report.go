// Package report renders a run for a terminal or for another program.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/quietmachineworks/agentlint/internal/check"
	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// Run is one execution, rendered whole so both formats say the same thing.
type Run struct {
	Root     string          `json:"root"`
	Read     []string        `json:"read"`
	Counts   Counts          `json:"counts"`
	Findings []check.Finding `json:"findings"`
}

// Counts is the inventory line a reader wants before the findings.
type Counts struct {
	Settings     int `json:"settings"`
	Agents       int `json:"agents"`
	Skills       int `json:"skills"`
	MCP          int `json:"mcpServers"`
	HookCommands int `json:"hookCommands"`
	Errors       int `json:"errors"`
	Warnings     int `json:"warnings"`
}

// Summarise counts what was read and what was found.
func Summarise(inv *inventory.Inventory, findings []check.Finding) Run {
	run := Run{Root: inv.Root, Read: inv.Read, Findings: findings}
	if run.Findings == nil {
		run.Findings = []check.Finding{}
	}
	run.Counts.Settings = len(inv.Settings)
	run.Counts.Agents = len(inv.Agents)
	run.Counts.Skills = len(inv.Skills)
	run.Counts.MCP = len(inv.MCP)
	for _, settings := range inv.Settings {
		for _, matchers := range settings.Hooks {
			for _, matcher := range matchers {
				run.Counts.HookCommands += len(matcher.Hooks)
			}
		}
	}
	for _, finding := range findings {
		if finding.Severity == check.Error {
			run.Counts.Errors++
			continue
		}
		run.Counts.Warnings++
	}
	return run
}

// JSON writes the run for another program.
func JSON(w io.Writer, run Run) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(run)
}

// Text writes the run for a terminal, grouped by file, errors first.
func Text(w io.Writer, run Run) error {
	fmt.Fprintf(w, "%s\n", run.Root)
	fmt.Fprintf(w, "%d settings, %d hook commands, %d MCP servers, %d subagents, %d skills\n",
		run.Counts.Settings, run.Counts.HookCommands, run.Counts.MCP, run.Counts.Agents, run.Counts.Skills)

	byFile := map[string][]check.Finding{}
	var order []string
	for _, finding := range run.Findings {
		if _, seen := byFile[finding.File]; !seen {
			order = append(order, finding.File)
		}
		byFile[finding.File] = append(byFile[finding.File], finding)
	}
	sort.Strings(order)

	for _, file := range order {
		fmt.Fprintf(w, "\n%s\n", relativise(file, run.Root))
		for _, finding := range byFile[file] {
			head := finding.Message
			if finding.Where != "" {
				head = finding.Where + ": " + head
			}
			fmt.Fprintf(w, "  %-8s %s\n", finding.Severity, head)
			fmt.Fprintf(w, "  %-8s %s\n", "", finding.Check)
			if finding.Fix != "" {
				fmt.Fprintf(w, "  %-8s %s\n", "", finding.Fix)
			}
		}
	}

	fmt.Fprintln(w)
	if run.Counts.Errors == 0 && run.Counts.Warnings == 0 {
		fmt.Fprintln(w, "nothing to repair")
		return nil
	}
	fmt.Fprintf(w, "%d error(s), %d warning(s)\n", run.Counts.Errors, run.Counts.Warnings)
	return nil
}

func relativise(path, root string) string {
	if relative, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(relative, "..") {
		return relative
	}
	return path
}
