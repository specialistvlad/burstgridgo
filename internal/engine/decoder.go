package engine

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// DecodeGridFile parses and decodes a single HCL grid file into a GridConfig struct.
func DecodeGridFile(filePath string) (*GridConfig, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL file %s: %s", filePath, diags.Error())
	}

	var config GridConfig
	diags = gohcl.DecodeBody(file.Body, nil, &config)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode HCL file %s: %s", filePath, diags.Error())
	}

	return &config, nil
}

// DecodeDefinitionFile parses and decodes a single HCL runner manifest file.
func DecodeDefinitionFile(filePath string) (*DefinitionConfig, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL runner definition file %s: %s", filePath, diags.Error())
	}

	var config DefinitionConfig
	diags = gohcl.DecodeBody(file.Body, nil, &config)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode HCL runner definition file %s: %s", filePath, diags.Error())
	}

	return &config, nil
}
