// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file defines the model for a Runner's lifecycle event hooks.
//
// Why a lifecycle model?
//
// This struct acts as the bridge between the declarative HCL world and the
// imperative Go code that executes the runner's logic. It provides a simple mapping
// from an event name in HCL (e.g., `on_run`) to a string that the execution engine
// can use to identify and invoke the correct Go handler function.
package model

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/specialistvlad/burstgridgo/internal/bggohcl"
)

// RunnerLifecycle maps a runner's events to Go handler names.
type RunnerLifecycle struct {
	OnRun string `hcl:"on_run,attr"`
}

// parseRunnerLifecycle finds and decodes the unique 'lifecycle' block from HCL.
func parseRunnerLifecycle(blocks hcl.Blocks) (RunnerLifecycle, hcl.Diagnostics) {
	var lifecycle RunnerLifecycle
	var diags hcl.Diagnostics

	lifecycleBlock, blockDiags := bggohcl.FindUniqueBlock(blocks, "lifecycle")
	diags = append(diags, blockDiags...)
	if diags.HasErrors() {
		return lifecycle, diags
	}

	// It's not an error for the lifecycle block to be absent.
	if lifecycleBlock == nil {
		return lifecycle, diags
	}

	decodeDiags := gohcl.DecodeBody(lifecycleBlock.Body, nil, &lifecycle)
	diags = append(diags, decodeDiags...)

	return lifecycle, diags
}
