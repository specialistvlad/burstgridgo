package bggohcl

import (
	"github.com/hashicorp/hcl/v2"
)

// FindUniqueBlock searches a slice of blocks for all blocks of a given name.
// It returns a diagnostic error if more than one block of that name is found.
// If no block is found, it returns nil.
func FindUniqueBlock(blocks hcl.Blocks, name string) (*hcl.Block, hcl.Diagnostics) {
	var found *hcl.Block
	var diags hcl.Diagnostics

	for _, block := range blocks {
		if block.Type == name {
			if found != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate \"" + name + "\" block",
					Detail:   "Only one \"" + name + "\" block is allowed.",
					Subject:  &block.DefRange,
				})
			}
			found = block
		}
	}

	return found, diags
}
