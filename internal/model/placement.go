// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file defines the model for step placement and scheduling hints.
//
// Why model placement?
//
// In a distributed system, *where* a step runs can be as important as *what* it
// runs. The `Placement` struct is designed to capture user intent about scheduling
// constraints, such as required hardware labels or data locality. This provides a
// structured contract for a distributed scheduler to make intelligent decisions
// about assigning steps to specific workers.
// TODO: Update documentation
package model

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/specialistvlad/burstgridgo/internal/bggoexpr"
)

// Placement defines scheduling and placement constraints.
type Placement struct {
	Labels      hcl.Expression `hcl:"labels,attr"`
	Constraints hcl.Expression `hcl:"constraints,attr"`
	ShardBy     hcl.Expression `hcl:"shard_by,attr"`
}

// Expressions returns a slice of all HCL expressions defined in the Placement block.
func (p *Placement) Expressions() []hcl.Expression {
	if p == nil {
		return nil
	}
	return []hcl.Expression{p.Labels, p.Constraints, p.ShardBy}
}

// parsePlacement finds and decodes a 'placement' block from HCL.
func parsePlacement(blocks hcl.Blocks) (*Placement, []hcl.Expression, hcl.Diagnostics) {
	return bggoexpr.ParseBlock[*Placement](blocks, "placement")
}
