package socketio_client

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"time"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/zishang520/engine.io-client-go/transports"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io-client-go/socket"
)

// Module implements the registry.Module interface for this package.
type Module struct{}

// Input defines the arguments for creating a socketio_client resource.
type Input struct {
	URL                string `hcl:"url"`
	Namespace          string `hcl:"namespace,optional"`
	InsecureSkipVerify bool   `hcl:"insecure_skip_verify,optional"`
}

// CreateSocketIOClient is the 'create' handler for the asset.
func CreateSocketIOClient(ctx context.Context, input *Input) (*socket.Socket, error) {
	logger := ctxlog.FromContext(ctx).With("asset", "socketio_client", "url", input.URL)
	logger.Info("Creating new client instance...")

	parsedURL, err := url.Parse(input.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	opts := socket.DefaultOptions()
	opts.SetPath(parsedURL.Path)
	if input.InsecureSkipVerify {
		logger.Warn("Skipping TLS certificate verification")
		opts.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	opts.SetTransports(types.NewSet(transports.WebSocket))

	connectChan := make(chan error, 1)

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
	manager := socket.NewManager(baseURL, opts)
	io := manager.Socket(input.Namespace, opts)

	io.Once(types.EventName("connect"), func(...any) {
		logger.Debug("EVENT HANDLER: 'connect' event fired")
		logger.Info("Successfully connected", "sid", io.Id())
		connectChan <- nil
	})

	io.Once(types.EventName("connect_error"), func(errs ...any) {
		err := errs[0].(error)
		logger.Debug("EVENT HANDLER: 'connect_error' event fired", "error", err)
		connectChan <- err
	})

	logger.Debug("Initiating connection...")
	io.Connect()
	logger.Debug("io.Connect() called, now entering select block to wait for event...")

	select {
	case err := <-connectChan:
		if err != nil {
			io.Disconnect()
			return nil, fmt.Errorf("socket.io connection failed: %w", err)
		}
		// Connection succeeded, return the persistent client.
		return io, nil
	case <-ctx.Done():
		io.Disconnect()
		return nil, fmt.Errorf("context cancelled while waiting for socket.io connection")
	case <-time.After(15 * time.Second): // Generous timeout for connection
		io.Disconnect()
		return nil, fmt.Errorf("timed out after 15s waiting for socket.io connection")
	}
}

// DestroySocketIOClient is the 'destroy' handler.
func DestroySocketIOClient(client *socket.Socket) error {
	slog.Info("Destroying socket.io client instance", "sid", client.Id())
	client.Disconnect()
	return nil
}

// Register registers the asset handlers and interface with the engine.
func (m *Module) Register(r *registry.Registry) {
	r.RegisterAssetHandler("CreateSocketIOClient", &registry.RegisteredAssetHandler{
		NewInput: func() any { return new(Input) },
		CreateFn: CreateSocketIOClient,
	})
	r.RegisterAssetHandler("DestroySocketIOClient", &registry.RegisteredAssetHandler{
		DestroyFn: DestroySocketIOClient,
	})
	// Revert to registering the raw socket client type.
	r.RegisterAssetInterface("socketio_client", reflect.TypeOf((*socket.Socket)(nil)))
}
