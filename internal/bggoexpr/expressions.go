package bggoexpr

import (
	"reflect"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/specialistvlad/burstgridgo/internal/bggohcl"
)

// TraversalKey generates a stable, canonical string representation for an hcl.Traversal,
// suitable for use as a map key.
func TraversalKey(t hcl.Traversal) string {
	// e.g., var.foo[0].bar
	return string(hclwrite.TokensForTraversal(t).Bytes())
}

// Expressioner is an interface for HCL block structs that can provide a list
// of their contained HCL expressions. This is used by the generic parseBlock
// function to collect all expressions for dependency analysis.
type Expressioner interface {
	Expressions() []hcl.Expression
}

// extractReferencesAndFunctions walks through HCL expressions to find all unique
// variable traversals and function calls. The returned slices are sorted to
// ensure a deterministic order.
func extractReferencesAndFunctions(exprs ...hcl.Expression) ([]hcl.Traversal, []string) {
	traversals := make(map[string]hcl.Traversal)
	functions := make(map[string]struct{})

	for _, expr := range exprs {
		if expr == nil {
			continue
		}

		// Use the built-in Variables() method for robust variable collection.
		for _, traversal := range expr.Variables() {
			key := TraversalKey(traversal)
			traversals[key] = traversal
		}

		// Walk the syntax tree to find what Variables() doesn't give us: function calls.
		if syntaxExpr, ok := expr.(hclsyntax.Expression); ok {
			walkForFunctions(syntaxExpr, functions)
		}
	}

	// Convert maps to slices for the return value.
	traversalSlice := make([]hcl.Traversal, 0, len(traversals))
	traversalKeys := make([]string, 0, len(traversals))

	for k := range traversals {
		traversalKeys = append(traversalKeys, k)
	}
	sort.Strings(traversalKeys) // Sort keys for deterministic output

	for _, k := range traversalKeys {
		traversalSlice = append(traversalSlice, traversals[k])
	}

	functionSlice := make([]string, 0, len(functions))
	for f := range functions {
		functionSlice = append(functionSlice, f)
	}
	sort.Strings(functionSlice) // Sort for deterministic output

	return traversalSlice, functionSlice
}

// walkForFunctions recursively walks the AST, looking only for function calls.
func walkForFunctions(expr hclsyntax.Expression, functions map[string]struct{}) {
	// (Implementation from before, kept the same)
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *hclsyntax.FunctionCallExpr:
		functions[e.Name] = struct{}{}
		for _, arg := range e.Args {
			walkForFunctions(arg, functions)
		}
	case *hclsyntax.BinaryOpExpr:
		walkForFunctions(e.LHS, functions)
		walkForFunctions(e.RHS, functions)
	case *hclsyntax.ConditionalExpr:
		walkForFunctions(e.Condition, functions)
		walkForFunctions(e.TrueResult, functions)
		walkForFunctions(e.FalseResult, functions)
	case *hclsyntax.UnaryOpExpr:
		walkForFunctions(e.Val, functions)
	case *hclsyntax.TemplateExpr:
		for _, part := range e.Parts {
			walkForFunctions(part, functions)
		}
	case *hclsyntax.TemplateWrapExpr:
		walkForFunctions(e.Wrapped, functions)
	case *hclsyntax.TupleConsExpr:
		for _, item := range e.Exprs {
			walkForFunctions(item, functions)
		}
	case *hclsyntax.ObjectConsExpr:
		for _, item := range e.Items {
			walkForFunctions(item.KeyExpr, functions)
			walkForFunctions(item.ValueExpr, functions)
		}
	case *hclsyntax.ForExpr:
		walkForFunctions(e.CollExpr, functions)
		walkForFunctions(e.KeyExpr, functions)
		walkForFunctions(e.ValExpr, functions)
		walkForFunctions(e.CondExpr, functions)
	case *hclsyntax.IndexExpr:
		walkForFunctions(e.Collection, functions)
		walkForFunctions(e.Key, functions)
	case *hclsyntax.SplatExpr:
		walkForFunctions(e.Source, functions)
		walkForFunctions(e.Each, functions)
	case *hclsyntax.ParenthesesExpr:
		walkForFunctions(e.Expression, functions)
	}
}

// parseBlock provides a generic way to find and decode a unique HCL block.
// It uses a generic type 'T' which must be a pointer to a struct that
// implements the Expressioner interface.
func ParseBlock[T Expressioner](blocks hcl.Blocks, blockName string) (T, []hcl.Expression, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var zero T // This will be the zero value for the type T (e.g., nil for a pointer)

	block, blockDiags := bggohcl.FindUniqueBlock(blocks, blockName)
	diags = append(diags, blockDiags...)
	if block == nil || blockDiags.HasErrors() {
		return zero, nil, diags
	}

	// We need to create a new instance of the struct that T is a pointer to.
	// E.g., if T is *Timeouts, this creates a new Timeouts{}.
	val := reflect.New(reflect.TypeOf(zero).Elem())
	content := val.Interface().(T)

	decodeDiags := gohcl.DecodeBody(block.Body, nil, content)
	diags = append(diags, decodeDiags...)
	if decodeDiags.HasErrors() {
		return zero, nil, diags
	}

	return content, content.Expressions(), diags
}
