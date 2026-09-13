package check_test

import (
	"path/filepath"
	"reflect"
	"sort"
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
	inv, err := inventory.Load(root, root)
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

// A report that pastes a credential into a terminal, a log or a CI annotation
// has leaked it, so a finding on a secret-bearing field names the field and
// the shape of the problem and never the value.
func TestCredentialValuesNeverReachAFinding(t *testing.T) {
	reported := 0
	for _, finding := range run(t) {
		if strings.Contains(finding.Message, "12345") || strings.Contains(finding.Message, "99") {
			t.Fatalf("a value from a secret-bearing field reached the report: %v", finding)
		}
		if strings.HasPrefix(finding.Where, "env") || strings.HasPrefix(finding.Where, "apiKeyHelper") {
			reported++
		}
	}
	if reported == 0 {
		t.Fatal("the malformed secret-bearing fields were not reported at all")
	}
}

func messages(findings []check.Finding, name string) []string {
	var out []string
	for _, finding := range find(findings, name) {
		out = append(out, finding.Where+": "+finding.Message)
	}
	return out
}

func TestBareToolAllowanceIsReportedOnceWithWhatItSwallows(t *testing.T) {
	got := messages(run(t), "permission-rule")
	broad := 0
	for _, message := range got {
		if strings.Contains(message, "grants every use of Bash") {
			broad++
			if !strings.Contains(message, "decide nothing") {
				t.Fatalf("the rules it swallows are not counted: %s", message)
			}
		}
	}
	if broad != 1 {
		t.Fatalf("want one breadth finding, got %d in %v", broad, got)
	}
}

func TestADeniedRuleKillsTheAllowance(t *testing.T) {
	for _, message := range messages(run(t), "permission-rule") {
		if strings.Contains(message, "rm -rf /tmp/build") && strings.Contains(message, "deny") {
			return
		}
	}
	t.Fatal("an allowance a denial already covers was not reported")
}

func TestNarrowerRuleUnderAGlobIsReported(t *testing.T) {
	for _, message := range messages(run(t), "permission-rule") {
		if strings.Contains(message, "Bash(git status)") && strings.Contains(message, "Bash(git *)") {
			return
		}
	}
	t.Fatal("a rule already covered by a glob in the same list was not reported")
}

func TestAnOverriddenScalarIsReported(t *testing.T) {
	var overridden []string
	for _, finding := range find(run(t), "settings-precedence") {
		overridden = append(overridden, finding.Where)
	}
	sort.Strings(overridden)
	want := []string{"cleanupPeriodDays", "model"}
	if !reflect.DeepEqual(overridden, want) {
		t.Fatalf("want %v, got %v", want, overridden)
	}
}

// No published order places a settings.local.json beside the user file against
// the levels below it, so nothing here may claim that it wins or loses.
func TestUndocumentedScopeMakesNoPrecedenceClaim(t *testing.T) {
	for _, finding := range find(run(t), "settings-precedence") {
		if strings.HasSuffix(finding.File, filepath.Join("config", "settings.local.json")) {
			t.Fatalf("claimed an order the documentation does not give: %v", finding)
		}
		if strings.Contains(finding.Message, "user local") {
			t.Fatalf("used an unranked scope as the winner: %v", finding)
		}
	}
}

// permissions and hooks are merged by the runtime rather than replaced, so a
// key set in both files is two contributions and not a contradiction.
func TestMergedKeysAreNotReportedAsOverridden(t *testing.T) {
	for _, finding := range find(run(t), "settings-precedence") {
		if finding.Where == "permissions" || finding.Where == "hooks" {
			t.Fatalf("a merged key was called an override: %v", finding)
		}
	}
}

func TestOneNameInstalledTwiceIsReported(t *testing.T) {
	findings := find(run(t), "name-collision")
	if len(findings) != 1 || findings[0].Where != "good" {
		t.Fatalf("got %v", findings)
	}
}

func TestUnstartableMCPServersAreReported(t *testing.T) {
	got := messages(run(t), "mcp-unresolvable")
	if len(got) != 3 {
		t.Fatalf("want the missing command, the url-less http server and the shapeless one, got %v", got)
	}
	for _, name := range []string{"gone", "nameless", "shapeless"} {
		found := false
		for _, message := range got {
			if strings.HasPrefix(message, name+" ") {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s was not reported: %v", name, got)
		}
	}
	// A POSIX path taken for a bare command name reports as missing from PATH
	// rather than as missing, which is what Windows saw first.
	for _, message := range got {
		if strings.HasPrefix(message, "gone ") && !strings.Contains(message, "does not exist") {
			t.Fatalf("a path was judged as a PATH lookup: %s", message)
		}
	}
}

func TestMintedCredentialsAreFoundInBothSettingsAndMCP(t *testing.T) {
	got := messages(run(t), "secret-in-config")
	var inSettings, inMCP bool
	for _, message := range got {
		if strings.HasPrefix(message, "env.ANTHROPIC_API_KEY") {
			inSettings = true
		}
		if strings.HasPrefix(message, "leaky.env.SERVICE_TOKEN") {
			inMCP = true
		}
		if strings.Contains(message, "sk-ant") || strings.Contains(message, "ghp_") {
			t.Fatalf("the credential itself reached the report: %s", message)
		}
	}
	if !inSettings || !inMCP {
		t.Fatalf("settings=%v mcp=%v in %v", inSettings, inMCP, got)
	}
}

// A value that points at a secret is not a secret, and a linter that cannot
// tell the difference makes every correct configuration look wrong.
func TestReferencesAndPlainValuesAreNotCalledSecrets(t *testing.T) {
	for _, message := range messages(run(t), "secret-in-config") {
		if strings.Contains(message, "SAFE_REFERENCE") || strings.Contains(message, "PATH_EXTRA") {
			t.Fatalf("a value that is not a credential was reported: %s", message)
		}
	}
}

// A plugin's tree on disk is not what runs. The cache keeps every version ever
// fetched and the marketplace keeps every plugin ever offered, so an inventory
// that walks either one prices an agent heavier than the one that starts and
// reports defects in files nothing reads.
func TestOnlyInstalledAndEnabledPluginContentIsLoaded(t *testing.T) {
	root, err := filepath.Abs("../../testdata/config")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	inv, err := inventory.Load(root, root)
	if err != nil {
		t.Fatal(err)
	}
	var stems []string
	for _, skill := range inv.Skills {
		stems = append(stems, skill.Stem)
	}
	sort.Strings(stems)
	want := []string{"good", "good", "live-skill", "misnamed"}
	if !reflect.DeepEqual(stems, want) {
		t.Fatalf("want %v, got %v", want, stems)
	}
	for _, skill := range inv.Skills {
		if strings.Contains(skill.Path, "stale") || strings.Contains(skill.Path, "marketplaces") {
			t.Fatalf("read a tree the agent never loads: %s", skill.Path)
		}
	}
}

// A stale version of one plugin is not the same skill installed twice.
func TestStaleVersionsAreNotReportedAsCollisions(t *testing.T) {
	for _, finding := range find(run(t), "name-collision") {
		if finding.Where == "live-skill" {
			t.Fatalf("a cached older version was called a collision: %v", finding)
		}
	}
}
