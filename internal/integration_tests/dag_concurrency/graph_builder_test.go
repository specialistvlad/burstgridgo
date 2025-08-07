package integration_tests

import (
	"context"
	"strings"
	"testing"

	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/registry"
)

func TestBuild_CycleDetection(t *testing.T) {
	t.Parallel()

	// Arrange: Create steps with a circular dependency (A -> B -> A)
	// using the new format-agnostic config types.
	stepA := &config.Step{
		Name:       "A",
		RunnerType: "test",
		Arguments:  nil, // Arguments are not needed for this test.
		DependsOn:  []string{"test.B"},
	}
	stepB := &config.Step{
		Name:       "B",
		RunnerType: "test",
		Arguments:  nil,
		DependsOn:  []string{"test.A"},
	}

	// Build the config.Model that dag.Build now expects.
	model := &config.Model{
		Grid: &config.Grid{
			Steps: []*config.Step{stepA, stepB},
		},
	}

	// Act: Attempt to create a graph, passing the new model.
	_, err := dag.Build(context.Background(), model, registry.New())

	// Assert: Check that an error indicating a cycle was returned.
	if err == nil {
		t.Fatal("Build() should have returned an error for a cyclic dependency, but it did not.")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("Expected error to contain the word 'cycle', but got: %v", err)
	}
}
