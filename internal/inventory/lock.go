package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Lockfile is what the skills CLI records about the skills it manages. It
// sits beside the one source tree the CLI links into every agent directory,
// and carries what that tree cannot show: where each skill came from and what
// it hashed to when it was installed.
type Lockfile struct {
	Path   string
	Tree   string // one directory per managed skill
	Global bool   // the user's lockfile, whose tree layout is fixed; a project's is not
	Skills map[string]LockEntry
	Err    error
}

// LockEntry is one managed skill.
type LockEntry struct {
	Source          string `json:"source"`
	SourceType      string `json:"sourceType"`
	SkillPath       string `json:"skillPath"`
	SkillFolderHash string `json:"skillFolderHash"`
}

// loadLocks reads the user's lockfile and the project's, whichever exist. A
// machine that does not use the CLI has neither, and reads nothing here.
func loadLocks(project string) []Lockfile {
	var out []Lockfile
	if home, err := os.UserHomeDir(); err == nil {
		out = appendLock(out, filepath.Join(home, ".agents", ".skill-lock.json"), filepath.Join(home, ".agents", "skills"), true)
	}
	if project != "" {
		out = appendLock(out, filepath.Join(project, "skills-lock.json"), filepath.Join(project, ".agents", "skills"), false)
	}
	return out
}

func appendLock(locks []Lockfile, path, tree string, global bool) []Lockfile {
	data, err := os.ReadFile(path)
	if err != nil {
		return locks
	}
	lock := Lockfile{Path: path, Tree: tree, Global: global}
	var doc struct {
		Skills map[string]LockEntry `json:"skills"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		lock.Err = err
	}
	lock.Skills = doc.Skills
	return append(locks, lock)
}
