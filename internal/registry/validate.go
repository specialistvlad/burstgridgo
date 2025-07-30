package registry

import (
	"fmt"
	"reflect"
	"strings"
)

// validateRegistry performs a strict parity check between HCL manifests and Go structs.
func (r *Registry) ValidateRegistry() error {
	var errs []string

	for runnerType, def := range r.DefinitionRegistry {
		handler, ok := r.HandlerRegistry[def.Lifecycle.OnRun]
		if !ok {
			// This case should be caught by other mechanisms, but we check for completeness.
			continue
		}

		// Get all declared input names from the HCL manifest.
		hclInputs := make(map[string]struct{})
		for _, inputDef := range def.Inputs {
			hclInputs[inputDef.Name] = struct{}{}
		}

		// Get all field names from the Go Input struct using reflection.
		goInputs := make(map[string]struct{})
		inputStruct := handler.NewInput()
		if inputStruct != nil {
			val := reflect.TypeOf(inputStruct).Elem()
			for i := 0; i < val.NumField(); i++ {
				field := val.Field(i)
				tag := strings.Split(field.Tag.Get("hcl"), ",")[0]
				if tag != "" && tag != "-" {
					goInputs[tag] = struct{}{}
				}
			}
		}

		// Compare the two sets.
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
