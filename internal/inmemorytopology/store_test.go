package inmemorytopology

import (
	"context"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddAndGetNode(t *testing.T) {
	s := New()
	ctx := context.Background()
	addr, _ := nodeid.Parse("step.test.0")
	testNode := &node.Node{ID: *addr} // FIX: Dereference addr

	// Add the node
	err := s.AddNode(ctx, testNode)
	require.NoError(t, err)

	// Get the node
	retrievedNode, ok := s.GetNode(ctx, *addr) // FIX: Dereference addr
	require.True(t, ok)
	assert.Equal(t, testNode, retrievedNode)
}

func TestDependencies(t *testing.T) {
	s := New()
	ctx := context.Background()
	addr1, _ := nodeid.Parse("step.a.0")
	addr2, _ := nodeid.Parse("step.b.0")
	node1 := &node.Node{ID: *addr1} // FIX: Dereference addr1
	node2 := &node.Node{ID: *addr2} // FIX: Dereference addr2

	s.AddNode(ctx, node1)
	s.AddNode(ctx, node2)

	// Add dependency: b depends on a
	err := s.AddDependency(ctx, *addr1, *addr2) // FIX: Dereference both
	require.NoError(t, err)

	// Check dependencies of b
	deps, err := s.DependenciesOf(ctx, *addr2) // FIX: Dereference addr2
	require.NoError(t, err)
	require.Len(t, deps, 1)
	assert.True(t, addr1.Equal(&deps[0])) // FIX: Correct comparison logic
}
