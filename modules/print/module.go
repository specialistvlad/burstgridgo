package print

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

type PrintRunner struct{}

// Config defines the HCL structure for this module.
type Config struct {
	Input cty.Value `hcl:"input"`
}

func (r *PrintRunner) Run(mod engine.Module, ctx *hcl.EvalContext) (cty.Value, error) {
	log.Printf("    ⚙️  Executing print runner for module '%s'...", mod.Name)

	var config Config
	// Use the provided context to decode the body, resolving the input expression.
	if diags := gohcl.DecodeBody(mod.Body, ctx, &config); diags.HasErrors() {
		return cty.NullVal(cty.DynamicPseudoType), diags
	}

	log.Printf("    🖨️  Printing input for module '%s':", mod.Name)

	// Iterate over the map and print key-value pairs
	if config.Input.Type().IsMapType() || config.Input.Type().IsObjectType() {
		it := config.Input.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			vStr := v.AsString() // Corrected line
			fmt.Printf("      %s = %s\n", k.AsString(), vStr)
		}
	} else {
		fmt.Printf("      %s\n", config.Input.GoString())
	}

	// This module produces no output for other modules to consume.
	return cty.NullVal(cty.DynamicPseudoType), nil
}

func init() {
	engine.Registry["print"] = &PrintRunner{}
	log.Println("🔌 print runner registered.")
}
