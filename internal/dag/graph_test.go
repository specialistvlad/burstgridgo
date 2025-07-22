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

	// Arrange: Create modules with a circular dependency (A -> B -> A).
	moduleA := &engine.Module{
		Name:      "A",
		Runner:    "test",
		Body:      hcl.EmptyBody(),
		DependsOn: []string{"B"},
	}
	moduleB := &engine.Module{
		Name:      "B",
		Runner:    "test",
		Body:      hcl.EmptyBody(),
		DependsOn: []string{"A"},
	}
	modules := []*engine.Module{moduleA, moduleB}

	// Act: Attempt to create a graph, which should fail.
	_, err := NewGraph(modules)

	// Assert: Check that an error indicating a cycle was returned.
	if err == nil {
		t.Fatal("NewGraph should have returned an error for a cyclic dependency, but it did not.")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("Expected error to contain the word 'cycle', but got: %v", err)
	}
}
