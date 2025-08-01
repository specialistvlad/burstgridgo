// Package config defines the format-agnostic configuration model for the
// application, along with the core interfaces (Loader, Converter) for
// loading and interpreting configuration from various sources.
//
// The `config.Model` is the single source of truth for the `dag` and
// `executor` packages. Concrete implementations of the interfaces, such as
// for HCL, are provided in separate packages.
package config
