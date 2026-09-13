package inventory

import (
	"os"
	"path/filepath"
	"runtime"
)

// Scope is a level of the settings stack. Higher wins.
//
// The chain is the documented one, and it has a gap this tool respects: a
// settings.local.json beside the user file is written by the agent and read by
// it, but no published order places it against the levels below, so it is
// carried with an unknown rank and never used to claim that something else
// decides nothing.
type Scope struct {
	Name string
	Rank int
}

var (
	ScopeUser         = Scope{"user", 1}
	ScopeUserLocal    = Scope{"user local", 0}
	ScopeProject      = Scope{"shared project", 2}
	ScopeProjectLocal = Scope{"project local", 3}
	ScopeManaged      = Scope{"managed", 5}
	ScopePlugin       = Scope{"plugin", 0}
)

// Ordered reports whether this scope's place in the stack is documented.
func (s Scope) Ordered() bool { return s.Rank > 0 }

// managedPaths are where an organization deploys a policy, per platform.
func managedPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/Library/Application Support/ClaudeCode/managed-settings.json"}
	case "windows":
		return []string{
			filepath.FromSlash(`C:/Program Files/ClaudeCode/managed-settings.json`),
			filepath.FromSlash(`C:/ProgramData/ClaudeCode/managed-settings.json`),
		}
	default:
		return []string{"/etc/claude-code/managed-settings.json"}
	}
}

// projectPaths are the two files a repository carries, under its own .claude.
func projectPaths(project string) []struct {
	path  string
	scope Scope
} {
	root := filepath.Join(project, ".claude")
	return []struct {
		path  string
		scope Scope
	}{
		{filepath.Join(root, "settings.json"), ScopeProject},
		{filepath.Join(root, "settings.local.json"), ScopeProjectLocal},
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
