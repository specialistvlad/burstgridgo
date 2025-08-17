package inmemorystore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetAndGetStatus(t *testing.T) {
	s := New()
	ctx := context.Background()
	addr, err := nodeid.Parse("step.test.0")
	require.NoError(t, err)

	// Get status of a node that doesn't exist yet
	status, err := s.GetStatus(ctx, *addr)
	require.NoError(t, err)
	assert.Equal(t, node.StatusPending, status)

	// Set status
	err = s.SetStatus(ctx, *addr, node.StatusRunning)
	require.NoError(t, err)

	// Get status again
	status, err = s.GetStatus(ctx, *addr)
	require.NoError(t, err)
	assert.Equal(t, node.StatusRunning, status)
}

func TestSetAndGetOutput(t *testing.T) {
	s := New()
	ctx := context.Background()
	addr, err := nodeid.Parse("step.test.0")
	require.NoError(t, err)

	// Get output for a node that doesn't exist yet should be nil
	output, err := s.GetOutput(ctx, *addr)
	require.NoError(t, err)
	assert.Nil(t, output)

	// Set output
	expectedOutput := map[string]any{"status_code": 200}
	err = s.SetOutput(ctx, *addr, expectedOutput)
	require.NoError(t, err)

	// Get output
	retrievedOutput, err := s.GetOutput(ctx, *addr)
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, retrievedOutput)
}

func TestSetAndGetError(t *testing.T) {
	s := New()
	ctx := context.Background()
	addr, err := nodeid.Parse("step.test.0")
	require.NoError(t, err)

	// Get error for a node that doesn't exist yet should be nil
	retrievedErr, err := s.GetError(ctx, *addr)
	require.NoError(t, err)
	assert.Nil(t, retrievedErr)

	// Set error
	expectedErr := errors.New("a test error occurred")
	err = s.SetError(ctx, *addr, expectedErr)
	require.NoError(t, err)

	// Get error
	retrievedErr, err = s.GetError(ctx, *addr)
	require.NoError(t, err)
	require.Error(t, retrievedErr)
	assert.Equal(t, expectedErr, retrievedErr)
}

// TestStore_ConcurrentAccess verifies that the store can be safely accessed by
// multiple goroutines simultaneously without data races or lost writes.
func TestStore_ConcurrentAccess(t *testing.T) {
	s := New()
	ctx := context.Background()
	numGoroutines := 100
	var wg sync.WaitGroup

	wg.Add(numGoroutines)

	// Phase 1: Concurrent Writes
	// All goroutines will write to a unique node ID.
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			addr, err := nodeid.Parse(fmt.Sprintf("step.concurrent.%d", i))
			if err != nil {
				t.Errorf("failed to parse nodeid: %v", err)
				return
			}
			// Set all three state types for each node
			s.SetStatus(ctx, *addr, node.StatusCompleted)
			s.SetOutput(ctx, *addr, i) // Use the loop index as a unique output
			s.SetError(ctx, *addr, fmt.Errorf("error for node %d", i))
		}(i)
	}

	wg.Wait() // Wait for all writes to complete

	// Phase 2: Concurrent Reads / Verification
	// Verify that all data written in the first phase is present and correct.
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			addr, err := nodeid.Parse(fmt.Sprintf("step.concurrent.%d", i))
			if err != nil {
				t.Errorf("failed to parse nodeid: %v", err)
				return
			}

			status, err := s.GetStatus(ctx, *addr)
			assert.NoError(t, err)
			assert.Equal(t, node.StatusCompleted, status, "mismatched status for node %d", i)

			output, err := s.GetOutput(ctx, *addr)
			assert.NoError(t, err)
			assert.Equal(t, i, output, "mismatched output for node %d", i)

			nodeErr, err := s.GetError(ctx, *addr)
			assert.NoError(t, err)
			assert.EqualError(t, nodeErr, fmt.Sprintf("error for node %d", i), "mismatched error for node %d", i)
		}(i)
	}

	wg.Wait() // Wait for all reads to complete
}
