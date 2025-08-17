package testutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/app"
	"github.com/specialistvlad/burstgridgo/internal/handlers"
	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/stretchr/testify/require"
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
func RunIntegrationTest(t *testing.T, files map[string]string, handlers *handlers.Handlers) *HarnessResult {
	t.Helper()
	return RunIntegrationTestWithContext(context.Background(), t, files, handlers)
}

// RunIntegrationTestWithContext provides a standardized harness for running integration
// tests with a specific context provided by the caller.
func RunIntegrationTestWithContext(ctx context.Context, t *testing.T, files map[string]string, hndls *handlers.Handlers) *HarnessResult {
	t.Helper()

	// 1. Create a temporary root directory for the test.
	tmpDir, err := os.MkdirTemp("", ".tmp-integration-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	gridDir := filepath.Join(tmpDir, "grid")
	modulesDir := filepath.Join(tmpDir, "modules")
	require.NoError(t, os.Mkdir(gridDir, 0755))
	require.NoError(t, os.Mkdir(modulesDir, 0755))

	// 2. Write all HCL files to the temporary directory.
	//    The test provides relative paths (e.g., "modules/x/manifest.hcl"),
	//    which naturally creates the subdirectory structure within the root tmpDir.
	for name, content := range files {
		filePath := filepath.Join(tmpDir, name)
		dir := filepath.Dir(filePath)
		require.NoError(t, os.MkdirAll(dir, 0755))
		err = os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// 3. Prepare handlers and register all provided test modules.
	registry := registry.New(hndls)

	// 4. Configure the app to use the dedicated, non-overlapping subdirectories.
	appConfig := &app.Config{
		GridPath:    gridDir,
		ModulesPath: modulesDir,
		LogLevel:    "debug",
		LogFormat:   "text",
		WorkerCount: 4,
	}

	logBuffer := &SafeBuffer{}

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
		// Inject the prepared handlers into the app.
		testApp = app.NewApp(ctx, logBuffer, appConfig, registry)
	}()

	if panicErr != nil {
		return &HarnessResult{
			LogOutput: logBuffer.String(),
			Err:       fmt.Errorf("application startup panicked | %v", panicErr),
			App:       nil,
		}
	}

	// TODO: FIX THIS IN THE FUTURE. LOW PRIORITY
	// Instead of calling the full app.Run(), we call the loader methods
	// directly. This makes the tests for parsing more focused and reliable,
	// bypassing unrelated logic in the Run() method and ensuring errors propagate.
	var runErr error
	if err := testApp.LoadModules(); err != nil {
		runErr = err
	} else if err := testApp.LoadGrids(); err != nil {
		// We run LoadGrids() as well to get a complete "load phase" test.
		runErr = err
	}

	if os.Getenv("BGGO_TEST_LOGS") == "true" {
		t.Logf("--- Full Log Output for %s ---\n%s", t.Name(), logBuffer.String())
	}

	return &HarnessResult{
		LogOutput: logBuffer.String(),
		Err:       runErr,
		App:       testApp,
	}
}
