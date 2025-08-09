package dag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	g := New()
	require.NotNil(t, g)
	assert.NotNil(t, g.nodes)
	assert.Empty(t, g.nodes)
}

func TestAddNode(t *testing.T) {
	g := New()

	g.AddNode("a")
	assert.Len(t, g.nodes, 1)
	nodeA, ok := g.nodes["a"]
	require.True(t, ok)
	assert.Equal(t, "a", nodeA.id)
	assert.NotNil(t, nodeA.deps)
	assert.NotNil(t, nodeA.dependents)

	g.AddNode("a") // Test idempotency
	assert.Len(t, g.nodes, 1)

	g.AddNode("b")
	assert.Len(t, g.nodes, 2)
	_, ok = g.nodes["b"]
	assert.True(t, ok)
}

func TestAddEdge(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		g := New()
		g.AddNode("a")
		g.AddNode("b")

		err := g.AddEdge("a", "b") // b depends on a
		require.NoError(t, err)

		nodeA := g.nodes["a"]
		nodeB := g.nodes["b"]

		assert.Contains(t, nodeA.dependents, "b")
		assert.Equal(t, nodeB, nodeA.dependents["b"])
		assert.Contains(t, nodeB.deps, "a")
		assert.Equal(t, nodeA, nodeB.deps["a"])
	})

	t.Run("error cases", func(t *testing.T) {
		g := New()
		g.AddNode("a")
		g.AddNode("b")

		err := g.AddEdge("dne", "a")
		assert.ErrorContains(t, err, "source node not found")

		err = g.AddEdge("a", "dne")
		assert.ErrorContains(t, err, "destination node not found")

		err = g.AddEdge("a", "a")
		assert.ErrorContains(t, err, "self-referential edge")
	})
}

func TestDetectCycles(t *testing.T) {
	t.Run("empty graph has no cycles", func(t *testing.T) {
		g := New()
		assert.NoError(t, g.DetectCycles())
	})

	t.Run("graph with nodes but no edges has no cycles", func(t *testing.T) {
		g := New()
		g.AddNode("a")
		g.AddNode("b")
		g.AddNode("c")
		assert.NoError(t, g.DetectCycles())
	})

	t.Run("valid dag has no cycles", func(t *testing.T) {
		g := New()
		g.AddNode("a")
		g.AddNode("b")
		g.AddNode("c")
		g.AddNode("d")
		require.NoError(t, g.AddEdge("a", "b"))
		require.NoError(t, g.AddEdge("b", "c"))
		require.NoError(t, g.AddEdge("a", "c")) // Transitive edge
		require.NoError(t, g.AddEdge("c", "d"))
		assert.NoError(t, g.DetectCycles())
	})

	t.Run("simple direct cycle is detected", func(t *testing.T) {
		g := New()
		g.AddNode("a")
		g.AddNode("b")
		require.NoError(t, g.AddEdge("a", "b"))
		require.NoError(t, g.AddEdge("b", "a")) // Cycle
		err := g.DetectCycles()
		assert.Error(t, err)
		assert.ErrorContains(t, err, "cycle detected")
	})

	t.Run("longer cycle is detected", func(t *testing.T) {
		g := New()
		g.AddNode("a")
		g.AddNode("b")
		g.AddNode("c")
		g.AddNode("d")
		require.NoError(t, g.AddEdge("a", "b"))
		require.NoError(t, g.AddEdge("b", "c"))
		require.NoError(t, g.AddEdge("c", "d"))
		require.NoError(t, g.AddEdge("d", "a")) // Cycle back to the start
		err := g.DetectCycles()
		assert.Error(t, err)
		assert.ErrorContains(t, err, "cycle detected")
	})

	t.Run("cycle in a disjoint component is detected", func(t *testing.T) {
		g := New()
		// Component 1 (valid)
		g.AddNode("a")
		g.AddNode("b")
		require.NoError(t, g.AddEdge("a", "b"))

		// Component 2 (has a cycle)
		g.AddNode("x")
		g.AddNode("y")
		g.AddNode("z")
		require.NoError(t, g.AddEdge("x", "y"))
		require.NoError(t, g.AddEdge("y", "z"))
		require.NoError(t, g.AddEdge("z", "y")) // Cycle

		err := g.DetectCycles()
		assert.Error(t, err)
		assert.ErrorContains(t, err, "cycle detected")
	})
}
