package dag

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/engine"
)

// buildDepsStruct populates the `deps` struct for a step handler by mapping live
// resource instances to the fields of the struct based on the 'uses' block.
func (e *Executor) buildDepsStruct(node *Node, runnerDef *engine.RunnerDefinition, handler *engine.RegisteredHandler, evalCtx *hcl.EvalContext) (any, error) {
	depsStruct := handler.NewDeps()
	if node.StepConfig.Uses == nil || node.StepConfig.Uses.Body == nil {
		return depsStruct, nil // No `uses` block, return empty deps struct.
	}

	// The uses block maps local names (fields in Deps struct) to resource IDs.
	// e.g., { client = resource.http_client.shared }
	usesMap := make(map[string]hcl.Expression)
	attrs, diags := node.StepConfig.Uses.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}
	for _, attr := range attrs {
		usesMap[attr.Name] = attr.Expr
	}

	depsValue := reflect.ValueOf(depsStruct).Elem()
	for i := 0; i < depsValue.NumField(); i++ {
		field := depsValue.Type().Field(i)
		lookupKey := field.Tag.Get("hcl")

		// A field must have an `hcl` tag to be considered for injection.
		if lookupKey == "" || lookupKey == "-" {
			continue
		}

		// Find which resource this field maps to from the `uses` block
		resourceExpr, ok := usesMap[lookupKey]
		if !ok {
			continue // This field in the Deps struct isn't set in the HCL.
		}

		// The expression should be a variable traversal, e.g., `resource.http_client.shared`
		vars := resourceExpr.Variables()
		if len(vars) != 1 {
			return nil, fmt.Errorf("field '%s' in 'uses' block must be a direct reference to a single resource", lookupKey)
		}
		resourceID, err := traversableToID(vars[0])
		if err != nil {
			return nil, err
		}

		// Get the live resource object instance
		instance, found := e.resourceInstances.Load(resourceID)
		if !found {
			return nil, fmt.Errorf("step '%s' requires resource '%s' which has not been created", node.ID, resourceID)
		}

		// Type-check: ensure the live object can be assigned to the field.
		instanceType := reflect.TypeOf(instance)
		fieldType := field.Type

		// Check for compatibility. If the target field is an interface, check if the
		// instance implements it. Otherwise, check if the instance is assignable to the field type.
		if fieldType.Kind() == reflect.Interface {
			if !instanceType.Implements(fieldType) {
				return nil, fmt.Errorf("type mismatch for '%s': resource '%s' of type %v does not implement required interface %v", lookupKey, resourceID, instanceType, fieldType)
			}
		} else {
			if !instanceType.AssignableTo(fieldType) {
				return nil, fmt.Errorf("type mismatch for '%s': resource '%s' of type %v is not assignable to field of type %v", lookupKey, resourceID, instanceType, fieldType)
			}
		}

		// Set the field in the deps struct
		depsValue.Field(i).Set(reflect.ValueOf(instance))
	}

	return depsStruct, nil
}

// traversableToID converts an HCL traversal for a resource into its canonical string ID.
func traversableToID(v hcl.Traversal) (string, error) {
	if len(v) < 3 {
		return "", fmt.Errorf("invalid resource traversal")
	}
	root := v.RootName()
	if root != "resource" {
		return "", fmt.Errorf("expected a 'resource' traversal, got '%s'", root)
	}
	return fmt.Sprintf("resource.%s.%s", v[1].(hcl.TraverseAttr).Name, v[2].(hcl.TraverseAttr).Name), nil
}
