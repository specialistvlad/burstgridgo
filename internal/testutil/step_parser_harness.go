package testutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/model"
	"github.com/stretchr/testify/require"
)

// StepTestCase defines a single scenario for testing the parsing of a `step` block.
type StepTestCase struct {
	Name string
	// HCL should contain only the content *inside* the `step "test" "a" { ... }` block.
	// It can be written as a readable, indented multi-line string.
	HCL string
	// ExpectErr should be true if a parsing error is expected.
	ExpectErr bool
	// ErrContains is a substring that must appear in the error message if ExpectErr is true.
	ErrContains string
	// Validate is a function that performs assertions on the successfully parsed *model.Step.
	// It is only called if ExpectErr is false.
	Validate func(t *testing.T, s *model.Step)
}

// unindent removes common leading whitespace from a multi-line string,
// allowing for readable, indented HCL snippets in Go tests.
func unindent(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return ""
	}

	// Remove leading/trailing empty lines that are common with multi-line literals
	if strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		return ""
	}

	// Find the minimum indentation of non-empty lines
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := 0
		for _, r := range line {
			if r == ' ' || r == '\t' {
				indent++
			} else {
				break
			}
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return strings.Join(lines, "\n")
	}

	// Strip the common indentation from each line
	var b strings.Builder
	for i, line := range lines {
		if len(line) >= minIndent {
			b.WriteString(line[minIndent:])
		} else {
			b.WriteString(strings.TrimSpace(line))
		}
		if i < len(lines)-1 {
			b.WriteRune('\n')
		}
	}
	return b.String()
}

// RunStepParsingTests provides a reusable harness for testing the parsing of HCL `step` blocks.
// It iterates through a table of test cases, handling boilerplate and common assertions.
func RunStepParsingTests(t *testing.T, cases []StepTestCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			// The harness now automatically cleans the HCL snippet before parsing.
			cleanHCL := unindent(tc.HCL)
			fullHCL := fmt.Sprintf(`step "test" "a" {
%s
}`, cleanHCL)

			result, steps := RunHCLGridTest(t, fullHCL)

			if tc.ExpectErr {
				require.Error(t, result.Err, "Expected a parsing error, but got none")
				if tc.ErrContains != "" {
					require.Contains(t, result.Err.Error(), tc.ErrContains, "Error message did not contain the expected text")
				}
				return
			}

			require.NoError(t, result.Err, "Expected successful parsing, but got an error")
			require.Len(t, steps, 1, "Expected exactly one step to be parsed")

			if tc.Validate != nil {
				tc.Validate(t, steps[0])
			}
		})
	}
}
