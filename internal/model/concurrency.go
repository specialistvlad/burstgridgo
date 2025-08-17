// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// // This file models the configuration for a step's concurrency and rate limiting.
//
// Why model these separately?
//
// Concurrency and rate limiting are complex but distinct scheduling constraints.
// By modeling them in dedicated structs (`Concurrency`, `RateLimit`), we create a
// clear, domain-specific representation of the user's intent. This allows the
// scheduler component to easily consume this configuration and apply the correct
// token buckets or concurrent execution limits without needing to interpret
// generic key-value attributes.
package model

import (
	"github.com/hashicorp/hcl/v2"
)

// Concurrency defines the concurrency constraints for a step.
type Concurrency struct {
	Limit  hcl.Expression `hcl:"limit,attr"`
	PerKey hcl.Expression `hcl:"per_key,attr"`
	Order  hcl.Expression `hcl:"order,attr"`
}

// Expressions returns a slice of all HCL expressions defined in the Concurrency block.
func (c *Concurrency) Expressions() []hcl.Expression {
	if c == nil {
		return nil
	}
	return []hcl.Expression{c.Limit, c.PerKey, c.Order}
}

// RateLimit defines the rate limiting behavior for a step.
type RateLimit struct {
	Limit hcl.Expression `hcl:"limit,attr"`
	Per   hcl.Expression `hcl:"per,attr"`
	Burst hcl.Expression `hcl:"burst,attr"`
	Key   hcl.Expression `hcl:"key,attr"`
}

// Expressions returns a slice of all HCL expressions defined in the RateLimit block.
func (rl *RateLimit) Expressions() []hcl.Expression {
	if rl == nil {
		return nil
	}
	return []hcl.Expression{rl.Limit, rl.Per, rl.Burst, rl.Key}
}
