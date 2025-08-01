package integration_tests

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

// TestHclFeatures_OptionalArgumentDefault_FromFile tests that an optional argument
// with a default value defined in a manifest is applied correctly when the
// argument is omitted in the step definition.
func TestHclFeatures_OptionalArgumentDefault_FromFile(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	manifestHCL := `
		runner "defaulter" {
		  lifecycle {
		    on_run = "OnRunDefaulter"
		  }
		  input "required" {
		    type = string
		  }
		  input "mode" {
		    type    = string
		    default = "standard"
		  }
		  input "metadata" {
		    type    = map(string)
		    default = {
		      "source" = "test-suite"
		    }
		  }
		}
	`
	gridHCL := `
		step "defaulter" "A" {
			arguments {
				required = "must-be-present"
			}
		}
	`
	files := map[string]string{
		"modules/defaulter/manifest.hcl": manifestHCL,
		"main.hcl":                       gridHCL,
	}

	// Define the mock module and its data structures inside the test.
	type defaulterInput struct {
		Mode     string            `bggo:"mode"`
		Required string            `bggo:"required"`
		Metadata map[string]string `bggo:"metadata"`
	}

	var capturedInput defaulterInput
	var mu sync.Mutex

	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunDefaulter",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(defaulterInput) },
			InputType: reflect.TypeOf(defaulterInput{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn: func(_ context.Context, _ any, inputRaw any) (any, error) {
				mu.Lock()
				capturedInput = *inputRaw.(*defaulterInput)
				mu.Unlock()
				return nil, nil
			},
		},
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "app.Run() returned an unexpected error")

	expectedInput := defaulterInput{
		Mode:     "standard",
		Required: "must-be-present",
		Metadata: map[string]string{
			"source": "test-suite",
		},
	}

	mu.Lock()
	defer mu.Unlock()
	if diff := cmp.Diff(expectedInput, capturedInput); diff != "" {
		t.Errorf("Captured input mismatch (-want +got):\n%s", diff)
	}
}
