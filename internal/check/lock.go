package check

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// SkillLock reads what a skill manager recorded against what its tree holds.
// Two defects are visible no other way: a skill edited in place, which the
// next sync silently overwrites, and a link or an entry left behind when the
// other one went away.
type SkillLock struct{}

func (SkillLock) Name() string { return "skill-lock" }

func (SkillLock) Run(inv *inventory.Inventory) []Finding {
	var findings []Finding
	for _, lock := range inv.Locks {
		if lock.Err != nil {
			findings = append(findings, Finding{
				Check: "skill-lock", Severity: Warning, File: lock.Path,
				Message: fmt.Sprintf("does not parse, so nothing it records could be checked: %v", lock.Err),
			})
			continue
		}
		names := make([]string, 0, len(lock.Skills))
		for name := range lock.Skills {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			entry := lock.Skills[name]
			dir := filepath.Join(lock.Tree, name)
			if _, err := os.Stat(dir); err != nil {
				// A project lockfile's tree layout is not verified here, so a
				// missing folder there says nothing.
				if lock.Global {
					findings = append(findings, Finding{
						Check: "skill-lock", Severity: Error, File: lock.Path, Where: name,
						Message: fmt.Sprintf("records a skill with nothing installed at %s", dir),
						Fix:     fmt.Sprintf("skills remove %s, or reinstall it from %s", name, entry.Source),
					})
				}
				continue
			}
			// Only a git tree hash is recomputable offline. Local and git
			// sources carry a shape the CLI itself does not version.
			if len(entry.SkillFolderHash) != 40 || matchesTree(dir, entry.SkillFolderHash) {
				continue
			}
			findings = append(findings, Finding{
				Check: "skill-lock", Severity: Warning, File: lock.Path, Where: name,
				Message: fmt.Sprintf("%s no longer matches the hash recorded at install, so it was edited in place and the next `skills update` overwrites it", dir),
				Fix:     "move the change upstream, or copy the folder out of the manager's tree before updating",
			})
		}
	}
	return append(findings, judgeLinks(inv)...)
}

// judgeLinks reads the agent's own skill folder for links into a manager's
// tree: one that leads nowhere, or one the lockfile does not know.
func judgeLinks(inv *inventory.Inventory) []Finding {
	dir := filepath.Join(inv.Root, "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var findings []Finding
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink == 0 {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		target, err := filepath.EvalSymlinks(path)
		if err != nil {
			findings = append(findings, Finding{
				Check: "skill-lock", Severity: Error, File: path,
				Message: "is a link to a folder that no longer exists, so the skill it named is gone",
				Fix:     "remove the link, or reinstall the skill through its manager",
			})
			continue
		}
		for _, lock := range inv.Locks {
			if !within(lock.Tree, target) {
				continue
			}
			if _, known := lock.Skills[entry.Name()]; known {
				continue
			}
			findings = append(findings, Finding{
				Check: "skill-lock", Severity: Warning, File: path,
				Message: fmt.Sprintf("links into %s, but %s has no entry for it, so the manager can neither update nor remove it", lock.Tree, lock.Path),
				Fix:     "reinstall it through the manager, or copy it out and drop the link",
			})
		}
	}
	return findings
}

func within(tree, path string) bool {
	tree, err := filepath.EvalSymlinks(tree)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(tree, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// matchesTree recomputes the hash the skills CLI records for a skill fetched
// from GitHub: the git tree object of its folder upstream. A checkout drops
// the executable bit the upstream tree kept, so a shebang stands in for it on
// a second attempt.
func matchesTree(dir, want string) bool {
	if got, err := treeHash(dir, false); err == nil && got == want {
		return true
	}
	got, err := treeHash(dir, true)
	return err == nil && got == want
}

func treeHash(dir string, shebangExecutable bool) (string, error) {
	sum, _, err := tree(dir, shebangExecutable)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sum), nil
}

type treeEntry struct {
	mode, name string
	sum        []byte
}

// tree hashes a folder the way git does: blobs and subtrees as objects,
// entries sorted with a directory's name read as if it ended in a slash, and
// CRLF undone the way a checkout on Windows applied it.
func tree(dir string, shebangExecutable bool) ([]byte, bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false, err
	}
	var items []treeEntry
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)
		if entry.IsDir() {
			if name == ".git" || name == "node_modules" {
				continue
			}
			sum, empty, err := tree(path, shebangExecutable)
			if err != nil {
				return nil, false, err
			}
			if !empty {
				items = append(items, treeEntry{"40000", name, sum})
			}
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, false, err
		}
		if !bytes.ContainsRune(content, 0) {
			content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
		}
		mode := "100644"
		if shebangExecutable && bytes.HasPrefix(content, []byte("#!")) {
			mode = "100755"
		}
		items = append(items, treeEntry{mode, name, object("blob", content)})
	}
	if len(items) == 0 {
		return nil, true, nil
	}
	sort.Slice(items, func(i, j int) bool { return sortKey(items[i]) < sortKey(items[j]) })
	var body bytes.Buffer
	for _, item := range items {
		body.WriteString(item.mode)
		body.WriteByte(' ')
		body.WriteString(item.name)
		body.WriteByte(0)
		body.Write(item.sum)
	}
	return object("tree", body.Bytes()), false, nil
}

func sortKey(e treeEntry) string {
	if e.mode == "40000" {
		return e.name + "/"
	}
	return e.name
}

func object(kind string, content []byte) []byte {
	h := sha1.New()
	fmt.Fprintf(h, "%s %d\x00", kind, len(content))
	h.Write(content)
	return h.Sum(nil)
}
