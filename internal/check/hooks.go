package check

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// HookResolvable answers the question the schema cannot: does the program a
// hook names exist on this machine, and may it be executed.
type HookResolvable struct{}

func (HookResolvable) Name() string { return "hook-unresolvable" }

// interpreters run a script named in a later argument rather than themselves.
var interpreters = map[string]bool{
	"bash": true, "sh": true, "zsh": true, "dash": true,
	"node": true, "bun": true, "deno": true,
	"python": true, "python3": true, "ruby": true, "perl": true,
	"env": true, "pwsh": true, "powershell": true,
}

func (h HookResolvable) Run(inv *inventory.Inventory) []Finding {
	var findings []Finding
	for _, settings := range inv.Settings {
		for event, matchers := range settings.Hooks {
			for _, matcher := range matchers {
				for _, hook := range matcher.Hooks {
					if finding, ok := h.judge(settings.Path, event, matcher.Matcher, hook); ok {
						findings = append(findings, finding)
					}
				}
			}
		}
	}
	return findings
}

func (h HookResolvable) judge(file, event, matcher string, hook inventory.HookCommand) (Finding, bool) {
	target, kind := resolveTarget(hook.Command)
	if kind == targetUnverifiable {
		return Finding{}, false
	}
	where := fmt.Sprintf("%s[%s]", event, orNone(matcher))
	switch kind {
	case targetPath:
		info, err := os.Stat(target)
		if err != nil {
			return Finding{
				Check: "hook-unresolvable", Severity: Error, File: file, Where: where,
				Message: fmt.Sprintf("runs %s, which does not exist", target),
				Fix:     "point the hook at a program that is on this machine, or drop the entry",
			}, true
		}
		if info.IsDir() {
			return Finding{
				Check: "hook-unresolvable", Severity: Error, File: file, Where: where,
				Message: fmt.Sprintf("runs %s, which is a directory", target),
			}, true
		}
		if info.Mode()&0o111 == 0 && !passedToInterpreter(hook.Command, target) {
			return Finding{
				Check: "hook-unresolvable", Severity: Error, File: file, Where: where,
				Message: fmt.Sprintf("runs %s, which is not executable", target),
				Fix:     fmt.Sprintf("chmod +x %s", target),
			}, true
		}
	case targetLookup:
		if _, err := exec.LookPath(target); err != nil {
			return Finding{
				Check: "hook-unresolvable", Severity: Error, File: file, Where: where,
				Message: fmt.Sprintf("runs %q, which is not on PATH", target),
				Fix:     "name the program by absolute path, or install it",
			}, true
		}
	}
	return Finding{}, false
}

type targetKind int

const (
	targetUnverifiable targetKind = iota
	targetPath
	targetLookup
)

// resolveTarget picks the program a hook command runs, expanding what can be
// expanded off the agent's runtime. A command whose target depends on a
// variable only the runtime sets is reported as unverifiable, never as broken.
func resolveTarget(command string) (string, targetKind) {
	command = strings.TrimSpace(command)
	if command == "" || strings.ContainsAny(command, "|&;") || strings.HasPrefix(command, "[") {
		return "", targetUnverifiable
	}
	fields := splitFields(command)
	if len(fields) == 0 {
		return "", targetUnverifiable
	}
	target := fields[0]
	if interpreters[filepath.Base(target)] {
		script := ""
		for _, field := range fields[1:] {
			if strings.HasPrefix(field, "-") || strings.Contains(field, "=") {
				continue
			}
			script = field
			break
		}
		if script == "" {
			return "", targetUnverifiable
		}
		target = script
	}
	expanded, ok := expand(target)
	if !ok {
		return "", targetUnverifiable
	}
	if strings.ContainsRune(expanded, filepath.Separator) {
		return expanded, targetPath
	}
	return expanded, targetLookup
}

// expand resolves the variables whose value this process can know.
func expand(token string) (string, bool) {
	if strings.HasPrefix(token, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		token = home + strings.TrimPrefix(token, "~")
	}
	for _, name := range []string{"HOME", "CLAUDE_CONFIG_DIR"} {
		value := os.Getenv(name)
		if value == "" {
			continue
		}
		token = strings.ReplaceAll(token, "${"+name+"}", value)
		token = strings.ReplaceAll(token, "$"+name, value)
	}
	if strings.Contains(token, "$") {
		return "", false
	}
	return token, true
}

func passedToInterpreter(command, target string) bool {
	fields := splitFields(command)
	return len(fields) > 1 && interpreters[filepath.Base(fields[0])] && fields[0] != target
}

// splitFields splits on whitespace while keeping quoted runs together.
func splitFields(command string) []string {
	var fields []string
	var current strings.Builder
	var quote rune
	for _, char := range command {
		switch {
		case quote != 0 && char == quote:
			quote = 0
		case quote == 0 && (char == '"' || char == '\''):
			quote = char
		case quote == 0 && (char == ' ' || char == '\t'):
			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}
	return fields
}

func orNone(matcher string) string {
	if matcher == "" {
		return "no matcher"
	}
	return matcher
}
