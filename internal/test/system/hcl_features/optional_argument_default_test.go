package system

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/vk/burstgridgo/internal/testutil"
	"github.com/zclconf/go-cty/cty"
)

// mockDefaulterModule is a self-contained module for this test.
type mockDefaulterModule struct {
	wg            *sync.WaitGroup
	capturedInput *defaulterInput
	mu            sync.Mutex
}

// defaulterInput is the Go struct for the runner's arguments.
type defaulterInput struct {
	Mode     string `hcl:"mode,optional"`
	Required string `hcl:"required"`
}

// Register registers the "defaulter" runner.
func (m *mockDefaulterModule) Register(r *registry.Registry) {
	r.RegisterHandler("OnRunDefaulter", &registry.RegisteredHandler{
		NewInput: func() any { return new(defaulterInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (cty.Value, error) {
			m.mu.Lock()
			m.capturedInput = inputRaw.(*defaulterInput)
			m.mu.Unlock()
			m.wg.Done()
			return cty.NilVal, nil
		},
	})

	// Corrected: Create the cty.Value first, then take its address.
	defaultValue := cty.StringVal("standard")

	r.DefinitionRegistry["defaulter"] = &schema.RunnerDefinition{
		Type:      "defaulter",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunDefaulter"},
		Inputs: []*schema.InputDefinition{
			{Name: "required"},
			{
				Name:     "mode",
				Optional: true,
				Default:  &defaultValue, // Assign the pointer to the created value.
			},
		},
	}
}

// Test for: optional argument default
func TestHclFeatures_OptionalArgumentDefault(t *testing.T) {
	// --- Arrange ---
	hcl := `
		step "defaulter" "A" {
			arguments {
				required = "must-be-present"
			}
		}
	`
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	appConfig := &app.AppConfig{GridPath: gridPath}
	mockModule := &mockDefaulterModule{wg: &wg}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait()

	// --- Assert ---
	if mockModule.capturedInput == nil {
		t.Fatal("Spy did not capture any input.")
	}

	expectedInput := &defaulterInput{
		Mode:     "standard",
		Required: "must-be-present",
	}

	if diff := cmp.Diff(expectedInput, mockModule.capturedInput); diff != "" {
		t.Errorf("Captured input mismatch (-want +got):\n%s", diff)
	}
}
