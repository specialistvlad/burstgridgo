package integration_tests

import (
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestHCLLanguage_ParsesMultipleLocalsBlocks(t *testing.T) {
	gridHCL := `
		locals {
			hostname = "example.com"
		}

		locals {
			port = 8080
		}

		step "print" "a" {
			arguments {
				// Note: reference resolution is a runtime task.
				// This test only confirms the parser accepts the blocks.
				endpoint = "http://${local.hostname}:${local.port}"
			}
		}
	`
	result, _ := testutil.RunHCLGridTest(t, gridHCL)

	// Assert that the parser accepts multiple `locals` blocks
	// without returning an "Unsupported block type" error.
	require.NoError(t, result.Err)
}

func TestHCLLanguage_ParsesVariableBlock(t *testing.T) {
	gridHCL := `
		variable "name" {
		  type    = string
		  default = "World"
		}
		step "print" "a" {
		  arguments {
		    message = "Hello, ${var.name}"
		  }
		}
	`
	result, _ := testutil.RunHCLGridTest(t, gridHCL)

	// For now, we just assert that the parser accepts the `variable` block
	// without returning an "Unsupported block type" error.
	require.NoError(t, result.Err)
}
