package app

import (
	"bytes"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/vk/burstgridgo/internal/registry"
)

// SafeBuffer is a thread-safe buffer for capturing log output in tests.
type SafeBuffer struct {
	b  bytes.Buffer
	mu sync.Mutex
}

func (b *SafeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *SafeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// ExecutionRecord holds the start and end times for a single step's execution.
// It is now public and can be shared across different test packages.
type ExecutionRecord struct {
	Start time.Time
	End   time.Time
}

// SetupAppTest creates a new app instance for system testing.
func SetupAppTest(t *testing.T, appConfig *AppConfig, modules ...registry.Module) (*App, *SafeBuffer) {
	t.Helper()

	logBuffer := &SafeBuffer{}
	appConfig.LogLevel = "debug"
	testApp := NewApp(logBuffer, appConfig, modules...)

	t.Cleanup(func() {
		if os.Getenv("BGGO_TEST_LOGS") == "true" {
			t.Logf("--- Full Log Output for %s ---\n%s", t.Name(), logBuffer.String())
		}
	})

	return testApp, logBuffer
}
