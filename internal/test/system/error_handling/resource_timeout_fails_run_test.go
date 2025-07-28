package system

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/vk/burstgridgo/internal/testutil"
)

// mockTimeoutResourceModule contains the logic for an asset that can time out.
type mockTimeoutResourceModule struct{}

// Register registers the "timeout_resource" asset.
func (m *mockTimeoutResourceModule) Register(r *registry.Registry) {
	// The Create handler will wait on a timer longer than our test's
	// context timeout, but it will exit immediately if the context is canceled.
	r.RegisterAssetHandler("CreateTimeoutResource", &registry.RegisteredAssetHandler{
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
	// A destroy handler is needed for a valid lifecycle definition.
	r.RegisterAssetHandler("DestroyTimeoutResource", &registry.RegisteredAssetHandler{
		DestroyFn: func(resource any) error { return nil },
	})
	r.AssetDefinitionRegistry["timeout_resource"] = &schema.AssetDefinition{
		Type: "timeout_resource",
		Lifecycle: &schema.AssetLifecycle{
			Create:  "CreateTimeoutResource",
			Destroy: "DestroyTimeoutResource",
		},
	}
}

// Test for: resource connection times out during creation
func TestErrorHandling_ResourceTimeout_FailsRun(t *testing.T) {
	// --- Arrange ---
	hcl := `
		resource "timeout_resource" "A" {
			arguments {}
		}
	`
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	appConfig := &app.AppConfig{GridPath: gridPath}
	testApp, _ := testutil.SetupAppTest(t, appConfig, &mockTimeoutResourceModule{})

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
