package integration_tests

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
	"github.com/zclconf/go-cty/cty"
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

// mockStateModule now only registers the Go handlers.
type mockStateModule struct {
	wg         *sync.WaitGroup
	finalValue *atomic.Int32
}

// Register registers the "stateful_counter" asset and "counter_op" runner Go handlers.
func (m *mockStateModule) Register(r *registry.Registry) {
	// --- "stateful_counter" Asset: Go Handlers ---
	r.RegisterAssetHandler("CreateStatefulCounter", &registry.RegisteredAsset{
		NewInput: func() any { return new(schema.StepArgs) },
		CreateFn: func(context.Context, any) (any, error) {
			return new(statefulCounter), nil
		},
	})
	r.RegisterAssetHandler("DestroyStatefulCounter", &registry.RegisteredAsset{
		DestroyFn: func(any) error { return nil },
	})

	// --- "counter_op" Runner: Go Handler ---
	type opDeps struct {
		Counter *statefulCounter `hcl:"counter"`
	}
	type opInput struct {
		Action string `hcl:"action"`
	}
	r.RegisterRunner("OnRunCounterOp", &registry.RegisteredRunner{
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
				m.finalValue.Store(val)
				return cty.ObjectVal(map[string]cty.Value{
					"value": cty.NumberIntVal(int64(val)),
				}), nil
			default:
				return cty.NilVal, fmt.Errorf("unknown action: %s", input.Action)
			}
		},
	})
}

// Test for: Resource state is correctly modified across multiple steps.
func TestCoreExecution_ResourceStateModification(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write HCL manifests for the asset and runner.
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
			lifecycle { on_run = "OnRunCounterOp" }
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
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "stateful_counter"), 0755); err != nil {
		t.Fatalf("failed to create asset module dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "counter_op"), 0755); err != nil {
		t.Fatalf("failed to create runner module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/stateful_counter/manifest.hcl"), []byte(assetManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write asset manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/counter_op/manifest.hcl"), []byte(runnerManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write runner manifest: %v", err)
	}

	// 2. The user's grid file.
	gridHCL := `
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
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(3)

	var finalValue atomic.Int32
	// 3. Configure the app to use the temporary directory for module discovery.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockStateModule{wg: &wg, finalValue: &finalValue}
	testApp, _ := app.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait()

	// --- Assert ---
	finalCount := finalValue.Load()
	if finalCount != 2 {
		t.Errorf("expected final counter value to be 2, but got %d", finalCount)
	}
}
