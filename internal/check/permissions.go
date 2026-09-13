package check

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// PermissionRules judges a permission list against itself: what a rule grants,
// what another rule already granted, and what a denial has already taken back.
type PermissionRules struct{}

func (PermissionRules) Name() string { return "permission-rule" }

// buckets are ordered by the precedence the runtime applies: a denial beats an
// ask, and an ask beats an allowance.
var buckets = []string{"deny", "ask", "allow"}

// wholeTool names the tools where granting the tool grants everything it can
// do. Read and Grep are absent: breadth there is a choice, not a surprise.
var wholeTool = map[string]bool{
	"Bash": true, "Write": true, "Edit": true, "MultiEdit": true,
	"NotebookEdit": true, "PowerShell": true, "Agent": true, "Skill": true,
}

type rule struct {
	raw      string
	tool     string
	argument string
	bounded  bool
	index    int
	bucket   string
}

func (PermissionRules) Run(inv *inventory.Inventory) []Finding {
	var findings []Finding
	for _, settings := range inv.Settings {
		parsed := parseRules(settings)
		findings = append(findings, judgeBreadth(settings.Path, parsed)...)
		findings = append(findings, judgeOverlap(settings.Path, parsed)...)
	}
	return findings
}

func parseRules(settings inventory.Settings) map[string][]rule {
	out := map[string][]rule{}
	block, ok := settings.Raw["permissions"]
	if !ok {
		return out
	}
	// The permissions object holds defaultMode alongside the three lists, so
	// each list is decoded on its own or one scalar loses the whole block.
	var permissions map[string]json.RawMessage
	if err := json.Unmarshal(block, &permissions); err != nil {
		return out
	}
	for _, bucket := range buckets {
		var rules []string
		if err := json.Unmarshal(permissions[bucket], &rules); err != nil {
			continue
		}
		for index, raw := range rules {
			out[bucket] = append(out[bucket], parseRule(raw, index, bucket))
		}
	}
	return out
}

func parseRule(raw string, index int, bucket string) rule {
	parsed := rule{raw: raw, tool: raw, index: index, bucket: bucket}
	if open := strings.Index(raw, "("); open > 0 && strings.HasSuffix(raw, ")") {
		parsed.tool = raw[:open]
		parsed.argument = raw[open+1 : len(raw)-1]
		parsed.bounded = true
	}
	return parsed
}

// judgeBreadth reports an allowance that covers everything a tool can do.
func countDead(allowances []rule, broad rule) int {
	dead := 0
	for _, other := range allowances {
		if other.index != broad.index && other.tool == broad.tool && other.bounded {
			dead++
		}
	}
	return dead
}

func judgeBreadth(file string, parsed map[string][]rule) []Finding {
	var findings []Finding
	for _, allowed := range parsed["allow"] {
		if !wholeTool[allowed.tool] {
			continue
		}
		if allowed.bounded && strings.Trim(allowed.argument, "* ") != "" {
			continue
		}
		message := fmt.Sprintf("%s grants every use of %s without a prompt", allowed.raw, allowed.tool)
		if dead := countDead(parsed["allow"], allowed); dead > 0 {
			message += fmt.Sprintf(", which is why %d narrower %s rule(s) below it decide nothing", dead, allowed.tool)
		}
		findings = append(findings, Finding{
			Check: "permission-rule", Severity: Warning, File: file,
			Where:   fmt.Sprintf("permissions.allow.%d", allowed.index),
			Message: message,
			Fix:     "bound it to the commands you meant, or drop it and answer the prompt",
		})
	}
	return findings
}

// judgeOverlap reports a rule that can never decide anything, because a rule
// the runtime reaches first already covers it.
func judgeOverlap(file string, parsed map[string][]rule) []Finding {
	var findings []Finding
	for position, bucket := range buckets {
		for _, candidate := range parsed[bucket] {
			for _, earlier := range buckets[:position+1] {
				for _, other := range parsed[earlier] {
					if other.raw == candidate.raw && other.bucket == candidate.bucket && other.index >= candidate.index {
						continue
					}
					if !covers(other, candidate) {
						continue
					}
					// A bare tool allowance is already one finding of its own,
					// and every rule it swallows is that same defect.
					if !other.bounded && other.bucket == "allow" && wholeTool[other.tool] {
						goto next
					}
					findings = append(findings, Finding{
						Check: "permission-rule", Severity: Warning, File: file,
						Where:   fmt.Sprintf("permissions.%s.%d", bucket, candidate.index),
						Message: shadowMessage(candidate, other),
						Fix:     fmt.Sprintf("drop %s, or narrow %s so both decide something", candidate.raw, other.raw),
					})
					goto next
				}
			}
		next:
		}
	}
	return findings
}

func shadowMessage(candidate, other rule) string {
	if candidate.bucket == other.bucket {
		return fmt.Sprintf("%s is already covered by %s in the same list", candidate.raw, other.raw)
	}
	return fmt.Sprintf("%s never applies: %s is in %s, which the runtime reaches first", candidate.raw, other.raw, other.bucket)
}

// covers reports whether one rule decides everything another would.
func covers(broad, narrow rule) bool {
	if broad.raw == narrow.raw && broad.bucket == narrow.bucket && broad.index == narrow.index {
		return false
	}
	if broad.tool != narrow.tool {
		return false
	}
	if !broad.bounded {
		return true
	}
	if !narrow.bounded {
		return false
	}
	if broad.argument == narrow.argument {
		return broad.bucket != narrow.bucket || broad.index < narrow.index
	}
	return globMatches(broad.argument, narrow.argument)
}

// globMatches reports whether a pattern covers a literal or a narrower pattern,
// with * standing for any run of characters, as the rule syntax defines it.
func globMatches(pattern, value string) bool {
	if !strings.Contains(pattern, "*") {
		return false
	}
	var builder strings.Builder
	builder.WriteString("^")
	for _, part := range strings.Split(pattern, "*") {
		builder.WriteString(regexp.QuoteMeta(part))
		builder.WriteString(".*")
	}
	expression := strings.TrimSuffix(builder.String(), ".*") + "$"
	if strings.HasSuffix(pattern, "*") {
		expression = strings.TrimSuffix(expression, "$") + ".*$"
	}
	matcher, err := regexp.Compile(expression)
	if err != nil {
		return false
	}
	return matcher.MatchString(value)
}
