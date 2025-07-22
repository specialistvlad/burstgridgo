// File: modules/socketio/module.go

package socketio

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
	"github.com/zishang520/engine.io-client-go/transports"
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

// ctyValueToInterface recursively converts a cty.Value to a standard Go interface{} for emitting.
func ctyValueToInterface(val cty.Value) (interface{}, error) {
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
		out := make(map[string]interface{})
		for it := val.ElementIterator(); it.Next(); {
			k, v := it.Element()
			keyStr := k.AsString()
			valInterface, err := ctyValueToInterface(v)
			if err != nil {
				return nil, err
			}
			out[keyStr] = valInterface
		}
		return out, nil
	}

	if val.Type().IsTupleType() || val.Type().IsListType() {
		var out []interface{}
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

// interfaceToCtyValue converts a generic Go interface{} from the socket library into a cty.Value.
func interfaceToCtyValue(data interface{}) (cty.Value, error) {
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
	case map[string]interface{}:
		attrs := make(map[string]cty.Value)
		for key, val := range v {
			ctyVal, err := interfaceToCtyValue(val)
			if err != nil {
				return cty.NilVal, err
			}
			attrs[key] = ctyVal
		}
		return cty.ObjectVal(attrs), nil
	case []interface{}:
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

	done := make(chan interface{}, 1)

	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to parse URL: %w", err)
	}

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
	log.Printf("    dialing %s...", baseURL)

	opts := socket.DefaultOptions()
	opts.SetPath(parsedURL.Path)
	if config.InsecureSkipVerify {
		log.Printf("        ⚠️  Skipping TLS certificate verification for module '%s'", mod.Name)
		opts.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	opts.SetTransports(types.NewSet(transports.WebSocket))

	manager := socket.NewManager(baseURL, opts)
	namespace := "/"
	if config.Namespace != "" {
		namespace = config.Namespace
	}
	io := manager.Socket(namespace, opts)

	io.On(types.EventName("connect"), func(...any) {
		log.Printf("    🔌 Successfully connected to %s (namespace: %s, sid: %s)", config.URL, namespace, io.Id())
		if config.EmitEvent != "" {
			data, err := ctyValueToInterface(config.EmitData)
			if err != nil {
				done <- err
				return
			}
			log.Printf("    ➡️  Emitting event '%s' with data: %+v", config.EmitEvent, data)
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

	io.On(types.EventName(config.OnEvent), func(data ...any) {
		log.Printf("    ⬅️  Received SUCCESS event '%s' with data: %v", config.OnEvent, data)
		// Convert the generic response data into a cty.Value.
		if len(data) > 0 {
			ctyVal, err := interfaceToCtyValue(data)
			if err != nil {
				done <- err
				return
			}
			done <- ctyVal // Send the entire converted value on success.
			return
		}
		done <- fmt.Errorf("success event received with no data")
	})

	select {
	case result := <-done:
		log.Printf("    🔚 Event loop finished for '%s', disconnecting...", mod.Name)
		io.Disconnect()
		switch res := result.(type) {
		case cty.Value: // This is our new success case.
			// Return the result wrapped in an object with the key 'response_data'.
			return cty.ObjectVal(map[string]cty.Value{
				"response_data": res,
			}), nil
		case error: // This is a failure case.
			return cty.NullVal(cty.DynamicPseudoType), res
		default:
			return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("unexpected result type from event handler")
		}
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
