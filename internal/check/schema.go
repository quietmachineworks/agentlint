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
				Message: leaf.ErrorKind.LocalizedString(englishPrinter()), Fix: note,
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
