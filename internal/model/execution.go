// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file defines the model for step execution timeouts.
//
// Why model timeouts?
//
// The `Timeouts` struct provides a clean, declarative way for users to enforce
// Service Level Objectives (SLOs) on their steps. It captures all time-based
// constraints a step must adhere to throughout its lifecycle. This allows the
// scheduler and execution engine to monitor progress and proactively terminate
// any step instance that violates these constraints, preventing stuck or runaway
// processes from consuming resources indefinitely.
//
// # How Timeouts Work
//
// The four timeouts correspond to different phases of a step's lifecycle for a single attempt:
//
//  1. `start`: A "schedule-to-start" guard. The timer begins when the scheduler
//     marks the step as ready to run (i.e., all dependencies are met). The step
//     must be picked up by a worker and begin execution before this timer expires.
//
//  2. `queue`: A subset of `start`, this is the maximum time a step can wait in a
//     worker's internal queue after being assigned but before its logic actually
//     starts running.
//
//  3. `execution`: A "start-to-finish" guard. The timer starts the moment the
//     runner's logic begins and stops when it completes. Exceeding this will
//     cancel the current attempt, possibly triggering a retry.
//
//  4. `deadline`: An absolute wall-clock time after which the step attempt will be
//     cancelled. This takes precedence over all other timeouts and retry settings.
package model

import (
	"github.com/hashicorp/hcl/v2"
)

// Timeouts defines various timeout constraints for a step.
type Timeouts struct {
	// Execution is the maximum duration for a single attempt of the step,
	// measured from the moment execution begins until it concludes. If this
	// duration is exceeded, the current attempt is cancelled.
	Execution hcl.Expression `hcl:"execution,attr"`

	// Start is the maximum duration the step can wait to begin execution after
	// being scheduled. The timer starts when the step's dependencies are met
	// and it is considered "ready to run".
	Start hcl.Expression `hcl:"start,attr"`

	// Queue is the maximum duration the step can wait in a worker's internal
	// run queue before its logic starts. This is typically a subset of the
	// `Start` timeout.
	Queue hcl.Expression `hcl:"queue,attr"`

	// Deadline is an absolute wall-clock time (can be specified as a future
	// timestamp or a duration from the workflow's start) after which the step
	// will be cancelled. It overrides all other timeouts and retries if it
	// is the sooner to expire.
	Deadline hcl.Expression `hcl:"deadline,attr"`
}

// Expressions returns a slice of all HCL expressions defined in the Timeouts block.
func (t *Timeouts) Expressions() []hcl.Expression {
	if t == nil {
		return nil
	}
	return []hcl.Expression{t.Execution, t.Start, t.Queue, t.Deadline}
}
