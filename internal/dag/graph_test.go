// ./internal/dag/graph_test.go

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
	// We use engine.Step as per the updated architecture.
	stepA := &engine.Step{
		Name:       "A",
		RunnerType: "test", // Renamed from Runner to RunnerType
		Arguments:  hcl.EmptyBody(),
		DependsOn:  []string{"B"},
	}
	stepB := &engine.Step{
		Name:       "B",
		RunnerType: "test", // Renamed from Runner to RunnerType
		Arguments:  hcl.EmptyBody(),
		DependsOn:  []string{"A"},
	}
	steps := []*engine.Step{stepA, stepB} // Use a slice of *engine.Step

	// Act: Attempt to create a graph, which should fail.
	_, err := NewGraph(steps) // Pass the slice of *engine.Step

	// Assert: Check that an error indicating a cycle was returned.
	if err == nil {
		t.Fatal("NewGraph should have returned an error for a cyclic dependency, but it did not.")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("Expected error to contain the word 'cycle', but got: %v", err)
	}
}
