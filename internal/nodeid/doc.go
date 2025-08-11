// internal/nodeid/doc.go

/*
Package nodeid provides a structured, type-safe representation for node
identifiers within the system, based on the canonical format `path`.

The format is defined as a dot-separated sequence of segments,
e.g., `a.b[0].c[1].d`.

This package enforces the identifier schema and centralizes all
formatting and parsing logic, improving maintainability and robustness.
*/
package nodeid
