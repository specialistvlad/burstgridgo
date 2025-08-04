package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()

	// --- Test Case Definitions ---
	testCases := []struct {
		name        string
		args        []string
		setup       func(t *testing.T) (args []string, cleanup func())
		checkErr    func(t *testing.T, err error)
		checkOutput func(t *testing.T, output string)
	}{
		{
			name: "Success Path - Should Exit Cleanly",
			args: []string{"-h"},
			checkErr: func(t *testing.T, err error) {
				require.NoError(t, err, "run() should not return an error for a clean exit")
			},
			checkOutput: func(t *testing.T, output string) {
				require.Contains(t, output, "Usage:")
			},
		},
		{
			name: "Error Path - Argument Parsing Fails",
			args: []string{"--invalid-flag"},
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err, "run() should return an error for a parsing failure")
				require.Contains(t, err.Error(), "flag provided but not defined: -invalid-flag")
			},
		},
		{
			name: "Error Path - Recovers from Panic",
			setup: func(t *testing.T) (args []string, cleanup func()) {
				// Create a syntactically invalid HCL file that will cause a panic
				invalidHCL := `step "print" "A" { arguments {`
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
			t.Parallel()

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
			err := run(out, args)

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
