// internal/nodeid/address.go
package nodeid

import (
	"fmt"
	"reflect"
	"strings"
)

// String serializes the Address into its canonical path string representation.
func (a *Address) String() string {
	if a == nil {
		return ""
	}

	var sb strings.Builder
	for i, segment := range a.Path {
		if i > 0 {
			sb.WriteRune('.')
		}
		sb.WriteString(segment.Name)
		if segment.Index != -1 {
			sb.WriteString(fmt.Sprintf("[%d]", segment.Index))
		}
	}

	return sb.String()
}

// Equal checks for deep equality between two Address pointers.
func (a *Address) Equal(other *Address) bool {
	if a == nil || other == nil {
		return a == other
	}
	return reflect.DeepEqual(a.Path, other.Path)
}
