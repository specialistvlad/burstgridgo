package engine

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/schema"
)

// DecodeGridFile parses and decodes a single HCL grid file into a GridConfig struct.
func DecodeGridFile(ctx context.Context, filePath string) (*schema.GridConfig, error) {
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

// DecodeDefinitionFile parses and decodes a single HCL module manifest file.
func DecodeDefinitionFile(ctx context.Context, filePath string) (*schema.DefinitionConfig, error) {
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
