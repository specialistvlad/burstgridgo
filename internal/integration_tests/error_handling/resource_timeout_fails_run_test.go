package integration_tests

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
)

// mockTimeoutResourceModule contains the logic for an asset that can time out.
// It now only registers the Go handlers.
type mockTimeoutResourceModule struct{}

// Register registers the "timeout_resource" asset's Go handlers.
func (m *mockTimeoutResourceModule) Register(r *registry.Registry) {
	r.RegisterAssetHandler("CreateTimeoutResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(struct{}) },
		CreateFn: func(ctx context.Context, input any) (any, error) {
			select {
			case <-time.After(1 * time.Second):
				return nil, nil // Should not be reached
			case <-ctx.Done():
				return nil, ctx.Err() // Will be reached due to test timeout
			}
		},
	})
	r.RegisterAssetHandler("DestroyTimeoutResource", &registry.RegisteredAsset{
		DestroyFn: func(resource any) error { return nil },
	})
}

// Test for: resource connection times out during creation
func TestErrorHandling_ResourceTimeout_FailsRun(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write the HCL manifest for the asset.
	manifestHCL := `
		asset "timeout_resource" {
			lifecycle {
				create  = "CreateTimeoutResource"
				destroy = "DestroyTimeoutResource"
			}
		}
	`
	moduleDir := filepath.Join(tempDir, "modules", "timeout_resource")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "manifest.hcl"), []byte(manifestHCL), 0600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// 2. Define the user's grid file.
	gridHCL := `
		resource "timeout_resource" "A" {
			arguments {}
		}
	`
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	// 3. Configure the app for module discovery.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	testApp, _ := app.SetupAppTest(t, appConfig, &mockTimeoutResourceModule{})

	// Create a context with a very short timeout (50ms).
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// --- Act ---
	runErr := testApp.Run(ctx, appConfig)

	// --- Assert ---
	if runErr == nil {
		t.Fatal("app.Run() should have returned a timeout error, but it returned nil")
	}

	if !errors.Is(runErr, context.DeadlineExceeded) {
		t.Errorf("expected the error chain to contain context.DeadlineExceeded, but it did not. Got: %v", runErr)
	}
}
