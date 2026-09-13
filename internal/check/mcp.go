package check

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// MCPResolvable answers for a server the same question HookResolvable answers
// for a hook: does the program it names exist here.
//
// A server that cannot start costs a connection attempt on every session and
// reports nothing when it fails.
type MCPResolvable struct{}

func (MCPResolvable) Name() string { return "mcp-unresolvable" }

func (MCPResolvable) Run(inv *inventory.Inventory) []Finding {
	var findings []Finding
	for _, server := range inv.MCP {
		where := fmt.Sprintf("%s (%s)", server.Name, server.Scope)
		kind := server.Type
		if kind == "" && server.Command != "" {
			kind = "stdio"
		}
		switch kind {
		case "http", "sse":
			if strings.TrimSpace(server.URL) == "" {
				findings = append(findings, Finding{
					Check: "mcp-unresolvable", Severity: Error, File: server.File, Where: where,
					Message: fmt.Sprintf("declared as %s with no url, so it can never connect", kind),
				})
			}
		case "stdio":
			if finding, ok := judgeCommand(server, where); ok {
				findings = append(findings, finding)
			}
		default:
			findings = append(findings, Finding{
				Check: "mcp-unresolvable", Severity: Error, File: server.File, Where: where,
				Message: fmt.Sprintf("has neither a command nor a url, so there is nothing to start (type %q)", server.Type),
			})
		}
	}
	return findings
}

func judgeCommand(server inventory.MCPServer, where string) (Finding, bool) {
	command := strings.TrimSpace(server.Command)
	if command == "" {
		return Finding{
			Check: "mcp-unresolvable", Severity: Error, File: server.File, Where: where,
			Message: "declared as stdio with no command",
		}, true
	}
	expanded, ok := expand(command)
	if !ok {
		return Finding{}, false
	}
	if strings.ContainsRune(expanded, filepath.Separator) {
		info, err := os.Stat(expanded)
		if err != nil {
			return Finding{
				Check: "mcp-unresolvable", Severity: Error, File: server.File, Where: where,
				Message: fmt.Sprintf("starts %s, which does not exist", expanded),
				Fix:     "reinstall the server, or drop the entry",
			}, true
		}
		if info.Mode()&0o111 == 0 {
			return Finding{
				Check: "mcp-unresolvable", Severity: Error, File: server.File, Where: where,
				Message: fmt.Sprintf("starts %s, which is not executable", expanded),
				Fix:     fmt.Sprintf("chmod +x %s", expanded),
			}, true
		}
		return Finding{}, false
	}
	if _, err := exec.LookPath(expanded); err != nil {
		return Finding{
			Check: "mcp-unresolvable", Severity: Error, File: server.File, Where: where,
			Message: fmt.Sprintf("starts %q, which is not on PATH", expanded),
			Fix:     "name the program by absolute path, or install it",
		}, true
	}
	return Finding{}, false
}
