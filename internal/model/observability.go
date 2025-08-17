// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file defines the models for observability and telemetry settings.
//
// Why model observability settings?
//
// These structs (`Tracing`, `Metrics`) provide a formal schema for how a user
// can inject custom metadata into the system's telemetry. This allows for
// fine-grained control over tracing attributes and metric emission on a
// per-step basis, which is essential for debugging and monitoring complex workflows
// in a production environment.
// TODO: Describe how it works in details and fields descriptions
package model

import (
	"github.com/hashicorp/hcl/v2"
)

// Tracing defines opentelemetry tracing settings.
type Tracing struct {
	Attributes hcl.Expression `hcl:"attributes,attr"`
	SampleRate hcl.Expression `hcl:"sample_rate,attr"`
}

// Expressions returns a slice of all HCL expressions defined in the Tracing block.
func (t *Tracing) Expressions() []hcl.Expression {
	if t == nil {
		return nil
	}
	return []hcl.Expression{t.Attributes, t.SampleRate}
}

// Metrics defines settings for metric emission.
type Metrics struct {
	Emit hcl.Expression `hcl:"emit,attr"`
}

// Expressions returns a slice of all HCL expressions defined in the Metrics block.
func (m *Metrics) Expressions() []hcl.Expression {
	if m == nil {
		return nil
	}
	return []hcl.Expression{m.Emit}
}
