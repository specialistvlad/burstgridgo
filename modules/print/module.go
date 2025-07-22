package print

import (
	"fmt"
	"log/slog"

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
	var config Config
	if diags := gohcl.DecodeBody(mod.Body, ctx, &config); diags.HasErrors() {
		return cty.NullVal(cty.DynamicPseudoType), diags
	}

	slog.Info("Printing input", "module", mod.Name)

	if config.Input.Type().IsMapType() || config.Input.Type().IsObjectType() {
		it := config.Input.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			fmt.Printf("      %s = %s\n", k.AsString(), v.AsString())
		}
	} else {
		fmt.Printf("      %s\n", config.Input.GoString())
	}

	return cty.NullVal(cty.DynamicPseudoType), nil
}

func init() {
	engine.Registry["print"] = &PrintRunner{}
	slog.Debug("Runner registered", "runner", "print")
}
