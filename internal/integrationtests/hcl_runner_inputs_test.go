package integration_tests

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestRunnerParsing_Input(t *testing.T) {
	t.Parallel()

	t.Run("Success: Parses full input block definitions", func(t *testing.T) {
		t.Parallel()
		hcl := `
		runner "test" {
			lifecycle { on_run = "OnRun" }

			input "url" {
			type        = string
			description = "The target URL."
			}

			input "retries" {
			type        = number
			description = "Number of retries."
			default     = 3
			}

			input "enabled" {
			type    = bool
			default = true
			}
		}`

		runner, err := testutil.RunRunnerParsingTest(t, hcl)
		require.NoError(t, err)
		require.NotNil(t, runner, "Parsed runner should not be nil")
		require.NotNil(t, runner.Inputs)
		require.Len(t, runner.Inputs, 3, "Should have parsed exactly 3 inputs")

		// --- Assert on 'url' (required string) ---
		urlInput, ok := runner.Inputs["url"]
		require.True(t, ok, "Input 'url' should be present")
		require.Equal(t, "url", urlInput.Name)
		require.Equal(t, "The target URL.", urlInput.Description)
		require.True(t, urlInput.Type.Equals(cty.String), "Type should be cty.String")
		require.Nil(t, urlInput.Default, "Default should be nil for a required input")

		// --- Assert on 'retries' (optional number) ---
		retriesInput, ok := runner.Inputs["retries"]
		require.True(t, ok, "Input 'retries' should be present")
		require.Equal(t, "retries", retriesInput.Name)
		require.Equal(t, "Number of retries.", retriesInput.Description)
		require.True(t, retriesInput.Type.Equals(cty.Number), "Type should be cty.Number")
		require.NotNil(t, retriesInput.Default, "Default should be present")

		expectedDefaultRetries := cty.NumberIntVal(3)
		if diff := cmp.Diff(expectedDefaultRetries, *retriesInput.Default, cmpopts.IgnoreUnexported(cty.Value{})); diff != "" {
			t.Errorf("Default value for 'retries' mismatch (-want +got):\n%s", diff)
		}

		// --- Assert on 'enabled' (optional bool) ---
		enabledInput, ok := runner.Inputs["enabled"]
		require.True(t, ok, "Input 'enabled' should be present")
		require.True(t, enabledInput.Type.Equals(cty.Bool), "Type should be cty.Bool")
		require.NotNil(t, enabledInput.Default, "Default should be present")

		expectedDefaultEnabled := cty.True
		if diff := cmp.Diff(expectedDefaultEnabled, *enabledInput.Default, cmpopts.IgnoreUnexported(cty.Value{})); diff != "" {
			t.Errorf("Default value for 'enabled' mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("Failure: Invalid input block definitions", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name        string
			hcl         string
			errContains string
		}{
			{
				name: "missing type attribute",
				hcl: `
			runner "test" {
			input "a" {
			description = "..."
			}
			}`,
				errContains: "Missing 'type' attribute",
			},
			{
				name: "invalid type keyword",
				hcl: `
			runner "test" {
			input "a" {
			type = integer
			}
			}`,
				errContains: "not a valid type",
			},
			{
				name: "default value type mismatch",
				hcl: `
			runner "test" {
			input "a" {
			type    = string
			default = 123
			}
			}`,
				errContains: "Invalid default value type",
			},
			{
				name: "duplicate input name",
				hcl: `
			runner "test" {
			input "a" {
			type = string
			}
			input "a" {
			type = number
			}
			}`,
				errContains: "Duplicate input definition",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := testutil.RunRunnerParsingTest(t, tc.hcl)
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			})
		}
	})

	t.Run("Panic on unimplemented complex type", func(t *testing.T) {
		t.Parallel()
		hcl := `
			runner "test" {
			input "a" {
			type = list
			}
			}`

		// --- FIX: Changed from PanicsWithError to PanicsWithValue to match a string panic ---
		require.PanicsWithValue(t, "PANIC: The complex type 'list' is not yet implemented.", func() {
			// We only care about the panic, not the return values.
			_, _ = testutil.RunRunnerParsingTest(t, hcl)
		})
	})
}
