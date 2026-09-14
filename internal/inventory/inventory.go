// Package inventory discovers what an agent installation actually carries.
//
// Layouts differ by version, by OS and by install method, so everything here
// reports what it found rather than asserting where things live.
package inventory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Settings is one settings file, kept as raw JSON so unknown keys survive.
type Settings struct {
	Path   string
	Scope  Scope
	Raw    map[string]json.RawMessage
	Hooks  map[string][]HookMatcher
	Parsed bool
	Err    error
}

// HookMatcher is one matcher block inside a hook event.
type HookMatcher struct {
	Matcher string        `json:"matcher"`
	Hooks   []HookCommand `json:"hooks"`
}

// HookCommand is a single hook entry.
type HookCommand struct {
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Timeout *float64 `json:"timeout"`
	Async   bool     `json:"async"`
	If      string   `json:"if"`
}

// Definition is a Markdown-defined skill, subagent or command.
type Definition struct {
	Path        string
	Stem        string
	Frontmatter Frontmatter
}

// Inventory is everything one run read.
type Inventory struct {
	Root     string
	Project  string
	Settings []Settings
	// HookSources carry hooks without being settings files: a plugin ships
	// them, the runtime runs them, and nothing else about a settings file
	// applies to their shape.
	HookSources []Settings
	Plugins     []Plugin
	Agents      []Definition
	Skills      []Definition
	MCP         []MCPServer
	Locks       []Lockfile
	Read        []string
}

// DefaultRoot is the configuration directory, overridable for tests and CI.
func DefaultRoot() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude"
	}
	return filepath.Join(home, ".claude")
}

// Load reads every kind of item under root that this tool judges.
func Load(root, project string) (*Inventory, error) {
	inv := &Inventory{Root: root, Project: project}
	for _, candidate := range settingsPaths(root, project) {
		if !exists(candidate.path) {
			continue
		}
		inv.Settings = append(inv.Settings, loadSettings(candidate.path, candidate.scope))
		inv.Read = append(inv.Read, candidate.path)
	}
	inv.Plugins = loadPlugins(root, inv.Settings)
	inv.Agents = loadDefinitions(filepath.Join(root, "agents"), "")
	if len(inv.Agents) > 0 {
		inv.Read = append(inv.Read, filepath.Join(root, "agents"))
	}
	for _, dir := range skillDirs(root, inv.Plugins) {
		found := loadDefinitions(dir, "SKILL.md")
		if len(found) == 0 {
			continue
		}
		inv.Skills = append(inv.Skills, found...)
		inv.Read = append(inv.Read, dir)
	}
	for _, path := range pluginFiles(inv.Plugins, "hooks.json") {
		entry := loadSettings(path, ScopePlugin)
		if !entry.Parsed && entry.Err == nil {
			continue
		}
		inv.HookSources = append(inv.HookSources, entry)
		inv.Read = append(inv.Read, path)
	}
	servers, readMCP := loadMCP(root, inv.Plugins)
	inv.MCP = servers
	inv.Read = append(inv.Read, readMCP...)
	inv.Locks = loadLocks(project)
	for _, lock := range inv.Locks {
		inv.Read = append(inv.Read, lock.Path)
	}
	return inv, nil
}

// AddCommandLine reads the settings the agent is given as --settings: a file,
// or the same JSON inline. It is real configuration at a published place in
// the stack, and nothing else validates it.
func (inv *Inventory) AddCommandLine(arg string) error {
	var s Settings
	if strings.HasPrefix(strings.TrimSpace(arg), "{") {
		s = parseSettings("--settings", ScopeCommandLine, []byte(arg))
		if s.Err != nil {
			return fmt.Errorf("--settings is not valid JSON: %w", s.Err)
		}
	} else {
		if !exists(arg) {
			return fmt.Errorf("--settings: cannot read %s", arg)
		}
		s = loadSettings(arg, ScopeCommandLine)
	}
	inv.Settings = append(inv.Settings, s)
	inv.Read = append(inv.Read, s.Path)
	return nil
}

// settingsPaths are every file that contributes, lowest level first.
func settingsPaths(root, project string) []struct {
	path  string
	scope Scope
} {
	type candidate = struct {
		path  string
		scope Scope
	}
	out := []candidate{
		{filepath.Join(root, "settings.json"), ScopeUser},
		{filepath.Join(root, "settings.local.json"), ScopeUserLocal},
	}
	if project != "" {
		out = append(out, projectPaths(project)...)
	}
	for _, path := range managedPaths() {
		out = append(out, candidate{path, ScopeManaged})
	}
	return out
}

func loadSettings(path string, scope Scope) Settings {
	data, err := os.ReadFile(path)
	if err != nil {
		return Settings{Path: path, Scope: scope, Err: err}
	}
	return parseSettings(path, scope, data)
}

func parseSettings(path string, scope Scope, data []byte) Settings {
	s := Settings{Path: path, Scope: scope}
	if err := json.Unmarshal(data, &s.Raw); err != nil {
		s.Err = err
		return s
	}
	s.Parsed = true
	if block, ok := s.Raw["hooks"]; ok {
		if err := json.Unmarshal(block, &s.Hooks); err != nil {
			s.Err = err
		}
	}
	return s
}

// loadDefinitions reads dir. With basename set, each child directory is one
// item named by that directory and holding that file; otherwise every .md file
// below dir is one item.
//
// A skill manager links one source tree into the agent's folder, so every entry
// here may be a symlink and the walk has to follow them.
func loadDefinitions(dir, basename string) []Definition {
	if basename != "" {
		return loadContainers(dir, basename)
	}
	return loadMarkdown(dir)
}

func loadContainers(dir, basename string) []Definition {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Definition
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name(), basename)
		info, statErr := os.Stat(path)
		if statErr != nil || info.IsDir() {
			continue
		}
		if definition, ok := read(path, entry.Name()); ok {
			out = append(out, definition)
		}
	}
	return out
}

func loadMarkdown(dir string) []Definition {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Definition
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		info, statErr := os.Stat(path)
		if statErr != nil {
			continue
		}
		if info.IsDir() {
			out = append(out, loadMarkdown(path)...)
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if definition, ok := read(path, strings.TrimSuffix(entry.Name(), ".md")); ok {
			out = append(out, definition)
		}
	}
	return out
}

func read(path, stem string) (Definition, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, false
	}
	return Definition{Path: path, Stem: stem, Frontmatter: ParseFrontmatter(string(data))}, true
}

// skillDirs are the folders that hold skills: the agent's own, plus every one
// a plugin brings. A plugin's skills are carried on every prompt exactly like
// the agent's own, so an inventory that skips them understates the load.
func skillDirs(root string, plugins []Plugin) []string {
	dirs := []string{filepath.Join(root, "skills")}
	for _, plugin := range Loaded(plugins) {
		dirs = append(dirs, filepath.Join(plugin.Path, "skills"))
	}
	return dirs
}
