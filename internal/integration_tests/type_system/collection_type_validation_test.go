package type_system_test

import (
	"context"
	"reflect"
	"regexp"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

func TestStartupValidation_CollectionTypeMismatch_Fails(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	manifestHCL := `
		runner "collection_runner" {
			lifecycle { on_run = "OnRunMismatch" }
			input "urls" {
				type = list(string) // Manifest wants a list of strings
			}
		}
	`
	// The mock module registers a Go struct with []int, which is a mismatch.
	type mismatchInput struct {
		Urls []int `bggo:"urls"`
	}
	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunMismatch",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(mismatchInput) },
			InputType: reflect.TypeOf(mismatchInput{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn:        func(context.Context, any, any) (any, error) { return nil, nil },
		},
	}

	files := map[string]string{
		"modules/collections/manifest.hcl": manifestHCL,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.Error(t, result.Err, "app.New() should have panicked, but it did not")
	errStr := result.Err.Error()

	expectedErrPattern := `(?s).*type mismatch.*Manifest requires 'list of string'.*Go struct field 'Urls' provides compatible type 'list of number'`
	require.Regexp(t, regexp.MustCompile(expectedErrPattern), errStr, "The error message did not match the expected pattern")
}

func TestCoreExecution_ListType_Success(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	manifestHCL := `
		runner "collection_runner" {
			lifecycle { on_run = "OnRunList" }
			input "urls" { type = list(string) }
		}
	`
	gridHCL := `
		step "collection_runner" "A" {
			arguments {
				urls = ["a.com", "b.com", "c.com"]
			}
		}
	`
	files := map[string]string{
		"modules/collections/manifest.hcl": manifestHCL,
		"grid.hcl":                         gridHCL,
	}

	type listInput struct {
		Urls []string `bggo:"urls"`
	}
	var capturedInput listInput
	var mu sync.Mutex
	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunList",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(listInput) },
			InputType: reflect.TypeOf(listInput{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn: func(_ context.Context, _ any, input any) (any, error) {
				mu.Lock()
				capturedInput = *input.(*listInput)
				mu.Unlock()
				return nil, nil
			},
		},
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "Run failed unexpectedly")

	expected := listInput{
		Urls: []string{"a.com", "b.com", "c.com"},
	}
	mu.Lock()
	defer mu.Unlock()
	if diff := cmp.Diff(expected, capturedInput); diff != "" {
		t.Errorf("Captured list mismatch (-want +got):\n%s", diff)
	}
}

func TestCoreExecution_MapType_Success(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	manifestHCL := `
		runner "collection_runner" {
			lifecycle { on_run = "OnRunMap" }
			input "headers" { type = map(string) }
		}
	`
	gridHCL := `
		step "collection_runner" "A" {
			arguments {
				headers = {
					"Content-Type" = "application/json"
					"X-Request-ID" = "abc-123"
				}
			}
		}
	`
	files := map[string]string{
		"modules/collections/manifest.hcl": manifestHCL,
		"grid.hcl":                         gridHCL,
	}

	type mapInput struct {
		Headers map[string]string `bggo:"headers"`
	}
	var capturedInput mapInput
	var mu sync.Mutex
	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunMap",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(mapInput) },
			InputType: reflect.TypeOf(mapInput{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn: func(_ context.Context, _ any, input any) (any, error) {
				mu.Lock()
				capturedInput = *input.(*mapInput)
				mu.Unlock()
				return nil, nil
			},
		},
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "Run failed unexpectedly")

	expected := mapInput{
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Request-ID": "abc-123",
		},
	}
	mu.Lock()
	defer mu.Unlock()
	if diff := cmp.Diff(expected, capturedInput); diff != "" {
		t.Errorf("Captured map mismatch (-want +got):\n%s", diff)
	}
}

func TestErrorHandling_CollectionElementTypeMismatch_FailsRun(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	manifestHCL := `
		runner "collection_runner" {
			lifecycle { on_run = "OnRunPorts" }
			input "ports" { type = list(number) }
		}
	`
	gridHCL := `
		step "collection_runner" "A" {
			arguments {
				// This list is invalid because the last element is not a number.
				ports = [80, 443, "not-a-port"]
			}
		}
	`
	files := map[string]string{
		"modules/collections/manifest.hcl": manifestHCL,
		"grid.hcl":                         gridHCL,
	}

	type portsInput struct {
		Ports []int `bggo:"ports"`
	}
	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunPorts",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(portsInput) },
			InputType: reflect.TypeOf(portsInput{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn:        func(context.Context, any, any) (any, error) { return nil, nil },
		},
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.Error(t, result.Err, "Run should have failed")
	errStr := result.Err.Error()

	// Corrected the assertion to check for the actual error message.
	require.Contains(t, errStr, "a number is required")
}
