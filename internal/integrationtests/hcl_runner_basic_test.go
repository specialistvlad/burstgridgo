package integration_tests

import (
	"strings"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/model"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestRunnerParsing_Success(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		hcl      string
		validate func(t *testing.T, r *model.Runner)
	}{
		{
			name: "full definition with description and lifecycle",
			hcl: `
			runner "test" {
				description = "A test runner with a description."
				lifecycle {
					on_run = "OnRunTest"
				}
			}
			`,
			validate: func(t *testing.T, r *model.Runner) {
				require.Equal(t, "A test runner with a description.", r.Description)
				require.Equal(t, "OnRunTest", r.Lifecycle.OnRun)

				require.NotNil(t, r.FSInformation)
				require.NotEmpty(t, r.FSInformation.FilePath)
				// The test harness creates a predictable structure we can check against.
				require.True(t, strings.HasSuffix(r.FSInformation.FilePath, "/modules/test/manifest.hcl"), "FilePath mismatch")
			},
		},
		{
			name: "minimal definition with only lifecycle",
			hcl: `
			runner "test" {
				lifecycle {
					on_run = "OnRunTest"
				}
			}
			`,
			validate: func(t *testing.T, r *model.Runner) {
				require.Empty(t, r.Description, "Description should be the zero value")
				require.Equal(t, "OnRunTest", r.Lifecycle.OnRun)
			},
		},
		{
			name: "definition with only description",
			hcl: `
			runner "test" {
				description = "No lifecycle here."
				// lifecycle block is intentionally omitted
			}
			`,
			validate: func(t *testing.T, r *model.Runner) {
				require.Equal(t, "No lifecycle here.", r.Description)
				require.Empty(t, r.Lifecycle.OnRun, "Lifecycle.OnRun should be empty when block is omitted")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runner, err := testutil.RunRunnerParsingTest(t, tc.hcl)
			require.NoError(t, err)
			tc.validate(t, runner)
		})
	}
}

func TestRunnerParsing_Failure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		hcl         string
		errContains string
	}{
		{
			name: "duplicate lifecycle block",
			hcl: `
			runner "test" {
				lifecycle {}
				lifecycle {}
			}
			`,
			errContains: `Duplicate "lifecycle" block`,
		},
		{
			name: "unknown attribute",
			hcl: `
			runner "test" {
				author = "test"
			}
			`,
			errContains: "Unsupported argument",
		},
		{
			name: "unknown nested block",
			hcl: `
			runner "test" {
				metadata {}
			}
			`,
			errContains: "Unsupported block type",
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
}
