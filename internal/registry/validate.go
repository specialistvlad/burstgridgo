package registry

import (
	"fmt"
	"strings"
)

// ValidateRegistry performs a strict parity check between the inputs declared
// in HCL manifests and the fields defined in the corresponding Go input structs.
// It uses reflection to read the `bggo` tags from the Go struct and ensures
// the two sets of inputs are identical. This prevents runtime errors caused by
// a mismatch between a module's Go code and its public manifest.
func (r *Registry) ValidateRegistry() error {
	var errs []string

	for runnerType, def := range r.DefinitionRegistry {
		handler, ok := r.HandlerRegistry[def.Lifecycle.OnRun]
		if !ok {
			// This case is unlikely as it's caught earlier, but we check for completeness.
			continue
		}

		// If the handler doesn't define an input type, there's nothing to validate.
		if handler.InputType == nil {
			if len(def.Inputs) > 0 {
				errs = append(errs, fmt.Sprintf("runner '%s': manifest declares inputs, but Go handler has no input struct", runnerType))
			}
			continue
		}

		// 1. Get all declared input names from the HCL manifest.
		hclInputs := make(map[string]struct{})
		for name := range def.Inputs {
			hclInputs[name] = struct{}{}
		}

		// 2. Get all field names from the Go Input struct using bggo tags.
		goInputs := make(map[string]struct{})
		inputType := handler.InputType
		for i := 0; i < inputType.NumField(); i++ {
			field := inputType.Field(i)
			if !field.IsExported() {
				continue
			}

			tag := field.Tag.Get("bggo")
			tagName := strings.Split(tag, ",")[0]

			if tagName != "" && tagName != "-" {
				goInputs[tagName] = struct{}{}
			}
		}

		// 3. Compare the two sets to find any mismatches.
		for name := range goInputs {
			if _, ok := hclInputs[name]; !ok {
				errs = append(errs, fmt.Sprintf("runner '%s': Go struct has field '%s' not declared in manifest", runnerType, name))
			}
		}
		for name := range hclInputs {
			if _, ok := goInputs[name]; !ok {
				errs = append(errs, fmt.Sprintf("runner '%s': manifest declares input '%s' not found in Go struct", runnerType, name))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("registry validation failed:\n- %s", strings.Join(errs, "\n- "))
	}

	return nil
}
