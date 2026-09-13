package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// Plugin is one plugin the agent actually loads.
//
// A plugin's tree on disk is not what runs. The cache keeps every version ever
// fetched, and the marketplace tree keeps every plugin ever offered, installed
// or not. Counting either one prices an agent heavier than the one that starts,
// and reports defects in files nothing reads.
type Plugin struct {
	Key     string
	Path    string
	Enabled bool
}

type installedFile struct {
	Plugins map[string][]struct {
		Scope       string `json:"scope"`
		InstallPath string `json:"installPath"`
	} `json:"plugins"`
}

func loadPlugins(root string, settings []Settings) []Plugin {
	data, err := os.ReadFile(filepath.Join(root, "plugins", "installed_plugins.json"))
	if err != nil {
		return nil
	}
	var installed installedFile
	if err := json.Unmarshal(data, &installed); err != nil {
		return nil
	}
	enabled := enabledPlugins(settings)

	var plugins []Plugin
	for key, entries := range installed.Plugins {
		for _, entry := range entries {
			path := entry.InstallPath
			if path == "" {
				continue
			}
			if !filepath.IsAbs(path) {
				path = filepath.Join(root, path)
			}
			if !exists(path) {
				continue
			}
			plugins = append(plugins, Plugin{Key: key, Path: path, Enabled: enabled[key]})
		}
	}
	sort.Slice(plugins, func(i, j int) bool {
		if plugins[i].Key != plugins[j].Key {
			return plugins[i].Key < plugins[j].Key
		}
		return plugins[i].Path < plugins[j].Path
	})
	return plugins
}

// enabledPlugins reads the switch from every settings file, letting the higher
// level decide when two of them disagree.
func enabledPlugins(settings []Settings) map[string]bool {
	enabled := map[string]bool{}
	ordered := append([]Settings{}, settings...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Scope.Rank < ordered[j].Scope.Rank })
	for _, entry := range ordered {
		block, ok := entry.Raw["enabledPlugins"]
		if !ok {
			continue
		}
		var declared map[string]bool
		if err := json.Unmarshal(block, &declared); err != nil {
			continue
		}
		for key, on := range declared {
			enabled[key] = on
		}
	}
	return enabled
}

// Loaded are the plugins whose files the agent reads on every prompt.
func Loaded(plugins []Plugin) []Plugin {
	var live []Plugin
	for _, plugin := range plugins {
		if plugin.Enabled {
			live = append(live, plugin)
		}
	}
	return live
}
