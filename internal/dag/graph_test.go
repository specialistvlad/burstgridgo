package dag

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/engine"
)

func TestNewGraph_CycleDetection(t *testing.T) {
	t.Parallel() // Mark this test as safe to run in parallel.

	// Arrange: Create steps with a circular dependency (A -> B -> A).
	// Use the full, unambiguous address for dependencies: "runner_type.instance_name".
	stepA := &engine.Step{
		Name:       "A",
		RunnerType: "test",
		Arguments:  &engine.StepArgs{Body: hcl.EmptyBody()},
		DependsOn:  []string{"test.B"},
	}
	stepB := &engine.Step{
		Name:       "B",
		RunnerType: "test",
		Arguments:  &engine.StepArgs{Body: hcl.EmptyBody()},
		DependsOn:  []string{"test.A"},
	}
	gridConfig := &engine.GridConfig{Steps: []*engine.Step{stepA, stepB}}

	// Act: Attempt to create a graph, which should fail due to a cycle.
	_, err := NewGraph(gridConfig)

	// Assert: Check that an error indicating a cycle was returned.
	if err == nil {
		t.Fatal("NewGraph should have returned an error for a cyclic dependency, but it did not.")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("Expected error to contain the word 'cycle', but got: %v", err)
	}
}
