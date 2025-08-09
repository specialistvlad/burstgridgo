package testutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/hcl_adapter"
	"github.com/vk/burstgridgo/internal/registry"
)

// SafeBuffer is a thread-safe buffer for capturing log output in tests.
type SafeBuffer struct {
	b  bytes.Buffer
	mu sync.Mutex
}

// Write implements the io.Writer interface for SafeBuffer.
func (b *SafeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

// String implements the fmt.Stringer interface for SafeBuffer.
func (b *SafeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// HarnessResult holds the outcomes of an integration test run.
type HarnessResult struct {
	LogOutput string
	Err       error
	App       *app.App
}

// RunIntegrationTest provides a standardized harness for running integration tests
// using a default background context.
func RunIntegrationTest(t *testing.T, files map[string]string, modules ...registry.Module) *HarnessResult {
	t.Helper()
	return RunIntegrationTestWithContext(context.Background(), t, files, modules...)
}

// RunIntegrationTestWithContext provides a standardized harness for running integration
// tests with a specific context provided by the caller.
func RunIntegrationTestWithContext(ctx context.Context, t *testing.T, files map[string]string, modules ...registry.Module) *HarnessResult {
	t.Helper()

	// 1. Create a temporary root directory for the test.
	tmpDir, err := os.MkdirTemp("", "integration-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	// 2. Write all HCL files to the temporary directory.
	for name, content := range files {
		filePath := filepath.Join(tmpDir, name)
		dir := filepath.Dir(filePath)
		require.NoError(t, os.MkdirAll(dir, 0755))
		err = os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// 3. Configure the app to use the temporary directory.
	appConfig := &app.AppConfig{
		GridPath:    tmpDir,
		ModulesPath: tmpDir,
		LogLevel:    "debug",
		LogFormat:   "text",
		WorkerCount: 4,
	}

	logBuffer := &SafeBuffer{}
	loader := hcl_adapter.NewLoader()

	var testApp *app.App
	var panicErr any
	func() {
		defer func() {
			if r := recover(); r != nil {
				if os.Getenv("BGGO_TEST_LOGS") == "true" {
					t.Logf("--- HARNESS RECOVERED PANIC ---\n%q", fmt.Sprintf("%v", r))
				}
				panicErr = r
			}
		}()
		testApp = app.NewApp(logBuffer, appConfig, loader, modules...)
	}()

	if panicErr != nil {
		return &HarnessResult{
			LogOutput: logBuffer.String(),
			Err:       fmt.Errorf("application startup panicked: %v", panicErr),
			App:       nil,
		}
	}

	// 4. Run the application logic with the provided context.
	runErr := testApp.Run(ctx, appConfig)

	if os.Getenv("BGGO_TEST_LOGS") == "true" {
		t.Logf("--- Full Log Output for %s ---\n%s", t.Name(), logBuffer.String())
	}

	return &HarnessResult{
		LogOutput: logBuffer.String(),
		Err:       runErr,
		App:       testApp,
	}
}
