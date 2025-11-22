package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	// NOTE: Cannot use t.Parallel() at the top level because some subtests call os.Chdir()
	// which modifies global process state and causes race conditions.

	// --- Test Case Definitions ---
	testCases := []struct {
		name        string
		skip        *string
		args        []string
		setup       func(t *testing.T) (args []string, cleanup func())
		checkErr    func(t *testing.T, err error)
		checkOutput func(t *testing.T, output string)
	}{
		{
			name: "Success Path - Should Exit Cleanly",
			skip: nil,
			args: []string{"-h"},
			checkErr: func(t *testing.T, err error) {
				require.NoError(t, err, "run() should not return an error for a clean exit")
			},
			checkOutput: func(t *testing.T, output string) {
				require.Contains(t, output, "Usage:")
			},
		},
		{
			name: "Success Path - Full Run",
			skip: nil,
			setup: func(t *testing.T) (args []string, cleanup func()) {
				// Create a minimal, valid HCL file for a successful run.
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "main.hcl")
				err := os.WriteFile(filePath, []byte(""), 0600)
				require.NoError(t, err)

				err = os.Mkdir(filepath.Join(tempDir, "modules"), 0755)
				require.NoError(t, err)

				// The test will run with the path to this file as its argument.
				// The app will look for the 'modules' dir relative to this path's parent.
				return []string{filePath}, nil
			},
			checkErr: func(t *testing.T, err error) {
				require.NoError(t, err, "run() should not return an error on a successful run")
			},
			checkOutput: func(t *testing.T, output string) {
				// Check for a log message that indicates a successful completion.
				require.Contains(t, output, "Execution finished.")
			},
		},
		{
			name: "Error Path - Argument Parsing Fails",
			skip: nil,
			args: []string{"--invalid-flag"},
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err, "run() should return an error for a parsing failure")
				require.Contains(t, err.Error(), "flag provided but not defined: -invalid-flag")
			},
		},
		{
			name: "Error Path - Recovers from Panic",
			skip: func() *string {
				s := "TODO A bit hard to simulate it. Skipping for now."
				return &s
			}(),
			setup: func(t *testing.T) (args []string, cleanup func()) {
				invalidHCL := ``
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "main.hcl")
				err := os.WriteFile(filePath, []byte(invalidHCL), 0600)
				require.NoError(t, err)

				// The test will run with the path to this file as its argument
				return []string{filePath}, func() { os.RemoveAll(tempDir) }
			},
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err, "run() should have returned an error after recovering from a panic")
				errStr := err.Error()
				require.Contains(t, errStr, "application startup panicked")
				require.Contains(t, errStr, "failed to parse")
			},
		},
	}

	// --- Test Runner ---
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() here because tests call os.Chdir()
			// which modifies global process state
			if tc.skip != nil {
				t.Skip(*tc.skip)
			}

			// --- Arrange ---
			out := &bytes.Buffer{}
			args := tc.args

			if tc.setup != nil {
				var cleanup func()
				args, cleanup = tc.setup(t)
				if cleanup != nil {
					t.Cleanup(cleanup)
				}
			}

			// --- Act ---
			// We need to change the working directory for this test to be isolated.
			// The application uses default paths like "modules" relative to the CWD.
			originalWD, err := os.Getwd()
			require.NoError(t, err)

			// Find the temporary directory from the file path arg.
			if len(args) > 0 {
				tempDir := filepath.Dir(args[0])
				err = os.Chdir(tempDir)
				require.NoError(t, err)
				t.Cleanup(func() { os.Chdir(originalWD) })
			}

			err = run(out, args)

			// --- Assert ---
			if tc.checkErr != nil {
				tc.checkErr(t, err)
			}
			if tc.checkOutput != nil {
				tc.checkOutput(t, out.String())
			}
		})
	}
}
