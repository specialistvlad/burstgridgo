// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// Package model provides the Go struct representation of the BurstGrid HCL
// configuration. Its core purpose is to create a strongly-typed, in-memory model
// of the user's definitions by parsing the raw HCL files.
//
// # Core Concepts
//
// The model is built around a few key structures:
//
//   - Grid: The root container representing an entire workspace. It aggregates all
//     steps parsed from one or more .hcl files.
//
//   - Runner: The reusable "template" or "definition" of a task. It defines a
//     contract, specifying the required inputs, expected outputs, and lifecycle logic.
//
//   - Step: An "instance" or "invocation" of a Runner. It represents a single node
//     in the execution graph and contains the specific configuration (arguments,
//     timeouts, retry logic, etc.) for that invocation.
//
//   - FSInfo: Metadata that links every Step and Runner back to its source file. This
//     is critical for providing clear error messages and for resolving file-based
//     modules and scopes.
//
// Why a separate model package?
//
// This package acts as a critical intermediate layer. It organizes raw HCL
// expressions into a predictable structure, which serves as the foundation for
// subsequent processing stages like validation and graph construction.
//
// The key advantages of this approach are:
//
//  1. Structured Validation: Before attempting to evaluate any expressions, we can
//     traverse the Go model to perform static checks on the overall shape of the
//     configuration, catching structural and logical errors early.
//
//  2. Foundation for Graph Building: The model is the direct input for the graph
//     builder. A builder can consume the `Grid`'s `Steps`, look up their `Runner`
//     definitions to validate arguments, and inspect expressions to resolve the
//     final execution DAG.
//
//  3. Isolate the Executor: While components like the graph builder must be aware
//     of HCL to evaluate expressions, the final execution engine does not. The
//     executor can be designed to work with a simpler, fully-resolved data
//     structure derived from this model.
//
// In short, this package provides the ideal, structured blueprint from which the
// final, executable graph is built. It turns free-form HCL into a validated,
// traversable Go representation ready for the next compilation stages.
package model
