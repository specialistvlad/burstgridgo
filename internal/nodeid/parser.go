// internal/nodeid/parser.go
package nodeid

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// segmentRegex is used to parse a single segment of a path, e.g., `name` or `name[1]`.
var segmentRegex = regexp.MustCompile(`^([a-zA-Z0-9_.-]+)(?:\[(\d+)\])?$`)

// isValidSegmentName checks for undesirable but technically valid names.
func isValidSegmentName(name string) bool {
	if name == "." || name == ".." || name == "-" {
		return false
	}
	return true
}

// Parse creates a new Address struct by parsing its canonical string representation.
func Parse(rawID string) (*Address, error) {
	if rawID == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	addr := &Address{}
	for _, segmentStr := range strings.Split(rawID, ".") {
		if segmentStr == "" {
			return nil, fmt.Errorf("identifier path contains empty segment")
		}

		matches := segmentRegex.FindStringSubmatch(segmentStr)
		if matches == nil {
			return nil, fmt.Errorf("invalid path segment format: %q", segmentStr)
		}

		name := matches[1]
		if !isValidSegmentName(name) {
			return nil, fmt.Errorf("invalid segment name: %q", name)
		}

		segment := NewPathSegment(name)
		if len(matches) > 2 && matches[2] != "" {
			index, err := strconv.Atoi(matches[2])
			if err != nil {
				// Unreachable due to regex `\d+`
				return nil, fmt.Errorf("internal error parsing index: %w", err)
			}
			segment.Index = index
		}
		addr.Path = append(addr.Path, segment)
	}

	return addr, nil
}
