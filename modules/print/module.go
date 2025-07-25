package print

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// Input defines the arguments for the print runner.
// It now expects a native Go map, which gohcl can decode correctly.
type Input struct {
	Value map[string]string `hcl:"input"`
}

// Deps is an empty struct because this runner does not use any resources.
type Deps struct{}

// OnRunPrint is the handler for the 'print' runner's on_run lifecycle event.
func OnRunPrint(ctx context.Context, deps *Deps, input *Input) (cty.Value, error) {
	slog.Info("Printing input")

	if input.Value == nil {
		fmt.Println("      (null)")
		return cty.NilVal, nil
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(input.Value))
	for k := range input.Value {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("      %s = %q\n", k, input.Value[k])
	}

	return cty.NilVal, nil
}

// init registers the handler with the engine.
func init() {
	engine.RegisterHandler("OnRunPrint", &engine.RegisteredHandler{
		NewInput: func() any { return new(Input) },
		NewDeps:  func() any { return new(Deps) },
		Fn:       OnRunPrint,
	})
}
