package socketio

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/zclconf/go-cty/cty"
	"github.com/zishang520/engine.io-client-go/transports"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io-client-go/socket"
)

// Module implements the registry.Module interface for this package.
type Module struct{}

// Input defines the arguments for the socketio runner.
type Input struct {
	URL                string    `hcl:"url"`
	Namespace          string    `hcl:"namespace,optional"`
	OnEvent            string    `hcl:"on_event"`
	EmitEvent          string    `hcl:"emit_event,optional"`
	EmitData           cty.Value `hcl:"emit_data,optional"`
	Timeout            string    `hcl:"timeout,optional"`
	InsecureSkipVerify bool      `hcl:"insecure_skip_verify,optional"`
}

// Deps is an empty struct because this runner does not use any resources.
type Deps struct{}

// opResult is a private struct to safely pass results through the done channel.
type opResult struct {
	value cty.Value
	err   error
}

// OnRunSocketIO is the handler for the 'socketio' runner's on_run lifecycle event.
func OnRunSocketIO(ctx context.Context, deps *Deps, input *Input) (cty.Value, error) {
	logger := ctxlog.FromContext(ctx).With("runner", "socketio", "url", input.URL, "onEvent", input.OnEvent, "emitEvent", input.EmitEvent)
	logger.Debug("Handler started")
	defer logger.Debug("Handler finished")

	var isConnected atomic.Bool

	timeout, err := time.ParseDuration(input.Timeout)
	if err != nil {
		logger.Warn("Failed to parse timeout, using default 10s", "inputTimeout", input.Timeout, "error", err)
		timeout = 10 * time.Second
	}
	logger.Debug("Operation timeout configured", "timeout", timeout)

	done := make(chan opResult, 1)
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	parsedURL, err := url.Parse(input.URL)
	if err != nil {
		logger.Error("Failed to parse URL", "error", err)
		return cty.NilVal, fmt.Errorf("failed to parse URL: %w", err)
	}
	logger.Debug("URL parsed successfully", "scheme", parsedURL.Scheme, "host", parsedURL.Host, "path", parsedURL.Path)

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
	opts := socket.DefaultOptions()
	opts.SetPath(parsedURL.Path)
	if input.InsecureSkipVerify {
		logger.Warn("Skipping TLS certificate verification")
		opts.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	opts.SetTransports(types.NewSet(transports.WebSocket))
	logger.Debug("Socket options configured", "transports", "WebSocket")

	logger.Debug("Creating new Socket.IO manager", "baseURL", baseURL, "path", parsedURL.Path)
	manager := socket.NewManager(baseURL, opts)
	io := manager.Socket(input.Namespace, opts)
	defer func() {
		logger.Debug("Disconnecting socket client")
		io.Disconnect()
	}()

	// --- Event Listeners ---
	logger.Debug("Setting up event listeners")
	io.OnAny(func(args ...any) {
		if len(args) == 0 {
			return
		}
		eventName, ok := args[0].(string)
		if !ok {
			logger.Warn("OnAny listener received event where name was not a string")
			return
		}
		var eventData []any
		if len(args) > 1 {
			eventData = args[1:]
		}
		jsonData, _ := json.Marshal(eventData)
		logger.Debug("EVENT RECEIVED", "event", eventName, "data", string(jsonData))
	})

	io.On(types.EventName("connect"), func(...any) {
		logger.Debug("EVENT HANDLER: 'connect' event fired")
		isConnected.Store(true)
		logger.Info("Successfully connected", "namespace", input.Namespace, "sid", io.Id())
		if input.EmitEvent != "" {
			data, err := ctyValueToInterface(input.EmitData)
			if err != nil {
				logger.Error("Failed to convert emit_data to interface", "error", err)
				done <- opResult{err: err}
				return
			}
			jsonData, _ := json.Marshal(data)
			logger.Info("Emitting event", "event", input.EmitEvent, "data", string(jsonData))
			io.Emit(input.EmitEvent, data)
		}
	})

	io.On(types.EventName("connect_error"), func(errs ...any) {
		err := errs[0].(error)
		logger.Debug("EVENT HANDLER: 'connect_error' event fired", "error", err)
		logger.Error("Connection error received", "error", err)
		done <- opResult{err: err}
	})

	io.On(types.EventName(input.OnEvent), func(data ...any) {
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
			logger.Debug("Success event received with no data payload")
			responseData = cty.NullVal(cty.DynamicPseudoType)
		}

		outputObject := cty.ObjectVal(map[string]cty.Value{"response_data": responseData})
		logger.Debug("Successfully processed success event, sending result to channel")
		done <- opResult{value: outputObject}
	})

	// --- Execution Block ---
	logger.Debug("Initiating connection...")
	io.Connect()
	logger.Debug("io.Connect() called, now entering select block to wait for event...")

	select {
	case <-opCtx.Done():
		var errMsg string
		if isConnected.Load() {
			errMsg = fmt.Sprintf("timed out after connecting while waiting for event '%s'", input.OnEvent)
		} else {
			errMsg = "timed out while waiting for initial connection"
		}
		logger.Error("SELECT CASE: Operation context finished", "reason", opCtx.Err(), "detail", errMsg)
		return cty.NilVal, fmt.Errorf("%s", errMsg)
	case res := <-done:
		logger.Debug("SELECT CASE: Result received from 'done' channel")
		if res.err != nil {
			logger.Error("Runner failed with an event-driven error", "error", res.err)
			return cty.NilVal, res.err
		}
		logger.Debug("Runner succeeded")
		return res.value, nil
	}
}

// Register registers the handler with the engine.
func (m *Module) Register(r *registry.Registry) {
	r.RegisterHandler("OnRunSocketIO", &registry.RegisteredHandler{
		NewInput: func() any { return new(Input) },
		NewDeps:  func() any { return new(Deps) },
		Fn:       OnRunSocketIO,
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
