package integration_tests

import (
	"testing"
)

// Test for: conditional meta arg
func TestHclFeatures_ConditionalMetaArg(t *testing.T) {
	t.Skip("Feature not yet implemented: conditional 'count' meta-argument. See ADR-004 and ADR-006.")

	// --- Test Implementation (for when feature is ready) ---

	// // This test would run twice, once with should_run = true and once with false.
	//
	// // 1. Arrange: Create a mock runner that records its calls.
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

	// // 2. Define HCL grid with a conditional 'count'.
	// hcl := `
	// 	variable "should_run" {
	// 		type    = bool
	// 		default = true // This would be overridden in the test
	// 	}
	//
	// 	step "counter" "conditional_step" {
	// 		count = var.should_run ? 1 : 0
	//
	// 		arguments {}
	// 	}
	// `
	// tempDir := t.TempDir()
	// gridPath := filepath.Join(tempDir, "main.hcl")
	// if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
	// 	t.Fatalf("failed to write hcl file: %v", err)
	// }

	// // (Test would need a way to pass variables into the HCL context)
	//
	// // 3. Act & 4. Assert
	// // The test would assert that with should_run = true, the call count is 1.
	// // The test would assert that with should_run = false, the call count is 0.
}
