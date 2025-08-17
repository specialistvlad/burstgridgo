package testutil

import "github.com/specialistvlad/burstgridgo/internal/handlers"

// SimpleModule is a test helper for easily creating a mock module that
// registers a single runner or asset handler.
type SimpleModule struct {
	RunnerName string
	Runner     *handlers.RegisteredHandler

	AssetName string
	Asset     *handlers.RegisteredAsset
}

// Register implements the registry.Module interface.
func (m *SimpleModule) Register(r *handlers.Handlers) {
	if m.RunnerName != "" && m.Runner != nil {
		r.RegisterRunner(m.RunnerName, m.Runner)
	}
	if m.AssetName != "" && m.Asset != nil {
		r.RegisterAssetHandler(m.AssetName, m.Asset)
	}
}
