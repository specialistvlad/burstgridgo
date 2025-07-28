package engine

import (
	"context"
	"fmt"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
)

// LoadGridConfig finds, parses, and merges one or more HCL grid files from a
// given path into a single, unified GridConfig object.
func LoadGridConfig(ctx context.Context, gridPath string) (*schema.GridConfig, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("LoadGridConfig started.", "path", gridPath)
	// This will hold the combined configuration from all parsed files.
	gridConfig := &schema.GridConfig{}

	hclFiles, err := ResolveGridPath(ctx, gridPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve grid path '%s': %w", gridPath, err)
	}

	if len(hclFiles) == 0 {
		logger.Warn("No .hcl files found at the specified path.", "path", gridPath)
		return gridConfig, nil
	}

	logger.Info("Found HCL files to process.", "count", len(hclFiles), "path", gridPath)
	for _, file := range hclFiles {
		logger.Debug("Resolved HCL file.", "path", file)
	}

	// Parse all files and merge them into a single config.
	for _, file := range hclFiles {
		cfg, err := DecodeGridFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to load grid file '%s': %w", file, err)
		}
		gridConfig.Resources = append(gridConfig.Resources, cfg.Resources...)
		gridConfig.Steps = append(gridConfig.Steps, cfg.Steps...)
	}

	logger.Debug("Finished loading and merging grid files.", "total_steps", len(gridConfig.Steps), "total_resources", len(gridConfig.Resources))
	return gridConfig, nil
}

// DiscoverModules scans a directory for HCL manifest files (*.hcl), decodes
// them, and populates the definition registries.
func DiscoverModules(ctx context.Context, dirPath string, r *registry.Registry) error {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Starting module discovery.", "path", dirPath)
	hclFiles, err := ResolveGridPath(ctx, dirPath)
	if err != nil {
		return fmt.Errorf("error finding module definitions in %s: %w", dirPath, err)
	}

	for _, file := range hclFiles {
		defConfig, err := DecodeDefinitionFile(ctx, file)
		if err != nil {
			logger.Warn("Failed to decode module definition, skipping.", "path", file, "error", err)
			continue
		}
		if defConfig.Runner != nil {
			runnerType := defConfig.Runner.Type
			if _, exists := r.DefinitionRegistry[runnerType]; exists {
				logger.Warn("Duplicate runner definition found, overwriting.", "type", runnerType, "path", file)
			}
			logger.Debug("Discovered runner definition.", "type", runnerType, "path", file)
			r.DefinitionRegistry[runnerType] = defConfig.Runner
		}
		if defConfig.Asset != nil {
			assetType := defConfig.Asset.Type
			if _, exists := r.AssetDefinitionRegistry[assetType]; exists {
				logger.Warn("Duplicate asset definition found, overwriting.", "type", assetType, "path", file)
			}
			logger.Debug("Discovered asset definition.", "type", assetType, "path", file)
			r.AssetDefinitionRegistry[assetType] = defConfig.Asset
		}
	}
	logger.Debug("Module discovery finished.")
	return nil
}
