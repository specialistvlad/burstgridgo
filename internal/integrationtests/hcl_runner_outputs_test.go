package integration_tests

import (
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestRunnerParsing_Output(t *testing.T) {
	t.Parallel()

	t.Run("Success: Parses full output block definitions", func(t *testing.T) {
		t.Parallel()
		hcl := `
		runner "test" {
			output "id" {
				type        = string
				description = "The unique ID of the created resource."
			}

			output "success" {
				type = bool
			}
		}`

		runner, err := testutil.RunRunnerParsingTest(t, hcl)
		require.NoError(t, err)
		require.NotNil(t, runner, "Parsed runner should not be nil")
		require.NotNil(t, runner.Outputs)
		require.Len(t, runner.Outputs, 2, "Should have parsed exactly 2 outputs")

		// --- Assert on 'id' (string with description) ---
		idOutput, ok := runner.Outputs["id"]
		require.True(t, ok, "Output 'id' should be present")
		require.Equal(t, "id", idOutput.Name)
		require.Equal(t, "The unique ID of the created resource.", idOutput.Description)
		require.True(t, idOutput.Type.Equals(cty.String), "Type should be cty.String")

		// --- Assert on 'success' (bool without description) ---
		successOutput, ok := runner.Outputs["success"]
		require.True(t, ok, "Output 'success' should be present")
		require.Equal(t, "success", successOutput.Name)
		require.Empty(t, successOutput.Description, "Description should be empty")
		require.True(t, successOutput.Type.Equals(cty.Bool), "Type should be cty.Bool")
	})

	t.Run("Failure: Invalid output block definitions", func(t *testing.T) {
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
					output "a" {
						description = "..."
					}
				}`,
				errContains: `Missing required argument`,
			},
			{
				name: "invalid type keyword",
				hcl: `
				runner "test" {
					output "a" {
						type = foobar
					}
				}`,
				errContains: "not a valid type",
			},
			{
				name: "duplicate output name",
				hcl: `
				runner "test" {
					output "a" { type = string }
					output "a" { type = number }
				}`,
				errContains: "Duplicate output definition",
			},
			{
				name: "unsupported attribute",
				hcl: `
				runner "test" {
					output "a" {
						type    = string
						default = "abc"
					}
				}`,
				errContains: `Unsupported argument`,
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
}
