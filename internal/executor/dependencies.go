package executor

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/registry"
)

// buildDepsStruct populates the `deps` struct for a step handler.
func (e *Executor) buildDepsStruct(ctx context.Context, node *dag.Node, handler *registry.RegisteredRunner, evalCtx *hcl.EvalContext) (any, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Building dependency struct.", "step", node.ID)
	depsStruct := handler.NewDeps()
	if node.StepConfig.Uses == nil || node.StepConfig.Uses.Body == nil {
		logger.Debug("Step has no 'uses' block, returning empty deps.", "step", node.ID)
		return depsStruct, nil
	}

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
		if lookupKey == "" || lookupKey == "-" {
			continue
		}

		resourceExpr, ok := usesMap[lookupKey]
		if !ok {
			continue
		}

		vars := resourceExpr.Variables()
		if len(vars) != 1 {
			return nil, fmt.Errorf("field '%s' in 'uses' must be a direct reference to one resource", lookupKey)
		}
		resourceID, err := traversableToID(vars[0])
		if err != nil {
			return nil, err
		}

		instance, found := e.resourceInstances.Load(resourceID)
		if !found {
			return nil, fmt.Errorf("step '%s' requires resource '%s', which has not been created", node.ID, resourceID)
		}

		instanceType := reflect.TypeOf(instance)
		fieldType := field.Type

		if fieldType.Kind() == reflect.Interface {
			if !instanceType.Implements(fieldType) {
				return nil, fmt.Errorf("type mismatch for '%s': resource %v does not implement required interface %v", lookupKey, instanceType, fieldType)
			}
		} else if !instanceType.AssignableTo(fieldType) {
			return nil, fmt.Errorf("type mismatch for '%s': resource of type %v is not assignable to field of type %v", lookupKey, instanceType, fieldType)
		}

		logger.Debug("Injecting resource dependency.", "step", node.ID, "field", lookupKey, "resourceID", resourceID)
		depsValue.Field(i).Set(reflect.ValueOf(instance))
	}

	logger.Debug("Successfully built dependency struct.", "step", node.ID)
	return depsStruct, nil
}

// traversableToID converts an HCL traversal for a resource into its canonical string ID.
func traversableToID(v hcl.Traversal) (string, error) {
	if len(v) < 3 {
		return "", fmt.Errorf("invalid resource traversal")
	}
	if v.RootName() != "resource" {
		return "", fmt.Errorf("expected a 'resource' traversal, got '%s'", v.RootName())
	}
	return fmt.Sprintf("resource.%s.%s", v[1].(hcl.TraverseAttr).Name, v[2].(hcl.TraverseAttr).Name), nil
}
