package bggohcl

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// TraversalKey generates a stable, canonical string representation for an hcl.Traversal,
// suitable for use as a map key.
func TraversalKey(t hcl.Traversal) string {
	// e.g., var.foo[0].bar
	return string(hclwrite.TokensForTraversal(t).Bytes())
}
