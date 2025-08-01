package socketio

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/zishang520/engine.io-client-go/transports"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io-client-go/socket"
)

// Module implements the registry.Module interface for this package.
type Module struct{}

// Input defines the arguments for the socketio runner.
type Input struct {
	URL                string         `bggo:"url"`
	Namespace          string         `bggo:"namespace"`
	OnEvent            string         `bggo:"on_event"`
	EmitEvent          string         `bggo:"emit_event"`
	EmitData           map[string]any `bggo:"emit_data"`
	Timeout            string         `bggo:"timeout"`
	InsecureSkipVerify bool           `bggo:"insecure_skip_verify"`
}

// Output defines the data structure returned by the runner.
type Output struct {
	ResponseData any `cty:"response_data"`
}

// Deps is an empty struct because this runner does not use any resources.
type Deps struct{}

// opResult is a private struct to safely pass results through the done channel.
type opResult struct {
	value *Output
	err   error
}

// OnRunSocketIO is the handler for the 'socketio' runner's on_run lifecycle event.
func OnRunSocketIO(ctx context.Context, deps *Deps, input *Input) (*Output, error) {
	logger := ctxlog.FromContext(ctx).With("runner", "socketio", "url", input.URL, "onEvent", input.OnEvent, "emitEvent", input.EmitEvent)
	logger.Debug("Handler started")
	defer logger.Debug("Handler finished")

	var isConnected atomic.Bool

	timeout, err := time.ParseDuration(input.Timeout)
	if err != nil {
		logger.Warn("Failed to parse timeout, using default 10s", "inputTimeout", input.Timeout, "error", err)
		timeout = 10 * time.Second
	}

	done := make(chan opResult, 1)
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	parsedURL, err := url.Parse(input.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
	opts := socket.DefaultOptions()
	opts.SetPath(parsedURL.Path)

	if input.InsecureSkipVerify {
		logger.Warn("Skipping TLS certificate verification")
		opts.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	opts.SetTransports(types.NewSet(transports.WebSocket))

	manager := socket.NewManager(baseURL, opts)
	io := manager.Socket(input.Namespace, opts)
	defer func() {
		logger.Debug("Disconnecting socket client")
		io.Disconnect()
	}()

	// --- Event Listeners ---
	io.On(types.EventName("connect"), func(...any) {
		isConnected.Store(true)
		logger.Info("Successfully connected", "namespace", input.Namespace, "sid", io.Id())
		if input.EmitEvent != "" {
			jsonData, _ := json.Marshal(input.EmitData)
			logger.Info("Emitting event", "event", input.EmitEvent, "data", string(jsonData))
			io.Emit(input.EmitEvent, input.EmitData)
		}
	})

	io.On(types.EventName("connect_error"), func(errs ...any) {
		done <- opResult{err: errs[0].(error)}
	})

	io.On(types.EventName(input.OnEvent), func(data ...any) {
		var responseData any
		if len(data) > 0 {
			responseData = data[0]
		}
		done <- opResult{value: &Output{ResponseData: responseData}}
	})

	// --- Execution Block ---
	io.Connect()

	select {
	case <-opCtx.Done():
		var errMsg string
		if isConnected.Load() {
			errMsg = fmt.Sprintf("timed out after connecting while waiting for event '%s'", input.OnEvent)
		} else {
			errMsg = "timed out while waiting for initial connection"
		}
		return nil, fmt.Errorf("%s", errMsg)
	case res := <-done:
		return res.value, res.err
	}
}

// Register registers the handler with the engine.
func (m *Module) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunSocketIO", &registry.RegisteredRunner{
		NewInput:  func() any { return new(Input) },
		InputType: reflect.TypeOf(Input{}),
		NewDeps:   func() any { return new(Deps) },
		Fn:        OnRunSocketIO,
	})
}
