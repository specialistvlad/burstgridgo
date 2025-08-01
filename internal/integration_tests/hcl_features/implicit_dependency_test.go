package integration_tests

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

type sourceOutput struct {
	Message string `cty:"message"`
	ID      int    `cty:"id"`
}

type mockSourceSpyModule struct {
	sourceOutput  sourceOutput
	capturedInput sourceOutput
}

func (m *mockSourceSpyModule) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunSource", &registry.RegisteredRunner{
		NewInput:  func() any { return new(struct{}) },
		InputType: reflect.TypeOf(struct{}{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn:        func(context.Context, any, any) (*sourceOutput, error) { return &m.sourceOutput, nil },
	})

	type spyInput struct {
		Input sourceOutput `bggo:"input"`
	}
	r.RegisterRunner("OnRunSpy", &registry.RegisteredRunner{
		NewInput:  func() any { return new(spyInput) },
		InputType: reflect.TypeOf(spyInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (any, error) {
			m.capturedInput = inputRaw.(*spyInput).Input
			return nil, nil
		},
	})
}

func TestHclFeatures_ImplicitDependency(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	sourceManifestHCL := `
		runner "source" {
			lifecycle {
				on_run = "OnRunSource"
			}
			output "data" {
				type = any
			}
		}
	`
	spyManifestHCL := `
		runner "spy" {
			lifecycle {
				on_run = "OnRunSpy"
			}
			input "input" {
				type = any
			}
		}
	`
	gridHCL := `
		step "source" "A" {
			arguments {}
		}
		step "spy" "B" {
			arguments {
				input = step.source.A.output
			}
		}
	`
	files := map[string]string{
		"modules/source/manifest.hcl": sourceManifestHCL,
		"modules/spy/manifest.hcl":    spyManifestHCL,
		"main.hcl":                    gridHCL,
	}

	expectedData := sourceOutput{
		Message: "hello from source",
		ID:      123,
	}
	mockModule := &mockSourceSpyModule{sourceOutput: expectedData}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err)

	if diff := cmp.Diff(expectedData, mockModule.capturedInput); diff != "" {
		t.Errorf("Captured input mismatch (-want +got):\n%s", diff)
	}
}
