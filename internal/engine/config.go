package engine

import "github.com/hashicorp/hcl/v2"

// Module represents a generic module block from the HCL file.
type Module struct {
	Name      string   `hcl:"name,label"`
	Runner    string   `hcl:"runner"`
	Body      hcl.Body `hcl:",remain"`
	DependsOn []string `hcl:"depends_on,optional"`
}

// Config represents the top-level structure of an HCL file.
type Config struct {
	Modules []*Module `hcl:"module,block"`
}
