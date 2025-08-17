package env_vars

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/specialistvlad/burstgridgo/internal/handlers"
)

// Module implements the registry.Module interface for this package.
type Module struct{}

// Deps is an empty struct because this runner does not use any resources.
type Deps struct{}

// Input defines the arguments that can be passed to the runner.
type Input struct {
	Include     []string          `bggo:"include,optional"`
	Required    []string          `bggo:"required,optional"`
	Defaults    map[string]string `bggo:"defaults,optional"`
	Prefix      string            `bggo:"prefix,optional"`
	StripPrefix bool              `bggo:"strip_prefix,optional"`
}

// Output defines the data structure returned by the runner.
type Output struct {
	Vars map[string]string `cty:"vars"`
}

// OnRunEnvVars is the handler for the 'env_vars' runner.
func OnRunEnvVars(ctx context.Context, deps *Deps, input *Input) (*Output, error) {
	candidateKeys := make(map[string]struct{})

	// Unify keys from 'include', 'defaults', AND 'required'. This ensures any
	// key that is explicitly mentioned is always considered.
	for _, key := range input.Include {
		candidateKeys[key] = struct{}{}
	}
	for key := range input.Defaults {
		candidateKeys[key] = struct{}{}
	}
	for _, key := range input.Required {
		candidateKeys[key] = struct{}{}
	}

	// If no explicit keys were provided, fall back to discovery via prefix or all.
	if len(candidateKeys) == 0 {
		if input.Prefix != "" {
			for _, e := range os.Environ() {
				if strings.HasPrefix(e, input.Prefix) {
					key := strings.SplitN(e, "=", 2)[0]
					candidateKeys[key] = struct{}{}
				}
			}
		} else {
			// Default behavior: consider all environment variables.
			for _, e := range os.Environ() {
				key := strings.SplitN(e, "=", 2)[0]
				candidateKeys[key] = struct{}{}
			}
		}
	}

	// Build the results map by checking the environment and then defaults.
	results := make(map[string]string)
	for key := range candidateKeys {
		value, found := os.LookupEnv(key)
		if !found {
			defaultValue, hasDefault := input.Defaults[key]
			if hasDefault {
				value = defaultValue
				found = true
			}
		}

		if found {
			resultKey := key
			if input.StripPrefix && input.Prefix != "" && strings.HasPrefix(key, input.Prefix) {
				resultKey = strings.TrimPrefix(key, input.Prefix)
			}
			results[resultKey] = value
		}
	}

	// Enforce that all required keys are present by checking the original sources.
	// This validation remains a crucial final check.
	if len(input.Required) > 0 {
		for _, reqKey := range input.Required {
			_, inEnv := os.LookupEnv(reqKey)
			_, inDefaults := input.Defaults[reqKey]
			if !inEnv && !inDefaults {
				return nil, fmt.Errorf("required environment variable '%s' is not set and has no default", reqKey)
			}
		}
	}

	return &Output{Vars: results}, nil
}

// Register registers the handler with the engine.
func (m *Module) Register(r *handlers.Handlers) {
	r.RegisterRunner("OnRunEnvVars", &handlers.RegisteredHandler{
		Input:     func() any { return new(Input) },
		InputType: reflect.TypeOf(Input{}),
		Deps:      func() any { return new(Deps) },
		Fn:        OnRunEnvVars,
	})
}
