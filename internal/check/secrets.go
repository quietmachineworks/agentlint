package check

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// SecretInConfig reports a credential written into a configuration file.
//
// It names the field and the shape and never the value. A finding that pastes
// the key into a terminal, a log or a CI annotation has leaked what it came to
// warn about.
type SecretInConfig struct{}

func (SecretInConfig) Name() string { return "secret-in-config" }

// minted are the credential formats that announce themselves. A value matching
// one of these is a secret whatever the field is called.
var minted = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"an Anthropic key", regexp.MustCompile(`^sk-ant-[A-Za-z0-9_-]{20,}`)},
	{"an OpenAI key", regexp.MustCompile(`^sk-[A-Za-z0-9]{32,}`)},
	{"a GitHub token", regexp.MustCompile(`^(gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})`)},
	{"an AWS access key", regexp.MustCompile(`^(AKIA|ASIA)[A-Z0-9]{16}$`)},
	{"a Google key", regexp.MustCompile(`^AIza[A-Za-z0-9_-]{30,}$`)},
	{"a Slack token", regexp.MustCompile(`^xox[abprs]-[A-Za-z0-9-]{10,}`)},
	{"a private key", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)},
	{"a bearer token", regexp.MustCompile(`^Bearer\s+[A-Za-z0-9._-]{20,}$`)},
}

// reference is a value that points at a secret rather than holding one.
var reference = regexp.MustCompile(`^\$[A-Za-z_{]|^\$\{|^[A-Za-z0-9_./~-]*\bop://|^<|^$`)

var opaque = regexp.MustCompile(`^[A-Za-z0-9+/_-]{32,}={0,2}$`)

func (SecretInConfig) Run(inv *inventory.Inventory) []Finding {
	var findings []Finding
	for _, settings := range inv.Settings {
		if !settings.Parsed {
			continue
		}
		findings = append(findings, scanSettings(settings)...)
	}
	for _, server := range inv.MCP {
		findings = append(findings, scanPairs(server.File, fmt.Sprintf("%s.env", server.Name), server.Env)...)
		findings = append(findings, scanPairs(server.File, fmt.Sprintf("%s.headers", server.Name), server.Headers)...)
	}
	return findings
}

func scanSettings(settings inventory.Settings) []Finding {
	block, ok := settings.Raw["env"]
	if !ok {
		return nil
	}
	var environment map[string]string
	if err := json.Unmarshal(block, &environment); err != nil {
		return nil
	}
	return scanPairs(settings.Path, "env", environment)
}

func scanPairs(file, where string, pairs map[string]string) []Finding {
	names := make([]string, 0, len(pairs))
	for name := range pairs {
		names = append(names, name)
	}
	sort.Strings(names)

	var findings []Finding
	for _, name := range names {
		shape, found := classify(name, pairs[name])
		if !found {
			continue
		}
		findings = append(findings, Finding{
			Check: "secret-in-config", Severity: Error, File: file,
			Where:   fmt.Sprintf("%s.%s", where, name),
			Message: shape,
			Fix:     "read it from the environment or a secret manager, and rotate this one: a config file is backed up, synced and often committed",
		})
	}
	return findings
}

// classify names the shape of a value that should not be sitting in a file.
func classify(name, value string) (string, bool) {
	value = strings.TrimSpace(value)
	if reference.MatchString(value) {
		return "", false
	}
	for _, format := range minted {
		if format.pattern.MatchString(value) {
			return fmt.Sprintf("holds %s in plain text", format.name), true
		}
	}
	if sensitive([]string{name}) && opaque.MatchString(value) {
		return fmt.Sprintf("holds %d opaque characters under a name that reads as a credential", len(value)), true
	}
	return "", false
}
