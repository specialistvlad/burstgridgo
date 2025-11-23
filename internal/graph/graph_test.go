package graph

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/inmemorystore"
	"github.com/specialistvlad/burstgridgo/internal/inmemorytopology"
	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestGraph creates a graph manager with in-memory stores for testing
func createTestGraph() Graph {
	topology := inmemorytopology.New()
	nodeState := inmemorystore.New()
	return New(topology, nodeState)
}

// addNodeToGraph is a helper that adds a node to the graph's topology store
func addNodeToGraph(t *testing.T, g Graph, id string, nodeType string) *node.Node {
	t.Helper()
	ctx := context.Background()
	addr, err := nodeid.Parse(id)
	require.NoError(t, err)

	n := &node.Node{
		ID:   *addr,
		Type: nodeType,
		RawConfig: map[string]any{
			"test": "config",
		},
	}

	// Access the underlying topology store to add the node
	// (Graph interface doesn't expose AddNode - that's for the builder to use)
	manager := g.(*Manager)
	err = manager.topology.AddNode(ctx, n)
	require.NoError(t, err)

	return n
}

// addDependency is a helper that adds a dependency to the graph's topology store
func addDependency(t *testing.T, g Graph, from, to string) {
	t.Helper()
	ctx := context.Background()
	fromAddr, err := nodeid.Parse(from)
	require.NoError(t, err)
	toAddr, err := nodeid.Parse(to)
	require.NoError(t, err)

	manager := g.(*Manager)
	err = manager.topology.AddDependency(ctx, *fromAddr, *toAddr)
	require.NoError(t, err)
}

func TestNode_GetExisting(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	// Add a node to the topology
	testNode := addNodeToGraph(t, g, "step.test.0", "print")

	// Retrieve the node via Graph interface
	retrieved, ok := g.Node(ctx, testNode.ID)
	require.True(t, ok)
	assert.Equal(t, testNode, retrieved)
	assert.Equal(t, "print", retrieved.Type)
}

func TestNode_NotFound(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	addr, err := nodeid.Parse("step.nonexistent.0")
	require.NoError(t, err)

	// Try to retrieve a node that doesn't exist
	retrieved, ok := g.Node(ctx, *addr)
	assert.False(t, ok)
	assert.Nil(t, retrieved)
}

func TestAllNodes_Empty(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	nodes := g.AllNodes(ctx)
	assert.Empty(t, nodes)
}

func TestAllNodes_Multiple(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	// Add multiple nodes
	node1 := addNodeToGraph(t, g, "step.first.0", "print")
	node2 := addNodeToGraph(t, g, "step.second.0", "http")
	node3 := addNodeToGraph(t, g, "step.third.0", "file")

	// Retrieve all nodes
	nodes := g.AllNodes(ctx)
	require.Len(t, nodes, 3)

	// Verify all nodes are present (order not guaranteed)
	nodeMap := make(map[string]*node.Node)
	for _, n := range nodes {
		nodeMap[n.ID.String()] = n
	}

	assert.Equal(t, node1, nodeMap[node1.ID.String()])
	assert.Equal(t, node2, nodeMap[node2.ID.String()])
	assert.Equal(t, node3, nodeMap[node3.ID.String()])
}

func TestDependenciesOf_NoDeps(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	// Add a node with no dependencies
	testNode := addNodeToGraph(t, g, "step.test.0", "print")

	// Query dependencies
	deps, err := g.DependenciesOf(ctx, testNode.ID)
	require.NoError(t, err)
	assert.Empty(t, deps)
}

func TestDependenciesOf_MultipleDeps(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	// Create a dependency chain: third depends on both first and second
	node1 := addNodeToGraph(t, g, "step.first.0", "print")
	node2 := addNodeToGraph(t, g, "step.second.0", "http")
	node3 := addNodeToGraph(t, g, "step.third.0", "file")

	addDependency(t, g, node1.ID.String(), node3.ID.String())
	addDependency(t, g, node2.ID.String(), node3.ID.String())

	// Query dependencies of node3
	deps, err := g.DependenciesOf(ctx, node3.ID)
	require.NoError(t, err)
	require.Len(t, deps, 2)

	// Verify dependencies (order not guaranteed)
	depMap := make(map[string]*node.Node)
	for _, d := range deps {
		depMap[d.ID.String()] = d
	}

	assert.Equal(t, node1, depMap[node1.ID.String()])
	assert.Equal(t, node2, depMap[node2.ID.String()])
}

func TestDependenciesOf_ReturnsFullNodes(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	// Create nodes
	node1 := addNodeToGraph(t, g, "step.first.0", "print")
	node2 := addNodeToGraph(t, g, "step.second.0", "http")

	addDependency(t, g, node1.ID.String(), node2.ID.String())

	// Query dependencies
	deps, err := g.DependenciesOf(ctx, node2.ID)
	require.NoError(t, err)
	require.Len(t, deps, 1)

	// Verify we get full node objects, not just IDs
	assert.Equal(t, node1.ID, deps[0].ID)
	assert.Equal(t, node1.Type, deps[0].Type)
	assert.Equal(t, node1.RawConfig, deps[0].RawConfig)
}

func TestNodeStatus_Default(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	// Add a node but don't set its status
	testNode := addNodeToGraph(t, g, "step.test.0", "print")

	// Status should default to Pending
	status, ok := g.NodeStatus(ctx, testNode.ID)
	assert.True(t, ok)
	assert.Equal(t, node.StatusPending, status)
}

func TestNodeStatus_AfterUpdate(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	testNode := addNodeToGraph(t, g, "step.test.0", "print")

	// Mark as running
	err := g.MarkRunning(ctx, testNode.ID)
	require.NoError(t, err)

	// Verify status updated
	status, ok := g.NodeStatus(ctx, testNode.ID)
	assert.True(t, ok)
	assert.Equal(t, node.StatusRunning, status)
}

func TestMarkRunning(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	testNode := addNodeToGraph(t, g, "step.test.0", "print")

	// Mark as running
	err := g.MarkRunning(ctx, testNode.ID)
	require.NoError(t, err)

	// Verify status
	status, ok := g.NodeStatus(ctx, testNode.ID)
	require.True(t, ok)
	assert.Equal(t, node.StatusRunning, status)
}

func TestMarkCompleted_WithOutput(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	testNode := addNodeToGraph(t, g, "step.test.0", "http")

	// Mark as completed with output
	expectedOutput := map[string]any{
		"status_code": 200,
		"body":        "success",
	}
	err := g.MarkCompleted(ctx, testNode.ID, expectedOutput)
	require.NoError(t, err)

	// Verify status
	status, ok := g.NodeStatus(ctx, testNode.ID)
	require.True(t, ok)
	assert.Equal(t, node.StatusCompleted, status)

	// Verify output was stored (access via underlying store)
	manager := g.(*Manager)
	output, err := manager.nodeState.GetOutput(ctx, testNode.ID)
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestMarkFailed_WithError(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	testNode := addNodeToGraph(t, g, "step.test.0", "http")

	// Mark as failed with error
	expectedErr := errors.New("connection timeout")
	err := g.MarkFailed(ctx, testNode.ID, expectedErr)
	require.NoError(t, err)

	// Verify status
	status, ok := g.NodeStatus(ctx, testNode.ID)
	require.True(t, ok)
	assert.Equal(t, node.StatusFailed, status)

	// Verify error was stored (access via underlying store)
	manager := g.(*Manager)
	storedErr, err := manager.nodeState.GetError(ctx, testNode.ID)
	require.NoError(t, err)
	assert.Equal(t, expectedErr, storedErr)
}

func TestMarkSkipped(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	testNode := addNodeToGraph(t, g, "step.test.0", "print")

	// Mark as skipped
	err := g.MarkSkipped(ctx, testNode.ID)
	require.NoError(t, err)

	// Verify status
	status, ok := g.NodeStatus(ctx, testNode.ID)
	require.True(t, ok)
	assert.Equal(t, node.StatusSkipped, status)
}

func TestGraph_StateTransitions(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()

	testNode := addNodeToGraph(t, g, "step.lifecycle.0", "http")

	// Start: Pending
	status, ok := g.NodeStatus(ctx, testNode.ID)
	require.True(t, ok)
	assert.Equal(t, node.StatusPending, status)

	// Transition: Pending → Running
	err := g.MarkRunning(ctx, testNode.ID)
	require.NoError(t, err)
	status, ok = g.NodeStatus(ctx, testNode.ID)
	require.True(t, ok)
	assert.Equal(t, node.StatusRunning, status)

	// Transition: Running → Completed
	output := map[string]any{"result": "success"}
	err = g.MarkCompleted(ctx, testNode.ID, output)
	require.NoError(t, err)
	status, ok = g.NodeStatus(ctx, testNode.ID)
	require.True(t, ok)
	assert.Equal(t, node.StatusCompleted, status)
}

func TestGraph_ConcurrentAccess(t *testing.T) {
	g := createTestGraph()
	ctx := context.Background()
	numGoroutines := 100
	var wg sync.WaitGroup

	// Phase 1: Concurrent node addition and status updates
	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(i int) {
			defer wg.Done()

			// Add node
			nodeID := fmt.Sprintf("step.concurrent.%d", i)
			addr, err := nodeid.Parse(nodeID)
			if err != nil {
				t.Errorf("failed to parse nodeid: %v", err)
				return
			}

			n := &node.Node{
				ID:        *addr,
				Type:      "test",
				RawConfig: map[string]any{"index": i},
			}

			// Add to topology
			manager := g.(*Manager)
			manager.topology.AddNode(ctx, n)

			// Update state: Pending → Running → Completed
			g.MarkRunning(ctx, *addr)
			g.MarkCompleted(ctx, *addr, map[string]any{"index": i})
		}(i)
	}

	wg.Wait() // Wait for all writes to complete

	// Phase 2: Concurrent reads and verification
	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(i int) {
			defer wg.Done()

			nodeID := fmt.Sprintf("step.concurrent.%d", i)
			addr, err := nodeid.Parse(nodeID)
			if err != nil {
				t.Errorf("failed to parse nodeid: %v", err)
				return
			}

			// Verify node exists
			n, ok := g.Node(ctx, *addr)
			assert.True(t, ok, "node %d should exist", i)
			assert.NotNil(t, n, "node %d should not be nil", i)

			// Verify status
			status, ok := g.NodeStatus(ctx, *addr)
			assert.True(t, ok, "status for node %d should exist", i)
			assert.Equal(t, node.StatusCompleted, status, "node %d should be completed", i)

			// Verify output
			manager := g.(*Manager)
			output, err := manager.nodeState.GetOutput(ctx, *addr)
			assert.NoError(t, err)
			assert.Equal(t, map[string]any{"index": i}, output, "output for node %d mismatch", i)
		}(i)
	}

	wg.Wait() // Wait for all reads to complete

	// Verify all nodes are present
	allNodes := g.AllNodes(ctx)
	assert.Len(t, allNodes, numGoroutines, "should have all concurrent nodes")
}
