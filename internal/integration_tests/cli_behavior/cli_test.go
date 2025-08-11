package integration_tests

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/specialistvlad/burstgridgo/internal/app"
	"github.com/specialistvlad/burstgridgo/internal/cli"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		args           []string
		expectExit     bool
		expectErr      bool
		expectedConfig *app.AppConfig
		checkOutput    func(t *testing.T, output string)
	}{
		{
			name: "Happy Path with all flags",
			args: []string{
				"-grid", "/test/grid",
				"--modules-path=/test/modules",
				"--log-level=debug",
				"--log-format=text",
				"--workers=50",
				"--healthcheck-port=8080",
			},
			expectedConfig: &app.AppConfig{
				GridPath:        "/test/grid",
				ModulesPath:     "/test/modules",
				LogLevel:        "debug",
				LogFormat:       "text",
				WorkerCount:     50,
				HealthcheckPort: 8080,
			},
		},
		{
			name:       "Shorthand flag and defaults",
			args:       []string{"-g", "/short/path"},
			expectExit: false,
			expectErr:  false,
			expectedConfig: &app.AppConfig{
				GridPath:        "/short/path",
				ModulesPath:     "modules",
				LogLevel:        "info",
				LogFormat:       "json",
				WorkerCount:     10,
				HealthcheckPort: 0,
			},
		},
		{
			name:       "Positional argument for path",
			args:       []string{"/positional/path"},
			expectExit: false,
			expectErr:  false,
			expectedConfig: &app.AppConfig{
				GridPath:        "/positional/path",
				ModulesPath:     "modules",
				LogLevel:        "info",
				LogFormat:       "json",
				WorkerCount:     10,
				HealthcheckPort: 0,
			},
		},
		{
			name:       "Help flag triggers clean exit",
			args:       []string{"-h"},
			expectExit: true,
			expectErr:  false,
			checkOutput: func(t *testing.T, output string) {
				require.True(t, strings.Contains(output, "Usage:"), "Expected help text to be printed")
			},
		},
		{
			name:       "No path triggers clean exit with usage",
			args:       []string{},
			expectExit: true,
			expectErr:  false,
			checkOutput: func(t *testing.T, output string) {
				require.True(t, strings.Contains(output, "Usage:"), "Expected help text to be printed")
			},
		},
		{
			name:      "Invalid log level returns an error",
			args:      []string{"--log-level=foo", "/path"},
			expectErr: true,
		},
		{
			name:      "Invalid log format returns an error",
			args:      []string{"--log-format=yaml", "/path"},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// --- Arrange ---
			out := &bytes.Buffer{}

			// --- Act ---
			appConfig, shouldExit, err := cli.Parse(tc.args, out)

			// --- Assert ---
			if tc.expectErr {
				require.Error(t, err)
				_, isExitError := err.(*cli.ExitError)
				require.True(t, isExitError, "Expected error to be of type ExitError")
				return // End test here if an error is expected
			}
			require.NoError(t, err)

			require.Equal(t, tc.expectExit, shouldExit)

			if tc.expectedConfig != nil {
				if diff := cmp.Diff(tc.expectedConfig, appConfig); diff != "" {
					t.Errorf("AppConfig mismatch (-want +got):\n%s", diff)
				}
			}

			if tc.checkOutput != nil {
				tc.checkOutput(t, out.String())
			}
		})
	}
}
