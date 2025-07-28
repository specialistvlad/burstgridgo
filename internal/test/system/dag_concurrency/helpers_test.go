package system

import (
	"context"
	"sync"
	"time"

	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/vk/burstgridgo/internal/testutil"
	"github.com/zclconf/go-cty/cty"
)

// mockSleeperModule is a self-contained module for concurrency tests.
type mockSleeperModule struct {
	wg             *sync.WaitGroup
	executionTimes map[string]*testutil.ExecutionRecord // Corrected type
	mu             sync.Mutex
	sleepDuration  time.Duration
}

// Register registers the "sleeper" runner.
func (m *mockSleeperModule) Register(r *registry.Registry) {
	type sleeperInput struct {
		ID string `hcl:"id"`
	}
	r.RegisterHandler("OnRunSleeper", &registry.RegisteredHandler{
		NewInput: func() any { return new(sleeperInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (cty.Value, error) {
			defer m.wg.Done()
			input := inputRaw.(*sleeperInput)

			startTime := time.Now()
			time.Sleep(m.sleepDuration)
			endTime := time.Now()

			m.mu.Lock()
			m.executionTimes[input.ID] = &testutil.ExecutionRecord{Start: startTime, End: endTime} // Corrected type
			m.mu.Unlock()

			return cty.NilVal, nil
		},
	})
	r.DefinitionRegistry["sleeper"] = &schema.RunnerDefinition{
		Type:      "sleeper",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunSleeper"},
		Inputs:    []*schema.InputDefinition{{Name: "id"}},
	}
}
