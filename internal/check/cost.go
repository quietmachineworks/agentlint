package check

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/quietmachineworks/agentlint/internal/inventory"
)

// defaultTimeout is what the runtime allows a hook that declares none.
const defaultTimeout = 60.0

// HookCost prices a matcher by what it costs per matching call: how many hooks
// sit on it and the worst case they add up to. It never reports a defect. An
// expensive hook is a choice, and this names the price of the choice.
type HookCost struct{}

func (HookCost) Name() string { return "hook-cost" }

func (HookCost) Run(inv *inventory.Inventory) []Finding {
	var findings []Finding
	for _, settings := range append(append([]inventory.Settings{}, inv.Settings...), inv.HookSources...) {
		events := make([]string, 0, len(settings.Hooks))
		for event := range settings.Hooks {
			events = append(events, event)
		}
		sort.Strings(events)
		for _, event := range events {
			count := map[string]int{}
			worst := map[string]float64{}
			var matchers []string
			for _, matcher := range settings.Hooks[event] {
				if _, seen := count[matcher.Matcher]; !seen {
					matchers = append(matchers, matcher.Matcher)
				}
				for _, hook := range matcher.Hooks {
					count[matcher.Matcher]++
					timeout := defaultTimeout
					if hook.Timeout != nil {
						timeout = *hook.Timeout
					}
					worst[matcher.Matcher] += timeout
				}
			}
			for _, matcher := range matchers {
				// The floor is fixed. Every configuration has one hook on
				// Bash, and a report that lists it lists nothing.
				if count[matcher] < 3 && worst[matcher] < 2*defaultTimeout {
					continue
				}
				findings = append(findings, Finding{
					Check: "hook-cost", Severity: Warning, File: settings.Path,
					Where:   fmt.Sprintf("%s[%s]", event, orNone(matcher)),
					Message: fmt.Sprintf("%s on %s, %s worst case per matching call", plural(count[matcher], "hook"), breadth(matcher), seconds(worst[matcher])),
					Fix:     "each runs on every call this matcher admits; narrow the matcher, drop what is not needed per call, and give each entry a timeout",
				})
			}
		}
	}
	return findings
}

// breadth says how many calls a matcher admits, in words a reader can price.
func breadth(matcher string) string {
	switch {
	case matcher == "":
		return "every call of this event"
	case strings.Contains(matcher, "Bash") || strings.Contains(matcher, "Write") || strings.Contains(matcher, "Edit"):
		return fmt.Sprintf("the hot matcher %q", matcher)
	}
	return strconv.Quote(matcher)
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func seconds(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) + "s" }
