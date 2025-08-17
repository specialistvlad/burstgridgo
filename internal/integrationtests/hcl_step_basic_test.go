package integration_tests

import (
	"strings"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestGridLoader_ParsesStepArguments(t *testing.T) {
	gridHCL := `
		step "print" "hello" {
			arguments {
				message = "Hello from the grid!"
			}
		}
	`
	result, steps := testutil.RunHCLGridTest(t, gridHCL)

	require.NoError(t, result.Err)
	require.Len(t, steps, 1)

	step := steps[0]
	require.Equal(t, "print", step.RunnerType)
	require.Equal(t, "hello", step.Name)
	require.NotNil(t, step.Arguments)
	require.Contains(t, step.Arguments, "message")

	require.NotNil(t, step.FSInformation)
	require.NotEmpty(t, step.FSInformation.FilePath)
	require.True(t, strings.HasSuffix(step.FSInformation.FilePath, "/grid/main.hcl"), "FilePath mismatch")
}
