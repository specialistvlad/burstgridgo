// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file provides a struct to formally recognize the HCL `locals` block.
//
// Why recognize but not process this block?
//
// The purpose of this struct is to make the HCL parser aware of the `locals`
// block. This prevents the parser from throwing an "unrecognized block" error,
// allowing users to include standard HCL constructs in their files. The actual
// processing and evaluation of locals is handled by the HCL evaluation context
// at a later stage, not during initial model parsing.
package model

// (A nearly identical comment would go into `variable.go`)

import "github.com/hashicorp/hcl/v2"

// hclLocalsBlock is a struct to allow the parser to recognize `locals` blocks.
// Its body is not processed at this stage.
type hclLocalsBlock struct {
	Body hcl.Body `hcl:",remain"`
}
