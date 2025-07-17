package engine

// Runner defines the interface that all modules must implement to be executable.
type Runner interface {
	Run(m Module) error
}

// Registry is a map that holds all the registered module runners,
// keyed by the runner name specified in the HCL config.
var Registry = make(map[string]Runner)
