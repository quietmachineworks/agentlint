package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// MCPServer is one server definition, wherever it was declared.
type MCPServer struct {
	File    string
	Scope   string
	Name    string
	Type    string
	Command string
	Args    []string
	URL     string
	Env     map[string]string
	Headers map[string]string
}

type mcpFile struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
	Projects   map[string]struct {
		MCPServers map[string]MCPServer `json:"mcpServers"`
	} `json:"projects"`
}

// mcpPaths are the files that declare servers for an agent rooted at root.
// The user-scoped one is a sibling of the configuration directory rather than a
// child of it, which is the reason this is not a simple walk.
func mcpPaths(root string) []struct{ path, scope string } {
	parent := filepath.Dir(root)
	return []struct{ path, scope string }{
		{filepath.Join(parent, ".claude.json"), "user"},
		{filepath.Join(root, ".mcp.json"), "project"},
		{filepath.Join(parent, ".mcp.json"), "project"},
	}
}

func loadMCP(root string, plugins []Plugin) ([]MCPServer, []string) {
	var servers []MCPServer
	var read []string
	candidates := mcpPaths(root)
	for _, path := range pluginFiles(plugins, ".mcp.json") {
		candidates = append(candidates, struct{ path, scope string }{path, "plugin"})
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate.path)
		if err != nil {
			continue
		}
		var parsed mcpFile
		if err := json.Unmarshal(data, &parsed); err != nil {
			continue
		}
		read = append(read, candidate.path)
		servers = append(servers, flatten(parsed.MCPServers, candidate.path, candidate.scope)...)
		for project, entry := range parsed.Projects {
			servers = append(servers, flatten(entry.MCPServers, candidate.path, "project:"+filepath.Base(project))...)
		}
	}
	sort.Slice(servers, func(i, j int) bool {
		if servers[i].File != servers[j].File {
			return servers[i].File < servers[j].File
		}
		return servers[i].Name < servers[j].Name
	})
	return servers, read
}

func flatten(declared map[string]MCPServer, file, scope string) []MCPServer {
	out := make([]MCPServer, 0, len(declared))
	for name, server := range declared {
		server.Name, server.File, server.Scope = name, file, scope
		out = append(out, server)
	}
	return out
}

// pluginFiles are the definition files plugins bring. They are carried exactly
// like the agent's own, and an inventory that skips them prices a lighter agent
// than the one that runs.
func pluginFiles(plugins []Plugin, name string) []string {
	var found []string
	for _, plugin := range Loaded(plugins) {
		for _, candidate := range []string{
			filepath.Join(plugin.Path, name),
			filepath.Join(plugin.Path, "hooks", name),
		} {
			if exists(candidate) {
				found = append(found, candidate)
			}
		}
	}
	sort.Strings(found)
	return found
}
