package integration_tests

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

// --- Test-Specific Mocks ---

// statefulCounter is our mock resource.
type statefulCounter struct {
	value atomic.Int32
}

func (c *statefulCounter) Increment() {
	c.value.Add(1)
}

func (c *statefulCounter) Get() int32 {
	return c.value.Load()
}

// mockStateModule uses the new pure-Go contract.
type mockStateModule struct {
	finalValue *atomic.Int32
}

// Register registers the asset and runner Go handlers.
func (m *mockStateModule) Register(r *registry.Registry) {
	// --- "stateful_counter" Asset: Go Handlers ---
	r.RegisterAssetHandler("CreateStatefulCounter", &registry.RegisteredAsset{
		NewInput: func() any { return new(struct{}) },
		CreateFn: func(context.Context, any) (any, error) {
			return new(statefulCounter), nil
		},
	})
	r.RegisterAssetHandler("DestroyStatefulCounter", &registry.RegisteredAsset{
		DestroyFn: func(any) error { return nil },
	})

	// --- "counter_op" Runner: Go Handler ---
	type opDeps struct {
		Counter *statefulCounter `bggo:"counter"`
	}
	type opInput struct {
		Action string `bggo:"action"`
	}
	type opOutput struct {
		Value int32 `cty:"value"`
	}

	r.RegisterRunner("OnRunCounterOp", &registry.RegisteredRunner{
		NewInput:  func() any { return new(opInput) },
		InputType: reflect.TypeOf(opInput{}), // This line was missing
		NewDeps:   func() any { return new(opDeps) },
		Fn: func(_ context.Context, depsRaw any, inputRaw any) (*opOutput, error) {
			deps := depsRaw.(*opDeps)
			input := inputRaw.(*opInput)

			switch input.Action {
			case "increment":
				deps.Counter.Increment()
				return nil, nil // Return nil for no output
			case "get":
				val := deps.Counter.Get()
				m.finalValue.Store(val)
				return &opOutput{Value: val}, nil
			default:
				return nil, fmt.Errorf("unknown action: %s", input.Action)
			}
		},
	})
}

// TestCoreExecution_ResourceStateModification validates that a resource's state
// can be modified by multiple steps in a dependency chain.
func TestCoreExecution_ResourceStateModification(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	assetManifestHCL := `
		asset "stateful_counter" {
			lifecycle {
				create = "CreateStatefulCounter"
				destroy = "DestroyStatefulCounter"
			}
		}
	`
	runnerManifestHCL := `
		runner "counter_op" {
			lifecycle {
				on_run = "OnRunCounterOp"
			}
			uses "counter" {
				asset_type = "stateful_counter"
			}
			input "action" {
				type = string
			}
			output "value" {
				type = number
			}
		}
	`
	gridHCL := `
		resource "stateful_counter" "shared" {}

		step "counter_op" "inc_A" {
			uses {
				counter = resource.stateful_counter.shared
			}
			arguments {
				action = "increment"
			}
		}

		step "counter_op" "inc_B" {
			uses {
				counter = resource.stateful_counter.shared
			}
			arguments {
				action = "increment"
			}
			depends_on = ["counter_op.inc_A"]
		}

		step "counter_op" "get_final" {
			uses {
				counter = resource.stateful_counter.shared
			}
			arguments {
				action = "get"
			}
			depends_on = ["counter_op.inc_B"]
		}
	`
	files := map[string]string{
		"modules/stateful_counter/manifest.hcl": assetManifestHCL,
		"modules/counter_op/manifest.hcl":       runnerManifestHCL,
		"main.hcl":                              gridHCL,
	}

	var finalValue atomic.Int32
	mockModule := &mockStateModule{finalValue: &finalValue}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "test run failed unexpectedly")

	finalCount := finalValue.Load()
	require.Equal(t, int32(2), finalCount, "expected final counter value to be 2")
}
