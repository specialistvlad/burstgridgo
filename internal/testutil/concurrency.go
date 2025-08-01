package testutil

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/vk/burstgridgo/internal/registry"
)

// MockSleeperModule is a shared, self-contained module for concurrency tests.
// It records the execution time of each step that uses it.
type MockSleeperModule struct {
	ExecutionTimes map[string]*ExecutionRecord
	mu             sync.Mutex
	sleepDuration  time.Duration
	completionChan chan<- string
}

// NewMockSleeperModule creates a new sleeper module for testing.
func NewMockSleeperModule(completionChan chan<- string, sleep time.Duration) *MockSleeperModule {
	return &MockSleeperModule{
		ExecutionTimes: make(map[string]*ExecutionRecord),
		sleepDuration:  sleep,
		completionChan: completionChan,
	}
}

// Register registers the "sleeper" runner's Go handler.
func (m *MockSleeperModule) Register(r *registry.Registry) {
	type sleeperInput struct {
		ID string `bggo:"id"`
	}

	r.RegisterRunner("OnRunSleeper", &registry.RegisteredRunner{
		NewInput:  func() any { return new(sleeperInput) },
		InputType: reflect.TypeOf(sleeperInput{}), // This line was missing
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (any, error) {
			input := inputRaw.(*sleeperInput)

			startTime := time.Now()
			time.Sleep(m.sleepDuration)
			endTime := time.Now()

			m.mu.Lock()
			m.ExecutionTimes[input.ID] = &ExecutionRecord{Start: startTime, End: endTime}
			m.mu.Unlock()

			if m.completionChan != nil {
				m.completionChan <- input.ID
			}
			return nil, nil
		},
	})
}
