package check_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/quietmachineworks/agentlint/internal/check"
	"github.com/quietmachineworks/agentlint/internal/inventory"
)

func run(t *testing.T) []check.Finding {
	t.Helper()
	root, err := filepath.Abs("../../testdata/config")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	inv, err := inventory.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return check.Run(inv, check.All())
}

func find(findings []check.Finding, name string) []check.Finding {
	var out []check.Finding
	for _, finding := range findings {
		if finding.Check == name {
			out = append(out, finding)
		}
	}
	return out
}

func TestMisspeltSettingKeyIsAnErrorWithASuggestion(t *testing.T) {
	for _, finding := range find(run(t), "settings-unknown-key") {
		if finding.Where != "permisions" {
			continue
		}
		if finding.Severity != check.Error {
			t.Fatalf("severity is %v", finding.Severity)
		}
		if finding.Fix == "" {
			t.Fatal("no suggestion offered")
		}
		return
	}
	t.Fatal("permisions was not caught")
}

func TestUndocumentedSettingKeyIsOnlyAWarning(t *testing.T) {
	for _, finding := range find(run(t), "settings-unknown-key") {
		if finding.Where == "feedbackDrafts" {
			if finding.Severity != check.Warning {
				t.Fatal("an undocumented key must not fail a run")
			}
			return
		}
	}
	t.Fatal("feedbackDrafts was not reported")
}

func TestUnknownHookEventIsCaught(t *testing.T) {
	findings := find(run(t), "settings-unknown-hook-event")
	if len(findings) != 1 || findings[0].Where != "PostToolYuse" {
		t.Fatalf("got %v", findings)
	}
}

func TestHookPointingAtAMissingScriptIsAnError(t *testing.T) {
	findings := find(run(t), "hook-unresolvable")
	if len(findings) != 1 {
		t.Fatalf("want exactly the missing script, got %d: %v", len(findings), findings)
	}
	if findings[0].Severity != check.Error {
		t.Fatal("a hook that cannot run is an error")
	}
}

// A command whose target only the runtime can expand is unverifiable, and an
// unverifiable hook is never reported as broken.
func TestRuntimeOnlyVariableIsNotReported(t *testing.T) {
	for _, finding := range find(run(t), "hook-unresolvable") {
		if filepath.Base(finding.Where) == "PreToolUse[Bash]" {
			t.Fatal("reported a hook it could not resolve")
		}
	}
}

func TestSkillNameMustMatchItsFolder(t *testing.T) {
	findings := find(run(t), "skill-frontmatter")
	if len(findings) != 1 {
		t.Fatalf("want only the misnamed skill, got %d: %v", len(findings), findings)
	}
}

func TestAgentWithoutFrontmatterIsCaught(t *testing.T) {
	findings := find(run(t), "agent-frontmatter")
	if len(findings) != 1 || findings[0].Severity != check.Error {
		t.Fatalf("got %v", findings)
	}
}

// A regex in a published schema is the likeliest place for it to be a
// simplification of the parser it describes, so a rule the pattern rejects is
// reported as suspect and never fails a run on its own.
func TestPatternMismatchIsOnlyAWarning(t *testing.T) {
	for _, finding := range find(run(t), "settings-schema") {
		if !strings.HasPrefix(finding.Where, "permissions.allow") {
			continue
		}
		if finding.Severity != check.Warning {
			t.Fatalf("a pattern mismatch must not be an error: %v", finding)
		}
		if finding.Fix == "" {
			t.Fatal("no caveat attached to a pattern mismatch")
		}
		return
	}
	t.Fatal("the rule the published pattern rejects was not reported")
}

// One defect is one finding: the schema also rejects an unknown hook event, and
// the dedicated check says it better.
func TestUnknownHookEventIsNotReportedTwice(t *testing.T) {
	for _, finding := range find(run(t), "settings-schema") {
		if len(finding.Where) >= 5 && finding.Where[:5] == "hooks" {
			t.Fatalf("duplicate of settings-unknown-hook-event: %v", finding)
		}
	}
}
