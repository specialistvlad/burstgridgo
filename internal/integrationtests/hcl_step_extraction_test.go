package integration_tests

import (
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/bggohcl"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHCLExtraction_IsUniqueAndSorted(t *testing.T) {
	gridHCL := `
		step "print" "a" {
			arguments {
				arg1 = var.foo
				arg2 = var.foo
				arg3 = upper("B")
				arg4 = lower("A")
			}
		}
	`
	result, steps := testutil.RunHCLGridTest(t, gridHCL)

	require.NoError(t, result.Err)
	require.Len(t, steps, 1)
	step := steps[0]

	// Assert on uniqueness
	require.Len(t, step.Expressions.References(), 1, "Should have found only one unique variable reference")
	require.Len(t, step.Expressions.CalledFunctions(), 2, "Should have found two unique function calls")

	// Assert on content and sort order
	refStrings := []string{bggohcl.TraversalKey(step.Expressions.References()[0])}
	require.Equal(t, []string{"var.foo"}, refStrings)
	require.Equal(t, []string{"lower", "upper"}, step.Expressions.CalledFunctions(), "Function calls should be sorted alphabetically")
}

func TestHCLExtraction_ComplexTraversals(t *testing.T) {
	gridHCL := `
		step "print" "a" {
			arguments {
				arg1 = var.list[0]
				arg2 = var.map["key-name"]
				arg3 = var.resources[*].id
			}
		}
	`
	result, steps := testutil.RunHCLGridTest(t, gridHCL)
	require.NoError(t, result.Err)
	require.Len(t, steps, 1)
	step := steps[0]

	require.Len(t, step.Expressions.References(), 3, "Should find three distinct complex traversals")

	refStrings := make([]string, len(step.Expressions.References()))
	for i, ref := range step.Expressions.References() {
		refStrings[i] = bggohcl.TraversalKey(ref)
	}

	// NOTE: For a splat expression, the HCL library correctly identifies the
	// dependency as the collection itself (`var.resources`), not the full expression.
	expectedRefs := []string{
		`var.list[0]`,
		`var.map["key-name"]`,
		`var.resources`,
	}
	assert.ElementsMatch(t, expectedRefs, refStrings)
}

func TestHCLExtraction_AllExpressionTypes(t *testing.T) {
	gridHCL := `
		step "test" "comprehensive" {
			arguments {
				binary   = len(var.list) + abs(-1)
				cond     = var.cond ? upper("A") : lower("B")
				template = "val is ${upper(var.name)}"
				tuple    = [upper("x")]
				object   = { key = upper("y") }
				for      = [for v in var.items: upper(v)]
				indexed  = var.all_vals[0]
				splat    = var.all_objs[*].name
				parens   = (upper("z"))
			}
		}
	`
	result, steps := testutil.RunHCLGridTest(t, gridHCL)
	require.NoError(t, result.Err)
	require.Len(t, steps, 1)
	step := steps[0]

	// --- Assert on References ---
	refStrings := make([]string, len(step.Expressions.References()))
	for i, ref := range step.Expressions.References() {
		refStrings[i] = bggohcl.TraversalKey(ref)
	}
	// NOTE: For an index expression, the HCL library returns the full traversal.
	// For a splat, it returns the collection.
	expectedRefs := []string{
		"var.all_objs",
		"var.all_vals[0]",
		"var.cond",
		"var.items",
		"var.list",
		"var.name",
	}
	require.Equal(t, expectedRefs, refStrings, "Extracted variable references should be complete and sorted")

	// --- Assert on Functions ---
	expectedFns := []string{
		"abs",
		"len",
		"lower",
		"upper",
	}
	require.Equal(t, expectedFns, step.Expressions.CalledFunctions(), "Extracted function calls should be complete, unique, and sorted")
}

func TestHCLExtraction_FindsTryAndCanFunctions(t *testing.T) {
	t.Skip("TODO: Not yet implemented")
	/* HCL Snippet:
	arguments = {
		a = try(var.optional_object.attr, "default")
		b = can(var.numeric_val + 1)
	}
	*/
	// Expected References: `var.optional_object`, `var.numeric_val`.
	// Expected CalledFunctions: `try`, `can`.
}

func TestHCLExtraction_LegacySplatExpressions(t *testing.T) {
	t.Skip("TODO: Not yet implemented")
	/* HCL Snippet:
	arguments = {
		a = var.list_of_objects.*.id
	}
	*/
	// Expected: Reference for `var.list_of_objects.*.id` is correctly extracted.
}
