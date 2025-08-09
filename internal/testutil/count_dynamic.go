package testutil

import (
	"context"
	"reflect"
	"testing"

	"github.com/vk/burstgridgo/internal/registry"
	"github.com/zclconf/go-cty/cty"
)

// RunDynamicCountTest provides a standardized harness for testing dynamic count scenarios.
// It sets up a common three-step graph (provider -> dynamic_step -> consumer)
// and allows the caller to specify the value that the provider outputs for the 'count'.
// It returns the test harness result and the data that was ultimately captured by the consumer.
func RunDynamicCountTest(t *testing.T, countValue cty.Value) (*HarnessResult, cty.Value) {
	t.Helper()

	manifestsHCL := `
		runner "number_provider" {
			lifecycle { on_run = "OnRunNumberProvider" }
			output "value" {
				type = any // Use 'any' to test invalid type handling
			}
		}

		runner "print_indexed" {
			lifecycle { on_run = "OnRunPrintIndexed" }
			output "index_val" {
				type = number
			}
		}

		runner "consumer" {
			lifecycle { on_run = "OnRunConsumer" }
			input "data" {
				type = any
			}
		}
	`

	gridHCL := `
		step "number_provider" "A" {
			arguments {}
		}

		step "print_indexed" "B" {
			count = step.number_provider.A.output.value
		}

		step "consumer" "C" {
			arguments {
				data = step.print_indexed.B[*].output
			}
		}
	`
	files := map[string]string{
		"modules/manifests.hcl": manifestsHCL,
		"main.hcl":              gridHCL,
	}

	var consumedData cty.Value

	numberProviderModule := &SimpleModule{
		RunnerName: "OnRunNumberProvider",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(struct{}) },
			InputType: reflect.TypeOf(struct{}{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn: func(ctx context.Context, deps, input any) (any, error) {
				// This module's output is controlled by the test case.
				return map[string]cty.Value{"value": countValue}, nil
			},
		},
	}

	printModule := &SimpleModule{
		RunnerName: "OnRunPrintIndexed",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(struct{}) },
			InputType: reflect.TypeOf(struct{}{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn: func(ctx context.Context, deps, input any) (any, error) {
				// A real module would need the eval context to get count.index.
				// For this test, we just need it to run and produce some output.
				return map[string]cty.Value{"index_val": cty.NumberIntVal(0)}, nil
			},
		},
	}

	consumerModule := &SimpleModule{
		RunnerName: "OnRunConsumer",
		Runner: &registry.RegisteredRunner{
			NewInput: func() any {
				return new(struct {
					Data cty.Value `bggo:"data"`
				})
			},
			InputType: reflect.TypeOf(struct {
				Data cty.Value `bggo:"data"`
			}{}),
			NewDeps: func() any { return new(struct{}) },
			Fn: func(ctx context.Context, deps, inputRaw any) (any, error) {
				// Capture the data passed to the consumer for assertion.
				consumedData = inputRaw.(*struct {
					Data cty.Value `bggo:"data"`
				}).Data
				return nil, nil
			},
		},
	}

	result := RunIntegrationTest(t, files, numberProviderModule, printModule, consumerModule)
	return result, consumedData
}
