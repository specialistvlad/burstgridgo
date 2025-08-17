// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file defines models for features that manage state across executions.
//
// Why model Cache and Dedupe?
//
// These features are designed to make workflows more efficient and robust.
// `Cache` allows the system to skip re-running a step if its inputs haven't
// changed, saving time and resources. `Dedupe` prevents multiple, concurrent,
// and identical steps from running wastefully. Modeling them as explicit structs
// provides a clear schema for these complex stateful behaviors.
package model

import (
	"github.com/hashicorp/hcl/v2"
)

// Cache defines caching behavior for a step's output.
type Cache struct {
	Enabled hcl.Expression `hcl:"enabled,attr"`
	Key     hcl.Expression `hcl:"key,attr"`
	TTL     hcl.Expression `hcl:"ttl,attr"`
	Scope   hcl.Expression `hcl:"scope,attr"`
	Restore hcl.Expression `hcl:"restore,attr"`
}

// Expressions returns a slice of all HCL expressions defined in the Cache block.
func (c *Cache) Expressions() []hcl.Expression {
	if c == nil {
		return nil
	}
	return []hcl.Expression{c.Enabled, c.Key, c.TTL, c.Scope, c.Restore}
}

// Dedupe defines deduplication behavior for step execution.
type Dedupe struct {
	Key    hcl.Expression `hcl:"key,attr"`
	Action hcl.Expression `hcl:"action,attr"`
	Scope  hcl.Expression `hcl:"scope,attr"`
}

// Expressions returns a slice of all HCL expressions defined in the Dedupe block.
func (d *Dedupe) Expressions() []hcl.Expression {
	if d == nil {
		return nil
	}
	return []hcl.Expression{d.Key, d.Action, d.Scope}
}
