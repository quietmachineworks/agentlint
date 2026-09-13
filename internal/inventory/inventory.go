// Package inventory discovers what an agent installation actually carries.
//
// Layouts differ by version, by OS and by install method, so everything here
// reports what it found rather than asserting where things live.
package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Settings is one settings file, kept as raw JSON so unknown keys survive.
type Settings struct {
	Path   string
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
	Settings []Settings
	Agents   []Definition
	Skills   []Definition
	MCP      []MCPServer
	Read     []string
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
func Load(root string) (*Inventory, error) {
	inv := &Inventory{Root: root}
	for _, name := range []string{"settings.json", "settings.local.json"} {
		path := filepath.Join(root, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		inv.Settings = append(inv.Settings, loadSettings(path))
		inv.Read = append(inv.Read, path)
	}
	inv.Agents = loadDefinitions(filepath.Join(root, "agents"), "")
	if len(inv.Agents) > 0 {
		inv.Read = append(inv.Read, filepath.Join(root, "agents"))
	}
	for _, dir := range skillDirs(root) {
		found := loadDefinitions(dir, "SKILL.md")
		if len(found) == 0 {
			continue
		}
		inv.Skills = append(inv.Skills, found...)
		inv.Read = append(inv.Read, dir)
	}
	servers, readMCP := loadMCP(root)
	inv.MCP = servers
	inv.Read = append(inv.Read, readMCP...)
	return inv, nil
}

func loadSettings(path string) Settings {
	s := Settings{Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		s.Err = err
		return s
	}
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
func skillDirs(root string) []string {
	dirs := []string{filepath.Join(root, "skills")}
	_ = filepath.WalkDir(filepath.Join(root, "plugins"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return nil
		}
		if entry.Name() == "skills" {
			dirs = append(dirs, path)
			return filepath.SkipDir
		}
		return nil
	})
	return dirs
}
