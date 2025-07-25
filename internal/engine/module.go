package engine

import (
	"fmt"
	"log/slog"
	"reflect"
)

// --- Step/Runner Registries ---

// RegisteredHandler holds the compiled Go parts of a runner's lifecycle function.
type RegisteredHandler struct {
	NewInput func() any // Factory for the 'arguments' struct.
	NewDeps  func() any // Factory for the 'uses' struct.
	Fn       any        // The handler function itself, e.g., OnRun().
}

// HandlerRegistry stores the Go functions for runners, registered by name.
var HandlerRegistry = make(map[string]*RegisteredHandler)

// DefinitionRegistry stores the parsed HCL runner definitions, keyed by runner type.
var DefinitionRegistry = make(map[string]*RunnerDefinition)

// RegisterHandler is called by init() in runner Go files.
func RegisterHandler(name string, handler *RegisteredHandler) {
	if _, exists := HandlerRegistry[name]; exists {
		panic(fmt.Sprintf("runner handler with name '%s' already registered", name))
	}
	slog.Debug("Registering runner handler", "name", name)
	HandlerRegistry[name] = handler
}

// --- Asset/Resource Registries ---

// RegisteredAssetHandler holds the Go functions for an asset's lifecycle.
type RegisteredAssetHandler struct {
	NewInput  func() any // Factory for the 'arguments' struct.
	CreateFn  any        // The Create() handler function.
	DestroyFn any        // The Destroy() handler function.
}

// AssetHandlerRegistry stores the Go functions for assets, registered by name.
var AssetHandlerRegistry = make(map[string]*RegisteredAssetHandler)

// AssetDefinitionRegistry stores the parsed HCL asset definitions.
var AssetDefinitionRegistry = make(map[string]*AssetDefinition)

// AssetInterfaceRegistry maps an asset type string to its Go interface type.
var AssetInterfaceRegistry = make(map[string]reflect.Type)

// RegisterAssetHandler is called by init() in asset Go files.
func RegisterAssetHandler(name string, handler *RegisteredAssetHandler) {
	if _, exists := AssetHandlerRegistry[name]; exists {
		panic(fmt.Sprintf("asset handler with name '%s' already registered", name))
	}
	slog.Debug("Registering asset handler", "name", name)
	AssetHandlerRegistry[name] = handler
}

// RegisterAssetInterface registers the Go interface contract for an asset type.
func RegisterAssetInterface(assetType string, iface reflect.Type) {
	if _, exists := AssetInterfaceRegistry[assetType]; exists {
		panic(fmt.Sprintf("interface for asset type '%s' already registered", assetType))
	}
	slog.Debug("Registering asset interface", "assetType", assetType, "interface", iface.String())
	AssetInterfaceRegistry[assetType] = iface
}

// --- Discovery ---

// DiscoverModules scans a directory for HCL manifest files and loads them.
func DiscoverModules(dirPath string) error {
	hclFiles, err := ResolveGridPath(dirPath)
	if err != nil {
		return fmt.Errorf("error finding module definitions in %s: %w", dirPath, err)
	}

	for _, file := range hclFiles {
		defConfig, err := DecodeDefinitionFile(file)
		if err != nil {
			slog.Warn("Failed to decode module definition", "path", file, "error", err)
			continue
		}
		// Register Runner definitions
		if defConfig.Runner != nil {
			runnerType := defConfig.Runner.Type
			if _, exists := DefinitionRegistry[runnerType]; exists {
				slog.Warn("Duplicate runner definition found, overwriting", "type", runnerType, "path", file)
			}
			slog.Debug("Discovered runner definition", "type", runnerType, "path", file)
			DefinitionRegistry[runnerType] = defConfig.Runner
		}
		// Register Asset definitions
		if defConfig.Asset != nil {
			assetType := defConfig.Asset.Type
			if _, exists := AssetDefinitionRegistry[assetType]; exists {
				slog.Warn("Duplicate asset definition found, overwriting", "type", assetType, "path", file)
			}
			slog.Debug("Discovered asset definition", "type", assetType, "path", file)
			AssetDefinitionRegistry[assetType] = defConfig.Asset
		}
	}
	return nil
}
