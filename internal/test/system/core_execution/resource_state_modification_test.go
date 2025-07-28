package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/vk/burstgridgo/internal/testutil"
	"github.com/zclconf/go-cty/cty"
)

// --- Test-Specific Mocks ---

// statefulCounter is our mock resource. It holds a simple integer value
// that can be modified by steps.
type statefulCounter struct {
	value atomic.Int32
}

func (c *statefulCounter) Increment() {
	c.value.Add(1)
}

func (c *statefulCounter) Get() int32 {
	return c.value.Load()
}

// mockStateModule is a self-contained module for this specific test.
// It holds a reference to the final value captured by the "get" step.
type mockStateModule struct {
	wg         *sync.WaitGroup
	finalValue *atomic.Int32
}

// Register registers the "stateful_counter" asset and "counter_op" runner.
func (m *mockStateModule) Register(r *registry.Registry) {
	// --- "stateful_counter" Asset ---
	r.RegisterAssetHandler("CreateStatefulCounter", &registry.RegisteredAssetHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		CreateFn: func(context.Context, any) (any, error) {
			return new(statefulCounter), nil // Return a new, zeroed counter.
		},
	})
	r.RegisterAssetHandler("DestroyStatefulCounter", &registry.RegisteredAssetHandler{
		DestroyFn: func(any) error { return nil },
	})
	r.AssetDefinitionRegistry["stateful_counter"] = &schema.AssetDefinition{
		Type: "stateful_counter",
		Lifecycle: &schema.AssetLifecycle{
			Create:  "CreateStatefulCounter",
			Destroy: "DestroyStatefulCounter",
		},
	}

	// --- "counter_op" Runner ---
	type opDeps struct {
		Counter *statefulCounter `hcl:"counter"`
	}
	type opInput struct {
		Action string `hcl:"action"`
	}
	r.RegisterHandler("OnRunCounterOp", &registry.RegisteredHandler{
		NewInput: func() any { return new(opInput) },
		NewDeps:  func() any { return new(opDeps) },
		Fn: func(_ context.Context, depsRaw any, inputRaw any) (cty.Value, error) {
			defer m.wg.Done()
			deps := depsRaw.(*opDeps)
			input := inputRaw.(*opInput)

			switch input.Action {
			case "increment":
				deps.Counter.Increment()
				return cty.NilVal, nil
			case "get":
				val := deps.Counter.Get()
				m.finalValue.Store(val) // Store the final value for the test to assert.
				return cty.ObjectVal(map[string]cty.Value{
					"value": cty.NumberIntVal(int64(val)),
				}), nil
			default:
				return cty.NilVal, fmt.Errorf("unknown action: %s", input.Action)
			}
		},
	})
	r.DefinitionRegistry["counter_op"] = &schema.RunnerDefinition{
		Type:      "counter_op",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunCounterOp"},
		Uses:      []*schema.UsesDefinition{{LocalName: "counter", AssetType: "stateful_counter"}},
		Inputs:    []*schema.InputDefinition{{Name: "action"}},
		Outputs:   []*schema.OutputDefinition{{Name: "value"}},
	}
}

// Test for: Resource state is correctly modified across multiple steps.
func TestCoreExecution_ResourceStateModification(t *testing.T) {
	// --- Arrange ---
	hcl := `
		resource "stateful_counter" "shared" {}

		step "counter_op" "inc_A" {
			uses { counter = resource.stateful_counter.shared }
			arguments { action = "increment" }
		}

		step "counter_op" "inc_B" {
			uses { counter = resource.stateful_counter.shared }
			arguments { action = "increment" }
			depends_on = ["counter_op.inc_A"]
		}

		step "counter_op" "get_final" {
			uses { counter = resource.stateful_counter.shared }
			arguments { action = "get" }
			depends_on = ["counter_op.inc_B"]
		}
	`
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(3) // Expect three steps to run.

	var finalValue atomic.Int32
	appConfig := &app.AppConfig{GridPath: gridPath}
	mockModule := &mockStateModule{wg: &wg, finalValue: &finalValue}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait() // Wait for all three steps to complete.

	// --- Assert ---
	finalCount := finalValue.Load()
	if finalCount != 2 {
		t.Errorf("expected final counter value to be 2, but got %d", finalCount)
	}
}
