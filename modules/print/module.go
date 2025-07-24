package print

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// Input defines the arguments for the print runner.
// Since the HCL input type is 'any', we accept it as a raw cty.Value.
type Input struct {
	Value cty.Value `hcl:"input"`
}

// OnRunPrint is the handler for the 'print' runner's on_run lifecycle event.
func OnRunPrint(ctx context.Context, input *Input) (any, error) {
	slog.Info("Printing input")

	if input.Value.Type().IsMapType() || input.Value.Type().IsObjectType() {
		it := input.Value.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			// This is a simplified print for now. A more robust version
			// would handle nested structures and different types.
			fmt.Printf("      %s = %s\n", k.AsString(), v.AsString())
		}
	} else {
		fmt.Printf("      %v\n", input.Value.GoString())
	}

	return nil, nil
}

// init registers the handler with the engine.
func init() {
	engine.RegisterHandler("OnRunPrint", &engine.RegisteredHandler{
		NewInput: func() any { return new(Input) },
		Fn:       OnRunPrint,
	})
}
