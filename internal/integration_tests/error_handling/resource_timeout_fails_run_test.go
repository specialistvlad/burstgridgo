package integration_tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

// mockTimeoutResourceModule contains the logic for an asset that can time out.
type mockTimeoutResourceModule struct{}

// Register registers the "timeout_resource" asset's Go handlers.
func (m *mockTimeoutResourceModule) Register(r *registry.Registry) {
	r.RegisterAssetHandler("CreateTimeoutResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(struct{}) },
		CreateFn: func(ctx context.Context, input any) (any, error) {
			select {
			case <-time.After(1 * time.Second):
				// This case should not be reached if the test timeout is shorter.
				return nil, errors.New("create function was not cancelled by context")
			case <-ctx.Done():
				return nil, ctx.Err() // This is the expected path.
			}
		},
	})
	r.RegisterAssetHandler("DestroyTimeoutResource", &registry.RegisteredAsset{
		DestroyFn: func(resource any) error { return nil },
	})
}

// TestErrorHandling_ResourceTimeout_FailsRun validates that a resource's
// Create handler is cancelled if it exceeds the context's deadline.
func TestErrorHandling_ResourceTimeout_FailsRun(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	manifestHCL := `
		asset "timeout_resource" {
			lifecycle {
				create  = "CreateTimeoutResource"
				destroy = "DestroyTimeoutResource"
			}
		}
	`
	gridHCL := `
		resource "timeout_resource" "A" {
			arguments {}
		}
	`
	files := map[string]string{
		"modules/timeout_resource/manifest.hcl": manifestHCL,
		"main.hcl":                              gridHCL,
	}

	// --- Act ---
	// Create a context with a very short timeout to trigger the failure.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Use the harness that accepts a custom context.
	result := testutil.RunIntegrationTestWithContext(ctx, t, files, &mockTimeoutResourceModule{})

	// --- Assert ---
	require.Error(t, result.Err, "app.Run() should have returned a timeout error, but it returned nil")
	require.ErrorIs(t, result.Err, context.DeadlineExceeded, "expected the error chain to contain context.DeadlineExceeded")
}
