package system

import (
	"testing"
)

// Test for: foreach meta arg
func TestHclFeatures_ForEachMetaArg(t *testing.T) {
	t.Skip("Feature not yet implemented: 'for_each' meta-argument. See ADR-004 and ADR-006.")

	// --- Test Implementation (for when feature is ready) ---

	// // 1. Arrange: Create a mock runner that records the key of each instance.
	// type recorderModule struct {
	// 	wg          *sync.WaitGroup
	// 	invocations *sync.Map // A thread-safe map to store which keys were processed.
	// }
	//
	// type recorderInput struct {
	// 	Key string `hcl:"key"`
	// }
	//
	// func (m *recorderModule) Register(r *registry.Registry) {
	// 	r.RegisterHandler("OnRunRecorder", &registry.RegisteredHandler{
	// 		NewInput: func() any { return new(recorderInput) },
	// 		NewDeps:  func() any { return new(struct{}) },
	// 		Fn: func(ctx context.Context, deps any, input any) (cty.Value, error) {
	// 			key := input.(*recorderInput).Key
	// 			m.invocations.Store(key, true)
	// 			m.wg.Done()
	// 			return cty.NilVal, nil
	// 		},
	// 	})
	// 	r.DefinitionRegistry["recorder"] = &schema.RunnerDefinition{
	// 		Type: "recorder",
	// 		Lifecycle: &schema.Lifecycle{OnRun: "OnRunRecorder"},
	// 		Inputs: []*schema.InputDefinition{{Name: "key"}},
	// 	}
	// }

	// // 2. Define HCL grid with the 'for_each' meta-argument.
	// hcl := `
	// 	variable "items" {
	// 		type    = set(string)
	// 		default = ["A", "B", "C"]
	// 	}
	//
	// 	step "recorder" "by_key" {
	// 		for_each = var.items
	//
	// 		arguments {
	// 			// 'each.key' and 'each.value' would be available in for_each.
	// 			key = each.key
	// 		}
	// 	}
	// `
	// tempDir := t.TempDir()
	// gridPath := filepath.Join(tempDir, "main.hcl")
	// if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
	// 	t.Fatalf("failed to write hcl file: %v", err)
	// }

	// var wg sync.WaitGroup
	// wg.Add(3) // Expect the runner to be called 3 times, once for each item.
	// var invocations sync.Map

	// appConfig := &app.AppConfig{GridPath: gridPath}
	// mockModule := &recorderModule{wg: &wg, invocations: &invocations}
	// testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

	// // 3. Act: Run the application.
	// runErr := testApp.Run(context.Background(), appConfig)
	// if runErr != nil {
	// 	t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	// }

	// wg.Wait()

	// // 4. Assert: Verify a runner was executed for each expected key.
	// expectedKeys := []string{"A", "B", "C"}
	// for _, key := range expectedKeys {
	// 	if _, ok := invocations.Load(key); !ok {
	// 		t.Errorf("expected runner to be invoked for key '%s', but it was not", key)
	// 	}
	// }
}
