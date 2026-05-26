package toolregistry

import "fmt"

// Validate checks a tool definition for structural errors beyond what YAML
// parsing catches. Returns a list of warnings (non-fatal) and errors (fatal).
func Validate(def *ToolDef) (warnings []string, errors []error) {
	if def.ID == "" {
		errors = append(errors, fmt.Errorf("tool id is required"))
	}
	if def.Binary == "" {
		errors = append(errors, fmt.Errorf("tool %q: binary is required", def.ID))
	}
	if def.Output.Format == "" {
		errors = append(errors, fmt.Errorf("tool %q: output.format is required", def.ID))
	}

	for key, p := range def.Parameters {
		switch p.Type {
		case ParamEnum:
			if len(p.Values) == 0 && p.Preset == "" {
				warnings = append(warnings, fmt.Sprintf("tool %q param %q: enum with no values or preset", def.ID, key))
			}
		case ParamString, ParamInt, ParamStringList, ParamPath:
			// OK
		default:
			errors = append(errors, fmt.Errorf("tool %q param %q: unknown type %q", def.ID, key, p.Type))
		}

		if p.Required && p.Flag == "" && p.Preset == "" {
			// Enum without flag but with value_flags is OK
			if p.Type != ParamEnum || len(p.ValueFlags) == 0 {
				warnings = append(warnings, fmt.Sprintf("tool %q param %q: required but has no flag", def.ID, key))
			}
		}
	}

	return warnings, errors
}

// ValidateAll validates all tools in the registry.
func (r *Registry) ValidateAll() (warnings []string, errors []error) {
	for _, def := range r.tools {
		w, e := Validate(def)
		warnings = append(warnings, w...)
		errors = append(errors, e...)
	}
	return warnings, errors
}
