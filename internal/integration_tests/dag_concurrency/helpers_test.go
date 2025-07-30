package integration_tests

import (
	"context"
	"sync"
	"time"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/zclconf/go-cty/cty"
)

// mockSleeperModule is a self-contained module for concurrency tests.
// It now only registers the Go handler. The HCL definition will be discovered from a file.
type mockSleeperModule struct {
	wg             *sync.WaitGroup
	executionTimes map[string]*app.ExecutionRecord
	mu             sync.Mutex
	sleepDuration  time.Duration
}

// Register registers the "sleeper" runner's Go handler.
func (m *mockSleeperModule) Register(r *registry.Registry) {
	type sleeperInput struct {
		ID string `hcl:"id"`
	}
	r.RegisterRunner("OnRunSleeper", &registry.RegisteredRunner{
		NewInput: func() any { return new(sleeperInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (cty.Value, error) {
			defer m.wg.Done()
			input := inputRaw.(*sleeperInput)

			startTime := time.Now()
			time.Sleep(m.sleepDuration)
			endTime := time.Now()

			m.mu.Lock()
			m.executionTimes[input.ID] = &app.ExecutionRecord{Start: startTime, End: endTime}
			m.mu.Unlock()

			return cty.NilVal, nil
		},
	})
}
