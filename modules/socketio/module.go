// File: modules/socketio/module.go

package socketio

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
	engineio "github.com/zishang520/engine.io-client-go/engine"
	"github.com/zishang520/engine.io-client-go/transports"
	"github.com/zishang520/engine.io/v2/events"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io-client-go/socket"
)

type SocketIoRunner struct{}

// Config defines the HCL structure for this module.
type Config struct {
	URL                string    `hcl:"url"`
	Namespace          string    `hcl:"namespace,optional"`
	OnEvent            string    `hcl:"on_event"`
	EmitEvent          string    `hcl:"emit_event,optional"`
	EmitData           cty.Value `hcl:"emit_data,optional"`
	Timeout            string    `hcl:"timeout,optional"`
	InsecureSkipVerify bool      `hcl:"insecure_skip_verify,optional"`
}

// Helper to convert HCL cty.Value to a format the library can use.
func convertCtyToInterface(val cty.Value) (interface{}, error) {
	if !val.IsKnown() || val.IsNull() {
		return nil, nil
	}
	switch val.Type() {
	case cty.String:
		return types.NewStringBufferString(val.AsString()), nil
	case cty.Number:
		f, _ := val.AsBigFloat().Float64()
		return f, nil
	case cty.Bool:
		return val.True(), nil
	default:
		return nil, fmt.Errorf("unsupported type for emit_data: %s", val.Type().FriendlyName())
	}
}

func (r *SocketIoRunner) Run(mod engine.Module, ctx *hcl.EvalContext) (cty.Value, error) {
	log.Printf("    ⚙️  Executing socketio runner for module '%s'...", mod.Name)

	var config Config
	if diags := gohcl.DecodeBody(mod.Body, ctx, &config); diags.HasErrors() {
		return cty.NullVal(cty.DynamicPseudoType), diags
	}
	log.Printf("    📄 Decoded config for '%s': URL=%s, OnEvent=%s, EmitEvent=%s", mod.Name, config.URL, config.OnEvent, config.EmitEvent)

	timeout, err := time.ParseDuration(config.Timeout)
	if err != nil {
		timeout = 10 * time.Second
		log.Printf("    ⏱️ Timeout not specified for '%s', using default: %s", mod.Name, timeout)
	} else {
		log.Printf("    ⏱️ Timeout configured for '%s': %s", mod.Name, timeout)
	}

	done := make(chan error, 1)

	// --- DEFINITIVE OPTIONS CREATION ---
	opts := socket.DefaultOptions()
	// opts.SetPath("/event-bridge/socket/socket.io/")

	engineOpts := engineio.DefaultSocketOptions()
	engineOpts.SetTransports(types.NewSet(transports.Polling, transports.WebSocket))

	if config.InsecureSkipVerify {
		log.Printf("      ⚠️  Skipping TLS certificate verification for module '%s'", mod.Name)

		// The correct method is SetTLSClientConfig, with "Client" in the name.
		engineOpts.SetTLSClientConfig(&tls.Config{
			InsecureSkipVerify: true,
		})
	}

	log.Printf("     dialing %s...", config.URL)
	manager := socket.NewManager(config.URL, opts)
	namespace := "/"
	if config.Namespace != "" {
		namespace = config.Namespace
	}
	io := manager.Socket(namespace, opts)

	// --- Define Event Handlers ---
	io.On(types.EventName("connect"), func(...any) {
		log.Printf("    🔌 Successfully connected to %s (namespace: %s, sid: %s)", config.URL, namespace, io.Id())
		if config.EmitEvent != "" {
			log.Printf("    ➡️  Emitting event '%s'", config.EmitEvent)
			data, err := convertCtyToInterface(config.EmitData)
			if err != nil {
				done <- err
				return
			}
			io.Emit(config.EmitEvent, data)
		}
	})

	io.On(types.EventName("connect_error"), func(errs ...any) {
		err := errs[0].(error)
		log.Printf("    ❗️ Connection error for module '%s': %v", mod.Name, err)
		done <- err
	})

	io.On(types.EventName("disconnect"), func(reason ...any) {
		log.Printf("    🔌 Disconnected from %s, reason: %v", config.URL, reason)
	})

	io.OnAny(events.Listener(func(args ...any) {
		if len(args) > 0 {
			event, ok := args[0].(types.EventName)
			if ok {
				log.Printf("    📡 Received generic event: '%s' with data: %v", event, args[1:])
			}
		}
	}))

	io.On(types.EventName(config.OnEvent), func(data ...any) {
		log.Printf("    ⬅️  Received SUCCESS event '%s' with data: %v", config.OnEvent, data)
		done <- nil
	})

	// --- Wait for Completion or Timeout ---
	select {
	case err := <-done:
		log.Printf("    🔚 Event loop finished for '%s', disconnecting...", mod.Name)
		io.Disconnect()
		if err != nil {
			return cty.NullVal(cty.DynamicPseudoType), err
		}
		return cty.NullVal(cty.DynamicPseudoType), nil
	case <-time.After(timeout):
		log.Printf("    ⌛️ Timed out waiting for event '%s' on module '%s'", config.OnEvent, mod.Name)
		io.Disconnect()
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("timed out waiting for event '%s' on module '%s'", config.OnEvent, mod.Name)
	}
}

func init() {
	engine.Registry["socketio"] = &SocketIoRunner{}
	log.Println("🔌 socketio runner registered.")
}
