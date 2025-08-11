package testutil

import "github.com/specialistvlad/burstgridgo/internal/registry"

// SimpleModule is a test helper for easily creating a mock module that
// registers a single runner or asset handler.
type SimpleModule struct {
	RunnerName string
	Runner     *registry.RegisteredRunner

	AssetName string
	Asset     *registry.RegisteredAsset
}

// Register implements the registry.Module interface.
func (m *SimpleModule) Register(r *registry.Registry) {
	if m.RunnerName != "" && m.Runner != nil {
		r.RegisterRunner(m.RunnerName, m.Runner)
	}
	if m.AssetName != "" && m.Asset != nil {
		r.RegisterAssetHandler(m.AssetName, m.Asset)
	}
}
