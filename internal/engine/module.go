package engine

import (
	"fmt"
	"log/slog"
)

// RegisteredHandler holds the compiled Go parts of a runner's lifecycle function.
type RegisteredHandler struct {
	// NewInput is a factory function that returns a new, zero-value instance
	// of the handler's specific input struct (e.g., new(MyInput)).
	NewInput func() any

	// NewState is a factory for the handler's state object. Can be nil.
	NewState func() any

	// Fn is the handler function itself.
	Fn any
}

// HandlerRegistry stores the Go functions registered via init() by name.
var HandlerRegistry = make(map[string]*RegisteredHandler)

// DefinitionRegistry stores the parsed HCL runner definitions, keyed by runner type.
var DefinitionRegistry = make(map[string]*RunnerDefinition)

// RegisterHandler is called by init() functions in runner Go files to make
// their handlers available to the engine.
func RegisterHandler(name string, handler *RegisteredHandler) {
	if _, exists := HandlerRegistry[name]; exists {
		// Panic on startup if there's a duplicate registration, as it's a developer error.
		panic(fmt.Sprintf("handler with name '%s' already registered", name))
	}
	slog.Debug("Registering handler", "name", name)
	HandlerRegistry[name] = handler
}

// DiscoverRunners scans a directory for HCL manifest files and loads them
// into the DefinitionRegistry.
func DiscoverRunners(dirPath string) error {
	// We use the existing traverser to find all HCL files recursively.
	hclFiles, err := ResolveGridPath(dirPath)
	if err != nil {
		return fmt.Errorf("error finding runner definitions in %s: %w", dirPath, err)
	}

	for _, file := range hclFiles {
		defConfig, err := DecodeDefinitionFile(file)
		if err != nil {
			slog.Warn("Failed to decode runner definition", "path", file, "error", err)
			continue
		}
		if defConfig != nil && defConfig.Runner != nil {
			runnerType := defConfig.Runner.Type
			if _, exists := DefinitionRegistry[runnerType]; exists {
				slog.Warn("Duplicate runner definition found, overwriting", "type", runnerType, "path", file)
			}
			slog.Debug("Discovered runner definition", "type", runnerType, "path", file)
			DefinitionRegistry[runnerType] = defConfig.Runner
		}
	}
	return nil
}
