package check

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// DescriptionShadowing reports two items whose descriptions claim the same
// request. They do not split the work: one wins every time, and the loser looks
// dormant when it is actually blocked.
//
// This measures the overlap and stops there. Which of the two should be
// narrowed needs usage, and usage lives in the transcripts.
type DescriptionShadowing struct{}

func (DescriptionShadowing) Name() string { return "description-shadowing" }

// overlapFloor is where two descriptions stop sharing a subject and start being
// the same text. Below it sit the neighbours: two hardening skills, one for
// Linux and one for Windows, share most of their vocabulary and none of their
// purpose, and reporting those is how a linter gets muted.
const overlapFloor = 0.7

// shared is the trigger language every description uses, which carries no
// signal about what an item claims.
var shared = map[string]bool{
	"use": true, "when": true, "this": true, "that": true, "with": true, "from": true,
	"into": true, "over": true, "your": true, "they": true, "them": true, "what": true,
	"which": true, "their": true, "these": true, "those": true, "than": true, "then": true,
	"user": true, "using": true, "used": true, "asks": true, "asked": true, "want": true,
	"wants": true, "needs": true, "should": true, "would": true, "could": true, "does": true,
	"also": true, "only": true, "such": true, "each": true, "including": true, "based": true,
}

var word = regexp.MustCompile(`[a-z][a-z0-9-]{3,}`)

type described struct {
	name   string
	path   string
	tokens map[string]bool
}

func (DescriptionShadowing) Run(inv *inventory.Inventory) []Finding {
	items := collect(inv.Skills, inv.Agents)
	var findings []Finding
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].name == items[j].name {
				continue
			}
			score, common := overlap(items[i].tokens, items[j].tokens)
			if score < overlapFloor {
				continue
			}
			findings = append(findings, describeOverlap(inv.Root, items[i], items[j], score, common))
		}
	}
	return findings
}

func describeOverlap(root string, left, right described, score float64, common []string) Finding {
	return Finding{
		Check: "description-shadowing", Severity: Warning, File: left.path, Where: left.name,
		Message: fmt.Sprintf("claims the same request as %s: %.0f%% of the trigger language is shared (%s)",
			right.name, score*100, strings.Join(common, ", ")),
		Fix: fmt.Sprintf("narrow one to what the other does not do, or strike it; the other is %s", trim(root, right.path)),
	}
}

func trim(root, path string) string {
	if relative, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(relative, "..") {
		return relative
	}
	return path
}

func collect(groups ...[]inventory.Definition) []described {
	var items []described
	for _, group := range groups {
		for _, definition := range group {
			description, ok := definition.Frontmatter.Get("description")
			if !ok || description == "" {
				continue
			}
			tokens := map[string]bool{}
			for _, token := range word.FindAllString(strings.ToLower(description), -1) {
				if !shared[token] {
					tokens[token] = true
				}
			}
			if len(tokens) < 4 {
				continue
			}
			items = append(items, described{name: definition.Stem, path: definition.Path, tokens: tokens})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	return items
}

// overlap is the Jaccard index of two token sets, with the tokens they share.
func overlap(left, right map[string]bool) (float64, []string) {
	if len(left) == 0 || len(right) == 0 {
		return 0, nil
	}
	var common []string
	for token := range left {
		if right[token] {
			common = append(common, token)
		}
	}
	union := len(left) + len(right) - len(common)
	if union == 0 {
		return 0, nil
	}
	score := float64(len(common)) / float64(union)
	sort.Strings(common)
	if len(common) > 6 {
		common = common[:6]
	}
	return score, common
}
