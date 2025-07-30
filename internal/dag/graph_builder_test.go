package dag

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
)

func TestBuild_CycleDetection(t *testing.T) {
	t.Parallel() // Mark this test as safe to run in parallel.

	// Arrange: Create steps with a circular dependency (A -> B -> A).
	// Use the full, unambiguous address for dependencies: "runner_type.instance_name".
	stepA := &schema.Step{
		Name:       "A",
		RunnerType: "test",
		Arguments:  &schema.StepArgs{Body: hcl.EmptyBody()},
		DependsOn:  []string{"test.B"},
	}
	stepB := &schema.Step{
		Name:       "B",
		RunnerType: "test",
		Arguments:  &schema.StepArgs{Body: hcl.EmptyBody()},
		DependsOn:  []string{"test.A"},
	}
	gridConfig := &schema.GridConfig{Steps: []*schema.Step{stepA, stepB}}

	// Act: Attempt to create a graph, passing a new registry.
	_, err := Build(context.Background(), gridConfig, registry.New())

	// Assert: Check that an error indicating a cycle was returned.
	if err == nil {
		t.Fatal("Build() should have returned an error for a cyclic dependency, but it did not.")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("Expected error to contain the word 'cycle', but got: %v", err)
	}
}
