// internal/nodeid/types.go
package nodeid

// PathSegment represents a single component of an address path, e.g., `name[index]`.
type PathSegment struct {
	Name  string
	Index int // -1 indicates no index is present.
}

// NewPathSegment creates a new path segment without an index.
func NewPathSegment(name string) PathSegment {
	return PathSegment{Name: name, Index: -1}
}

// NewPathSegmentWithIndex creates a new path segment that includes an index.
func NewPathSegmentWithIndex(name string, index int) PathSegment {
	return PathSegment{Name: name, Index: index}
}

// HasIndex returns true if the path segment has an explicit index.
func (ps PathSegment) HasIndex() bool {
	return ps.Index != -1
}

// Address is the structured representation of a unique node identifier.
// It is modeled as a path, broken into segments.
type Address struct {
	Path []PathSegment
	// Note: While the URI/URN model with queries was discussed, we are proceeding
	// with the simpler path-based model per the amended ADR. A Query field
	// could be added here in the future if needed.
}
