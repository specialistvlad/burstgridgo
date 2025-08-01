package integration_tests

import (
	"testing"
)

// Test for: count meta arg
func TestHclFeatures_CountMetaArg(t *testing.T) {
	t.Skip()
	t.Skip("Feature not yet implemented: 'count' meta-argument. See ADR-004 and ADR-006.")

	// --- Test Implementation (for when feature is ready) ---

	// // 1. Arrange: Create a mock runner that records each time it's called.
	// type counterModule struct {
	// 	wg    *sync.WaitGroup
	// 	calls *atomic.Int32
	// }
	//
	// func (m *counterModule) Register(r *registry.Registry) {
	// 	r.RegisterHandler("OnRunCounter", &registry.RegisteredHandler{
	// 		NewInput: func() any { return new(schema.StepArgs) },
	// 		NewDeps:  func() any { return new(struct{}) },
	// 		Fn: func(ctx context.Context, deps any, input any) (cty.Value, error) {
	// 			m.calls.Add(1)
	// 			m.wg.Done()
	// 			return cty.NilVal, nil
	// 		},
	// 	})
	// 	r.DefinitionRegistry["counter"] = &schema.RunnerDefinition{
	// 		Type: "counter",
	// 		Lifecycle: &schema.Lifecycle{OnRun: "OnRunCounter"},
	// 	}
	// }

	// // 2. Define HCL grid with the 'count' meta-argument.
	// hcl := `
	// 	step "counter" "A" {
	// 		count = 3
	// 		arguments {}
	// 	}
	// `
	// tempDir := t.TempDir()
	// gridPath := filepath.Join(tempDir, "main.hcl")
	// if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
	// 	t.Fatalf("failed to write hcl file: %v", err)
	// }

	// var wg sync.WaitGroup
	// wg.Add(3) // Expect the runner to be called 3 times.
	// var callCount atomic.Int32

	// appConfig := &app.AppConfig{GridPath: gridPath}
	// mockModule := &counterModule{wg: &wg, calls: &callCount}
	// testApp, _ := app.SetupAppTest(t, appConfig, mockModule)

	// // 3. Act: Run the application.
	// runErr := testApp.Run(context.Background(), appConfig)
	// if runErr != nil {
	// 	t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	// }

	// wg.Wait()

	// // 4. Assert: Verify the runner was executed the correct number of times.
	// finalCount := callCount.Load()
	// if finalCount != 3 {
	// 	t.Errorf("expected runner to be called 3 times due to count, but was called %d times", finalCount)
	// }
}
