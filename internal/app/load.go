package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
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
		cfg, err := LoadGridFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to load grid file '%s': %w", file, err)
		}
		gridConfig.Resources = append(gridConfig.Resources, cfg.Resources...)
		gridConfig.Steps = append(gridConfig.Steps, cfg.Steps...)
	}

	logger.Debug("Finished loading and merging grid files.", "total_steps", len(gridConfig.Steps), "total_resources", len(gridConfig.Resources))
	return gridConfig, nil
}

// LoadGridFile parses and decodes a single HCL grid file into a GridConfig struct.
func LoadGridFile(ctx context.Context, filePath string) (*schema.GridConfig, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Decoding grid file.", "path", filePath)
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL file %s: %s", filePath, diags.Error())
	}

	var config schema.GridConfig
	diags = gohcl.DecodeBody(file.Body, nil, &config)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode HCL file %s: %s", filePath, diags.Error())
	}

	logger.Debug("Successfully decoded grid file.", "path", filePath, "steps_found", len(config.Steps), "resources_found", len(config.Resources))
	return &config, nil
}

// LoadDefinitionFile parses and decodes a single HCL module manifest file.
func LoadDefinitionFile(ctx context.Context, filePath string) (*schema.DefinitionConfig, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Decoding module definition file.", "path", filePath)
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL module definition file %s: %s", filePath, diags.Error())
	}

	var config schema.DefinitionConfig
	diags = gohcl.DecodeBody(file.Body, nil, &config)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode HCL module definition file %s: %s", filePath, diags.Error())
	}

	logger.Debug("Successfully decoded module definition file.", "path", filePath)
	return &config, nil
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
		defConfig, err := LoadDefinitionFile(ctx, file)
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

// ResolveGridPath takes a path and returns a slice of all .hcl files found.
// If the path is a file, it returns a slice containing just that file.
// If the path is a directory, it recursively finds all .hcl files within it.
func ResolveGridPath(ctx context.Context, path string) ([]string, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Resolving grid path.", "path", path)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("grid path not found: %s", path)
	}
	if err != nil {
		return nil, fmt.Errorf("error accessing path %s: %w", path, err)
	}

	if info.IsDir() {
		logger.Debug("Path is a directory, scanning for HCL files.", "directory", path)
		return findHCLFilesRecursive(ctx, path)
	}

	logger.Debug("Path is a single file.", "file", path)
	// If it's a file, ensure it has the .hcl extension.
	if filepath.Ext(path) != ".hcl" {
		return nil, fmt.Errorf("specified file is not an .hcl file: %s", path)
	}
	return []string{path}, nil
}

// findHCLFilesRecursive scans a directory recursively for files with the .hcl extension.
func findHCLFilesRecursive(ctx context.Context, rootDir string) ([]string, error) {
	var hclFiles []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".hcl" {
			hclFiles = append(hclFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return hclFiles, nil
}
