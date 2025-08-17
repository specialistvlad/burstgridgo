// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// // This file defines the models for all error handling and retry strategies.
//
// Why encapsulate error handling?
//
// Step execution can fail for many reasons. This file consolidates all related
// configurable behaviors—such as retry attempts, backoff strategies, and fallback
// actions—into a set of dedicated structs. This provides a clear, structured
// representation of the user's desired failure policy, which the execution engine
// can then implement as a state machine.
// TODO: Describe how it works
package model

import (
	"github.com/hashicorp/hcl/v2"
)

// Backoff defines the strategy for delaying retries.
type Backoff struct {
	Strategy hcl.Expression `hcl:"strategy,attr"`
	Initial  hcl.Expression `hcl:"initial,attr"`
	Factor   hcl.Expression `hcl:"factor,attr"`
	Max      hcl.Expression `hcl:"max,attr"`
	Jitter   hcl.Expression `hcl:"jitter,attr"`
}

// Expressions returns a slice of all HCL expressions defined in the Backoff block.
func (b *Backoff) Expressions() []hcl.Expression {
	if b == nil {
		return nil
	}
	return []hcl.Expression{b.Strategy, b.Initial, b.Factor, b.Max, b.Jitter}
}

// Retry defines the retry behavior for a step.
type Retry struct {
	Attempts    hcl.Expression `hcl:"attempts,attr"`
	Backoff     *Backoff       `hcl:"backoff,block"`
	RetryOn     hcl.Expression `hcl:"retry_on,attr"`
	AbortOn     hcl.Expression `hcl:"abort_on,attr"`
	MaxDuration hcl.Expression `hcl:"max_duration,attr"`
}

// Expressions returns a slice of all HCL expressions defined in the Retry block.
func (r *Retry) Expressions() []hcl.Expression {
	if r == nil {
		return nil
	}
	exprs := []hcl.Expression{r.Attempts, r.RetryOn, r.AbortOn, r.MaxDuration}
	if r.Backoff != nil {
		exprs = append(exprs, r.Backoff.Expressions()...)
	}
	return exprs
}

// OnError defines behavior when a step fails.
type OnError struct {
	Action   hcl.Expression `hcl:"action,attr"`
	Fallback hcl.Expression `hcl:"fallback,attr"`
}

// Expressions returns a slice of all HCL expressions defined in the OnError block.
func (oe *OnError) Expressions() []hcl.Expression {
	if oe == nil {
		return nil
	}
	return []hcl.Expression{oe.Action, oe.Fallback}
}
