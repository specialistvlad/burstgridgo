package testutil

import "time"

// ExecutionRecord holds the start and end times for a single step's execution.
type ExecutionRecord struct {
	Start time.Time
	End   time.Time
}
