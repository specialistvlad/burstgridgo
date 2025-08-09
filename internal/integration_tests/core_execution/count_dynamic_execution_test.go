package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/testutil"
	"github.com/zclconf/go-cty/cty"
)

func TestCoreExecution_Count_Dynamic(t *testing.T) {
	t.Parallel()

	// Act
	result, consumedData := testutil.RunDynamicCountTest(t, cty.NumberIntVal(3))

	// Assert
	require.NoError(t, result.Err)

	// Verify all 3 instances of B ran
	testutil.AssertStepInstanceRan(t, result, "print_indexed", "B", 0)
	testutil.AssertStepInstanceRan(t, result, "print_indexed", "B", 1)
	testutil.AssertStepInstanceRan(t, result, "print_indexed", "B", 2)

	// Verify C ran
	testutil.AssertStepInstanceRan(t, result, "consumer", "C", 0)

	// Verify C received a list of 3 items
	require.True(t, consumedData.IsKnown() && !consumedData.IsNull(), "consumed data should be known")
	require.True(t, consumedData.Type().IsListType(), "consumed data should be a list")
	require.Equal(t, 3, consumedData.LengthInt(), "consumed data list should have 3 elements")
}

func TestCoreExecution_Count_Dynamic_Zero(t *testing.T) {
	t.Parallel()

	// Act
	result, consumedData := testutil.RunDynamicCountTest(t, cty.NumberIntVal(0))

	// Assert
	require.NoError(t, result.Err)

	// Verify B did not run, but C did (and received an empty list)
	require.NotContains(t, result.LogOutput, `step=step.print_indexed.B[0]`)
	testutil.AssertStepInstanceRan(t, result, "consumer", "C", 0)

	// Verify C received an empty list
	require.True(t, consumedData.IsKnown() && !consumedData.IsNull())
	require.True(t, consumedData.Type().IsTupleType(), "consumed data should be an empty tuple (which acts as an empty list)")
	require.Equal(t, 0, consumedData.LengthInt())
}
