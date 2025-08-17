package integration_tests

import (
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestErrorHandling_Count_Dynamic_InvalidType(t *testing.T) {
	t.Parallel()

	// Act
	result, _ := testutil.RunDynamicCountTest(t, cty.StringVal("three"))

	// Assert
	require.Error(t, result.Err)
	require.Contains(t, result.Err.Error(), "count for step step.print_indexed.B must be a number, but got string")
}

func TestErrorHandling_Count_Dynamic_Negative(t *testing.T) {
	t.Parallel()

	// Act
	result, _ := testutil.RunDynamicCountTest(t, cty.NumberIntVal(-1))

	// Assert
	require.Error(t, result.Err)
	require.Contains(t, result.Err.Error(), "count for step step.print_indexed.B cannot be negative")
}
