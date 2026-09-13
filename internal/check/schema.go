package check

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"

	"github.com/quietmachineworks/agentlint/internal/inventory"
	"github.com/quietmachineworks/agentlint/internal/schema"
)

// SettingsSchema validates a settings file against the published schema.
//
// The schema is the authority on what settings.json may contain, and nothing
// in a normal workflow ever runs it.
type SettingsSchema struct{}

func (SettingsSchema) Name() string { return "settings-schema" }

var (
	compileOnce sync.Once
	compiled    *jsonschema.Schema
	compileErr  error
)

func published() (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema.Raw()))
		if err != nil {
			compileErr = err
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("settings.json", document); err != nil {
			compileErr = err
			return
		}
		compiled, compileErr = compiler.Compile("settings.json")
	})
	return compiled, compileErr
}

func (s SettingsSchema) Run(inv *inventory.Inventory) []Finding {
	validator, err := published()
	if err != nil {
		return []Finding{{
			Check: "settings-schema", Severity: Warning, File: inv.Root,
			Message: fmt.Sprintf("the embedded schema did not compile, so nothing was validated against it: %v", err),
		}}
	}
	var findings []Finding
	for _, settings := range inv.Settings {
		if !settings.Parsed {
			findings = append(findings, Finding{
				Check: "settings-schema", Severity: Error, File: settings.Path,
				Message: fmt.Sprintf("does not parse as JSON: %v", settings.Err),
			})
			continue
		}
		encoded, err := json.Marshal(settings.Raw)
		if err != nil {
			continue
		}
		instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(encoded))
		if err != nil {
			continue
		}
		validationErr := validator.Validate(instance)
		if validationErr == nil {
			continue
		}
		var failure *jsonschema.ValidationError
		if !errorsAs(validationErr, &failure) {
			continue
		}
		for _, leaf := range leaves(failure) {
			where := strings.Join(leaf.InstanceLocation, ".")
			if coveredElsewhere(leaf) {
				continue
			}
			severity, note := weigh(leaf)
			findings = append(findings, Finding{
				Check: "settings-schema", Severity: severity, File: settings.Path, Where: where,
				Message: describe(leaf), Fix: note,
			})
		}
	}
	return findings
}

// coveredElsewhere drops what a dedicated check already reports with a better
// message, so one defect is never two findings.
func coveredElsewhere(leaf *jsonschema.ValidationError) bool {
	if len(leaf.InstanceLocation) == 0 || leaf.InstanceLocation[0] != "hooks" {
		return false
	}
	_, additional := leaf.ErrorKind.(*kind.AdditionalProperties)
	return additional
}

// describe renders one violation short enough to read in a terminal. A pattern
// failure otherwise prints the whole regex, which buries the value that failed.
func describe(leaf *jsonschema.ValidationError) string {
	if pattern, ok := leaf.ErrorKind.(*kind.Pattern); ok {
		if sensitive(leaf.InstanceLocation) {
			return "the value does not match the pattern this field is defined by"
		}
		return fmt.Sprintf("%s does not match the pattern this field is defined by", clamp(fmt.Sprintf("%v", pattern.Got), 90))
	}
	if sensitive(leaf.InstanceLocation) {
		return "the value here is not what this field is defined to hold"
	}
	return clamp(leaf.ErrorKind.LocalizedString(englishPrinter()), 200)
}

// secretish names a field whose value is a credential often enough that no
// report should carry it.
var secretish = []string{"key", "token", "secret", "password", "credential", "auth", "header"}

// sensitive reports whether a location may hold a credential. A finding there
// names the field and the shape of the problem, never the value: a report that
// pastes an API key into a terminal, a log or a CI annotation has leaked it.
func sensitive(location []string) bool {
	for _, segment := range location {
		if segment == "env" {
			return true
		}
		lowered := strings.ToLower(segment)
		for _, needle := range secretish {
			if strings.Contains(lowered, needle) {
				return true
			}
		}
	}
	return false
}

func clamp(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

// weigh ranks one schema violation.
//
// A regex in a published schema is the likeliest place for it to be a
// simplification of the parser it describes, so a pattern mismatch is reported
// as suspect rather than as broken.
func weigh(leaf *jsonschema.ValidationError) (Severity, string) {
	if _, pattern := leaf.ErrorKind.(*kind.Pattern); pattern {
		return Warning, "the published pattern rejects this; confirm against the runtime before rewriting it"
	}
	return Error, ""
}

// leaves returns the deepest causes, which name the actual instance location.
func leaves(err *jsonschema.ValidationError) []*jsonschema.ValidationError {
	if len(err.Causes) == 0 {
		return []*jsonschema.ValidationError{err}
	}
	var out []*jsonschema.ValidationError
	for _, cause := range err.Causes {
		out = append(out, leaves(cause)...)
	}
	return out
}
