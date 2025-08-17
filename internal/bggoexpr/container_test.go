package bggoexpr_test

import (
	"sync"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/specialistvlad/burstgridgo/internal/bggoexpr"
	"github.com/specialistvlad/burstgridgo/internal/bggohcl"
	"github.com/stretchr/testify/require"
)

// parseExpr is a test helper to quickly get an hcl.Expression from a string.
func parseExpr(t *testing.T, exprStr string) hcl.Expression {
	t.Helper()
	expr, diags := hclsyntax.ParseExpression([]byte(exprStr), "test.hcl", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "Expression parsing failed: %s", diags.Error())
	return expr
}

func TestContainer_AddAndExtract(t *testing.T) {
	c := bggoexpr.NewContainer()
	c.Add(
		parseExpr(t, `upper("hello")`),
		parseExpr(t, `var.foo.bar`),
		parseExpr(t, `lower(var.foo.baz)`),
		parseExpr(t, `var.foo.bar`), // Duplicate reference
	)

	// --- Assert on Functions (sorted, unique) ---
	expectedFuncs := []string{"lower", "upper"}
	require.Equal(t, expectedFuncs, c.CalledFunctions())

	// --- Assert on References (sorted, unique) ---
	refs := c.References()
	require.Len(t, refs, 2)
	refStrings := []string{
		bggohcl.TraversalKey(refs[0]),
		bggohcl.TraversalKey(refs[1]),
	}
	expectedRefs := []string{"var.foo.bar", "var.foo.baz"}
	require.Equal(t, expectedRefs, refStrings)
}

func TestContainer_Idempotency(t *testing.T) {
	c := bggoexpr.NewContainer()
	c.Add(parseExpr(t, `var.a + var.b`))

	// Call getters multiple times to ensure results are stable and cached
	require.Len(t, c.References(), 2)
	require.Len(t, c.References(), 2)
	require.Empty(t, c.CalledFunctions())
	require.Empty(t, c.CalledFunctions())
}

func TestContainer_AddAfterExtract(t *testing.T) {
	c := bggoexpr.NewContainer()
	c.Add(parseExpr(t, `var.first`))

	// First extraction
	require.Len(t, c.References(), 1)
	require.Equal(t, "var.first", bggohcl.TraversalKey(c.References()[0]))

	// Add more expressions
	c.Add(parseExpr(t, `var.second`), parseExpr(t, `my_func()`))

	// Second extraction should have new data
	require.Len(t, c.CalledFunctions(), 1)
	require.Equal(t, "my_func", c.CalledFunctions()[0])

	refs := c.References()
	require.Len(t, refs, 2)
	refStrings := []string{
		bggohcl.TraversalKey(refs[0]),
		bggohcl.TraversalKey(refs[1]),
	}
	require.Equal(t, []string{"var.first", "var.second"}, refStrings)
}

func TestContainer_ConcurrentAccess(t *testing.T) {
	c := bggoexpr.NewContainer()
	c.Add(
		parseExpr(t, `var.a`),
		parseExpr(t, `var.b`),
		parseExpr(t, `func_a()`),
	)

	var wg sync.WaitGroup
	numGoroutines := 100
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			// Alternate between the two getters
			if i%2 == 0 {
				require.Len(t, c.References(), 2)
			} else {
				require.Len(t, c.CalledFunctions(), 1)
			}
		}()
	}

	wg.Wait()
}

func TestContainer_EdgeCases(t *testing.T) {
	t.Run("Empty Container", func(t *testing.T) {
		c := bggoexpr.NewContainer()
		require.Empty(t, c.References())
		require.Empty(t, c.CalledFunctions())
	})

	t.Run("Adding Nil Expressions", func(t *testing.T) {
		c := bggoexpr.NewContainer()
		c.Add(nil, parseExpr(t, `var.a`), nil)
		require.Len(t, c.References(), 1)
		require.Equal(t, "var.a", bggohcl.TraversalKey(c.References()[0]))
	})
}
