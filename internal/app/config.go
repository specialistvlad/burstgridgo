package app

import "errors"

// Config holds all the necessary configuration for an App instance to run.
type Config struct {
	GridPath    string // hcl files
	ModulesPath string // hcl files + handlers

	LogFormat       string
	LogLevel        string
	HealthcheckPort int
	WorkerCount     int
}

func NewConfig(cfg Config) (*Config, error) {
	if cfg.GridPath == "" {
		return nil, errors.New("GridPath is a required configuration field and cannot be empty")
	}

	// Future validations for other fields can be added here.
	// For example: checking if LogLevel is a valid value.

	return &cfg, nil
}
