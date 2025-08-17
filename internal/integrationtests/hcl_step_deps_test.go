package integration_tests

import (
	"context"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/bggoexpr"
	"github.com/specialistvlad/burstgridgo/internal/handlers"
	"github.com/specialistvlad/burstgridgo/internal/model"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGridLoader_ParsesImplicitDependenciesAndFunctions(t *testing.T) {
	// --- Arrange ---
	gridHCL := `
		step "config" "first" {
			// This step is a dependency for the next one.
		}

		step "print" "second" {
			depends_on = [
				step.config.first,
			]

			for_each = var.iterations

			arguments {
				message = upper("Host is ${var.api_host}")
			}
		}
	`
	files := map[string]string{
		"grid/main.hcl": gridHCL,
	}

	// Register dummy handlers for the runners used in the grid.
	handlerStorage := handlers.New()
	mockHandler := &handlers.RegisteredHandler{
		Input: func() any { return new(struct{}) },
		Fn:    func(ctx context.Context, deps, input any) (any, error) { return nil, nil },
	}
	handlerStorage.RegisterHandler("OnRunConfig", mockHandler)
	handlerStorage.RegisterHandler("OnRunPrint", mockHandler)

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, handlerStorage)

	// --- Assert ---
	require.NoError(t, result.Err, "The application run should not produce an error")
	require.NotNil(t, result.App.Grid(), "Grid should be loaded")

	var targetStep *model.Step
	for _, s := range result.App.Grid().Steps {
		if s.Name == "second" {
			targetStep = s
			break
		}
	}
	require.NotNil(t, targetStep, "Could not find 'second' step in parsed grid")

	// --- Assert on References (Variables) ---
	var references []string
	for _, ref := range targetStep.Expressions.References() {
		references = append(references, bggoexpr.TraversalKey(ref))
	}
	expectedReferences := []string{
		"var.iterations",
		"var.api_host",
		"step.config.first",
	}
	assert.ElementsMatch(t, expectedReferences, references, "The extracted variable references do not match")

	// --- Assert on Called Functions ---
	expectedFunctions := []string{"upper"}
	assert.ElementsMatch(t, expectedFunctions, targetStep.Expressions.CalledFunctions(), "The extracted function calls do not match")
}
