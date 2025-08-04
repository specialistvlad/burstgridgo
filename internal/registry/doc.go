// Package registry provides the central "glue" for the module system.
//
// The Registry is responsible for storing mappings between the string identifiers
// used in manifests (e.g., "OnRunMyModule") and the actual compiled Go
// functions and types that implement the module's logic. It also holds the
// parsed, format-agnostic definitions from the manifests themselves.
//
// During application startup, the registry is populated and then validated to
// ensure that the Go code and the public-facing manifests are perfectly in
// sync, preventing a wide class of runtime errors.
package registry
