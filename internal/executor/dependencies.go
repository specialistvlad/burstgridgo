package executor

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/builder"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
)

// buildDepsStruct populates the `deps` struct for a step handler.
func (e *Executor) buildDepsStruct(ctx context.Context, node *builder.Node, handler *registry.RegisteredRunner) (any, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Building dependency struct.", "step", node.ID)
	depsStruct := handler.NewDeps()

	if node.StepConfig.Uses == nil {
		logger.Debug("Step has no 'uses' block, returning empty deps.", "step", node.ID)
		return depsStruct, nil
	}

	usesMap := node.StepConfig.Uses
	depsValue := reflect.ValueOf(depsStruct).Elem()
	depsType := depsValue.Type()

	for i := 0; i < depsValue.NumField(); i++ {
		field := depsType.Field(i)
		fieldLogger := logger.With("step", node.ID, "go_field", field.Name)

		tag := field.Tag.Get("bggo")
		if tag == "" || tag == "-" {
			fieldLogger.Debug("Dependency field has no 'bggo' tag, skipping.")
			continue
		}

		parts := strings.Split(tag, ",")
		lookupKey := parts[0]
		fieldLogger.Debug("Processing dependency field.", "tag", tag, "lookup_key", lookupKey)

		resourceExpr, ok := usesMap[lookupKey]
		if !ok {
			fieldLogger.Debug("No matching entry in 'uses' block for key, skipping.", "key", lookupKey)
			// This is not an error if the field is optional in the Go struct.
			// For now, we assume all `uses` are required if present.
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
		fieldLogger.Debug("Resolved resource dependency.", "resource_id", resourceID)

		instance, found := e.resourceInstances.Load(resourceID)
		if !found {
			return nil, fmt.Errorf("step '%s' requires resource '%s', which has not been created", node.ID, resourceID)
		}

		instanceType := reflect.TypeOf(instance)
		fieldType := field.Type

		if fieldType.Kind() == reflect.Interface {
			if !instanceType.Implements(fieldType) {
				err := fmt.Errorf("type mismatch for '%s': resource of type %v does not implement required interface %v", lookupKey, instanceType, fieldType)
				fieldLogger.Error("Dependency injection failed.", "error", err)
				return nil, err
			}
		} else if !instanceType.AssignableTo(fieldType) {
			err := fmt.Errorf("type mismatch for '%s': resource of type %v is not assignable to field of type %v", lookupKey, instanceType, fieldType)
			fieldLogger.Error("Dependency injection failed.", "error", err)
			return nil, err
		}

		fieldLogger.Debug("Injecting resource dependency.")
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
