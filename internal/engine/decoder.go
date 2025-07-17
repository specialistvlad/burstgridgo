package engine

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// DecodeHCLFile parses and decodes a single HCL file into a generic Config struct.
func DecodeHCLFile(filePath string) (*Config, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL file %s: %s", filePath, diags.Error())
	}

	var config Config
	diags = gohcl.DecodeBody(file.Body, nil, &config)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode HCL file %s: %s", filePath, diags.Error())
	}

	return &config, nil
}
