package inventory

import (
	"bufio"
	"strings"
)

// Frontmatter is the leading YAML block of a Markdown definition, read as the
// flat key/value map the runtime treats it as.
type Frontmatter struct {
	Present bool
	Fields  map[string]string
	Order   []string
}

// Get returns a field and whether it was set.
func (f Frontmatter) Get(key string) (string, bool) {
	v, ok := f.Fields[key]
	return v, ok
}

// ParseFrontmatter reads the block delimited by the first two `---` lines.
func ParseFrontmatter(text string) Frontmatter {
	fm := Frontmatter{Fields: map[string]string{}}
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	if !scanner.Scan() || strings.TrimRight(scanner.Text(), "\r") != "---" {
		return fm
	}
	var key string
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "---" {
			fm.Present = true
			return fm
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if key != "" {
				fm.Fields[key] += " " + strings.TrimSpace(line)
			}
			continue
		}
		name, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(name)
		fm.Fields[key] = strings.TrimSpace(value)
		fm.Order = append(fm.Order, key)
	}
	return fm
}

// SplitList reads a comma-separated frontmatter value such as allowed-tools.
func SplitList(value string) []string {
	value = strings.Trim(value, "[]")
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(strings.Trim(strings.TrimSpace(part), `"'`))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
