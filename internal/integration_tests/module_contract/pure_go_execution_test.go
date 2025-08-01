package module_contract_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

type pureStringManipulatorInput struct {
	Source string `bggo:"Source"`
	Suffix string `bggo:"Suffix"`
}

type pureStringManipulatorOutput struct {
	Result string `cty:"result"`
}

type pureStringManipulatorModule struct{}

func (m *pureStringManipulatorModule) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunPureStringManipulator", &registry.RegisteredRunner{
		NewInput:  func() any { return new(pureStringManipulatorInput) },
		InputType: reflect.TypeOf(pureStringManipulatorInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(ctx context.Context, deps any, input *pureStringManipulatorInput) (*pureStringManipulatorOutput, error) {
			if input.Source == "" {
				return nil, fmt.Errorf("input 'Source' cannot be empty")
			}
			result := fmt.Sprintf("%s-%s", input.Source, input.Suffix)
			return &pureStringManipulatorOutput{Result: result}, nil
		},
	})
}

func TestPureGoModuleExecution(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	manifestHCL := `
		runner "string_manipulator" {
			lifecycle {
				on_run = "OnRunPureStringManipulator"
			}
			input "Source" { type = string }
			input "Suffix" { type = string }
			output "result" { type = string }
		}
	`
	gridHCL := `
		step "string_manipulator" "add_suffix" {
			arguments {
				Source = "hello"
				Suffix = "world"
			}
		}
	`
	files := map[string]string{
		"modules/string_manipulator/manifest.hcl": manifestHCL,
		"main.hcl": gridHCL,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, &pureStringManipulatorModule{})

	// --- Assert ---
	assert.NoError(t, result.Err, "Expected the run to succeed, but it failed.")
	assert.Contains(t, result.LogOutput, `msg="✅ Finished step" step=step.string_manipulator.add_suffix`)
}
