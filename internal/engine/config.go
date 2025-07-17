package engine

import "github.com/hashicorp/hcl/v2"

// Module represents a generic module block from the HCL file.
// The engine only decodes the fields it needs to identify and dispatch the module.
// The Body field captures the rest of the block for the specific module to decode later.
type Module struct {
	Name   string   `hcl:"name,label"`
	Runner string   `hcl:"runner"`
	Body   hcl.Body `hcl:",remain"`
}

// Config represents the top-level structure of an HCL file.
type Config struct {
	Modules []*Module `hcl:"module,block"`
}
