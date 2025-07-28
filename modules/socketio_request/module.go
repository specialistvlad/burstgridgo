package socketio_request

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/zclconf/go-cty/cty"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io-client-go/socket"
)

// Module implements the registry.Module interface for this package.
type Module struct{}

// Input defines the arguments for the runner.
type Input struct {
	OnEvent   string    `hcl:"on_event"`
	EmitEvent string    `hcl:"emit_event"`
	EmitData  cty.Value `hcl:"emit_data,optional"`
	Timeout   string    `hcl:"timeout,optional"`
}

// Deps defines the injected resources from the 'uses' block.
type Deps struct {
	// Revert to expecting the raw socket client.
	Client *socket.Socket `hcl:"client"`
}

type opResult struct {
	value cty.Value
	err   error
}

// OnRunSocketIORequest is the handler for the runner.
func OnRunSocketIORequest(ctx context.Context, deps *Deps, input *Input) (cty.Value, error) {
	logger := ctxlog.FromContext(ctx)

	if deps.Client == nil {
		logger.Error("Socket.io client dependency was not injected")
		return cty.NilVal, fmt.Errorf("socket.io client dependency was not injected")
	}
	if !deps.Client.Connected() {
		logger.Error("Injected socket.io client is not connected", "sid", deps.Client.Id())
		return cty.NilVal, fmt.Errorf("injected socket.io client is not connected")
	}

	logger = logger.With("runner", "socketio_request", "sid", deps.Client.Id())
	logger.Info("Executing request", "emitEvent", input.EmitEvent, "onEvent", input.OnEvent)

	timeout, err := time.ParseDuration(input.Timeout)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to parse timeout: %w", err)
	}

	done := make(chan opResult, 1)
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	deps.Client.Once(types.EventName(input.OnEvent), func(data ...any) {
		logger.Debug("EVENT HANDLER: Success event received", "event", input.OnEvent)
		var responseData cty.Value
		if len(data) > 0 {
			ctyVal, err := interfaceToCtyValue(data[0])
			if err != nil {
				logger.Error("Failed to convert received data to cty.Value", "error", err)
				done <- opResult{err: err}
				return
			}
			responseData = ctyVal
		} else {
			responseData = cty.NullVal(cty.DynamicPseudoType)
		}
		outputObject := cty.ObjectVal(map[string]cty.Value{"response_data": responseData})
		done <- opResult{value: outputObject}
	})

	data, err := ctyValueToInterface(input.EmitData)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to convert emit_data to interface: %w", err)
	}
	jsonData, _ := json.Marshal(data)
	logger.Debug("Emitting event", "event", input.EmitEvent, "data", string(jsonData))
	deps.Client.Emit(input.EmitEvent, data)

	select {
	case <-opCtx.Done():
		return cty.NilVal, fmt.Errorf("timed out after %v waiting for event '%s'", timeout, input.OnEvent)
	case res := <-done:
		if res.err != nil {
			return cty.NilVal, res.err
		}
		logger.Info("Successfully received response event", "event", input.OnEvent)
		return res.value, nil
	}
}

// Register registers the handler with the engine.
func (m *Module) Register(r *registry.Registry) {
	r.RegisterHandler("OnRunSocketIORequest", &registry.RegisteredHandler{
		NewInput: func() any { return new(Input) },
		NewDeps:  func() any { return new(Deps) },
		Fn:       OnRunSocketIORequest,
	})
}

// ctyValueToInterface converts a cty.Value to a Go interface{}.
func ctyValueToInterface(val cty.Value) (any, error) {
	if !val.IsKnown() || val.IsNull() {
		return nil, nil
	}
	if val.Type().IsPrimitiveType() {
		switch val.Type() {
		case cty.String:
			return val.AsString(), nil
		case cty.Number:
			f, _ := val.AsBigFloat().Float64()
			return f, nil
		case cty.Bool:
			return val.True(), nil
		default:
			return nil, fmt.Errorf("unsupported primitive type: %s", val.Type().FriendlyName())
		}
	}
	if val.Type().IsObjectType() || val.Type().IsMapType() {
		out := make(map[string]any)
		for it := val.ElementIterator(); it.Next(); {
			k, v := it.Element()
			valInterface, err := ctyValueToInterface(v)
			if err != nil {
				return nil, err
			}
			out[k.AsString()] = valInterface
		}
		return out, nil
	}
	if val.Type().IsTupleType() || val.Type().IsListType() {
		var out []any
		for it := val.ElementIterator(); it.Next(); {
			_, v := it.Element()
			valInterface, err := ctyValueToInterface(v)
			if err != nil {
				return nil, err
			}
			out = append(out, valInterface)
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported cty.Type for conversion: %s", val.Type().FriendlyName())
}

// interfaceToCtyValue converts a Go interface{} to a cty.Value.
func interfaceToCtyValue(data any) (cty.Value, error) {
	if data == nil {
		return cty.NullVal(cty.DynamicPseudoType), nil
	}
	switch v := data.(type) {
	case string:
		return cty.StringVal(v), nil
	case float64:
		return cty.NumberFloatVal(v), nil
	case bool:
		return cty.BoolVal(v), nil
	case map[string]any:
		attrs := make(map[string]cty.Value)
		for key, val := range v {
			ctyVal, err := interfaceToCtyValue(val)
			if err != nil {
				return cty.NilVal, err
			}
			attrs[key] = ctyVal
		}
		return cty.ObjectVal(attrs), nil
	case []any:
		elems := make([]cty.Value, 0, len(v))
		for _, val := range v {
			ctyVal, err := interfaceToCtyValue(val)
			if err != nil {
				return cty.NilVal, err
			}
			elems = append(elems, ctyVal)
		}
		return cty.TupleVal(elems), nil
	default:
		return cty.NilVal, fmt.Errorf("unsupported type for conversion to cty.Value: %T", v)
	}
}
